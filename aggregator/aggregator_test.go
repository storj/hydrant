package aggregator

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
)

func TestWireEverythingUp(t *testing.T) {
	results := make(chan string)

	var destinationURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			fmt.Fprintf(w, `
			{"destinations": [{
				"url": "%s",
				"aggregation_interval": "1s",
				"queries": [{ "filter": "has(message)" }]
			}]}`, destinationURL)
		case http.MethodPost:
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Log(err)
				t.Fail()
				return
			}
			results <- string(data)
		}
	}))
	defer srv.Close()
	destinationURL = srv.URL

	p := new(filter.Parser)
	filter.SetBuiltins(p)

	a := NewAggregator([]*config.Source{config.NewSource(srv.URL)}, p)
	go a.Run(t.Context())

	<-a.WaitForFirstLoad()

	hydrant.Log(hydrant.WithSubmitter(t.Context(), a), "hello")

	t.Log(<-results)
}
