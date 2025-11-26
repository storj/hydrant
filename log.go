package hydrant

import (
	"context"
	"runtime"
	"time"
)

func Log(ctx context.Context, message string, annotations ...Annotation) {
	if submitter := GetSubmitter(ctx); submitter != nil {
		var pcs [1]uintptr
		runtime.Callers(2, pcs[:])

		fn := runtime.FuncForPC(pcs[0])
		file, line := fn.FileLine(pcs[0])

		sys := (&[7]Annotation{
			0: String("file", file),
			1: String("func", fn.Name()),
			2: Int("line", int64(line)),
			3: String("message", message),
		})[:4]

		if span := GetSpan(ctx); span != nil {
			sys = append(sys,
				Identifier("span_id", span.Id()),
				Identifier("task_id", span.Task()),
			)
		}

		sys = append(sys, Timestamp("timestamp", time.Now()))

		submitter.Submit(ctx, Event{
			System: sys,
			User:   annotations,
		})
	}
}
