package submitters

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zeebo/assert"

	"github.com/histdb/histdb/flathist"
	"storj.io/hydrant"
)

func TestHydrator_IssueWithManyTags(t *testing.T) {
	h := NewHydratorSubmitter()

	for i := range 256 {
		h.Submit(t.Context(), hydrant.Event{
			hydrant.String("name", fmt.Sprintf("foo-%d", i)),
			hydrant.Histogram("duration", flathist.NewHistogram()),
		})
	}

	h.Query([]byte(`_=**`), func(name []byte, hist *flathist.Histogram) bool {
		assert.That(t, strings.Contains(string(name), "_=duration,name=foo-"))
		return true
	})
}
