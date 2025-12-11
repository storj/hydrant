package hydrant

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/zeebo/mwc"
)

const (
	sysIdxDuration = iota
	sysIdxName
	sysIdxParentId
	sysIdxSpanId
	sysIdxStartTime
	sysIdxSuccess
	sysIdxTaskId
	sysIdxTimestamp
	sysIdxMax
)

type Span struct {
	ctx  context.Context
	sub  Submitter // we do this so that finding the root submitter doesn't have to walk the full span chain
	ev   Event
	buf  [sysIdxMax]Annotation
	once sync.Once
}

func (s *Span) Name() string         { x, _ := s.buf[sysIdxName].Value.String(); return x }
func (s *Span) StartTime() time.Time { x, _ := s.buf[sysIdxStartTime].Value.Timestamp(); return x }
func (s *Span) Id() uint64           { x, _ := s.buf[sysIdxSpanId].Value.Uint(); return x }
func (s *Span) Parent() uint64       { x, _ := s.buf[sysIdxParentId].Value.Uint(); return x }
func (s *Span) Task() uint64         { x, _ := s.buf[sysIdxTaskId].Value.Uint(); return x }

func (s *Span) Annotate(annotations ...Annotation) {
	s.ev = append(s.ev, annotations...)
}

func (s *Span) Done(err *error) {
	s.once.Do(func() {
		if s.sub == nil {
			return
		}

		now := time.Now()

		s.ev[sysIdxTimestamp] = Timestamp("timestamp", now)
		s.ev[sysIdxDuration] = Duration("duration", now.Sub(s.StartTime()))
		s.ev[sysIdxSuccess] = Bool("success", err == nil || *err == nil)

		s.sub.Submit((*contextSpan)(s), s.ev)
	})
}

func StartSpan(ctx context.Context, annotations ...Annotation) (context.Context, *Span) {
	// TODO: this escapes pcs and CallersFrames allocates :(
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	frame, _ := runtime.CallersFrames(pcs[:]).Next()

	return StartSpanNamed(ctx, frame.Function, annotations...)
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
		buf: [sysIdxMax]Annotation{
			sysIdxName:      String("name", name),
			sysIdxStartTime: Timestamp("start", time.Now()),
			sysIdxSpanId:    Identifier("span_id", id),
			sysIdxParentId:  Identifier("parent_id", parent),
			sysIdxTaskId:    Identifier("task_id", task),
		},
	}
	s.ev = s.buf[:]

	return (*contextSpan)(s), s
}
