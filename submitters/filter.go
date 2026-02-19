package submitters

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/zeebo/hmux"

	"storj.io/hydrant"
	"storj.io/hydrant/filter"
)

var filterEvalPool = sync.Pool{New: func() any { return new(filter.EvalState) }}

type FilterSubmitter struct {
	fil  *filter.Filter
	sub  Submitter
	live liveBuffer

	stats struct {
		received atomic.Uint64
		passed   atomic.Uint64
		filtered atomic.Uint64
	}
}

func NewFilterSubmitter(
	fil *filter.Filter,
	sub Submitter,
) *FilterSubmitter {
	return &FilterSubmitter{
		fil:  fil,
		sub:  sub,
		live: newLiveBuffer(),
	}
}

func (f *FilterSubmitter) Children() []Submitter {
	return []Submitter{f.sub}
}

func (f *FilterSubmitter) ExtraData() any {
	return map[string]string{"filter": f.fil.Filter()}
}

func (f *FilterSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	f.live.Record(ev)
	f.stats.received.Add(1)

	es := filterEvalPool.Get().(*filter.EvalState)
	if es.Evaluate(f.fil, ev) {
		f.stats.passed.Add(1)
		f.sub.Submit(ctx, ev)
	} else {
		f.stats.filtered.Add(1)
	}
	filterEvalPool.Put(es)
}

func (f *FilterSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree": constJSONHandler(treeify(f)),
		"/live": f.live.Handler(),
		"/sub":  f.sub.Handler(),
		"/stats": statsHandler(func() []stat {
			return []stat{
				{"received", f.stats.received.Load()},
				{"passed", f.stats.passed.Load()},
				{"filtered", f.stats.filtered.Load()},
			}
		}),
	}
}
