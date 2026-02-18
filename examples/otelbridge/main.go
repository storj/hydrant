// otelbridge demonstrates the bidirectional OTel integration. Hydrant events
// are exported to an OTLP collector, and the server also accepts incoming OTLP
// traces and logs.
//
// To see the export side, run an OTel collector or Jaeger instance:
//
//	docker run -p 4318:4318 -p 16686:16686 jaegertracing/all-in-one:latest
//
// Then run this example:
//
//	go run .
//
// Traces will appear in Jaeger at http://localhost:16686 and in the hydrant
// web UI at http://localhost:9912.
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
	"storj.io/hydrant/utils/otelutil"
)

func main() {
	sub, err := submitters.Environment{
		Filter:  filter.NewBuiltinEnvionment(),
		Process: process.DefaultStore,
	}.New(config.Config{
		Submitter: config.MultiSubmitter{
			// Store in-memory for the web UI.
			config.GrouperSubmitter{
				FlushInterval: 10 * time.Second,
				GroupBy:       []string{"name"},
				Submitter:     config.HydratorSubmitter{},
			},
			// Also export to an OTLP collector.
			config.OTelSubmitter{
				Endpoint:      "http://localhost:4318",
				FlushInterval: 5 * time.Second,
				MaxBatchSize:  1000,
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
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Serve the hydrant web UI plus OTLP receiver endpoints.
	http.Handle("/", sub.Handler())
	http.Handle("/v1/traces", otelutil.NewTraceReceiver(sub))
	http.Handle("/v1/logs", otelutil.NewLogReceiver(sub))

	fmt.Println("web UI at          http://localhost:9912")
	fmt.Println("OTLP receiver at   http://localhost:9912/v1/traces")
	fmt.Println("Jaeger UI at       http://localhost:16686 (if running)")
	panic(http.ListenAndServe(":9912", nil))
}

func doWork(ctx context.Context) {
	ctx, span := hydrant.StartSpanNamed(ctx, "process_order",
		hydrant.String("order_id", fmt.Sprintf("ord-%d", rand.IntN(10000))),
	)
	defer span.Done(nil)

	time.Sleep(time.Duration(rand.IntN(50)+10) * time.Millisecond)

	ctx, child := hydrant.StartSpanNamed(ctx, "db_insert")
	defer child.Done(nil)

	time.Sleep(time.Duration(rand.IntN(30)+5) * time.Millisecond)

	hydrant.Log(ctx, "order saved",
		hydrant.Int("items", int64(rand.IntN(5)+1)),
	)
}
