package submitters

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/zeebo/hmux"

	"storj.io/hydrant"
)

type NullSubmitter struct {
	live liveBuffer

	stats struct {
		received atomic.Uint64
	}
}

func NewNullSubmitter() *NullSubmitter {
	return &NullSubmitter{live: newLiveBuffer()}
}

func (n *NullSubmitter) Children() []Submitter {
	return []Submitter{}
}

func (n *NullSubmitter) ExtraData() any { return nil }

func (n *NullSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	n.live.Record(ev)
	n.stats.received.Add(1)
}

func (n *NullSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree": constJSONHandler(treeify(n)),
		"/live": n.live.Handler(),
		"/stats": statsHandler(func() []stat {
			return []stat{
				{"received", n.stats.received.Load()},
			}
		}),
	}
}
