// slog demonstrates bridging Go's log/slog into hydrant. All slog output
// becomes hydrant events, flowing through the pipeline with full type fidelity.
// If a span is active on the context, log events are linked to it.
//
// Run the example:
//
//	go run .
//
// Open http://localhost:9912 to see the events in the web UI.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/process"
	"storj.io/hydrant/submitters"
	"storj.io/hydrant/utils/slogutil"
)

func main() {
	sub, err := submitters.Environment{
		Filter:  filter.NewBuiltinEnvionment(),
		Process: process.DefaultStore,
	}.New(config.Config{
		Submitter: config.GrouperSubmitter{
			FlushInterval: 10 * time.Second,
			GroupBy:       []string{"name"},
			Submitter:     config.HydratorSubmitter{},
		},
	})
	if err != nil {
		panic(err)
	}

	hydrant.SetDefaultSubmitter(sub)
	go sub.Run(context.Background())

	// Set the default slog logger to write through hydrant.
	slogutil.SetDefault(nil)

	// Simulate work that uses both slog and hydrant spans.
	go func() {
		for {
			doWork(context.Background())
			time.Sleep(100 * time.Millisecond)
		}
	}()

	fmt.Println("web UI at http://localhost:9912")
	panic(http.ListenAndServe(":9912", sub.Handler()))
}

func doWork(ctx context.Context) {
	ctx, span := hydrant.StartSpanNamed(ctx, "process_order")
	defer span.Done(nil)

	orderID := fmt.Sprintf("ord-%d", rand.IntN(10000))
	items := rand.IntN(5) + 1

	// slog calls inside a span are linked via span_id and trace_id.
	slog.InfoContext(ctx, "processing order",
		slog.String("order_id", orderID),
		slog.Int("items", items),
	)

	time.Sleep(time.Duration(rand.IntN(80)+10) * time.Millisecond)

	slog.InfoContext(ctx, "order complete",
		slog.String("order_id", orderID),
		slog.Duration("elapsed", time.Duration(rand.IntN(80))*time.Millisecond),
	)
}
