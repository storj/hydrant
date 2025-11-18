package hydrant

import (
	"context"
	"runtime"
	"time"

	"github.com/zeebo/mwc"
)

type Span struct {
	ctx context.Context
	sub Submitter
	ev  Event
	sys [8]Annotation
}

func (s *Span) Name() string     { x, _ := s.sys[0].Value.String(); return x }
func (s *Span) Start() time.Time { x, _ := s.sys[1].Value.Timestamp(); return x }
func (s *Span) Id() uint64       { x, _ := s.sys[2].Value.Uint(); return x }
func (s *Span) Parent() uint64   { x, _ := s.sys[3].Value.Uint(); return x }
func (s *Span) Task() uint64     { x, _ := s.sys[4].Value.Uint(); return x }

func (s *Span) Annotate(annotations ...Annotation) {
	s.ev.User = append(s.ev.User, annotations...)
}

func (s *Span) Done(err *error) {
	if s.ev.System != nil {
		return
	}

	now := time.Now()
	s.sys[5] = Timestamp("timestamp", now)
	s.sys[6] = Duration("duration", now.Sub(s.Start()))
	s.sys[7] = Bool("success", err == nil || *err == nil)
	s.ev.System = s.sys[:]

	if s.sub != nil {
		s.sub.Submit(s.ev)
	}
}

func StartSpan(ctx context.Context, annotations ...Annotation) (context.Context, *Span) {
	var buf [1]uintptr
	runtime.Callers(2, buf[:])
	return StartSpanNamed(ctx, runtime.FuncForPC(buf[0]).Name(), annotations...)
}

func StartSpanNamed(ctx context.Context, name string, annotations ...Annotation) (context.Context, *Span) {
	var parent, task uint64
	if span := GetSpan(ctx); span != nil {
		parent, task = span.Id(), span.Task()
	}
	return StartRemoteSpanNamed(ctx, name, parent, task, annotations...)
}

func StartRemoteSpanNamed(ctx context.Context, name string, parent, task uint64, annotations ...Annotation) (context.Context, *Span) {
	var id uint64
	for id == 0 {
		id = mwc.Uint64()
	}
	for task == 0 {
		task = mwc.Uint64()
	}
	if parent == 0 {
		parent = task
	}

	s := &Span{
		ctx: ctx,
		sub: GetSubmitter(ctx),
		ev:  Event{User: annotations},
		sys: [8]Annotation{
			0: String("name", name),
			1: Timestamp("start", time.Now()),
			2: Uint("span_id", id),
			3: Uint("parent_id", parent),
			4: Uint("task_id", task),
		},
	}

	return (*contextSpan)(s), s
}
