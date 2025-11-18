package hydrant

import (
	"context"
	"testing"
)

func TestLog(t *testing.T) {
	var bs bufferSubmitter
	ctx := WithSubmitter(context.Background(), &bs)

	Log(ctx, "test message",
		String("user_key", "user_value"),
		Int("user_int", 42),
	)

	for _, ev := range bs {
		t.Logf("%+v", ev)
	}
}

//
// benchmarks
//

func BenchmarkLog(b *testing.B) {
	ctx := WithSubmitter(context.Background(), nullSubmitter{})

	b.Run("NoAnnotations", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			Log(ctx, "benchmark message")
		}
	})

	b.Run("WithAnnotations", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			Log(ctx, "benchmark message",
				String("user_key", "user_value"),
				Int("user_int", 42),
			)
		}
	})
}
