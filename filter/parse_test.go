package filter

import (
	"testing"

	"github.com/zeebo/assert"
)

func TestParse(t *testing.T) {
	var p Parser
	SetBuiltins(&p)

	filter, err := p.Parse(`(equal(key("foo"), "b\tar") && exists("test")) || less(rand(), 0.5)`)
	assert.NoError(t, err)

	t.Logf("prog: %v", filter.prog)
}

func BenchmarkParse(b *testing.B) {
	var p Parser
	p.SetFunction("equal", func(es *EvalState) bool { return true })
	p.SetFunction("exists", func(es *EvalState) bool { return true })
	p.SetFunction("less", func(es *EvalState) bool { return true })
	p.SetFunction("rand", func(es *EvalState) bool { return true })

	query := `(equal(key("foo"), "bar") && exists("test")) || less(rand(), 0.5)`

	b.ReportAllocs()

	for b.Loop() {
		_, _ = p.Parse(query)
	}
}
