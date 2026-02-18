// httpserver demonstrates the httputil middleware wrapping a real HTTP server.
// Spans are created automatically for each request with method, path, status,
// and response size annotations.
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
	"storj.io/hydrant/utils/httputil"
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

	// Application routes wrapped with hydrant middleware.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/users", handleUsers)
	mux.HandleFunc("GET /api/slow", handleSlow)
	mux.HandleFunc("GET /api/error", handleError)

	// Mount the web UI and the instrumented app on different paths.
	http.Handle("/ui/", http.StripPrefix("/ui", sub.Handler()))
	http.Handle("/", httputil.Wrap(mux))

	// Drive some traffic so there's data to look at.
	go driveTraffic()

	fmt.Println("app at    http://localhost:9912/api/users")
	fmt.Println("web UI at http://localhost:9912/ui/")
	panic(http.ListenAndServe(":9912", nil))
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Duration(rand.IntN(20)) * time.Millisecond)
	fmt.Fprintln(w, `[{"id":1,"name":"alice"},{"id":2,"name":"bob"}]`)
}

func handleSlow(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Duration(rand.IntN(500)+200) * time.Millisecond)
	fmt.Fprintln(w, `{"status":"ok"}`)
}

func handleError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, `{"error":"something broke"}`)
}

func driveTraffic() {
	paths := []string{"/api/users", "/api/users", "/api/users", "/api/slow", "/api/error"}
	for {
		time.Sleep(50 * time.Millisecond)
		path := paths[rand.IntN(len(paths))]
		resp, err := http.Get("http://localhost:9912" + path)
		if err == nil {
			resp.Body.Close()
		}
	}
}
