// remoteconfig demonstrates central configuration with RemoteSubmitter. A
// config server serves pipeline definitions over HTTP, and the client polls
// for changes and hot-swaps its pipeline without restarting.
//
// Run the example:
//
//	go run .
//
// The config server starts on :9913 and the hydrant client on :9912.
// Edit the pipeline by POSTing new JSON to the config server:
//
//	curl -X POST http://localhost:9913/config -d '{"submitter":{"kind":"null"}}'
//
// The client picks up the change on its next poll (every 10 seconds) or you
// can trigger it immediately by visiting http://localhost:9912.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"storj.io/hydrant"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/process"
	"storj.io/hydrant/submitters"
)

func main() {
	// Start the config server. It serves a pipeline config that the
	// RemoteSubmitter polls periodically.
	cfgSrv := newConfigServer()
	go func() {
		panic(http.ListenAndServe(":9913", cfgSrv))
	}()

	// Create a RemoteSubmitter that polls the config server.
	env := submitters.Environment{
		Filter:  filter.NewBuiltinEnvionment(),
		Process: process.DefaultStore,
	}
	remote := submitters.NewRemoteSubmitter(env, "http://localhost:9913/config")
	go remote.Run(context.Background())

	hydrant.SetDefaultSubmitter(remote)

	// Simulate work.
	go func() {
		for {
			doWork(context.Background())
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Serve the hydrant web UI through the RemoteSubmitter. It delegates
	// to whichever ConfiguredSubmitter is currently active.
	fmt.Println("hydrant UI at     http://localhost:9912")
	fmt.Println("config server at  http://localhost:9913/config")
	fmt.Println()
	fmt.Println("try changing the pipeline:")
	fmt.Println(`  curl http://localhost:9913/config              # see current config`)
	fmt.Println(`  curl -X POST http://localhost:9913/config \`)
	fmt.Println(`    -d '{"submitter":{"kind":"null"}}'           # drop all events`)
	panic(http.ListenAndServe(":9912", remote))
}

func doWork(ctx context.Context) {
	names := []string{"process_order", "send_email", "sync_inventory"}
	name := names[rand.IntN(len(names))]

	ctx, span := hydrant.StartSpanNamed(ctx, name,
		hydrant.String("customer", fmt.Sprintf("cust-%d", rand.IntN(100))),
	)
	defer span.Done(nil)

	time.Sleep(time.Duration(rand.IntN(80)+10) * time.Millisecond)

	hydrant.Log(ctx, "completed",
		hydrant.Int("items", int64(rand.IntN(10)+1)),
	)
}

// configServer is a trivial HTTP config server. GET returns the current
// config as JSON. POST replaces it.
type configServer struct {
	mu  sync.RWMutex
	cfg json.RawMessage
}

func newConfigServer() *configServer {
	return &configServer{
		cfg: json.RawMessage(`{
	"refresh_interval": "10s",
	"submitter": {
		"kind": "grouper",
		"flush_interval": "10s",
		"group_by": ["name"],
		"submitter": {"kind": "hydrator"}
	}
}`),
	}
}

func (s *configServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.mu.RLock()
		cfg := s.cfg
		s.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(cfg)

	case "POST":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !json.Valid(body) {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.cfg = body
		s.mu.Unlock()
		fmt.Fprintf(w, "config updated\n")

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
