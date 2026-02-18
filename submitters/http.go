package submitters

import (
	"bytes"
	"context"
	"net/http"
	"sync"
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

		batch:   make([]hydrant.Event, 0, batch),
		trigger: make(chan struct{}, 1),
	}
}

func (h *HTTPSubmitter) Children() []Submitter {
	return []Submitter{}
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
	h.mu.Lock()
	if len(h.batch) < cap(h.batch) {
		h.batch = append(h.batch, ev)
	} else {
		// TODO: note that it dropped the event
	}
	if len(h.batch) >= cap(h.batch) {
		h.Trigger()
	}
	h.mu.Unlock()
}

func (h *HTTPSubmitter) flush(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.batch) == 0 {
		return
	}

	buf := make([]byte, 0, 64)
	buf = hydrant.Event(h.process).AppendTo(buf)
	buf = rw.AppendVarint(buf, uint64(len(h.batch)))
	for _, ev := range h.batch {
		buf = ev.AppendTo(buf)
	}
	out := h.enc.EncodeAll(buf, nil)

	// TODO: connection pool configuration?
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(out))
	if err != nil {
		// TODO: log dropped packets? try again?
		h.batch = h.batch[:0]
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// TODO: log dropped packets? try again?
		h.batch = h.batch[:0]
		return
	}
	defer resp.Body.Close()

	// TODO: check http status code
	h.batch = h.batch[:0]
}

func (h *HTTPSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree": constJSONHandler(treeify(h)),
	}
}
