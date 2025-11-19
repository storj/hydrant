package filter

import (
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/mwc"

	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

func BenchmarkEval(b *testing.B) {
	var p Parser
	p.SetFunction("equal", (*EvalState).Equal)
	p.SetFunction("exists", (*EvalState).Exists)
	p.SetFunction("key", (*EvalState).Key)
	p.SetFunction("less", (*EvalState).Less)
	p.SetFunction("rand", func(es *EvalState) bool { es.Push(value.Float(mwc.Float64())); return true })

	filter, err := p.Parse(`equal(key("foo"), "bar") && exists("test") && less(rand(), 1)`)
	assert.NoError(b, err)

	ev := hydrant.Event{
		System: []hydrant.Annotation{{"foo", value.String("bar")}},
		User:   []hydrant.Annotation{{"test", value.Int(42)}},
	}

	for range 100 {
		ev.System = append(ev.System, hydrant.Annotation{"extra_sys", value.Bool(true)})
		ev.User = append(ev.User, hydrant.Annotation{"extra_user", value.Bool(true)})
	}

	var es EvalState

	b.ReportAllocs()

	for b.Loop() {
		assert.That(b, es.Evaluate(filter, ev))
	}
}
