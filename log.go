package hydrant

import (
	"context"
	"runtime"
	"time"
)

func Log(ctx context.Context, message string, annotations ...Annotation) {
	// TODO: this escapes pcs and CallersFrames allocates but every other option either depends on
	// internals or is broken :(
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	frame, _ := runtime.CallersFrames(pcs[:]).Next()

	ev := Event((&[7]Annotation{
		0: String("file", frame.File),
		1: String("func", frame.Function),
		2: Int("line", int64(frame.Line)),
		3: String("message", message),
	})[:4])

	if span := GetSpan(ctx); span != nil {
		ev = append(ev,
			SpanId("span_id", span.SpanId()),
			TraceId("trace_id", span.TraceId()),
		)
	}

	ev = append(ev, Timestamp("timestamp", time.Now()))
	ev = append(ev, annotations...)

	GetSubmitter(ctx).Submit(ctx, ev)
}
