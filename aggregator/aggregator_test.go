package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
)

func TestAggregator(t *testing.T) {
	env := new(filter.Environment)
	filter.SetBuiltins(env)
	var bs bufferSubmitter

	q, err := New(env, &bs, config.Query{
		Filter:  config.Expression("eq(key(name), test) && lt(key(dur), 1s)"),
		GroupBy: []config.Expression{"group"},
	})
	assert.NoError(t, err)

	ev := func(name string, dur time.Duration, group string, count int) hydrant.Event {
		return hydrant.Event{
			hydrant.String("name", name),
			hydrant.Duration("dur", dur),
			hydrant.String("group", group),
			hydrant.Int("count", int64(count)),
		}
	}

	q.Submit(t.Context(), ev("wrong", 500*time.Millisecond, "group1", 1))
	q.Submit(t.Context(), ev("test", 500*time.Millisecond, "group1", 1))
	q.Submit(t.Context(), ev("test", 500*time.Millisecond, "group1", 1))
	q.Submit(t.Context(), ev("test", 5000*time.Millisecond, "group1", 1))
	q.Submit(t.Context(), ev("test", 500*time.Millisecond, "group2", 10))
	q.Submit(t.Context(), ev("test", 50*time.Millisecond, "group2", 1))
	q.Submit(t.Context(), ev("test", 5000*time.Millisecond, "group2", 1))
	q.Flush(t.Context())
	q.Flush(t.Context()) // should do nothing

	for _, ev := range bs {
		t.Log(ev)
	}
}

type bufferSubmitter []hydrant.Event

func (bs *bufferSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	*bs = append(*bs, ev)
}
