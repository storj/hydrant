package hydrant

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
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
	sysIdxTraceId
	sysIdxTimestamp
	sysIdxMax
)

type Span struct {
	ctx  context.Context
	sub  Submitter // we do this so that finding the root submitter doesn't have to walk the full span chain
	ev   Event
	prev atomic.Pointer[Span]
	next atomic.Pointer[Span]
	root *listRoot
	buf  [sysIdxMax]Annotation
	mu   sync.Mutex
	done atomic.Bool
}

func (s *Span) Context() context.Context       { return (*contextSpan)(s) }
func (s *Span) ParentContext() context.Context { return s.ctx }
func (s *Span) IsDone() bool                   { return s.done.Load() }

func (s *Span) Name() string          { x, _ := s.buf[sysIdxName].Value.String(); return x }
func (s *Span) StartTime() time.Time  { x, _ := s.buf[sysIdxStartTime].Value.Timestamp(); return x }
func (s *Span) SpanId() [8]byte       { x, _ := s.buf[sysIdxSpanId].Value.SpanId(); return x }
func (s *Span) ParentSpanId() [8]byte { x, _ := s.buf[sysIdxParentId].Value.SpanId(); return x }
func (s *Span) TraceId() [16]byte     { x, _ := s.buf[sysIdxTraceId].Value.TraceId(); return x }

func (s *Span) Annotations() []Annotation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ev[sysIdxMax:]
}

func (s *Span) Annotate(annotations ...Annotation) {
	s.mu.Lock()
	s.ev = append(s.ev, annotations...)
	s.mu.Unlock()
}

func (s *Span) Done(err *error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.done.CompareAndSwap(false, true) {
		return
	}
	popSpan(s)

	now := time.Now()

	s.ev[sysIdxTimestamp] = Timestamp("timestamp", now)
	s.ev[sysIdxDuration] = Duration("duration", now.Sub(s.StartTime()))
	s.ev[sysIdxSuccess] = Bool("success", err == nil || *err == nil)

	s.sub.Submit((*contextSpan)(s), s.ev)
}

func StartSpan(ctx context.Context, annotations ...Annotation) (context.Context, *Span) {
	// TODO: this escapes pcs and CallersFrames allocates but every other option either depends on
	// internals or is broken :(
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	frame, _ := runtime.CallersFrames(pcs[:]).Next()

	return StartSpanNamed(ctx, frame.Function, annotations...)
}

func StartSpanNamed(ctx context.Context, name string, annotations ...Annotation) (context.Context, *Span) {
	if span := GetSpan(ctx); span != nil {
		return startChildSpanNamed(ctx, name, span, annotations...)
	}
	return StartRemoteSpanNamed(ctx, name, [8]byte{}, [16]byte{}, annotations...)
}

func startChildSpanNamed(ctx context.Context, name string, parent *Span, annotations ...Annotation) (context.Context, *Span) {
	var id [8]byte
	for id == [8]byte{} {
		_, _ = mwc.Read(id[:])
	}

	return createSpan(ctx, name, id, parent.SpanId(), parent.buf[sysIdxTraceId], annotations...)
}

func StartRemoteSpanNamed(ctx context.Context, name string, parentId [8]byte, traceId [16]byte, annotations ...Annotation) (context.Context, *Span) {
	var spanId [8]byte
	for spanId == [8]byte{} {
		_, _ = mwc.Read(spanId[:])
	}
	for traceId == [16]byte{} {
		_, _ = mwc.Read(traceId[:])
	}
	if parentId == [8]byte{} {
		parentId = spanId
	}

	return createSpan(ctx, name, spanId, parentId, TraceId("trace_id", traceId), annotations...)
}

func createSpan(ctx context.Context, name string, spanId, parentId [8]byte, traceId Annotation, annotations ...Annotation) (context.Context, *Span) {
	s := &Span{
		ctx: ctx,
		sub: GetSubmitter(ctx),
		buf: [sysIdxMax]Annotation{
			sysIdxName:      String("name", name),
			sysIdxStartTime: Timestamp("start", time.Now()),
			sysIdxSpanId:    SpanId("span_id", spanId),
			sysIdxParentId:  SpanId("parent_id", parentId),
			sysIdxTraceId:   traceId,
		},
	}
	s.ev = append(s.buf[:], annotations...)

	s.root = pushSpan(s)

	return (*contextSpan)(s), s
}
