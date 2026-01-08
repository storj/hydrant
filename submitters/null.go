package submitters

import (
	"context"
	"net/http"

	"github.com/zeebo/hmux"

	"storj.io/hydrant"
)

type NullSubmitter struct{}

func NewNullSubmitter() *NullSubmitter {
	return &NullSubmitter{}
}

func (n *NullSubmitter) Children() []Submitter {
	return []Submitter{}
}

func (n *NullSubmitter) Submit(ctx context.Context, ev hydrant.Event) {}

func (n *NullSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree": constHandler(treeify(n)),
	}
}
