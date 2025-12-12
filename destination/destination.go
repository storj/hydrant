package destination

import (
	"context"
	"sync"
	"time"

	"storj.io/hydrant"
	"storj.io/hydrant/aggregator"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/process"
	"storj.io/hydrant/protocol"
	"storj.io/hydrant/utils"
)

const (
	maxAggregationInterval = 24 * time.Hour
	minAggregationInterval = 10 * time.Second
)

type Destination struct {
	Config config.Destination

	aggregators []*aggregator.Aggregator
	submitter   hydrant.Submitter
	http        *protocol.HTTPSubmitter
}

var _ hydrant.Submitter = (*Destination)(nil)

func New(cfg config.Destination, self hydrant.Submitter, filterEnv *filter.Environment, store *process.Store) (*Destination, error) {
	submitter, http := self, (*protocol.HTTPSubmitter)(nil)
	if cfg.URL != "self" {
		http = protocol.NewHTTPSubmitter(cfg.URL, process.NewSelected(store, cfg.GlobalFields))
		submitter = http
	}

	aggregators := make([]*aggregator.Aggregator, 0, len(cfg.Queries))
	for i := range cfg.Queries {
		agg, err := aggregator.New(filterEnv, submitter, cfg.Queries[i])
		if err != nil {
			return nil, err
		}
		aggregators = append(aggregators, agg)
	}

	return &Destination{
		Config: cfg,

		aggregators: aggregators,
		submitter:   submitter,
		http:        http,
	}, nil
}

func (d *Destination) Run(ctx context.Context) {
	var wg sync.WaitGroup

	wg.Go(func() {
		if d.http != nil {
			d.http.Run(ctx)
		}
	})

	wg.Go(func() {
		interval := time.Duration(d.Config.AggregationInterval)
		interval = max(interval, minAggregationInterval)
		interval = min(interval, maxAggregationInterval)

		nextTick := time.After(utils.Jitter(interval))
		for {
			select {
			case <-ctx.Done():
				return
			case <-nextTick:
			}
			nextTick = time.After(utils.Jitter(interval))

			for _, agg := range d.aggregators {
				agg.Flush(ctx)
			}
		}
	})

	wg.Wait()

	for _, agg := range d.aggregators {
		agg.Flush(context.WithoutCancel(ctx))
	}
}

func (d *Destination) Submit(ctx context.Context, ev hydrant.Event) {
	for _, agg := range d.aggregators {
		agg.Submit(ctx, ev)
	}
}
