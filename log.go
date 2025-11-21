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
