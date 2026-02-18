package submitters

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/zeebo/swaparoo"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/internal/utils"
)

const (
	maxRemoteInterval = 24 * time.Hour
	minRemoteInterval = 10 * time.Second
)

type runningConfiguredSubmitter struct {
	sub     *ConfiguredSubmitter
	handler http.Handler
	cancel  func()
	done    chan struct{}
}

func (r *runningConfiguredSubmitter) stop() {
	if r.cancel != nil {
		r.cancel()
		<-r.done
		r.cancel = nil
	}
}

type RemoteSubmitter struct {
	url     string
	env     Environment
	trigger chan chan struct{}

	mu   sync.Mutex
	cfg  config.Config
	swap swaparoo.Tracker
	sub  [2]runningConfiguredSubmitter
}

func NewRemoteSubmitter(env Environment, url string) *RemoteSubmitter {
	return &RemoteSubmitter{
		env:     env,
		url:     url,
		trigger: make(chan chan struct{}, 1),
	}
}

func (r *RemoteSubmitter) Run(ctx context.Context) {
	var interval time.Duration
	var triggered chan struct{}

	for {
		cfg, err := r.getConfig(ctx)
		if err == nil {
			r.updateConfig(ctx, cfg)
		}

		interval = utils.Bound(cfg.RefreshInterval, [2]time.Duration{
			minRemoteInterval,
			maxRemoteInterval,
		})

		if triggered != nil {
			close(triggered)
			triggered = nil
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(utils.Jitter(interval)):
		case triggered = <-r.trigger:
		}
	}
}

func (r *RemoteSubmitter) getConfig(ctx context.Context) (cfg config.Config, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", r.url, nil)
	if err != nil {
		return config.Config{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return config.Config{}, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return config.Config{}, err
	}

	return cfg, nil
}

func (r *RemoteSubmitter) updateConfig(ctx context.Context, cfg config.Config) {
	next, err := r.env.New(cfg)
	if err != nil {
		// TODO: log this or something?
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// if the config hasn't changed, don't do anything
	if reflect.DeepEqual(cfg, r.cfg) {
		return
	}
	r.cfg = cfg

	// acquire a token to create the next configured submitter.
	tok := r.swap.Acquire()

	// create the next configured submitter.
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)
	r.sub[(tok.Gen()+1)%2] = runningConfiguredSubmitter{
		sub:     next,
		handler: next.Handler(),
		cancel:  cancel,
		done:    done,
	}

	// start it up.
	go func() {
		defer close(done)
		next.Run(ctx)
	}()

	// release the token so we can increment.
	tok.Release()

	// increment and wait so no one is using the old one and everyone is now using the next one.
	r.swap.Increment().Wait()

	// stop the old one.
	r.sub[tok.Gen()%2].stop()

	// clear it out to free memory
	r.sub[tok.Gen()%2] = runningConfiguredSubmitter{}
}

func (r *RemoteSubmitter) Trigger() {
	ch := make(chan struct{})
	select {
	case r.trigger <- ch:
		<-ch
	default:
	}
}

func (r *RemoteSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	tok := r.swap.Acquire()
	defer tok.Release()

	if sub := r.sub[tok.Gen()%2].sub; sub != nil {
		sub.Submit(ctx, ev)
	}
}

func (r *RemoteSubmitter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	tok := r.swap.Acquire()
	defer tok.Release()

	if handler := r.sub[tok.Gen()%2].handler; handler != nil {
		handler.ServeHTTP(w, req)
	} else {
		http.Error(w, "no submitter configured", http.StatusServiceUnavailable)
	}
}
