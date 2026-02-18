package hydrant

import (
	"context"
	"fmt"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/mwc"

	"storj.io/hydrant/internal/utils"
)

func TestSpan(t *testing.T) {
	var bs bufferSubmitter
	ctx := WithSubmitter(t.Context(), &bs)

	func() {
		assert.Equal(t, ActiveSpanCount(), 0)
		assert.Nil(t, GetSpan(ctx))

		ctx, span1 := StartSpanNamed(ctx, "span1",
			String("user_key", "user_value"),
			Int("user_int", 42),
		)
		defer span1.Done(nil)

		assert.Equal(t, GetSpan(ctx), span1)
		assert.Equal(t, span1.SpanId(), span1.ParentSpanId())

		ctx, span2 := StartSpanNamed(ctx, "span2",
			String("child_key", "child_value"),
		)
		defer span2.Done(nil)

		assert.Equal(t, ActiveSpanCount(), 2)
		for s := range IterateSpans {
			t.Logf("span: %p %v", s, s.Name())
		}

		assert.Equal(t, GetSpan(ctx), span2)
		assert.Equal(t, span2.ParentSpanId(), span1.SpanId())
		assert.Equal(t, span1.TraceId(), span2.TraceId())

		span2.Done(nil)

		t.Log("after done")
		assert.Equal(t, ActiveSpanCount(), 1)
		for s := range IterateSpans {
			t.Logf("span: %p %v", s, s.Name())
		}
	}()

	assert.Equal(t, ActiveSpanCount(), 0)
	for _, ev := range bs {
		t.Logf("%+v", ev)
	}
}

func TestIterateSpans(t *testing.T) {
	running := make(map[*Span]struct{})
	defer func() {
		for span := range running {
			span.Done(nil)
		}
		if !t.Failed() {
			assert.Equal(t, len(utils.Set(IterateSpans)), 0)
		}
	}()

	for range 1000 {
		switch mwc.Intn(10) {
		case 0, 1, 2:
			_, span := StartSpan(t.Context())
			running[span] = struct{}{}

		case 3, 4:
			for span := range running {
				span.Done(nil)
				delete(running, span)
				break
			}

		default:
			for span := range running {
				_, span := StartSpan(span.Context())
				running[span] = struct{}{}
				break
			}
		}

		assert.Equal(t, running, utils.Set(IterateSpans))
		assert.Equal(t, len(running), ActiveSpanCount())
	}
}

//
// benchmarks
//

func BenchmarkStartSpan(b *testing.B) {
	ctx := WithSubmitter(b.Context(), nullSubmitter{})

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
	ctx := WithSubmitter(b.Context(), nullSubmitter{})

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

func BenchmarkStartSpanParallel(b *testing.B) {
	ctx := WithSubmitter(b.Context(), nullSubmitter{})

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, span := StartSpan(ctx)
			span.Done(nil)
		}
	})
}
