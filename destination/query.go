package destination

import (
	"context"
	"sync"
	"unique"

	"github.com/zeebo/errs"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/group"
	"storj.io/hydrant/value"
)

var evalPool = sync.Pool{New: func() any { return new(filter.EvalState) }}

type aggregate struct {
	group []hydrant.Annotation
	anns  []hydrant.Annotation
}

type Query struct {
	filter    *filter.Filter
	grouper   *group.Grouper
	submitter hydrant.Submitter
	aggs      []string

	mu     sync.Mutex
	groups map[unique.Handle[string]]*aggregate
}

var _ hydrant.Submitter = (*Query)(nil)

func NewQuery(p *filter.Parser, submitter hydrant.Submitter, cfg config.Query) (*Query, error) {
	fil, err := p.Parse(cfg.Filter.String())
	if err != nil {
		return nil, err
	}

	var grouper *group.Grouper
	switch {
	case len(cfg.GroupBy) > 0 && len(cfg.AggregateOver) > 0:
		return nil, errs.New("cannot specify both group_by and aggregate_over")
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
		agg = &aggregate{group: q.grouper.Annotations(ev)}
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

		into, ok := lookup(key, agg.anns)
		if !ok {
			agg.anns = append(agg.anns, hydrant.Annotation{Key: key, Value: *val})
			continue
		}

		if into.Kind() != val.Kind() {
			continue
		}

		switch into.Kind() {
		case value.KindEmpty, value.KindString, value.KindBytes, value.KindBool, value.KindTimestamp:
		case value.KindInt:
			x, _ := into.Int()
			y, _ := val.Int()
			*into = value.Int(x + y)
		case value.KindUint:
			x, _ := into.Uint()
			y, _ := val.Uint()
			*into = value.Uint(x + y)
		case value.KindDuration:
			x, _ := into.Duration()
			y, _ := val.Duration()
			*into = value.Duration(x + y)
		case value.KindFloat:
			x, _ := into.Float()
			y, _ := val.Float()
			*into = value.Float(x + y)
		}
	}
}

func (q *Query) Flush(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, anns := range q.groups {
		q.submitter.Submit(ctx, hydrant.Event{
			System: anns.group,
			User:   anns.anns,
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
