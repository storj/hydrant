package hydrant

import (
	"context"
	"runtime"
	"time"
)

func Log(ctx context.Context, message string, annotations ...Annotation) {
	if submitter := GetSubmitter(ctx); submitter != nil {
		var buf [1]uintptr
		runtime.Callers(2, buf[:])

		fn := runtime.FuncForPC(buf[0])
		file, line := fn.FileLine(buf[0])

		submitter.Submit(ctx, Event{
			System: []Annotation{
				String("file", file),
				String("func", fn.Name()),
				Int("line", int64(line)),
				String("message", message),
				Timestamp("timestamp", time.Now()),
			},
			User: annotations,
		})
	}
}
