package hydrant

import (
	"context"
	"runtime"
	"time"

	"github.com/zeebo/mwc"
)

func Log(ctx context.Context, message string, annotations ...Annotation) {
	GetSubmitter(ctx).Submit(Event{
		System: []Annotation{
			String("message", message),
			Timestamp("timestamp", time.Now()),
		},
		User: annotations,
	})
}

type Span struct {
	ctx    context.Context
	ev     Event
	sysBuf [8]Annotation
}

func (s *Span) Name() string     { x, _ := s.sysBuf[0].Value.String(); return x }
func (s *Span) Start() time.Time { x, _ := s.sysBuf[1].Value.Timestamp(); return x }
func (s *Span) Id() uint64       { x, _ := s.sysBuf[2].Value.Uint(); return x }
func (s *Span) Parent() uint64   { x, _ := s.sysBuf[3].Value.Uint(); return x }
func (s *Span) Task() uint64     { x, _ := s.sysBuf[4].Value.Uint(); return x }

func (s *Span) Annotate(annotations ...Annotation) {
	s.ev.User = append(s.ev.User, annotations...)
}

func (s *Span) Done(err *error) {
	if s.ev.System == nil {
		now := time.Now()

		s.sysBuf[5] = Timestamp("timestamp", now)
		s.sysBuf[6] = Duration("duration", now.Sub(s.Start()))
		s.sysBuf[7] = Bool("success", err == nil || *err == nil)

		s.ev.System = s.sysBuf[:]

		GetSubmitter(s.ctx).Submit(s.ev)
	}
}

func StartSpanNamed(ctx context.Context, name string, annotations ...Annotation) (context.Context, *Span) {
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
		ev:  Event{User: annotations},
		sysBuf: [8]Annotation{
			0: String("name", name),
			1: Timestamp("start", time.Now()),
			2: Uint("span_id", id),
			3: Uint("parent_id", parent),
			4: Uint("task_id", task),
		},
	}

	return (*contextSpan)(s), s
}

func StartSpan(ctx context.Context, annotations ...Annotation) (context.Context, *Span) {
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
