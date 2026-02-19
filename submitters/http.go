package submitters

import (
	"bytes"
	"context"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/zeebo/hmux"

	"storj.io/hydrant"
	"storj.io/hydrant/internal/rw"
	"storj.io/hydrant/internal/utils"
)

type HTTPSubmitter struct {
	url      string
	process  []hydrant.Annotation
	interval time.Duration
	enc      *zstd.Encoder
	live     liveBuffer

	stats struct {
		received    atomic.Uint64
		dropped     atomic.Uint64
		flushes     atomic.Uint64
		flushErrors atomic.Uint64
		bytesSent   atomic.Uint64
	}

	mu      sync.Mutex
	batch   []hydrant.Event
	trigger chan struct{}
}

func NewHTTPSubmitter(
	url string,
	process []hydrant.Annotation,
	interval time.Duration,
	batch int,
) *HTTPSubmitter {
	enc, err := zstd.NewWriter(nil,
		zstd.WithWindowSize(1<<20),
		zstd.WithLowerEncoderMem(true),
	)
	if err != nil {
		panic(err) // this can only happen with invalid options
	}

	return &HTTPSubmitter{
		url:      url,
		process:  process,
		interval: interval,
		enc:      enc,
		live:     newLiveBuffer(),

		batch:   make([]hydrant.Event, 0, batch),
		trigger: make(chan struct{}, 1),
	}
}

func (h *HTTPSubmitter) Children() []Submitter {
	return []Submitter{}
}

func (h *HTTPSubmitter) ExtraData() any {
	return map[string]string{"endpoint": h.url}
}

func (h *HTTPSubmitter) Run(ctx context.Context) {
	nextTick := time.After(utils.Jitter(h.interval))
	for {
		select {
		case <-ctx.Done():
			// TODO: maybe we want to put a timeout or not flush at all? unsure
			h.flush(context.WithoutCancel(ctx))
			return
		case <-h.trigger:
		case <-nextTick:
		}
		nextTick = time.After(utils.Jitter(h.interval))
		h.flush(ctx)
	}
}

func (h *HTTPSubmitter) Trigger() {
	select {
	case h.trigger <- struct{}{}:
	default:
	}
}

func (h *HTTPSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	h.live.Record(ev)
	h.stats.received.Add(1)

	h.mu.Lock()
	if len(h.batch) < cap(h.batch) {
		h.batch = append(h.batch, ev)
	} else {
		h.stats.dropped.Add(1)
	}
	// trigger a flush slightly early to avoid dropping events.
	if len(h.batch) >= cap(h.batch)*2/3 {
		h.Trigger()
	}
	h.mu.Unlock()
}

func (h *HTTPSubmitter) flush(ctx context.Context) {
	h.mu.Lock()
	batch := slices.Clone(h.batch)
	h.batch = h.batch[:0]
	h.mu.Unlock()

	if len(batch) == 0 {
		return
	}

	buf := make([]byte, 0, 64)
	buf = hydrant.Event(h.process).AppendTo(buf)
	buf = rw.AppendVarint(buf, uint64(len(batch)))
	for _, ev := range batch {
		buf = ev.AppendTo(buf)
	}
	out := h.enc.EncodeAll(buf, nil)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(out))
	if err != nil {
		h.stats.flushErrors.Add(1)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.stats.flushErrors.Add(1)
		return
	}
	resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		h.stats.flushErrors.Add(1)
	} else {
		h.stats.flushes.Add(1)
		h.stats.bytesSent.Add(uint64(len(out)))
	}
}

func (h *HTTPSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree": constJSONHandler(treeify(h)),
		"/live": h.live.Handler(),
		"/stats": statsHandler(func() []stat {
			return []stat{
				{"received", h.stats.received.Load()},
				{"dropped", h.stats.dropped.Load()},
				{"flushes", h.stats.flushes.Load()},
				{"flush_errors", h.stats.flushErrors.Load()},
				{"bytes_sent", h.stats.bytesSent.Load()},
			}
		}),
	}
}
