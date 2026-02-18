package httputil

import (
	"bufio"
	"errors"
	"net"
	"net/http"

	"storj.io/hydrant"
)

// Handler wraps an http.Handler with span-based instrumentation.
type Handler struct {
	// Name returns the span name for a request. If nil, defaults to
	// r.Method + "  " + r.URL.Path (e.g. "GET /path").
	Name func(r *http.Request) string

	// Handler is the wrapped http.Handler.
	Handler http.Handler
}

func Wrap(h http.Handler) *Handler {
	return &Handler{Handler: h}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var name string
	if h.Name != nil {
		name = h.Name(r)
	} else {
		name = r.Method + " " + r.URL.Path
	}

	traceId, parentId := ExtractTraceparent(r)
	ctx, span := hydrant.StartRemoteSpanNamed(r.Context(), name, parentId, traceId,
		hydrant.String("http.method", r.Method),
		hydrant.String("http.path", r.URL.Path),
		hydrant.String("http.remote_addr", r.RemoteAddr),
	)
	r = r.WithContext(ctx)

	rw := &responseWriter{w: w}

	defer func() {
		var err error
		defer span.Done(&err)

		span.Annotate(
			hydrant.Int("http.status_code", int64(rw.status)),
			hydrant.Int("http.response_size", int64(rw.written)),
		)

		if rw.status >= 500 {
			err = errStatus
		}

		if v := recover(); v != nil {
			span.Annotate(hydrant.Bool("http.panic", true))
			panic(v)
		}
	}()

	h.Handler.ServeHTTP(rw, r)
}

var errStatus = errors.New("5xx status")

type responseWriter struct {
	w       http.ResponseWriter
	status  int
	written int64
	wroteH  bool
}

func (rw *responseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteH {
		rw.status = code
		rw.wroteH = true
	}
	rw.w.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteH {
		rw.status = 200
		rw.wroteH = true
	}
	n, err := rw.w.Write(b)
	rw.written += int64(n)
	return n, err
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.w.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("underlying ResponseWriter does not support Hijack")
}

// Unwrap returns the underlying http.ResponseWriter.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.w
}
