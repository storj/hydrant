package submitters

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/zeebo/hmux"

	"storj.io/hydrant"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/value"
)

var traceFilterPool = sync.Pool{New: func() any { return new(filter.EvalState) }}

type pendingTrace struct {
	spans []hydrant.Event
}

type traceEntry struct {
	traceID [16]byte
	spans   []hydrant.Event
}

type TraceBufferSubmitter struct {
	fil  *filter.Filter
	live liveBuffer

	mu        sync.Mutex
	traces    []traceEntry               // completed traces ring buffer
	completed map[[16]byte]int           // trace_id → slot index in traces
	pos       int                        // next slot in ring buffer
	pending   map[[16]byte]*pendingTrace // in-progress traces

	stats struct {
		received atomic.Uint64
		spans    atomic.Uint64
		traces   atomic.Uint64
		evicted  atomic.Uint64
		filtered atomic.Uint64
	}
}

func NewTraceBufferSubmitter(size int, fil *filter.Filter) *TraceBufferSubmitter {
	if size <= 0 {
		size = 64
	}
	return &TraceBufferSubmitter{
		fil:       fil,
		live:      newLiveBuffer(),
		traces:    make([]traceEntry, size),
		completed: make(map[[16]byte]int, size),
		pending:   make(map[[16]byte]*pendingTrace),
	}
}

func (t *TraceBufferSubmitter) Children() []Submitter {
	return []Submitter{}
}

func (t *TraceBufferSubmitter) ExtraData() any {
	return map[string]string{"filter": t.fil.Filter()}
}

func (t *TraceBufferSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	t.live.Record(ev)
	t.stats.received.Add(1)

	traceID, spanID, parentID, ok := extractTraceInfo(ev)
	if !ok {
		return
	}
	t.stats.spans.Add(1)

	isRoot := spanID == parentID

	t.mu.Lock()
	defer t.mu.Unlock()

	// Already completed — append late span.
	if idx, exists := t.completed[traceID]; exists {
		t.traces[idx].spans = append(t.traces[idx].spans, ev)
		return
	}

	// Already pending.
	if pt, exists := t.pending[traceID]; exists {
		pt.spans = append(pt.spans, ev)
		if isRoot {
			if t.passesFilter(ev) {
				t.insertCompleted(traceID, pt.spans)
			} else {
				t.stats.filtered.Add(1)
			}
			delete(t.pending, traceID)
		}
		return
	}

	// New root trace.
	if isRoot {
		if t.passesFilter(ev) {
			t.insertCompleted(traceID, []hydrant.Event{ev})
		} else {
			t.stats.filtered.Add(1)
		}
		return
	}

	// New pending trace.
	t.pending[traceID] = &pendingTrace{spans: []hydrant.Event{ev}}
}

// passesFilter evaluates the filter against ev. Must be called with t.mu held.
func (t *TraceBufferSubmitter) passesFilter(ev hydrant.Event) bool {
	es := traceFilterPool.Get().(*filter.EvalState)
	ok := es.Evaluate(t.fil, ev)
	traceFilterPool.Put(es)
	return ok
}

// insertCompleted adds a trace to the ring buffer. Must be called with t.mu held.
func (t *TraceBufferSubmitter) insertCompleted(traceID [16]byte, spans []hydrant.Event) {
	idx := t.pos % len(t.traces)
	t.pos++

	if old := t.traces[idx]; old.spans != nil {
		delete(t.completed, old.traceID)
		t.stats.evicted.Add(1)
	}

	t.traces[idx] = traceEntry{
		traceID: traceID,
		spans:   spans,
	}
	t.completed[traceID] = idx
	t.stats.traces.Add(1)
}

func (t *TraceBufferSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree":   constJSONHandler(treeify(t)),
		"/live":   t.live.Handler(),
		"/traces": http.HandlerFunc(t.tracesHandler),
		"/stats": statsHandler(func() []stat {
			t.mu.Lock()
			completedTraces := uint64(len(t.completed))
			pendingTraces := uint64(len(t.pending))
			t.mu.Unlock()
			return []stat{
				{"received", t.stats.received.Load()},
				{"spans", t.stats.spans.Load()},
				{"traces", t.stats.traces.Load()},
				{"completed_traces", completedTraces},
				{"pending_traces", pendingTraces},
				{"evicted", t.stats.evicted.Load()},
				{"filtered", t.stats.filtered.Load()},
			}
		}),
	}
}

type jsonTrace struct {
	TraceID string      `json:"trace_id"`
	Spans   []jsonEvent `json:"spans"`
}

func (t *TraceBufferSubmitter) tracesHandler(w http.ResponseWriter, r *http.Request) {
	t.mu.Lock()

	// Collect traces newest first.
	size := len(t.traces)
	out := make([]jsonTrace, 0, len(t.completed))
	for i := 0; i < size; i++ {
		idx := ((t.pos-1-i)%size + size) % size
		entry := t.traces[idx]
		if entry.spans == nil {
			continue
		}
		out = append(out, jsonTrace{
			TraceID: hex.EncodeToString(entry.traceID[:]),
			Spans:   serializeEvents(entry.spans),
		})
	}

	t.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func extractTraceInfo(ev hydrant.Event) (traceID [16]byte, spanID, parentID [8]byte, ok bool) {
	var hasTrace, hasSpan, hasParent bool
	for _, a := range ev {
		switch a.Value.Kind() {
		case value.KindTraceId:
			if a.Key == "trace_id" {
				traceID, hasTrace = a.Value.TraceId()
			}
		case value.KindSpanId:
			switch a.Key {
			case "span_id":
				spanID, hasSpan = a.Value.SpanId()
			case "parent_id":
				parentID, hasParent = a.Value.SpanId()
			}
		}
	}
	ok = hasTrace && hasSpan && hasParent
	return
}
