package submitters

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/zeebo/hmux"

	"github.com/histdb/histdb/flathist"
	"github.com/histdb/histdb/memindex"
	"github.com/histdb/histdb/query"
	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

type HydratorSubmitter struct {
	live liveBuffer

	stats struct {
		received atomic.Uint64
	}

	mu    sync.Mutex
	idx   memindex.T
	hists []*flathist.Histogram
}

func NewHydratorSubmitter() *HydratorSubmitter {
	return &HydratorSubmitter{live: newLiveBuffer()}
}

func (h *HydratorSubmitter) Children() []Submitter {
	return []Submitter{}
}

var hydratorSkipKinds = [...]bool{
	value.KindTraceId:   true,
	value.KindSpanId:    true,
	value.KindTimestamp: true,
	value.KindFloat:     true,
	value.KindDuration:  true,
}

func (h *HydratorSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	h.live.Record(ev)
	h.stats.received.Add(1)

	hasHist := false

	buf := make([]byte, 0, 64)
	for _, ann := range ev {
		if ann.Value.Kind() == value.KindHistogram {
			hasHist = true
			continue
		} else if hydratorSkipKinds[ann.Value.Kind()] {
			continue
		} else if strings.HasPrefix(ann.Key, "agg:") {
			continue
		} else if ann.Key == "_" {
			continue
		}

		buf = append(buf, ann.Key...)
		if ann.Value.Kind() != value.KindEmpty {
			buf = append(buf, '=')
		}

		switch ann.Value.Kind() {
		case value.KindString:
			x, _ := ann.Value.String()
			buf = append(buf, x...)

		case value.KindBytes:
			x, _ := ann.Value.Bytes()
			buf = append(buf, hex.EncodeToString(x)...)

		case value.KindInt:
			x, _ := ann.Value.Int()
			buf = strconv.AppendInt(buf, x, 10)

		case value.KindUint:
			x, _ := ann.Value.Uint()
			buf = strconv.AppendUint(buf, x, 10)

		case value.KindBool:
			x, _ := ann.Value.Bool()
			if x {
				buf = append(buf, "true"...)
			} else {
				buf = append(buf, "false"...)
			}
		}

		buf = append(buf, ',')
	}

	if !hasHist {
		metric := append(buf, '_')

		h.mu.Lock()
		_, id, _, created := h.idx.Add(metric, nil, nil)
		if created {
			h.hists = append(h.hists, flathist.NewHistogram())
		}
		into := h.hists[id]
		h.mu.Unlock()

		into.Observe(1)
		return
	}

	buf = append(buf, "_="...)

	for _, ann := range ev {
		x, ok := ann.Value.Histogram()
		if !ok {
			continue
		}
		metric := append(buf, ann.Key...)

		h.mu.Lock()
		_, id, _, created := h.idx.Add(metric, nil, nil)
		if created {
			h.hists = append(h.hists, flathist.NewHistogram())
		}
		into := h.hists[id]
		h.mu.Unlock()

		into.Merge(x)
	}
}

func (h *HydratorSubmitter) Query(q []byte, cb func(name []byte, hist *flathist.Histogram) bool) error {
	var qu query.Q
	if err := query.Parse(q, &qu); err != nil {
		return err
	}
	memindex.Iter(qu.Eval(&h.idx), func(id memindex.Id) bool {
		name, ok := h.idx.AppendNameById(id, nil)
		if !ok {
			return false
		}
		return cb(name, h.hists[id])
	})
	return nil
}

func (h *HydratorSubmitter) Keys(cb func([]byte) bool) bool {
	return h.idx.TagKeys(cb)
}

func (h *HydratorSubmitter) KeyValues(key []byte, cb func([]byte) bool) bool {
	return h.idx.TagValues(key, cb)
}

func (h *HydratorSubmitter) Annotations(cb func([]byte) bool) bool {
	return h.idx.Tags(cb)
}

func (h *HydratorSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree":  constJSONHandler(treeify(h)),
		"/live":  h.live.Handler(),
		"/query": http.HandlerFunc(h.queryHandler),
		"/stats": statsHandler(func() []stat {
			h.mu.Lock()
			metricsStored := uint64(len(h.hists))
			h.mu.Unlock()
			return []stat{
				{"received", h.stats.received.Load()},
				{"metrics_stored", metricsStored},
			}
		}),
	}
}

func (h *HydratorSubmitter) queryHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	q := []byte(r.Form.Get("q"))
	l := parseBool(r.Form.Get("l"), false)
	n := parseInt(r.Form.Get("n"), 20)
	e := parseInt(r.Form.Get("e"), 8)
	m := parseBool(r.Form.Get("m"), false)

	resp := queryResponse{
		Names: make([]string, 0),
		Data:  make([]histData, 0),
	}

	if m {
		into := flathist.NewHistogram()
		if err := h.Query(q, func(name []byte, hist *flathist.Histogram) bool {
			resp.Names = append(resp.Names, string(name))
			into.Merge(hist)
			return true
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(resp.Names) > 0 {
			resp.Data = append(resp.Data, generateHistogramResponse(l, n, e, into))
		}
	} else {
		if err := h.Query(q, func(name []byte, hist *flathist.Histogram) bool {
			resp.Names = append(resp.Names, string(name))
			resp.Data = append(resp.Data, generateHistogramResponse(l, n, e, hist))
			return true
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	enc.Encode(resp)
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

	Quantiles []histQuantile
}

type histQuantile struct {
	Q float64
	V float32
}

func generateHistogramResponse(l bool, n int, e int, hist *flathist.Histogram) histData {
	total, sum, avg, vari := hist.Summary()
	min, max := hist.Min(), hist.Max()

	resp := histData{
		Total:     total,
		Sum:       sum,
		Avg:       avg,
		Vari:      vari,
		Min:       min,
		Max:       max,
		Quantiles: make([]histQuantile, 0, n),
	}

	genQs(l, n, e, func(q float64) {
		resp.Quantiles = append(resp.Quantiles, histQuantile{
			Q: q,
			V: hist.Quantile(q),
		})
	})

	return resp
}

func genQs(l bool, n, e int, cb func(q float64)) {
	if l {
		for i := range n {
			cb(float64(i) / float64(n-1))
		}
	} else {
		t := int(math.Ceil(float64(n) / float64(e-1)))
		for i := range t {
			r := math.Pow(1.0/float64(e), float64(i))
			b := 1 - r
			d := r / float64(e)
			for j := range e - 1 {
				cb(b + d*float64(j))
			}
		}
		cb(1)
	}
}
