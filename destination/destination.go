package destination

import (
	"context"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/protocol"
)

type Destination struct {
	Config    config.Destination
	Submitter *protocol.HTTPSubmitter
}

func New(cfg config.Destination) (*Destination, error) {
	// TODO: construct query handlers
	return &Destination{
		Config:    cfg,
		Submitter: protocol.NewHTTPSubmitter(cfg.URL),
	}, nil
}

func (d *Destination) Run(ctx context.Context) {
	// TODO: query value aggregation interval
	d.Submitter.Run(ctx)
}

func (d *Destination) Submit(ctx context.Context, ev hydrant.Event) {
	for _, query := range d.Config.Queries {
		// TODO
		_ = query
	}
}

var _ hydrant.Submitter = (*Destination)(nil)
