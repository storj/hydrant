package filter

import (
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/hydrant"
)

func TestEvalShortCircuit(t *testing.T) {
	var es EvalState
	var p Environment
	SetBuiltins(&p)
	p.SetFunction("panic", func(es *EvalState) bool { t.Fatal("panic called"); return false })

	filter, err := p.Parse(`false() && panic(key(foo))`)
	assert.NoError(t, err)
	t.Log(filter.prog)
	assert.False(t, es.Evaluate(filter, hydrant.Event{}))
	assert.Equal(t, es.executed, 2)

	filter, err = p.Parse(`true() || panic(key(foo))`)
	assert.NoError(t, err)
	t.Log(filter.prog)
	assert.True(t, es.Evaluate(filter, hydrant.Event{}))
	assert.Equal(t, es.executed, 2)

	filter, err = p.Parse(`false() && panic(key(foo)) && panic(key(foo))`)
	assert.NoError(t, err)
	t.Log(filter.prog)
	assert.False(t, es.Evaluate(filter, hydrant.Event{}))
	assert.Equal(t, es.executed, 2)

	filter, err = p.Parse(`true() || panic(key(foo)) || panic(key(foo))`)
	assert.NoError(t, err)
	t.Log(filter.prog)
	assert.True(t, es.Evaluate(filter, hydrant.Event{}))
	assert.Equal(t, es.executed, 2)
}

func TestEvalDoubleKey(t *testing.T) {
	var es EvalState
	var p Environment
	SetBuiltins(&p)

	filter, err := p.Parse(`eq(key(key(foo)), bar)`)
	assert.NoError(t, err)
	t.Log(filter.prog)

	ev := hydrant.Event{
		hydrant.String("foo", "inner"),
		hydrant.String("inner", "bar"),
	}

	assert.True(t, es.Evaluate(filter, ev))
}

func TestEvalEmptyProgram(t *testing.T) {
	var es EvalState
	var p Environment
	SetBuiltins(&p)

	filter, err := p.Parse(``)
	assert.NoError(t, err)
	t.Log(filter.prog)

	assert.True(t, es.Evaluate(filter, hydrant.Event{}))
	assert.True(t, es.Evaluate(filter, hydrant.Event{
		hydrant.String("foo", "bar"),
	}))
}

//
// benchmarks
//

func BenchmarkEval(b *testing.B) {
	var p Environment
	SetBuiltins(&p)

	filter, err := p.Parse(`
		eq(key(foo), bar) && has(test) && lt(rand(), 1) && gte(key(dur), 1m)
	`)
	assert.NoError(b, err)
	b.Log("prog:", filter.prog)
	b.Log("vals:", anyfy(filter.vals))

	ev := hydrant.Event{
		hydrant.String("foo", "bar"),
		hydrant.Duration("dur", time.Minute),
		hydrant.Int("test", 42),
	}

	var es EvalState
	assert.That(b, es.Evaluate(filter, ev))
	b.Log("executed", es.executed, "instructions")

	b.ReportAllocs()
	for b.Loop() {
		es.Evaluate(filter, ev)
	}
}
