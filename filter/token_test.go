package filter

import (
	"fmt"
	"testing"

	"github.com/zeebo/assert"
)

var tokenCases = []struct {
	in  string
	out []string
}{
	{`equal("foo", "bar")` /*                  */, []string{`equal`, `(`, `foo`, `,`, `bar`, `)`}},
	{`equal("foo", "bar") && exists("test")` /**/, []string{`equal`, `(`, `foo`, `,`, `bar`, `)`, `&`, `exists`, `(`, `test`, `)`}},
	{`equal("foo", "bar") & exists("test")` /* */, []string{`equal`, `(`, `foo`, `,`, `bar`, `)`, `&`, `exists`, `(`, `test`, `)`}},
	{`equal("foo\"bar")` /*                    */, []string{`equal`, `(`, `foo\"bar`, `)`}},
	{`less(rand(), 0.5)` /*                    */, []string{`less`, `(`, `rand`, `(`, `)`, `,`, `0.5`, `)`}},
}

func TestToken(t *testing.T) {
	for i, c := range tokenCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			toks, err := tokens(c.in, nil)
			assert.NoError(t, err)

			var out []string
			for _, tk := range toks {
				if tk.isLiteral() {
					out = append(out, string(tk.literal(c.in)))
				} else {
					out = append(out, tk.String())
				}
			}

			assert.Equal(t, out, c.out)
		})
	}
}
