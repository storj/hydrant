package hydrant

import (
	"fmt"
	"time"

	"github.com/histdb/histdb/flathist"
	"storj.io/hydrant/internal/rw"
	"storj.io/hydrant/value"
)

type Event []Annotation

func (ev Event) AppendTo(buf []byte) []byte {
	buf = rw.AppendVarint(buf, uint64(len(ev)))
	for _, a := range ev {
		buf = a.AppendTo(buf)
	}
	return buf
}

func (ev *Event) ReadFrom(buf []byte) ([]byte, error) {
	r := rw.NewReader(buf)
	count := r.ReadVarint()
	buf, err := r.Done()
	if err != nil {
		return nil, err
	}

	next := make(Event, count)
	for i := range next {
		buf, err = next[i].ReadFrom(buf)
		if err != nil {
			return nil, err
		}
	}
	*ev = next

	return buf, nil
}

type Annotation struct {
	Key   string
	Value value.Value
}

func (a Annotation) AppendTo(buf []byte) []byte {
	buf = rw.AppendVarint(buf, uint64(len(a.Key)))
	buf = rw.AppendString(buf, a.Key)
	buf = a.Value.AppendTo(buf)
	return buf
}

func (a *Annotation) ReadFrom(buf []byte) ([]byte, error) {
	r := rw.NewReader(buf)
	a.Key = string(r.ReadBytes(r.ReadVarint()))
	rem, err := r.Done()
	if err != nil {
		return nil, err
	}
	return a.Value.ReadFrom(rem)
}

func (a Annotation) String() string {
	switch a.Value.Kind() {
	case value.KindHistogram:
		h, _ := a.Value.Histogram()
		tot, sum, avg, vari := h.Summary()
		min, max := h.Min(), h.Max()
		return fmt.Sprintf(
			"%s=[tot:%d sum:%0.1f avg:%0.1f var:%0.1f min:%0.1f max:%0.1f]",
			a.Key, tot, sum, avg, vari, min, max)
	case value.KindTraceId:
		x, _ := a.Value.TraceId()
		return fmt.Sprintf("%s=%x", a.Key, x)
	case value.KindSpanId:
		x, _ := a.Value.SpanId()
		return fmt.Sprintf("%s=%x", a.Key, x)
	}
	return fmt.Sprintf("%s=%v", a.Key, a.Value.AsAny())
}

func String(key, val string) Annotation {
	return Annotation{Key: key, Value: value.String(val)}
}

func Bytes(key string, val []byte) Annotation {
	return Annotation{Key: key, Value: value.Bytes(val)}
}

func Histogram(key string, val *flathist.Histogram) Annotation {
	return Annotation{Key: key, Value: value.Histogram(val)}
}

func TraceId(key string, val [16]byte) Annotation {
	return Annotation{Key: key, Value: value.TraceId(val)}
}

func SpanId(key string, val [8]byte) Annotation {
	return Annotation{Key: key, Value: value.SpanId(val)}
}

func Int(key string, val int64) Annotation {
	return Annotation{Key: key, Value: value.Int(val)}
}

func Uint(key string, val uint64) Annotation {
	return Annotation{Key: key, Value: value.Uint(val)}
}

func Duration(key string, val time.Duration) Annotation {
	return Annotation{Key: key, Value: value.Duration(val)}
}

func Float(key string, val float64) Annotation {
	return Annotation{Key: key, Value: value.Float(val)}
}

func Bool(key string, val bool) Annotation {
	return Annotation{Key: key, Value: value.Bool(val)}
}

func Timestamp(key string, val time.Time) Annotation {
	return Annotation{Key: key, Value: value.Timestamp(val)}
}
