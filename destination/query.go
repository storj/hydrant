package destination

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"unique"

	"github.com/histdb/histdb/flathist"
	"github.com/histdb/histdb/rwutils"

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
	dists    map[string]flathist.H
	excluded map[string]struct{}
}

type Query struct {
	filter    *filter.Filter
	grouper   *group.Grouper
	submitter hydrant.Submitter
	store     *flathist.S

	mu     sync.Mutex
	groups map[unique.Handle[string]]*aggregate
}

var _ hydrant.Submitter = (*Query)(nil)

func NewQuery(p *filter.Parser, submitter hydrant.Submitter, store *flathist.S, cfg config.Query) (*Query, error) {
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
		store:     store,
		groups:    make(map[unique.Handle[string]]*aggregate),
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
			dists:    make(map[string]flathist.H),
			excluded: make(map[string]struct{}),
		}

		q.groups[groupKey] = agg
	}

	for _, ann := range ev {
		if _, ok := agg.groupSet[ann.Key]; ok {
			continue
		}

		datum, ok := distributionize(ann.Value)
		if !ok {
			agg.excluded[ann.Key] = struct{}{}
			continue
		}

		into, ok := agg.dists[ann.Key]
		if !ok {
			into = q.store.New()
			agg.dists[ann.Key] = into
		}

		q.store.Observe(into, datum)
	}
}

func distributionize(v value.Value) (float32, bool) {
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

	for _, agg := range q.groups {
		if len(agg.excluded) > 0 {
			agg.event = append(agg.event, hydrant.String("excluded",
				strings.Join(slices.Collect(maps.Keys(agg.excluded)), ", ")))
		}
		for key, h := range agg.dists {
			var w rwutils.W
			flathist.AppendTo(q.store, h, &w)
			agg.event = append(agg.event, hydrant.Bytes(key, w.Done().Prefix()))
		}
		q.submitter.Submit(ctx, agg.event)
	}
	clear(q.groups)
}

func lookup(key string, anns []hydrant.Annotation) (*value.Value, bool) {
	for i := len(anns) - 1; i >= 0; i-- {
		if anns[i].Key == key {
			return &anns[i].Value, true
		}
	}
	return nil, false
}
