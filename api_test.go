package hydrant

import (
	"context"
	"fmt"
	"testing"
)

func TestLog(t *testing.T) {
	var bs bufferSubmitter
	ctx := WithSubmitter(context.Background(), &bs)

	Log(ctx, "test message",
		String("user_key", "user_value"),
		Int("user_int", 42),
	)

	t.Logf("%+v", bs)
}

func TestSpan(t *testing.T) {
	var bs bufferSubmitter
	ctx := WithSubmitter(context.Background(), &bs)

	ctx, span := StartSpan(ctx,
		String("user_key", "user_value"),
		Int("user_int", 42),
	)
	span.Done(nil)

	t.Logf("%+v", bs)
}

func BenchmarkLog(b *testing.B) {
	ctx := WithSubmitter(context.Background(), nullSubmitter{})

	b.ReportAllocs()
	for b.Loop() {
		Log(ctx, "benchmark message")
	}
}

func BenchmarkStartSpan(b *testing.B) {
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
	ctx, span := StartSpan(ctx)
	defer span.Done(&err)

	if depth > 0 {
		return recursiveSpan(ctx, depth-1)
	}
	return ctx.Err()
}

func BenchmarkStartSpanNamed(b *testing.B) {
	ctx := WithSubmitter(context.Background(), nullSubmitter{})

	for depth := range 5 {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = recursiveSpanNamed(ctx, depth)
			}
		})
	}
}

func recursiveSpanNamed(ctx context.Context, depth int) (err error) {
	ctx, span := StartSpanNamed(ctx, "benchmark")
	defer span.Done(&err)

	if depth > 0 {
		return recursiveSpanNamed(ctx, depth-1)
	}
	return ctx.Err()
}

type nullSubmitter struct{}

func (ns nullSubmitter) Submit(ev Event) {}

type bufferSubmitter []Event

func (bs *bufferSubmitter) Submit(ev Event) { *bs = append(*bs, ev) }
