package submitters

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"reflect"
	"sync"

	"github.com/zeebo/errs/v2"
	"github.com/zeebo/hmux"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/process"
)

type ConfiguredSubmitter struct {
	cfg      config.Config
	root     Submitter
	named    map[string]*lateSubmitter
	runnable []runnable
}

type Environment struct {
	Filter  *filter.Environment
	Process *process.Store
}

func (env Environment) New(cfg config.Config) (*ConfiguredSubmitter, error) {
	// collect all the names into a late binding submitter
	named := make(map[string]*lateSubmitter)
	for name := range cfg.Submitters {
		named[name] = newLateSubmitter()
	}

	// create a constructor with the environment and late bindings and construct all of the
	// submitters recursively, binding the late submitters as we go.
	cons := newConstructor(env, named)
	for name, cfg := range cfg.Submitters {
		sub, err := cons.Construct(cfg)
		if err != nil {
			return nil, errs.Errorf("constructing submitter %q: %w", name, err)
		}
		named[name].SetSubmitter(sub)
	}

	// construct the root submitter.
	root, err := cons.Construct(cfg.Submitter)
	if err != nil {
		return nil, errs.Errorf("constructing root submitter: %w", err)
	}

	return &ConfiguredSubmitter{
		cfg:      cfg,
		root:     root,
		named:    named,
		runnable: cons.Runnable(),
	}, nil
}

func (s *ConfiguredSubmitter) Config() config.Config {
	return s.cfg
}

func (s *ConfiguredSubmitter) Children() []Submitter {
	return []Submitter{s.root}
}

func (s *ConfiguredSubmitter) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, rsub := range s.runnable {
		wg.Go(func() { rsub.Run(ctx) })
	}
	wg.Wait()
}

func (s *ConfiguredSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	s.root.Submit(ctx, ev)
}

//go:embed static
var static embed.FS

func (s *ConfiguredSubmitter) Handler() http.Handler {
	subs := hmux.Dir{}
	names := make(map[string]string)
	for name, sub := range s.named {
		subs["/"+name] = sub.Handler()
		names[name] = reflect.TypeOf(sub.sub).Elem().Name()
	}

	return hmux.Dir{
		// TODO: a bit weird that this is where static is injected, but it's hard to find a way
		// to do double wildcard merging because we return an http.Handler from this method.
		"*": http.FileServerFS(func() fs.FS { sub, _ := fs.Sub(static, "static"); return sub }()),

		"/tree":   constJSONHandler(treeify(s)),
		"/config": constJSONHandler(s.cfg),
		"/sub":    s.root.Handler(),
		"/names":  constJSONHandler(names),
		"/name":   subs,
	}
}
