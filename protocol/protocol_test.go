package protocol

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/histdb/histdb/flathist"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/process"
)

func TestProtocol(t *testing.T) {
	done := make(chan struct{})
	handler := NewHTTPHandler(new(loggingSub))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
		defer func() { recover() }() // in case a race and we got two calls
		close(done)
	}))
	defer srv.Close()

	sel := process.NewSelected(process.DefaultStore, []config.Expression{
		"go.os",
		"go.arch",
		"go.version",
		"go.main.path",
		"go.main.version",
		"go.main.sum",
		"vcs.time",
		"vcs.revision",
		"vcs.modified",
		"proc.starttime",
		"os.hostname",
		"os.ip",
	})

	sub := NewHTTPSubmitter(srv.URL, sel)
	go sub.Run(t.Context())

	ctx := hydrant.WithSubmitter(t.Context(), sub)

	hist := flathist.NewHistogram()
	for range 10 {
		hist.Observe(1)
		hydrant.Log(ctx, "some test log",
			hydrant.String("example.key", "example.value"),
			hydrant.Int("example.number", 42),
			hydrant.Histogram("example.hist", hist.Clone()),
		)
	}

	sub.Trigger()
	<-done
}

type loggingSub struct{}

func (*loggingSub) Submit(ctx context.Context, ev hydrant.Event) {
	fmt.Printf("event: %v\n", ev)
}
