package submitters

import (
	"context"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/zeebo/hmux"

	"storj.io/hydrant"
)

type MultiSubmitter struct {
	subs []Submitter
	live liveBuffer

	stats struct {
		received atomic.Uint64
	}
}

func NewMultiSubmitter(
	subs ...Submitter,
) *MultiSubmitter {
	return &MultiSubmitter{
		subs: subs,
		live: newLiveBuffer(),
	}
}

func (m *MultiSubmitter) Children() []Submitter {
	return m.subs
}

func (m *MultiSubmitter) ExtraData() any { return nil }

func (m *MultiSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	m.live.Record(ev)
	m.stats.received.Add(1)

	for _, sub := range m.subs {
		sub.Submit(ctx, ev)
	}
}

func (m *MultiSubmitter) Handler() http.Handler {
	subs := hmux.Dir{}
	for i, sub := range m.subs {
		subs["/"+strconv.Itoa(i)] = sub.Handler()
	}

	return hmux.Dir{
		"/tree": constJSONHandler(treeify(m)),
		"/live": m.live.Handler(),
		"/sub":  subs,
		"/stats": statsHandler(func() []stat {
			return []stat{
				{"received", m.stats.received.Load()},
			}
		}),
	}
}
