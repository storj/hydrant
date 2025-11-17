package hydrant

import (
	"context"
	"fmt"
	"hydrant/event"
	"runtime"
	"testing"
)

func BenchmarkCallers(b *testing.B) {
	for depth := range 10 {
		b.Run(fmt.Sprintf("depth=%d", depth+3), func(b *testing.B) {
			b.ReportAllocs()
			deep(depth, func() {
				var buf [1]uintptr
				for b.Loop() {
					_ = runtime.Callers(1+depth+2, buf[:])
				}
			})
		})
	}
}

func deep(x int, cb func()) {
	if x > 0 {
		deep(x-1, cb)
	} else {
		cb()
	}
}

func BenchmarkSpan(b *testing.B) {
	b.ReportAllocs()

	ctx := WithSubmitter(context.Background(), nullSubmitter{})

	for b.Loop() {
		thisIsTheName(ctx)
	}
}

func thisIsTheName(ctx context.Context) (err error) {
	ctx, span := StartSpanNamed(ctx, "foo")
	defer span.Done(&err)

	return ctx.Err()
}

type nullSubmitter struct{}

func (nullSubmitter) Submit(event event.Event) {}
