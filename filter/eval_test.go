package filter

import (
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

func BenchmarkEval(b *testing.B) {
	var p Parser
	SetBuiltins(&p)

	filter, err := p.Parse(`
		   equal(key(foo), bar)
		&& exists(test)
		&& less(rand(), 1)
		&& less(key(dur), 2m)
		&& true()
	`)
	assert.NoError(b, err)

	ev := hydrant.Event{
		System: []hydrant.Annotation{
			{"foo", value.String("bar")},
			{"dur", value.Duration(time.Minute)},
		},
		User: []hydrant.Annotation{
			{"test", value.Int(42)},
		},
	}

	var es EvalState

	b.ReportAllocs()

	for b.Loop() {
		assert.That(b, es.Evaluate(filter, ev))
	}
}
