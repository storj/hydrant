package main

import (
	"context"
	"encoding/hex"
	"strconv"
	"sync"

	"github.com/histdb/histdb/flathist"
	"github.com/histdb/histdb/memindex"
	"github.com/histdb/histdb/query"
	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

type MemStore struct {
	mu    sync.Mutex
	idx   memindex.T
	hists []*flathist.Histogram
}

func (m *MemStore) Index() *memindex.T                      { return &m.idx }
func (m *MemStore) Histogram(id uint32) *flathist.Histogram { return m.hists[id] }

func (m *MemStore) Submit(ctx context.Context, ev hydrant.Event) {
	hasHist := false

	buf := make([]byte, 0, 64)
	for _, ann := range ev {
		if ann.Value.Kind() == value.KindHistogram {
			hasHist = true
			continue
		}

		buf = append(buf, ann.Key...)
		buf = append(buf, '=')

		switch ann.Value.Kind() {
		case value.KindEmpty:

		case value.KindString:
			x, _ := ann.Value.String()
			buf = append(buf, x...)

		case value.KindBytes:
			x, _ := ann.Value.Bytes()
			buf = append(buf, hex.EncodeToString(x)...)

		case value.KindHistogram:
			// protected above

		case value.KindInt:
			x, _ := ann.Value.Int()
			buf = strconv.AppendInt(buf, x, 10)

		case value.KindUint:
			x, _ := ann.Value.Uint()
			buf = strconv.AppendUint(buf, x, 10)

		case value.KindDuration:
			x, _ := ann.Value.Duration()
			buf = append(buf, x.String()...)

		case value.KindFloat:
			x, _ := ann.Value.Float()
			buf = strconv.AppendFloat(buf, x, 'g', -1, 64)

		case value.KindBool:
			x, _ := ann.Value.Bool()
			if x {
				buf = append(buf, "true"...)
			} else {
				buf = append(buf, "false"...)
			}

		case value.KindTimestamp:
			x, _ := ann.Value.Timestamp()
			buf = append(buf, x.String()...)

		case value.KindIdentifier:
			x, _ := ann.Value.Identifier()
			buf = strconv.AppendUint(buf, x, 10)
		}

		buf = append(buf, ',')
	}

	if !hasHist {
		return
	}

	for _, ann := range ev {
		x, ok := ann.Value.Histogram()
		if !ok {
			continue
		}
		metric := append(buf, ann.Key...)

		m.mu.Lock()
		_, id, _, created := m.idx.Add(metric, nil, nil)
		if created {
			m.hists = append(m.hists, flathist.NewHistogram())
		}
		into := m.hists[id]
		m.mu.Unlock()

		into.Merge(x)
	}
}

func (m *MemStore) Query(q []byte, cb func(name []byte, hist *flathist.Histogram) bool) error {
	var qu query.Q
	if err := query.Parse(q, &qu); err != nil {
		return err
	}
	memindex.Iter(qu.Eval(&m.idx), func(id memindex.Id) bool {
		name, ok := m.idx.AppendNameById(id, nil)
		if !ok {
			return false
		}
		return cb(name, m.hists[id])
	})
	return nil
}

func (m *MemStore) Keys(cb func([]byte) bool) bool {
	return m.idx.TagKeys(cb)
}

func (m *MemStore) KeyValues(key []byte, cb func([]byte) bool) bool {
	return m.idx.TagValues(key, cb)
}

func (m *MemStore) Annotations(cb func([]byte) bool) bool {
	return m.idx.Tags(cb)
}
