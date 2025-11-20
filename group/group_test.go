package group

import (
	"runtime"
	"slices"
	"strings"
	"testing"

	"storj.io/hydrant"
)

func TestGrouperInclude(t *testing.T) {
	g := NewGrouper([]string{"key1", "key2"}, false)

	h1 := g.Group(hydrant.Event{
		System: []hydrant.Annotation{
			hydrant.String("key1", "value1"),
			hydrant.Int("key3", 42),
		},
		User: []hydrant.Annotation{
			hydrant.String("key2", "value2"),
		},
	})

	h2 := g.Group(hydrant.Event{
		System: []hydrant.Annotation{
			hydrant.String("key1", "value1"),
		},
		User: []hydrant.Annotation{
			hydrant.String("key2", "value2"),
		},
	})

	if h1 != h2 {
		t.Errorf("expected h1 and h2 to be equal: %v != %v", h1, h2)
	}

	t.Logf("%x", h1.Value())
	t.Logf("%x", h2.Value())
}

//
// benchmarks
//

func BenchmarkGrouper_Include_SpanByNameSuccess(b *testing.B) {
	var ev hydrant.Event
	ctx := hydrant.WithSubmitter(b.Context(), (*eventSubmitter)(&ev))
	func() { _, span := hydrant.StartSpan(ctx); span.Done(nil) }()

	// TODO: there should be some sort of event processing in the pipeline
	// that sorts and stuff.
	slices.SortStableFunc(ev.System, compareAnnotations)
	slices.SortStableFunc(ev.User, compareAnnotations)

	g := NewGrouper([]string{"name", "success"}, false)
	h := g.Group(ev)

	b.ReportAllocs()
	for b.Loop() {
		h = g.Group(ev)
	}

	b.Logf("%x", h.Value())
	runtime.KeepAlive(h)
}

type eventSubmitter hydrant.Event

func (es *eventSubmitter) Submit(ev hydrant.Event) {
	*es = eventSubmitter(ev)
}

func compareAnnotations(a, b hydrant.Annotation) int {
	return strings.Compare(a.Key, b.Key)
}
