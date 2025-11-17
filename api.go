package hydrant

import (
	"context"
	"runtime"
	"time"

	"github.com/zeebo/mwc"

	"storj.io/hydrant/event"
)

func Log(ctx context.Context, message string, annotations ...event.Annotation) {
	GetSubmitter(ctx).Submit(event.Event{
		System: []event.Annotation{
			event.OfString("message", message),
			event.OfTimestamp("timestamp", time.Now()),
		},
		User: annotations,
	})
}

type Span struct {
	ctx    context.Context
	ev     event.Event
	sysBuf [8]event.Annotation
}

func (s *Span) Name() string     { x, _ := s.sysBuf[0].Value.AsString(); return x }
func (s *Span) Start() time.Time { x, _ := s.sysBuf[1].Value.AsTimestamp(); return x }
func (s *Span) Id() uint64       { x, _ := s.sysBuf[2].Value.AsUint(); return x }
func (s *Span) Parent() uint64   { x, _ := s.sysBuf[3].Value.AsUint(); return x }
func (s *Span) Task() uint64     { x, _ := s.sysBuf[4].Value.AsUint(); return x }

func (s *Span) Annotate(annotations ...event.Annotation) {
	s.ev.User = append(s.ev.User, annotations...)
}

func (s *Span) Done(err *error) {
	if s.ev.System == nil {
		now := time.Now()

		s.sysBuf[5] = event.OfTimestamp("timestamp", now)
		s.sysBuf[6] = event.OfDuration("duration", now.Sub(s.Start()))
		s.sysBuf[7] = event.OfBool("success", err == nil || *err == nil)

		s.ev.System = s.sysBuf[:]

		GetSubmitter(s.ctx).Submit(s.ev)
	}
}

func StartSpanNamed(ctx context.Context, name string, annotations ...event.Annotation) (context.Context, *Span) {
	var id, parent, task uint64
	for id == 0 {
		id = mwc.Uint64()
	}
	if cs, ok := ctx.(*contextSpan); ok {
		parent, task = (*Span)(cs).Id(), (*Span)(cs).Task()
	} else if cs, _ = ctx.Value((*contextSpan)(nil)).(*contextSpan); ok {
		parent, task = (*Span)(cs).Id(), (*Span)(cs).Task()
	}
	for task == 0 {
		task = mwc.Uint64()
	}

	s := &Span{
		ctx: ctx,
		ev:  event.Event{User: annotations},
		sysBuf: [8]event.Annotation{
			0: event.OfString("name", name),
			1: event.OfTimestamp("start", time.Now()),
			2: event.OfUint("span_id", id),
			3: event.OfUint("parent_id", parent),
			4: event.OfUint("task_id", task),
		},
	}

	return (*contextSpan)(s), s
}

func StartSpan(ctx context.Context, annotations ...event.Annotation) (context.Context, *Span) {
	var buf [1]uintptr
	runtime.Callers(2, buf[:])
	return StartSpanNamed(ctx, runtime.FuncForPC(buf[0]).Name(), annotations...)
}

type contextSpan Span

func (cs *contextSpan) Deadline() (time.Time, bool) { return cs.ctx.Deadline() }
func (cs *contextSpan) Done() <-chan struct{}       { return cs.ctx.Done() }
func (cs *contextSpan) Err() error                  { return cs.ctx.Err() }

func (cs *contextSpan) Value(key any) any {
	if key == (*contextSpan)(nil) {
		return cs
	}
	return cs.ctx.Value(key)
}
