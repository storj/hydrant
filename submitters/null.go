package submitters

import (
	"context"
	"net/http"

	"github.com/zeebo/hmux"

	"storj.io/hydrant"
)

type NullSubmitter struct {
	live liveBuffer
}

func NewNullSubmitter() *NullSubmitter {
	return &NullSubmitter{live: newLiveBuffer()}
}

func (n *NullSubmitter) Children() []Submitter {
	return []Submitter{}
}

func (n *NullSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	n.live.Record(ev)
}

func (n *NullSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree": constJSONHandler(treeify(n)),
		"/live": n.live.Handler(),
	}
}
