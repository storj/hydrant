package protocol

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"storj.io/hydrant"
)

var (
	httpBatchSubmit = 30 * time.Second
	httpBatchMax    = 10_000
)

type HTTPSubmitter struct {
	url string

	mu      sync.Mutex
	batch   []hydrant.Event
	trigger chan struct{}
}

func NewHTTPSubmitter(url string) *HTTPSubmitter {
	return &HTTPSubmitter{
		url:     url,
		trigger: make(chan struct{}, 1),
	}
}

func (s *HTTPSubmitter) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.trigger:
			s.submitBatch(ctx)
		case <-time.After(jitter(httpBatchSubmit)):
			s.submitBatch(ctx)
		}
	}
}

func (s *HTTPSubmitter) submitBatch(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.batch) == 0 {
		return
	}

	var reqBody io.Reader

	// TODO: connection pool configuration?
	req, err := http.NewRequestWithContext(ctx, s.url, http.MethodPost, reqBody)
	if err != nil {
		// TODO: log dropped packets? try again?
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// TODO: log dropped packets? try again?
		return
	}
	defer resp.Body.Close()

	// TODO
}

func (s *HTTPSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	s.mu.Lock()
	s.batch = append(s.batch, ev)
	batchSize := len(s.batch)
	s.mu.Unlock()
	if batchSize >= httpBatchMax {
		select {
		case s.trigger <- struct{}{}:
		default:
		}
	}
}

var _ hydrant.Submitter = (*HTTPSubmitter)(nil)
