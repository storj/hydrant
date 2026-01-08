package submitters

import (
	"context"
	"encoding/json/jsontext"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zeebo/assert"

	"github.com/histdb/histdb/flathist"

	"storj.io/hydrant"
)

func TestRemote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("config requested")
		w.Write([]byte(`{
			"refresh_interval": "10s",
			"submitter": {"kind": "hydrator"}
		}`))
	}))
	defer srv.Close()

	rem := NewRemoteSubmitter(Environment{}, srv.URL)
	go rem.Run(context.Background())
	rem.Trigger()

	for i := range 11 {
		hist := flathist.NewHistogram()
		hist.Observe(float32(i))
		rem.Submit(t.Context(), hydrant.Event{
			hydrant.String("name", "foo"),
			hydrant.Histogram("data", hist),
		})
	}

	rec := httptest.NewRecorder()
	rem.ServeHTTP(rec, httptest.NewRequest("GET", "/sub/query?q=name=foo&n=5&l=t", nil))

	body, err := io.ReadAll(rec.Result().Body)
	assert.NoError(t, err)
	(*jsontext.Value)(&body).Indent(jsontext.Multiline(false))
	t.Logf("response: %s", body)
}
