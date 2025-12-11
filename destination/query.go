package destination

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
	"unique"

	"github.com/histdb/histdb/flathist"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/group"
	"storj.io/hydrant/value"
)

var evalPool = sync.Pool{New: func() any { return new(filter.EvalState) }}

type aggregate struct {
	event    hydrant.Event
	groupSet map[string]struct{}
	hists    map[string]*flathist.Histogram
	histOrd  []string
	excluded map[string]struct{}
}

type Query struct {
	filter    *filter.Filter
	grouper   *group.Grouper
	submitter hydrant.Submitter

	mu     sync.Mutex
	groups map[unique.Handle[string]]*aggregate
	epoch  time.Time
}

var _ hydrant.Submitter = (*Query)(nil)

func NewQuery(p *filter.Parser, submitter hydrant.Submitter, cfg config.Query) (*Query, error) {
	fil, err := p.Parse(cfg.Filter.String())
	if err != nil {
		return nil, err
	}

	var grouper *group.Grouper
	if len(cfg.GroupBy) > 0 {
		var keys []string
		for _, expr := range cfg.GroupBy {
			keys = append(keys, expr.String())
		}
		grouper = group.NewGrouper(keys)
	}

	return &Query{
		filter:    fil,
		grouper:   grouper,
		submitter: submitter,

		groups: make(map[unique.Handle[string]]*aggregate),
		epoch:  time.Now(),
	}, nil
}

func (q *Query) Submit(ctx context.Context, ev hydrant.Event) {
	es := evalPool.Get().(*filter.EvalState)
	defer evalPool.Put(es)

	if !es.Evaluate(q.filter, ev) {
		return
	}

	if q.grouper == nil {
		q.submitter.Submit(ctx, ev)
		return
	}

	// TODO: this mutex is too big. need an atomic updating aggregate
	q.mu.Lock()
	defer q.mu.Unlock()

	groupKey := q.grouper.Group(ev)

	agg := q.groups[groupKey]
	if agg == nil {
		group := q.grouper.Annotations(ev)
		groupSet := make(map[string]struct{}, len(group))
		for _, ann := range group {
			groupSet[ann.Key] = struct{}{}
		}

		agg = &aggregate{
			event:    group,
			groupSet: groupSet,
			hists:    make(map[string]*flathist.Histogram),
			excluded: make(map[string]struct{}),
		}

		q.groups[groupKey] = agg
	}

	for _, ann := range ev {
		if _, ok := agg.groupSet[ann.Key]; ok {
			continue
		}

		// if we got a full histogram, merge it into our existing one.
		if h, ok := ann.Value.Histogram(); ok {
			into, ok := agg.hists[ann.Key]
			if !ok {
				into = flathist.NewHistogram()
				agg.hists[ann.Key] = into
				agg.histOrd = append(agg.histOrd, ann.Key)
			}

			into.Merge(h)
			continue
		}

		datum, ok := observableValue(ann.Value)
		if !ok {
			agg.excluded[ann.Key] = struct{}{}
			continue
		}

		into, ok := agg.hists[ann.Key]
		if !ok {
			into = flathist.NewHistogram()
			agg.hists[ann.Key] = into
			agg.histOrd = append(agg.histOrd, ann.Key)
		}

		into.Observe(datum)
	}
}

func observableValue(v value.Value) (float32, bool) {
	switch v.Kind() {
	case value.KindInt:
		x, _ := v.Int()
		return float32(x), true
	case value.KindUint:
		x, _ := v.Uint()
		return float32(x), true
	case value.KindDuration:
		x, _ := v.Duration()
		return float32(x.Seconds()), true
	case value.KindTimestamp:
		x, _ := v.Timestamp()
		return float32(x.Unix()), true
	case value.KindFloat:
		x, _ := v.Float()
		return float32(x), true
	}
	return 0, false
}

func (q *Query) Flush(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()

	end := time.Now()

	for _, agg := range q.groups {
		agg.event = append(agg.event,
			hydrant.Timestamp("agg:start_time", q.epoch),
			hydrant.Timestamp("agg:end_time", end),
			hydrant.Duration("agg:duration", end.Sub(q.epoch)),
		)
		if len(agg.excluded) > 0 {
			agg.event = append(agg.event, hydrant.String("agg:excluded",
				strings.Join(slices.Collect(maps.Keys(agg.excluded)), ",")))
		}
		for _, key := range agg.histOrd {
			agg.event = append(agg.event, hydrant.Histogram(key, agg.hists[key]))
		}
		q.submitter.Submit(ctx, agg.event)
	}

	q.epoch = end
	clear(q.groups)
}
