package hydrant

import (
	"context"
	"runtime"
	"time"

	"github.com/zeebo/mwc"
)

func Log(ctx context.Context, message string, annotations ...Annotation) {
	if submitter := GetSubmitter(ctx); submitter != nil {
		var buf [1]uintptr
		runtime.Callers(2, buf[:])

		fn := runtime.FuncForPC(buf[0])
		file, line := fn.FileLine(buf[0])

		submitter.Submit(Event{
			System: []Annotation{
				String("message", message),
				Timestamp("timestamp", time.Now()),
				String("func", fn.Name()),
				String("file", file),
				Int("line", int64(line)),
			},
			User: annotations,
		})
	}
}

type Span struct {
	ctx context.Context
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

	if submitter := GetSubmitter(s.ctx); submitter != nil {
		submitter.Submit(s.ev)
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
