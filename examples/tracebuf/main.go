// tracebuf demonstrates the trace buffer submitter, which captures recent
// traces in a ring buffer for browsing in the web UI at :9912.
package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
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
		Submitter: config.TraceBufferSubmitter{
			// Filter: "not(key(success)) || eq(key(name), http_request)",
		},
	})
	if err != nil {
		panic(err)
	}

	hydrant.SetDefaultSubmitter(sub)
	go sub.Run(context.Background())

	go func() {
		for {
			switch rand.IntN(5) {
			case 0:
				simulateHTTPRequest(context.Background())
			case 1:
				simulateDBQuery(context.Background())
			case 2:
				simulateOrderPipeline(context.Background())
			case 3:
				simulateParallelFanout(context.Background())
			case 4:
				simulateFailingRequest(context.Background())
			}
			time.Sleep(time.Duration(100+rand.IntN(200)) * time.Millisecond)
		}
	}()

	fmt.Println("web UI at http://localhost:9912")
	panic(http.ListenAndServe(":9912", sub.Handler()))
}

// simulateHTTPRequest produces a simple 3-level trace:
// http_request -> auth_check -> db_lookup
func simulateHTTPRequest(ctx context.Context) {
	endpoints := []string{"/api/users", "/api/orders", "/api/products"}
	endpoint := endpoints[rand.IntN(len(endpoints))]

	ctx, span := hydrant.StartSpanNamed(ctx, "http_request",
		hydrant.String("endpoint", endpoint),
		hydrant.String("method", "GET"),
	)
	defer span.Done(nil)

	time.Sleep(jitter(5))

	authCheck(ctx)

	time.Sleep(jitter(10))
}

func authCheck(ctx context.Context) {
	ctx, span := hydrant.StartSpanNamed(ctx, "auth_check")
	defer span.Done(nil)

	// Near-instant checks to trigger min-width expansion in the waterfall.
	func() {
		_, s := hydrant.StartSpanNamed(ctx, "parse_token")
		defer s.Done(nil)
		time.Sleep(time.Duration(50+rand.IntN(100)) * time.Microsecond)
	}()
	func() {
		_, s := hydrant.StartSpanNamed(ctx, "verify_signature")
		defer s.Done(nil)
		time.Sleep(time.Duration(20+rand.IntN(80)) * time.Microsecond)
	}()

	time.Sleep(jitter(3))
	dbLookup(ctx, "users")
}

func dbLookup(ctx context.Context, table string) {
	_, span := hydrant.StartSpanNamed(ctx, "db_lookup",
		hydrant.String("table", table),
	)
	defer span.Done(nil)
	time.Sleep(jitter(8))
}

// simulateDBQuery produces a 2-level trace with multiple sequential child spans.
func simulateDBQuery(ctx context.Context) {
	tables := []string{"users", "orders", "products", "sessions"}
	table := tables[rand.IntN(len(tables))]

	ctx, span := hydrant.StartSpanNamed(ctx, "db_transaction",
		hydrant.String("table", table),
	)
	defer span.Done(nil)

	// Connection acquire
	func() {
		_, s := hydrant.StartSpanNamed(ctx, "conn_acquire")
		defer s.Done(nil)
		time.Sleep(jitter(2))
	}()

	// Query execution
	n := 1 + rand.IntN(3)
	for i := range n {
		func() {
			_, s := hydrant.StartSpanNamed(ctx, "query_exec",
				hydrant.Int("query_index", int64(i)),
				hydrant.Int("rows", int64(rand.IntN(500))),
			)
			defer s.Done(nil)
			time.Sleep(jitter(10))
		}()
	}

	// Commit
	func() {
		_, s := hydrant.StartSpanNamed(ctx, "commit")
		defer s.Done(nil)
		time.Sleep(jitter(3))
	}()
}

// simulateOrderPipeline produces a deeper trace: process_order -> validate ->
// charge_payment -> (db_write + send_notification in parallel)
func simulateOrderPipeline(ctx context.Context) {
	ctx, span := hydrant.StartSpanNamed(ctx, "process_order",
		hydrant.String("order_id", fmt.Sprintf("ORD-%04d", rand.IntN(10000))),
	)
	defer span.Done(nil)

	// validate
	func() {
		_, s := hydrant.StartSpanNamed(ctx, "validate_order")
		defer s.Done(nil)
		time.Sleep(jitter(5))
	}()

	// charge payment
	func() {
		ctx2, s := hydrant.StartSpanNamed(ctx, "charge_payment",
			hydrant.String("provider", "stripe"),
		)
		defer s.Done(nil)
		time.Sleep(jitter(30))

		hydrant.Log(ctx2, "payment charged",
			hydrant.String("status", "success"),
		)
	}()

	// parallel: db write + notification
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, s := hydrant.StartSpanNamed(ctx, "db_write",
			hydrant.String("table", "orders"),
		)
		defer s.Done(nil)
		time.Sleep(jitter(15))
	}()
	go func() {
		defer wg.Done()
		_, s := hydrant.StartSpanNamed(ctx, "send_notification",
			hydrant.String("channel", "email"),
		)
		defer s.Done(nil)
		time.Sleep(jitter(20))
	}()
	wg.Wait()
}

// simulateParallelFanout produces a wide trace with many parallel children.
func simulateParallelFanout(ctx context.Context) {
	var err error
	if rand.IntN(10) == 0 {
		err = fmt.Errorf("simulated failure")
	}

	ctx, span := hydrant.StartSpanNamed(ctx, "batch_process",
		hydrant.Int("batch_size", int64(4+rand.IntN(4))),
	)
	defer span.Done(&err)

	// fan out
	n := 4 + rand.IntN(4)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func() {
			defer wg.Done()
			ctx2, s := hydrant.StartSpanNamed(ctx, "process_item",
				hydrant.Int("item_index", int64(i)),
			)
			defer s.Done(nil)
			time.Sleep(jitter(20))

			// Some items do a nested call
			if rand.IntN(3) == 0 {
				_, s2 := hydrant.StartSpanNamed(ctx2, "enrich_item")
				defer s2.Done(nil)
				time.Sleep(jitter(10))
			}
		}()
	}
	wg.Wait()
}

// simulateFailingRequest produces a trace where a child span fails.
func simulateFailingRequest(ctx context.Context) {
	ctx, span := hydrant.StartSpanNamed(ctx, "http_request",
		hydrant.String("endpoint", "/api/checkout"),
		hydrant.String("method", "POST"),
	)
	defer span.Done(nil)

	time.Sleep(jitter(5))

	err := func() error {
		ctx2, s := hydrant.StartSpanNamed(ctx, "process_checkout")
		var err error
		defer s.Done(&err)

		time.Sleep(jitter(10))

		hydrant.Log(ctx2, "inventory check",
			hydrant.String("status", "checking"),
		)

		// Simulate failure
		if rand.IntN(2) == 0 {
			err = fmt.Errorf("out of stock")
			return err
		}

		time.Sleep(jitter(15))
		return nil
	}()
	_ = err
}

func jitter(baseMs int) time.Duration {
	return time.Duration(baseMs+rand.IntN(baseMs+1)) * time.Millisecond
}
