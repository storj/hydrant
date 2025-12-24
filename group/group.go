package group

import (
	"encoding/binary"
	"slices"
	"sync/atomic"
	"unique"

	"storj.io/hydrant"
	"storj.io/hydrant/utils"
)

type Grouper struct {
	keys  []string
	hints []atomic.Uint32
	set   map[string]struct{}
}

func NewGrouper(keys []string) *Grouper {
	g := &Grouper{
		keys:  slices.Sorted(slices.Values(keys)),
		hints: make([]atomic.Uint32, len(keys)),
		set:   utils.Set(slices.Values(keys)),
	}
	return g
}

func (g *Grouper) Group(ev hydrant.Event) unique.Handle[string] {
	buf := make([]byte, 0, 256)

	for i, key := range g.keys {
		if h := g.hints[i].Load(); h != 0 {
			h >>= 1
			if h < uint32(len(ev)) && ev[h].Key == key {
				buf = appendString(buf, ev[h].Key)
				buf = ev[h].Value.AppendTo(buf)
				continue
			}
		}

		for j := len(ev) - 1; j >= 0; j-- {
			if ev[j].Key == key {
				buf = appendString(buf, ev[j].Key)
				buf = ev[j].Value.AppendTo(buf)
				g.hints[i].Store(uint32(j)<<1 | 1)
				continue
			}
		}
	}

	return unique.Make(string(buf))
}

func (g *Grouper) Annotations(ev hydrant.Event) []hydrant.Annotation {
	var out []hydrant.Annotation

	for i, key := range g.keys {
		if h := g.hints[i].Load(); h != 0 {
			h >>= 1
			if h < uint32(len(ev)) && ev[h].Key == key {
				out = append(out, ev[h])
				continue
			}
		}

		for j := len(ev) - 1; j >= 0; j-- {
			if ev[j].Key == key {
				out = append(out, ev[j])
				g.hints[i].Store(uint32(j)<<1 | 1)
				continue
			}
		}
	}

	return out
}

func appendString(buf []byte, s string) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], uint64(len(s)))
	buf = append(buf, tmp[:]...)
	buf = append(buf, s...)
	return buf
}
