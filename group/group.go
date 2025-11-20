package group

import (
	"encoding/binary"
	"iter"
	"maps"
	"slices"
	"sync/atomic"
	"unique"

	"storj.io/hydrant"
)

type Grouper struct {
	keys  []string
	hints [3][]atomic.Uint32
	set   map[string]struct{}
	fn    func(hydrant.Event) unique.Handle[string]
}

func NewGrouper(keys []string, exclude bool) *Grouper {
	g := &Grouper{
		keys: slices.Sorted(slices.Values(keys)),
		hints: [3][]atomic.Uint32{
			make([]atomic.Uint32, len(keys)),
			make([]atomic.Uint32, len(keys)),
			make([]atomic.Uint32, len(keys)),
		},
		set: maps.Collect(seq2seq2(slices.Values(keys), struct{}{})),
	}
	if exclude {
		g.fn = g.groupExclude
	} else {
		g.fn = g.groupInclude
	}
	return g
}

func (g *Grouper) Group(ev hydrant.Event) unique.Handle[string] {
	return g.fn(ev)
}

func (g *Grouper) groupInclude(ev hydrant.Event) unique.Handle[string] {
	buf := make([]byte, 0, 256)

	lookup := func(anns []hydrant.Annotation, class int, i int, key string) bool {
		if h := g.hints[class][i].Load(); h != 0 {
			h >>= 1
			if h < uint32(len(anns)) && anns[h].Key == key {
				buf = appendString(buf, anns[h].Key)
				buf = anns[h].Value.Serialize(buf)
				return true
			}
		}

		for j := len(anns) - 1; j >= 0; j-- {
			if anns[j].Key == key {
				buf = appendString(buf, anns[j].Key)
				buf = anns[j].Value.Serialize(buf)
				g.hints[class][i].Store(uint32(j)<<1 | 1)
				return true
			}
		}

		return false
	}

	for i, key := range g.keys {
		_ = lookup(ev.System, 0, i, key) || lookup(ev.User, 1, i, key)
	}

	return unique.Make(string(buf))
}

func (g *Grouper) groupExclude(ev hydrant.Event) unique.Handle[string] {
	buf := make([]byte, 0, 256)

	for i := range ev.System {
		if _, ok := g.set[ev.System[i].Key]; !ok {
			buf = appendString(buf, ev.System[i].Key)
			buf = ev.System[i].Value.Serialize(buf)
		}
	}

	for i := range ev.User {
		if _, ok := g.set[ev.User[i].Key]; !ok {
			buf = appendString(buf, ev.User[i].Key)
			buf = ev.User[i].Value.Serialize(buf)
		}
	}

	return unique.Make(string(buf))
}

func appendString(buf []byte, s string) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], uint64(len(s)))
	buf = append(buf, tmp[:]...)
	buf = append(buf, s...)
	return buf
}

func seq2seq2[S, T any](s iter.Seq[S], v T) iter.Seq2[S, T] {
	return func(yield func(S, T) bool) {
		for x := range s {
			if !yield(x, v) {
				return
			}
		}
	}
}
