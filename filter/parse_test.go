package filter

import (
	"testing"

	"github.com/zeebo/assert"
)

func TestParse(t *testing.T) {
	var p Environment
	SetBuiltins(&p)

	filter, err := p.Parse(`eq(key("foo"), "b\tar") && (has("test") || lt(rand(), 0.5))`)
	assert.NoError(t, err)

	t.Logf("prog: %v", filter.prog)
	t.Logf("vals: %v", anyfy(filter.vals))
}

func BenchmarkParse(b *testing.B) {
	var p Environment
	SetBuiltins(&p)

	query := `(eq(key("foo"), "bar") && has("test")) || lt(rand(), 0.5)`

	b.ReportAllocs()

	for b.Loop() {
		_, _ = p.Parse(query)
	}
}
