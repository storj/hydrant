// basic demonstrates the core hydrant pipeline: spans flow through a grouper
// into an in-memory hydrator, queryable through the web UI at :9912.
package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/process"
	"storj.io/hydrant/submitters"
)

func main() {
	sub, err := submitters.Environment{
		Filter:  filter.NewBuiltinEnvionment(),
		Process: process.DefaultStore,
	}.New(config.Config{
		Submitter: config.GrouperSubmitter{
			FlushInterval: 10 * time.Second,
			GroupBy:       []string{"name", "endpoint"},
			Submitter:     config.HydratorSubmitter{},
		},
	})
	if err != nil {
		panic(err)
	}

	hydrant.SetDefaultSubmitter(sub)
	go sub.Run(context.Background())

	// Simulate work that produces spans.
	go func() {
		for {
			simulateRequest(context.Background())
			time.Sleep(50 * time.Millisecond)
		}
	}()

	fmt.Println("web UI at http://localhost:9912")
	panic(http.ListenAndServe(":9912", sub.Handler()))
}

func simulateRequest(ctx context.Context) {
	endpoints := []string{"/api/users", "/api/orders", "/api/health"}
	endpoint := endpoints[rand.IntN(len(endpoints))]

	ctx, span := hydrant.StartSpanNamed(ctx, "http_request",
		hydrant.String("endpoint", endpoint),
	)
	defer span.Done(nil)

	// Simulate some latency.
	latency := time.Duration(rand.IntN(100)+10) * time.Millisecond
	time.Sleep(latency)

	hydrant.Log(ctx, "handled request",
		hydrant.Duration("latency", latency),
		hydrant.Int("status", 200),
	)
}
