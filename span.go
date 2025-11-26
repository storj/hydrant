package hydrant

import (
	"context"
	"runtime"
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
	ctx context.Context
	sub Submitter
	ev  Event
	sys [sysIdxMax]Annotation
}

func (s *Span) Name() string         { x, _ := s.sys[sysIdxName].Value.String(); return x }
func (s *Span) StartTime() time.Time { x, _ := s.sys[sysIdxStartTime].Value.Timestamp(); return x }
func (s *Span) Id() uint64           { x, _ := s.sys[sysIdxSpanId].Value.Uint(); return x }
func (s *Span) Parent() uint64       { x, _ := s.sys[sysIdxParentId].Value.Uint(); return x }
func (s *Span) Task() uint64         { x, _ := s.sys[sysIdxTaskId].Value.Uint(); return x }

func (s *Span) Annotate(annotations ...Annotation) {
	s.ev.User = append(s.ev.User, annotations...)
}

func (s *Span) Done(err *error) {
	if s.ev.System != nil {
		return
	}

	now := time.Now()
	s.sys[sysIdxTimestamp] = Timestamp("timestamp", now)
	s.sys[sysIdxDuration] = Duration("duration", now.Sub(s.StartTime()))
	s.sys[sysIdxSuccess] = Bool("success", err == nil || *err == nil)
	s.ev.System = s.sys[:]

	if s.sub != nil {
		s.sub.Submit((*contextSpan)(s), s.ev)
	}
}

func StartSpan(ctx context.Context, annotations ...Annotation) (context.Context, *Span) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	return StartSpanNamed(ctx, runtime.FuncForPC(pcs[0]).Name(), annotations...)
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
		sys: [sysIdxMax]Annotation{
			sysIdxName:      String("name", name),
			sysIdxStartTime: Timestamp("start", time.Now()),
			sysIdxSpanId:    Identifier("span_id", id),
			sysIdxParentId:  Identifier("parent_id", parent),
			sysIdxTaskId:    Identifier("task_id", task),
		},
	}

	return (*contextSpan)(s), s
}
