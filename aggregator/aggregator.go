package aggregator

import (
	"context"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/zeebo/swaparoo"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/destination"
)

const (
	maxInterval = 24 * time.Hour
)

type Aggregator struct {
	cfgs []config.ConfigSource
	mu   sync.Mutex
	subs [2][][]runningDest
	swap swaparoo.Tracker
}

type runningDest struct {
	dest   *destination.Destination
	cancel func()
	done   chan struct{}
}

var _ hydrant.Submitter = (*Aggregator)(nil)

func NewAggregator(cfgs []config.ConfigSource) *Aggregator {
	return &Aggregator{
		cfgs: cfgs,
		subs: [2][][]runningDest{
			make([][]runningDest, len(cfgs)),
			make([][]runningDest, len(cfgs)),
		},
	}
}

func (a *Aggregator) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for sourceIdx, cfg := range a.cfgs {
		wg.Go(func() {
			refreshInterval := 15 * time.Minute
			for {
				if err := ctx.Err(); err != nil {
					return
				}
				csc, destinations, err := cfg.Load(ctx)
				if err != nil {
					// TODO: log this error?
				} else {
					refreshInterval = min(maxInterval, time.Duration(csc.RefreshInterval))
					a.updateDestinations(ctx, sourceIdx, destinations)
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(jitter(refreshInterval)):
				}
			}
		})
	}
	wg.Wait()
}

func (a *Aggregator) updateDestinations(ctx context.Context, sourceIdx int, destConfigs []config.Destination) {
	// exclusively update the destinations
	a.mu.Lock()
	defer a.mu.Unlock()

	// check if all the destinations are equal
	token := a.swap.Acquire()
	current := a.subs[token.Gen()%2]

	differs := len(current[sourceIdx]) != len(destConfigs)
	if !differs {
		for i, dest := range current[sourceIdx] {
			if !reflect.DeepEqual(dest.dest.Config, destConfigs[i]) {
				differs = true
				break
			}
		}
	}
	if !differs {
		token.Release()
		return
	}

	dests := make([]*destination.Destination, len(destConfigs))
	for i, destConfig := range destConfigs {
		dest, err := destination.New(destConfig)
		if err != nil {
			token.Release()
			return
		}
		dests[i] = dest
	}

	// make a copy of the current slice and release the token. after now we're
	// done with current and update the destinations for the source index.
	next := slices.Clone(current)
	token.Release()

	next[sourceIdx] = make([]runningDest, len(destConfigs))
	for i, dest := range dests {
		ctx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		next[sourceIdx][i] = runningDest{
			dest:   dest,
			cancel: cancel,
			done:   done,
		}
		go func() {
			defer close(done)
			next[sourceIdx][i].dest.Run(ctx)
		}()
	}

	// set the submitters for submit to use for the next generation.
	a.subs[(token.Gen()+1)%2] = next

	// increment and wait for everyone to be finished with them.
	gen := a.swap.Increment().Wait()

	// clean up the destinations that were just swapped from.
	for _, dest := range a.subs[gen%2][sourceIdx] {
		dest.cancel()
		<-dest.done
	}
}

func (a *Aggregator) Submit(ev hydrant.Event) {
	token := a.swap.Acquire()
	defer token.Release()

	for _, dests := range a.subs[token.Gen()%2] {
		for _, dest := range dests {
			dest.dest.Submit(ev)
		}
	}
}
