package main

import (
	"context"
	"sync"

	"storj.io/hydrant/config"
)

type FixedSource struct {
	mu sync.Mutex

	dests []config.Destination
}

func (f *FixedSource) Load(ctx context.Context) (cfg config.SourceConfig, dsts []config.Destination, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return config.SourceConfig{}, f.dests, nil
}

func (f *FixedSource) SetDestinations(dsts []config.Destination) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.dests = dsts
}
