package protocol

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"

	"storj.io/hydrant"
	"storj.io/hydrant/process"
	"storj.io/hydrant/rw"
)

var (
	httpBatchInterval = 30 * time.Second
	httpBatchMax      = 10_000
)

type HTTPSubmitter struct {
	url                string
	processAnnotations *process.Selected
	enc                *zstd.Encoder

	mu      sync.Mutex
	batch   []hydrant.Event
	trigger chan struct{}
}

func NewHTTPSubmitter(url string, processAnnotations *process.Selected) *HTTPSubmitter {
	encoder, err := zstd.NewWriter(nil,
		zstd.WithWindowSize(1<<20),
		zstd.WithLowerEncoderMem(true),
	)
	if err != nil {
		panic(err) // this can only happen with invalid options
	}

	return &HTTPSubmitter{
		url:                url,
		processAnnotations: processAnnotations,
		enc:                encoder,

		trigger: make(chan struct{}, 1),
	}
}

func (s *HTTPSubmitter) Run(ctx context.Context) {
	nextTick := time.After(jitter(httpBatchInterval))
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.trigger:
		case <-nextTick:
		}
		nextTick = time.After(jitter(httpBatchInterval))
		s.submitBatch(ctx)
	}
}

func (s *HTTPSubmitter) Trigger() {
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}

func (s *HTTPSubmitter) submitBatch(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.batch) == 0 {
		return
	}

	buf := make([]byte, 0, 64)
	buf = hydrant.Event(s.processAnnotations.Annotations()).AppendTo(buf)
	buf = rw.AppendVarint(buf, uint64(len(s.batch)))
	for _, ev := range s.batch {
		buf = ev.AppendTo(buf)
	}
	out := s.enc.EncodeAll(buf, nil)

	// TODO: connection pool configuration?
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(out))
	if err != nil {
		// TODO: log dropped packets? try again?
		s.batch = s.batch[:0]
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// TODO: log dropped packets? try again?
		s.batch = s.batch[:0]
		return
	}
	defer resp.Body.Close()

	// TODO: check http status code
	s.batch = s.batch[:0]
}

func (s *HTTPSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	s.mu.Lock()
	s.batch = append(s.batch, ev)
	batchSize := len(s.batch)
	s.mu.Unlock()
	if batchSize >= httpBatchMax {
		s.Trigger()
	}
}

var _ hydrant.Submitter = (*HTTPSubmitter)(nil)
