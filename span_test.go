package hydrant

import (
	"context"
	"fmt"
	"testing"

	"github.com/zeebo/assert"
)

func TestSpan(t *testing.T) {
	var bs bufferSubmitter
	ctx := WithSubmitter(context.Background(), &bs)

	func() {
		assert.Nil(t, GetSpan(ctx))

		ctx, span1 := StartSpan(ctx,
			String("user_key", "user_value"),
			Int("user_int", 42),
		)
		defer span1.Done(nil)

		assert.Equal(t, GetSpan(ctx), span1)
		assert.Equal(t, span1.Task(), span1.Parent())

		ctx, span2 := StartSpan(ctx,
			String("child_key", "child_value"),
		)
		defer span2.Done(nil)

		assert.Equal(t, GetSpan(ctx), span2)
		assert.Equal(t, span2.Parent(), span1.Id())
		assert.Equal(t, span1.Task(), span2.Task())
	}()

	for _, ev := range bs {
		t.Logf("%+v", ev)
	}
}

//
// benchmarks
//

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
