package hydrant

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkSpanNamed(b *testing.B) {
	ctx := WithSubmitter(context.Background(), nullSubmitter{})

	for depth := range 5 {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = recursiveSpan(ctx, depth)
			}
		})
	}
}

func recursiveSpan(ctx context.Context, depth int) (err error) {
	ctx, span := StartSpanNamed(ctx, "benchmark")
	defer span.Done(&err)

	if depth > 0 {
		return recursiveSpan(ctx, depth-1)
	}
	return ctx.Err()
}

type nullSubmitter struct{}

func (ns nullSubmitter) Submit(ev Event) {}
