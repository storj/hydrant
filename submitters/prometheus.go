package submitters

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/zeebo/hmux"

	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

var defaultPromBuckets = []float64{
	.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10,
}

type promLabel struct {
	key   string
	value string
}

type promSeries struct {
	labels  []promLabel
	count   uint64
	sum     float64
	buckets []uint64
	errors  uint64
}

type PrometheusSubmitter struct {
	namespace string
	buckets   []float64
	live      liveBuffer

	stats struct {
		received atomic.Uint64
		skipped  atomic.Uint64
	}

	mu     sync.Mutex
	series map[string]*promSeries
}

func NewPrometheusSubmitter(namespace string, buckets []float64) *PrometheusSubmitter {
	if namespace == "" {
		namespace = "hydrant"
	}
	if len(buckets) == 0 {
		buckets = defaultPromBuckets
	}
	buckets = slices.Clone(buckets)
	sort.Float64s(buckets)

	return &PrometheusSubmitter{
		namespace: namespace,
		buckets:   buckets,
		live:      newLiveBuffer(),
		series:    make(map[string]*promSeries),
	}
}

func (p *PrometheusSubmitter) Children() []Submitter {
	return []Submitter{}
}

func (p *PrometheusSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	p.live.Record(ev)
	p.stats.received.Add(1)

	// Only process events that have a duration histogram.
	hasDuration := false
	for _, ann := range ev {
		if ann.Key == "duration" && ann.Value.Kind() == value.KindHistogram {
			hasDuration = true
			break
		}
	}
	if !hasDuration {
		p.stats.skipped.Add(1)
		return
	}

	// Extract labels from string annotations (skip agg: metadata).
	var labels []promLabel
	for _, ann := range ev {
		if ann.Value.Kind() != value.KindString {
			continue
		}
		if strings.HasPrefix(ann.Key, "agg:") {
			continue
		}
		v, _ := ann.Value.String()
		labels = append(labels, promLabel{key: ann.Key, value: v})
	}
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].key < labels[j].key
	})

	// Build a stable key from the sorted labels.
	var keyBuf strings.Builder
	for i, l := range labels {
		if i > 0 {
			keyBuf.WriteByte(',')
		}
		keyBuf.WriteString(l.key)
		keyBuf.WriteByte('=')
		keyBuf.WriteString(l.value)
	}
	key := keyBuf.String()

	p.mu.Lock()
	defer p.mu.Unlock()

	s := p.series[key]
	if s == nil {
		s = &promSeries{
			labels:  labels,
			buckets: make([]uint64, len(p.buckets)),
		}
		p.series[key] = s
	}

	// Accumulate duration histogram into prometheus buckets.
	for _, ann := range ev {
		if ann.Key != "duration" {
			continue
		}
		h, ok := ann.Value.Histogram()
		if !ok {
			break
		}

		total, sum, _, _ := h.Summary()
		s.count += total
		s.sum += sum

		// Use CDF to fill bucket counts. CDF returns fraction < threshold,
		// which is close enough to le for float histograms.
		tot := float64(total)
		for i, bound := range p.buckets {
			s.buckets[i] += uint64(h.CDF(float32(bound)) * tot)
		}
		break
	}

	// Accumulate errors from success histogram.
	for _, ann := range ev {
		if ann.Key != "success" {
			continue
		}
		h, ok := ann.Value.Histogram()
		if !ok {
			break
		}

		// success is a bool: 0.0 = false, 1.0 = true.
		// CDF(0.5) gives fraction of values < 0.5, i.e. the false observations.
		total := float64(h.Total())
		s.errors += uint64(h.CDF(0.5) * total)
		break
	}
}

func (p *PrometheusSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree":    constJSONHandler(treeify(p)),
		"/live":    p.live.Handler(),
		"/metrics": http.HandlerFunc(p.metricsHandler),
		"/stats": statsHandler(func() []stat {
			p.mu.Lock()
			seriesActive := uint64(len(p.series))
			p.mu.Unlock()
			return []stat{
				{"received", p.stats.received.Load()},
				{"skipped", p.stats.skipped.Load()},
				{"series_active", seriesActive},
			}
		}),
	}
}

func (p *PrometheusSubmitter) metricsHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	// Snapshot the series under the lock.
	type snapshot struct {
		labels  []promLabel
		count   uint64
		sum     float64
		buckets []uint64
		errors  uint64
	}
	snaps := make([]snapshot, 0, len(p.series))
	for _, s := range p.series {
		snaps = append(snaps, snapshot{
			labels:  s.labels,
			count:   s.count,
			sum:     s.sum,
			buckets: slices.Clone(s.buckets),
			errors:  s.errors,
		})
	}
	p.mu.Unlock()

	sort.Slice(snaps, func(i, j int) bool {
		return promLabelsLess(snaps[i].labels, snaps[j].labels)
	})

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	ns := p.namespace

	// Duration histogram.
	fmt.Fprintf(w, "# HELP %s_duration_seconds Duration of events.\n", ns)
	fmt.Fprintf(w, "# TYPE %s_duration_seconds histogram\n", ns)
	for _, s := range snaps {
		ls := formatLabels(s.labels)
		for i, bound := range p.buckets {
			fmt.Fprintf(w, "%s_duration_seconds_bucket{%sle=\"%g\"} %d\n",
				ns, ls, bound, s.buckets[i])
		}
		fmt.Fprintf(w, "%s_duration_seconds_bucket{%sle=\"+Inf\"} %d\n",
			ns, ls, s.count)
		fmt.Fprintf(w, "%s_duration_seconds_sum{%s} %g\n",
			ns, trimTrailingComma(ls), s.sum)
		fmt.Fprintf(w, "%s_duration_seconds_count{%s} %d\n",
			ns, trimTrailingComma(ls), s.count)
	}

	// Events total counter.
	fmt.Fprintf(w, "# HELP %s_events_total Total number of events.\n", ns)
	fmt.Fprintf(w, "# TYPE %s_events_total counter\n", ns)
	for _, s := range snaps {
		fmt.Fprintf(w, "%s_events_total{%s} %d\n",
			ns, trimTrailingComma(formatLabels(s.labels)), s.count)
	}

	// Errors total counter.
	fmt.Fprintf(w, "# HELP %s_errors_total Total number of failed events.\n", ns)
	fmt.Fprintf(w, "# TYPE %s_errors_total counter\n", ns)
	for _, s := range snaps {
		fmt.Fprintf(w, "%s_errors_total{%s} %d\n",
			ns, trimTrailingComma(formatLabels(s.labels)), s.errors)
	}

	// Active spans gauge (point-in-time, not pipeline-driven).
	fmt.Fprintf(w, "# HELP %s_active_spans Number of currently active spans.\n", ns)
	fmt.Fprintf(w, "# TYPE %s_active_spans gauge\n", ns)
	fmt.Fprintf(w, "%s_active_spans %d\n", ns, hydrant.ActiveSpanCount())
}

// formatLabels returns a Prometheus label string like `name="foo",endpoint="/api",`.
// The trailing comma makes it easy to append le="..." for histograms.
func formatLabels(labels []promLabel) string {
	if len(labels) == 0 {
		return ""
	}
	var b strings.Builder
	for _, l := range labels {
		b.WriteString(l.key)
		b.WriteString(`="`)
		b.WriteString(escapeLabelValue(l.value))
		b.WriteString(`",`)
	}
	return b.String()
}

func trimTrailingComma(s string) string {
	return strings.TrimRight(s, ",")
}

func escapeLabelValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func promLabelsLess(a, b []promLabel) bool {
	for i := range a {
		if i >= len(b) {
			return false
		}
		if a[i].key != b[i].key {
			return a[i].key < b[i].key
		}
		if a[i].value != b[i].value {
			return a[i].value < b[i].value
		}
	}
	return len(a) < len(b)
}
