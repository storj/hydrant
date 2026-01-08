package group

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"storj.io/hydrant"
)

func TestGrouperInclude(t *testing.T) {
	g := NewGrouper([]string{"key1", "key2"})

	h1 := g.Group(hydrant.Event{
		hydrant.String("key1", "value1"),
		hydrant.Int("key3", 42),
		hydrant.String("key2", "value2"),
	})

	h2 := g.Group(hydrant.Event{
		hydrant.String("key1", "value1"),
		hydrant.String("key2", "value2"),
	})

	if h1 != h2 {
		t.Errorf("expected h1 and h2 to be equal: %v != %v", h1, h2)
	}

	t.Logf("%x", h1.Value())
	t.Logf("%x", h2.Value())

	t.Log(g.Annotations(hydrant.Event{
		hydrant.String("key1", "value1"),
		hydrant.Int("key3", 42),
		hydrant.String("key2", "value2"),
	}))
}

//
// benchmarks
//

func BenchmarkGrouper_Include_SpanByNameSuccess(b *testing.B) {
	var ev hydrant.Event
	ctx := hydrant.WithSubmitter(b.Context(), (*eventSubmitter)(&ev))
	func() { _, span := hydrant.StartSpan(ctx); span.Done(nil) }()

	g := NewGrouper([]string{"name", "success"})
	h := g.Group(ev)

	b.ReportAllocs()
	for b.Loop() {
		h = g.Group(ev)
	}

	b.Logf("%x", h.Value())
	runtime.KeepAlive(h)
}

type eventSubmitter hydrant.Event

func (es *eventSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	*es = eventSubmitter(ev)
}

func compareAnnotations(a, b hydrant.Annotation) int {
	return strings.Compare(a.Key, b.Key)
}
