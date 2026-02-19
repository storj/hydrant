package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zeebo/assert"

	"github.com/histdb/histdb/flathist"

	"storj.io/hydrant"
	"storj.io/hydrant/process"
	"storj.io/hydrant/submitters"
)

func TestProtocol(t *testing.T) {
	var exp, got loggingSub

	done := make(chan struct{})
	handler := NewReceiver(&got)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
		defer func() { recover() }() // in case a race and we got two calls
		close(done)
	}))
	defer srv.Close()

	sel := process.DefaultStore.Select([]string{
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

	hsub := submitters.NewHTTPSubmitter(srv.URL, sel, time.Minute, 10_000)
	go hsub.Run(t.Context())

	ctx := hydrant.WithSubmitter(t.Context(), submitters.NewMultiSubmitter(
		hsub,
		&exp,
	))

	hist := flathist.NewHistogram()
	for range 10 {
		hist.Observe(1)
		hydrant.Log(ctx, "some test log",
			hydrant.String("example.key", "example.value"),
			hydrant.Int("example.number", 42),
			hydrant.Histogram("example.hist", hist.Clone()),
		)
	}

	hsub.Trigger()
	<-done

	assert.Equal(t, len(exp), len(got))
}

type loggingSub []hydrant.Event

func (l *loggingSub) Submit(ctx context.Context, ev hydrant.Event) { *l = append(*l, ev) }
func (l *loggingSub) Children() []submitters.Submitter             { return nil }
func (l *loggingSub) Handler() http.Handler                        { return nil }
func (l *loggingSub) ExtraData() any                               { return nil }
