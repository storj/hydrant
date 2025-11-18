package hydrant

import (
	"context"
	"time"
)

type submitterKeyType struct{}

type Submitter interface {
	Submit(Event)
}

func GetSubmitter(ctx context.Context) (s Submitter) {
	switch ctx := ctx.(type) {
	case *contextSpan:
		s = ctx.sub
	default:
		s, _ = ctx.Value(submitterKeyType{}).(Submitter)
	}
	return s
}

func WithSubmitter(ctx context.Context, s Submitter) context.Context {
	return context.WithValue(ctx, submitterKeyType{}, s)
}

func GetSpan(ctx context.Context) (s *Span) {
	switch ctx := ctx.(type) {
	case *contextSpan:
		s = (*Span)(ctx)
	default:
		s, _ = ctx.Value((*Span)(nil)).(*Span)
	}
	return s
}

type contextSpan Span

func (cs *contextSpan) Deadline() (time.Time, bool) { return cs.ctx.Deadline() }
func (cs *contextSpan) Done() <-chan struct{}       { return cs.ctx.Done() }
func (cs *contextSpan) Err() error                  { return cs.ctx.Err() }

func (cs *contextSpan) Value(key any) any {
	switch key {
	case (*Span)(nil):
		return (*Span)(cs)
	case submitterKeyType{}:
		return cs.sub
	}
	return cs.ctx.Value(key)
}
