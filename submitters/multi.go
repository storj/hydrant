package submitters

import (
	"context"
	"net/http"
	"strconv"

	"github.com/zeebo/hmux"

	"storj.io/hydrant"
)

type MultiSubmitter struct {
	subs []Submitter
}

func NewMultiSubmitter(
	subs ...Submitter,
) *MultiSubmitter {
	return &MultiSubmitter{
		subs: subs,
	}
}

func (m *MultiSubmitter) Children() []Submitter {
	return m.subs
}

func (m *MultiSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
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
		"/sub":  subs,
	}
}
