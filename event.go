package hydrant

import (
	"fmt"
	"time"

	"storj.io/hydrant/value"
)

type Event struct {
	System []Annotation
	User   []Annotation
}

type Annotation struct {
	Key   string
	Value value.Value
}

func (a Annotation) String() string {
	return fmt.Sprintf("%s=%v", a.Key, a.Value.AsAny())
}

func String(key, val string) Annotation {
	return Annotation{Key: key, Value: value.String(val)}
}

func Bytes(key string, val []byte) Annotation {
	return Annotation{Key: key, Value: value.Bytes(val)}
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
