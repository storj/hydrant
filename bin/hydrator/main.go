package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/zeebo/clingy"
	"github.com/zeebo/errs/v2"
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

	var mem MemStore

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
				NewHandler(&mem),
			),
		),
	)
}
