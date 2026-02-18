// prometheus demonstrates exporting hydrant metrics in Prometheus format.
// Events flow through a grouper into both a hydrator (for the web UI) and a
// PrometheusSubmitter (for /metrics scraping).
//
// Run the example:
//
//	go run .
//
// Then scrape metrics:
//
//	curl http://localhost:9912/name/prom/metrics
//
// The hydrant web UI is available at http://localhost:9912.
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
		Submitter: config.NamedSubmitter("default"),
		Submitters: map[string]config.Submitter{
			"prom": config.PrometheusSubmitter{
				Namespace: "myapp",
				Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			"hyd": config.HydratorSubmitter{},
			"default": config.GrouperSubmitter{
				FlushInterval: 10 * time.Second,
				GroupBy:       []string{"name"},
				Submitter: config.MultiSubmitter{
					config.NamedSubmitter("hyd"),
					config.NamedSubmitter("prom"),
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	hydrant.SetDefaultSubmitter(sub)
	go sub.Run(context.Background())

	// Simulate work.
	go func() {
		for {
			doWork(context.Background())
			time.Sleep(50 * time.Millisecond)
		}
	}()

	fmt.Println("hydrant web UI at      http://localhost:9912")
	fmt.Println("prometheus metrics at  http://localhost:9912/name/prom/metrics")
	panic(http.ListenAndServe(":9912", sub.Handler()))
}

func doWork(ctx context.Context) {
	endpoints := []string{"/api/users", "/api/orders", "/api/health"}
	endpoint := endpoints[rand.IntN(len(endpoints))]

	var err error
	// Simulate occasional errors.
	if rand.IntN(10) == 0 {
		err = fmt.Errorf("simulated failure")
	}

	ctx, span := hydrant.StartSpanNamed(ctx, "http_request",
		hydrant.String("endpoint", endpoint),
	)
	defer span.Done(&err)

	time.Sleep(time.Duration(rand.IntN(200)+5) * time.Millisecond)

	hydrant.Log(ctx, "handled request",
		hydrant.Int("status", 200),
	)
}
