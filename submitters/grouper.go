package submitters

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
	"unique"

	"github.com/zeebo/hmux"

	"github.com/histdb/histdb/flathist"
	"storj.io/hydrant"
	"storj.io/hydrant/internal/group"
	"storj.io/hydrant/internal/utils"
	"storj.io/hydrant/value"
)

const (
	maxGroupInterval = 24 * time.Hour
	minGroupInterval = 10 * time.Second
)

type groupedEvents struct {
	event    hydrant.Event
	groupSet map[string]struct{}
	hists    map[string]*flathist.Histogram
	histOrd  []string
	excluded map[string]struct{}
}

type GrouperSubmitter struct {
	grouper  *group.Grouper
	sub      Submitter
	interval time.Duration
	live     liveBuffer

	mu     sync.Mutex
	groups map[unique.Handle[string]]*groupedEvents
}

func NewGrouperSubmitter(
	fields []string,
	interval time.Duration,
	sub Submitter,
) *GrouperSubmitter {
	return &GrouperSubmitter{
		grouper:  group.NewGrouper(fields),
		sub:      sub,
		interval: utils.Bound(interval, [2]time.Duration{minGroupInterval, maxGroupInterval}),
		live:     newLiveBuffer(),

		groups: make(map[unique.Handle[string]]*groupedEvents),
	}
}

func (g *GrouperSubmitter) Children() []Submitter {
	return []Submitter{g.sub}
}

func (g *GrouperSubmitter) Run(ctx context.Context) {
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			start = g.flush(context.WithoutCancel(ctx), start)
			return

		case <-time.After(g.interval):
			start = g.flush(ctx, start)
		}
	}
}

func (g *GrouperSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	g.live.Record(ev)

	key, ok := g.grouper.Group(ev)
	if !ok {
		return
	}

	// TODO: it'd be nice if this mutex was smaller or non-existent but we have to coordinate with
	// the flush call. perhaps we can swaparoo it. we would also either need to add a mutex to the
	// groupedEvents or make it use sync maps or something. the histograms are already concurrent.
	g.mu.Lock()
	defer g.mu.Unlock()

	ge := g.groups[key]
	if ge == nil {
		group := g.grouper.Annotations(ev)
		groupSet := make(map[string]struct{}, len(group))
		for _, ann := range group {
			groupSet[ann.Key] = struct{}{}
		}

		ge = &groupedEvents{
			event:    group,
			groupSet: groupSet,
			hists:    make(map[string]*flathist.Histogram),
			excluded: make(map[string]struct{}),
		}

		g.groups[key] = ge
	}

	for _, ann := range ev {
		// annotations in the group set are not included
		if _, ok := ge.groupSet[ann.Key]; ok {
			continue
		}

		// if we got a full histogram, merge it into our existing one.
		if h, ok := ann.Value.Histogram(); ok {
			into, ok := ge.hists[ann.Key]
			if !ok {
				into = flathist.NewHistogram()
				ge.hists[ann.Key] = into
				ge.histOrd = append(ge.histOrd, ann.Key)
			}

			into.Merge(h)
			continue
		}

		// if we can't convert it to a float, exclude it.
		datum, ok := observableValue(ann.Value)
		if !ok {
			ge.excluded[ann.Key] = struct{}{}
			continue
		}

		// otherwise, observe the value.
		into, ok := ge.hists[ann.Key]
		if !ok {
			into = flathist.NewHistogram()
			ge.hists[ann.Key] = into
			ge.histOrd = append(ge.histOrd, ann.Key)
		}

		into.Observe(datum)
	}
}

func observableValue(v value.Value) (float32, bool) {
	switch v.Kind() {
	case value.KindInt:
		x, _ := v.Int()
		return float32(x), true
	case value.KindUint:
		x, _ := v.Uint()
		return float32(x), true
	case value.KindDuration:
		x, _ := v.Duration()
		return float32(x.Seconds()), true
	case value.KindTimestamp:
		x, _ := v.Timestamp()
		return float32(x.Unix()), true
	case value.KindFloat:
		x, _ := v.Float()
		return float32(x), true
	case value.KindBool:
		x, _ := v.Bool()
		if x {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func (g *GrouperSubmitter) flush(ctx context.Context, start time.Time) time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()

	end := time.Now()

	for _, ge := range g.groups {
		ge.event = append(ge.event,
			hydrant.Timestamp("agg:start_time", start),
			hydrant.Timestamp("agg:end_time", end),
			hydrant.Duration("agg:duration", end.Sub(start)),
		)
		if len(ge.excluded) > 0 {
			excluded := strings.Join(slices.Collect(maps.Keys(ge.excluded)), ",")
			ge.event = append(ge.event,
				hydrant.String("agg:excluded", excluded),
			)
		}
		for _, key := range ge.histOrd {
			ge.event = append(ge.event, hydrant.Histogram(key, ge.hists[key]))
		}

		g.sub.Submit(ctx, ge.event)
	}

	clear(g.groups)

	return end
}

func (g *GrouperSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree": constJSONHandler(treeify(g)),
		"/live": g.live.Handler(),
		"/sub":  g.sub.Handler(),
	}
}
