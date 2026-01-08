package submitters

import (
	"context"
	"net/http"
	"sync"

	"github.com/zeebo/hmux"

	"storj.io/hydrant"
	"storj.io/hydrant/filter"
)

var filterEvalPool = sync.Pool{New: func() any { return new(filter.EvalState) }}

type FilterSubmitter struct {
	fil *filter.Filter
	sub Submitter
}

func NewFilterSubmitter(
	fil *filter.Filter,
	sub Submitter,
) *FilterSubmitter {
	return &FilterSubmitter{
		fil: fil,
		sub: sub,
	}
}

func (f *FilterSubmitter) Children() []Submitter {
	return []Submitter{f.sub}
}

func (f *FilterSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	es := filterEvalPool.Get().(*filter.EvalState)
	if es.Evaluate(f.fil, ev) {
		f.sub.Submit(ctx, ev)
	}
	filterEvalPool.Put(es)
}

func (f *FilterSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree": constHandler(treeify(f)),
		"/sub":  f.sub.Handler(),
	}
}
