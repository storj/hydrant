package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"net/http"
	"strconv"

	"github.com/zeebo/hmux"

	"github.com/histdb/histdb/flathist"

	"storj.io/hydrant"
	"storj.io/hydrant/protocol"
)

//go:embed static
var static embed.FS

func NewHandler(mem *MemStore) http.Handler {
	return hmux.Dir{
		"*": http.FileServerFS(func() fs.FS {
			sub, _ := fs.Sub(static, "static")
			return sub
		}()),

		"/api": hmux.Dir{
			"/query": hmux.Method{
				"GET": newQueryHandler(mem),
			},
			"/keys": hmux.Method{
				"GET": newKeysHandler(mem),
			},
			"/values": hmux.Method{
				"GET": newValuesHandler(mem),
			},
			"/annotations": hmux.Method{
				"GET": newAnnotationsHandler(mem),
			},
			"/collect": hmux.Method{
				"POST": protocol.NewHTTPHandler(mem),
			},
		},
	}
}

func newQueryHandler(mem *MemStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := hydrant.WithSubmitter(r.Context(), mem)

		ctx, span := hydrant.StartSpanNamed(ctx, "query_handler")
		defer span.Done(nil)

		hist := flathist.NewHistogram()
		hist.Observe(10)
		span.Annotate(hydrant.Histogram("histogram", hist))

		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		q := []byte(r.Form.Get("q")) // ugh []byte lol
		l := parseBool(r.Form.Get("l"), false)
		n := parseInt(r.Form.Get("n"), 20)
		e := parseInt(r.Form.Get("e"), 8)
		m := parseBool(r.Form.Get("m"), true)

		resp := queryResponse{
			Names: make([]string, 0),
			Data:  make([]histData, 0),
		}

		if m {
			into := flathist.NewHistogram()
			if err := mem.Query(q, func(name []byte, hist *flathist.Histogram) bool {
				resp.Names = append(resp.Names, string(name))
				into.Merge(hist)
				return true
			}); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if len(resp.Names) > 0 {
				resp.Data = append(resp.Data, generateHistogramResponse(ctx, l, n, e, into))
			}
		} else {
			if err := mem.Query(q, func(name []byte, hist *flathist.Histogram) bool {
				resp.Names = append(resp.Names, string(name))
				resp.Data = append(resp.Data, generateHistogramResponse(ctx, l, n, e, hist))
				return true
			}); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		enc.Encode(resp)
	})
}

func parseBool(s string, def bool) bool {
	x, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return x
}

func parseInt(s string, def int) int {
	x, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return x
}

type queryResponse struct {
	Names []string
	Data  []histData
}

type histData struct {
	Total uint64
	Sum   float64
	Avg   float64
	Vari  float64
	Min   float32
	Max   float32

	MinPrecision int

	Quantiles []histQuantile
}

type histQuantile struct {
	Q float64
	V float32
}

func generateHistogramResponse(ctx context.Context, l bool, n int, e int, hist *flathist.Histogram) histData {
	ctx, span := hydrant.StartSpan(ctx)
	defer span.Done(nil)

	total, sum, avg, vari := hist.Summary()
	min, max := hist.Min(), hist.Max()

	resp := histData{
		Total: uint64(total),
		Sum:   sum,
		Avg:   avg,
		Vari:  vari,
		Min:   min,
		Max:   max,

		MinPrecision: minPrec(l, n, e),
	}

	genQs(l, n, e, func(q float64) {
		resp.Quantiles = append(resp.Quantiles, histQuantile{
			Q: q,
			V: hist.Quantile(q),
		})
	})

	return resp
}

func minPrec(l bool, n, e int) int {
	for i := 0; ; i++ {
		got := make(map[string]struct{})
		f := fmt.Sprintf("%%0.%df", i)
		dup := false
		genQs(l, n, e, func(q float64) {
			v := fmt.Sprintf(f, q)
			if _, ok := got[v]; ok {
				dup = true
			}
			got[v] = struct{}{}
		})
		if !dup || i >= 16 {
			return i
		}
	}
}

func genQs(l bool, n, e int, cb func(q float64)) {
	if l {
		for q := 0.0; q < 1.0-(1/float64(n)); q += (1 / float64(n)) {
			cb(q)
		}
	} else {
		t := int(math.Ceil(float64(n) / float64(e-1)))
		for i := 0; i < t; i++ {
			r := math.Pow(1.0/float64(e), float64(i))
			b := 1 - r
			d := r / float64(e)
			for j := 0; j < e-1; j++ {
				cb(b + d*float64(j))
			}
		}
	}
	cb(1)
}

func newKeysHandler(mem *MemStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := hydrant.WithSubmitter(r.Context(), mem)

		ctx, span := hydrant.StartSpanNamed(ctx, "keys_handler")
		defer span.Done(nil)

		keys := make([]string, 0)
		mem.Keys(func(key []byte) bool {
			keys = append(keys, string(key))
			return true
		})

		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		enc.Encode(keys)
	})
}

func newValuesHandler(mem *MemStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := hydrant.WithSubmitter(r.Context(), mem)

		ctx, span := hydrant.StartSpanNamed(ctx, "values_handler")
		defer span.Done(nil)

		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		key := []byte(r.Form.Get("key"))
		values := make([]string, 0)
		mem.KeyValues(key, func(value []byte) bool {
			values = append(values, string(value))
			return true
		})

		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		enc.Encode(values)
	})
}

func newAnnotationsHandler(mem *MemStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := hydrant.WithSubmitter(r.Context(), mem)

		ctx, span := hydrant.StartSpanNamed(ctx, "annotations_handler")
		defer span.Done(nil)

		annotations := make([]string, 0)
		mem.Annotations(func(annotation []byte) bool {
			annotations = append(annotations, string(annotation))
			return true
		})

		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		enc.Encode(annotations)
	})
}
