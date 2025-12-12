package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/zeebo/clingy"
	"github.com/zeebo/errs/v2"

	"storj.io/hydrant/backend"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
)

func main() {
	ok, err := clingy.Environment{
		Root: new(root),
		Name: "hydrator",
		Args: os.Args[1:],
	}.Run(context.Background(), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}
	if !ok || err != nil {
		os.Exit(1)
	}
}

type root struct {
	config string
	addr   string
}

func (r *root) Setup(params clingy.Parameters) {
	r.config = params.Flag("config", "Path to configuration file", "config.json",
		clingy.Short('c'),
	).(string)
	r.addr = params.Flag("addr", "Address to bind HTTP server to", ":0",
		clingy.Short('a'),
	).(string)
}

func (r *root) Execute(ctx context.Context) (err error) {
	// TODO: this needs to load a config of some sort and use it to make some
	// sort of submitter that eventually submits down into a memstore.

	mem := new(MemStore)
	env := new(filter.Environment)
	filter.SetBuiltins(env)
	source := &FixedSource{dests: []config.Destination{
		{
			URL:                 "self",
			AggregationInterval: config.Duration(10 * time.Second),
			Queries: []config.Query{
				{
					Filter:  "has(span_id)",
					GroupBy: []config.Expression{"name", "success"},
				},
			},
		},
	}}

	b := backend.New([]backend.Source{source}, mem, env)

	go b.Run(ctx)
	b.Trigger()

	select {
	case <-b.FirstLoad():
	case <-ctx.Done():
		return ctx.Err()
	}

	lis, err := net.Listen("tcp", r.addr)
	if err != nil {
		return errs.Wrap(err)
	}
	defer lis.Close()

	log.SetOutput(clingy.Stdout(ctx))
	log.Println("Listening on", lis.Addr().String())

	return errs.Wrap(
		http.Serve(
			lis,
			handlers.LoggingHandler(
				clingy.Stdout(ctx),
				NewHandler(b, mem, source),
			),
		),
	)
}
