package hydrant

import (
	"context"
	"time"
)

var DefaultSubmitter Submitter

type submitterKeyType struct{}

type Submitter interface {
	Submit(context.Context, Event)
}

func GetSubmitter(ctx context.Context) (s Submitter) {
	if cs, ok := ctx.(*contextSpan); ok {
		return cs.sub
	}
	if s, ok := ctx.Value(submitterKeyType{}).(Submitter); ok {
		return s
	}
	return DefaultSubmitter
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
