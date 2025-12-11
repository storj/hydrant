package hydrant

import (
	"fmt"
	"time"

	"github.com/histdb/histdb/flathist"
	"storj.io/hydrant/value"
)

type Event []Annotation

type Annotation struct {
	Key   string
	Value value.Value
}

func (a Annotation) String() string {
	if h, ok := a.Value.Histogram(); ok {
		tot, sum, avg, vari := h.Summary()
		return fmt.Sprintf("%s=[tot:%v sum:%v avg:%v var:%v]", a.Key, tot, sum, avg, vari)
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

func Identifier(key string, val uint64) Annotation {
	return Annotation{Key: key, Value: value.Identifier(val)}
}
