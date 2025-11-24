package destination

import (
	"context"
	"sync"
	"time"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/process"
	"storj.io/hydrant/protocol"
)

const (
	maxAggregationInterval = 24 * time.Hour
	minAggregationInterval = time.Minute
)

type Destination struct {
	Config config.Destination

	queries   []*Query
	submitter *protocol.HTTPSubmitter
}

func New(cfg config.Destination, p *filter.Parser, s *process.Store) (
	*Destination, error) {
	submitter := protocol.NewHTTPSubmitter(cfg.URL, process.NewSelected(s, cfg.GlobalFields))

	queries := make([]*Query, 0, len(cfg.Queries))
	for i := range cfg.Queries {
		q, err := NewQuery(p, submitter, cfg.Queries[i])
		if err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}

	return &Destination{
		Config:    cfg,
		queries:   queries,
		submitter: submitter,
	}, nil
}

func (d *Destination) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() {
		d.submitter.Run(ctx)
	})
	wg.Go(func() {
		interval := time.Duration(d.Config.AggregationInterval)
		interval = max(interval, minAggregationInterval)
		interval = min(interval, maxAggregationInterval)
		nextTick := time.After(jitter(interval))
		for {
			select {
			case <-ctx.Done():
				return
			case <-nextTick:
			}
			nextTick = time.After(jitter(interval))
			for _, q := range d.queries {
				q.Flush(ctx)
			}
		}
	})
	wg.Wait()
}

func (d *Destination) Submit(ctx context.Context, ev hydrant.Event) {
	for _, query := range d.queries {
		query.Submit(ctx, ev)
	}
}

var _ hydrant.Submitter = (*Destination)(nil)
