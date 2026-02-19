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
	"storj.io/hydrant/value"
)

type traceEntry struct {
	traceID [16]byte
	spans   []hydrant.Event
	done    bool
}

type TraceBufferSubmitter struct {
	live liveBuffer

	mu     sync.Mutex
	traces []traceEntry
	index  map[[16]byte]int // trace_id → slot index
	pos    int

	stats struct {
		received atomic.Uint64
		spans    atomic.Uint64
		traces   atomic.Uint64
		evicted  atomic.Uint64
	}
}

func NewTraceBufferSubmitter(size int) *TraceBufferSubmitter {
	if size <= 0 {
		size = 64
	}
	return &TraceBufferSubmitter{
		live:   newLiveBuffer(),
		traces: make([]traceEntry, size),
		index:  make(map[[16]byte]int, size),
	}
}

func (t *TraceBufferSubmitter) Children() []Submitter {
	return []Submitter{}
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

	if idx, exists := t.index[traceID]; exists {
		t.traces[idx].spans = append(t.traces[idx].spans, ev)
		if isRoot {
			t.traces[idx].done = true
		}
		return
	}

	// Allocate new slot.
	idx := t.pos % len(t.traces)
	t.pos++

	// Evict old trace if slot was occupied.
	if old := t.traces[idx]; old.spans != nil {
		delete(t.index, old.traceID)
		t.stats.evicted.Add(1)
	}

	t.traces[idx] = traceEntry{
		traceID: traceID,
		spans:   []hydrant.Event{ev},
		done:    isRoot,
	}
	t.index[traceID] = idx
	t.stats.traces.Add(1)
}

func (t *TraceBufferSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree":   constJSONHandler(treeify(t)),
		"/live":   t.live.Handler(),
		"/traces": http.HandlerFunc(t.tracesHandler),
		"/stats": statsHandler(func() []stat {
			t.mu.Lock()
			activeTraces := uint64(len(t.index))
			t.mu.Unlock()
			return []stat{
				{"received", t.stats.received.Load()},
				{"spans", t.stats.spans.Load()},
				{"traces", t.stats.traces.Load()},
				{"active_traces", activeTraces},
				{"evicted", t.stats.evicted.Load()},
			}
		}),
	}
}

type jsonTrace struct {
	TraceID string      `json:"trace_id"`
	Done    bool        `json:"done"`
	Spans   []jsonEvent `json:"spans"`
}

func (t *TraceBufferSubmitter) tracesHandler(w http.ResponseWriter, r *http.Request) {
	t.mu.Lock()

	// Collect traces newest first.
	size := len(t.traces)
	out := make([]jsonTrace, 0, len(t.index))
	for i := 0; i < size; i++ {
		idx := ((t.pos - 1 - i) % size + size) % size
		entry := t.traces[idx]
		if entry.spans == nil || !entry.done {
			continue
		}
		out = append(out, jsonTrace{
			TraceID: hex.EncodeToString(entry.traceID[:]),
			Done:    entry.done,
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
