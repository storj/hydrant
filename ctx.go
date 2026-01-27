package hydrant

import (
	"context"
	"sync/atomic"
	"time"
)

type noopSubmitter struct{}

func (n noopSubmitter) Submit(ctx context.Context, ev Event) {}

var defaultSubmitter = func() *atomic.Pointer[Submitter] {
	v := new(atomic.Pointer[Submitter])
	var sub Submitter = new(noopSubmitter)
	v.Store(&sub)
	return v
}()

func SetDefaultSubmitter(s Submitter) {
	if s == nil {
		s = noopSubmitter{}
	}
	defaultSubmitter.Store(&s)
}

func GetDefaultSubmitter() Submitter {
	return *defaultSubmitter.Load()
}

type submitterKeyType struct{}

type Submitter interface {
	Submit(context.Context, Event)
}

func GetSubmitter(ctx context.Context) (s Submitter) {
	if cs, ok := ctx.(*contextSpan); ok {
		s = cs.sub
	} else if ctx != nil {
		s, _ = ctx.Value(submitterKeyType{}).(Submitter)
	}
	if s == nil {
		s = GetDefaultSubmitter()
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
