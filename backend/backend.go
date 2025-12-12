package backend

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
	"storj.io/hydrant/filter"
	"storj.io/hydrant/process"
	"storj.io/hydrant/utils"
)

const (
	maxInterval = 24 * time.Hour
	minInterval = time.Minute
)

type Source interface {
	Load(context.Context) (cfg config.SourceConfig, dsts []config.Destination, err error)
}

type runningDest struct {
	dest   *destination.Destination
	cancel func()
	done   chan struct{}
}

type Backend struct {
	sources   []Source
	self      hydrant.Submitter
	filterEnv *filter.Environment
	trigger   chan chan struct{}

	mu   sync.Mutex
	swap swaparoo.Tracker
	subs [2][][]runningDest

	once   sync.Once
	loaded chan struct{}
}

var _ hydrant.Submitter = (*Backend)(nil)

func New(sources []Source, self hydrant.Submitter, filterEnv *filter.Environment) *Backend {
	return &Backend{
		sources:   sources,
		self:      self,
		filterEnv: filterEnv,
		trigger:   make(chan chan struct{}),

		subs: [2][][]runningDest{
			make([][]runningDest, len(sources)),
			make([][]runningDest, len(sources)),
		},

		loaded: make(chan struct{}),
	}
}

func (b *Backend) FirstLoad() <-chan struct{} { return b.loaded }

func (b *Backend) Trigger() {
	ch := make(chan struct{})
	select {
	case b.trigger <- ch:
		<-ch
	default:
	}
}

func (b *Backend) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for sourceIdx, source := range b.sources {
		wg.Go(func() {
			var refreshInterval time.Duration
			var triggered chan struct{}

			for {
				csc, destinations, err := source.Load(ctx)
				if err != nil {
					// TODO: log this error?
				} else {
					refreshInterval = time.Duration(csc.RefreshInterval)
					b.updateDestinations(ctx, sourceIdx, destinations)
					b.once.Do(func() { close(b.loaded) })
				}

				if triggered != nil {
					close(triggered)
					triggered = nil
				}

				refreshInterval = min(refreshInterval, maxInterval)
				refreshInterval = max(refreshInterval, minInterval)

				select {
				case <-ctx.Done():
					return
				case <-time.After(utils.Jitter(refreshInterval)):
				case triggered = <-b.trigger:
				}
			}
		})
	}

	wg.Wait()
}

func (b *Backend) updateDestinations(ctx context.Context, sourceIdx int, destConfigs []config.Destination) {
	// exclusively update the destinations
	b.mu.Lock()
	defer b.mu.Unlock()

	// check if all the destinations are equal
	token := b.swap.Acquire()
	current := b.subs[token.Gen()%2]

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
		dest, err := destination.New(destConfig, b.self, b.filterEnv, process.GetStore(ctx))
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
	b.subs[(token.Gen()+1)%2] = next

	// increment and wait for everyone to be finished with them.
	gen := b.swap.Increment().Wait()

	// clean up the destinations that were just swapped from.
	for _, dest := range b.subs[gen%2][sourceIdx] {
		dest.cancel()
		<-dest.done
	}
}

func (b *Backend) Submit(ctx context.Context, ev hydrant.Event) {
	token := b.swap.Acquire()
	defer token.Release()

	for _, dests := range b.subs[token.Gen()%2] {
		for _, dest := range dests {
			dest.dest.Submit(ctx, ev)
		}
	}
}
