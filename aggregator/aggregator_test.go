package aggregator

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
)

func TestWireEverythingUp(t *testing.T) {
	var destinationURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			fmt.Fprintf(w, `
			{"destinations": [{
				"url": "%s",
			    "aggregation_interval": "1m",
			    "queries": [{ "filter": "has(message)" }]
			}]}`, destinationURL)
		case http.MethodPost:
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Log(err)
				t.Fail()
				return
			}
			t.Log(string(data))
		}
	}))
	defer srv.Close()
	destinationURL = srv.URL

	p := new(filter.Parser)
	filter.SetBuiltins(p)

	a := NewAggregator([]*config.Source{config.NewSource(srv.URL)}, p)

	ctx := context.Background()

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	wg.Go(func() { a.Run(ctxTimeout) })

	<-a.WaitForFirstLoad()

	ctx = hydrant.WithSubmitter(ctx, a)

	hydrant.Log(ctx, "hello")

	wg.Wait()
}
