package submitters

import (
	"context"
	"net/http"

	"storj.io/hydrant"
)

type lateSubmitter struct {
	sub Submitter
}

func newLateSubmitter() *lateSubmitter {
	return &lateSubmitter{}
}

func (l *lateSubmitter) SetSubmitter(sub Submitter) {
	l.sub = sub
}

func (l *lateSubmitter) Children() []Submitter {
	return l.sub.Children()
}

func (l *lateSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	l.sub.Submit(ctx, ev)
}

func (l *lateSubmitter) Handler() http.Handler {
	return l.sub.Handler()
}
