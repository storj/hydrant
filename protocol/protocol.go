package protocol

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"storj.io/hydrant"
)

var (
	httpBatchInterval = 30 * time.Second
	httpBatchMax      = 10_000
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

func (s *HTTPSubmitter) submitBatch(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.batch) == 0 {
		return
	}

	var reqBody bytes.Buffer
	// TODO: actually compress and format in a good way
	fmt.Fprintf(&reqBody, "%#v", s.batch)

	// TODO: connection pool configuration?
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, &reqBody)
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
		select {
		case s.trigger <- struct{}{}:
		default:
		}
	}
}

var _ hydrant.Submitter = (*HTTPSubmitter)(nil)
