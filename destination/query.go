package destination

import (
	"context"
	"sync"
	"unique"

	"github.com/zeebo/errs/v2"

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
	group []hydrant.Annotation
	aggs  map[string]flathist.H
}

type Query struct {
	filter    *filter.Filter
	grouper   *group.Grouper
	submitter hydrant.Submitter
	store     *flathist.S
	aggs      []string

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
	switch {
	case len(cfg.GroupBy) > 0 && len(cfg.AggregateOver) > 0:
		return nil, errs.Errorf("cannot specify both group_by and aggregate_over")
	case len(cfg.GroupBy) > 0:
		// TODO: support more than raw keys
		var keys []string
		for _, expr := range cfg.GroupBy {
			keys = append(keys, expr.String())
		}
		grouper = group.NewGrouper(keys, false)
	case len(cfg.AggregateOver) > 0:
		// TODO: support more than raw keys
		var keys []string
		for _, expr := range cfg.AggregateOver {
			keys = append(keys, expr.String())
		}
		grouper = group.NewGrouper(keys, true)
	default:
		// no grouping
	}

	aggs := make([]string, len(cfg.Aggregates))
	for i, agg := range cfg.Aggregates {
		aggs[i] = agg.String()
	}

	return &Query{
		filter:    fil,
		grouper:   grouper,
		submitter: submitter,
		store:     store,
		aggs:      aggs,
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

	handle := q.grouper.Group(ev)
	agg := q.groups[handle]
	if agg == nil {
		agg = &aggregate{
			group: q.grouper.Annotations(ev),
			aggs:  make(map[string]flathist.H),
		}
		q.groups[handle] = agg
	}

	for _, key := range q.aggs {
		val, ok := lookup(key, ev.System)
		if !ok {
			val, ok = lookup(key, ev.User)
			if !ok {
				continue
			}
		}

		into, ok := agg.aggs[key]
		if !ok {
			into = q.store.New()
			agg.aggs[key] = into
		}

		switch val.Kind() {
		case value.KindInt:
			x, _ := val.Int()
			q.store.Observe(into, float32(x))
		case value.KindUint:
			x, _ := val.Uint()
			q.store.Observe(into, float32(x))
		case value.KindDuration:
			x, _ := val.Duration()
			q.store.Observe(into, float32(x.Seconds()))
		case value.KindFloat:
			x, _ := val.Float()
			q.store.Observe(into, float32(x))
		}
	}
}

func (q *Query) Flush(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, anns := range q.groups {
		user := make([]hydrant.Annotation, 0, len(q.aggs))
		for key, h := range anns.aggs {
			var w rwutils.W
			flathist.AppendTo(q.store, h, &w)
			user = append(user, hydrant.Bytes(key, w.Done().Prefix()))
		}
		q.submitter.Submit(ctx, hydrant.Event{
			System: anns.group,
			User:   user,
		})
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
