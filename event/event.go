package event

import (
	"hydrant/value"
	"time"
)

type Event struct {
	System []Annotation
	User   []Annotation
}

type Annotation struct {
	Key   string
	Value value.Value
}

func OfString(key, val string) Annotation {
	return Annotation{Key: key, Value: value.OfString(val)}
}

func OfBytes(key string, val []byte) Annotation {
	return Annotation{Key: key, Value: value.OfBytes(val)}
}

func OfInt(key string, val int64) Annotation {
	return Annotation{Key: key, Value: value.OfInt(val)}
}

func OfUint(key string, val uint64) Annotation {
	return Annotation{Key: key, Value: value.OfUint(val)}
}

func OfDuration(key string, val time.Duration) Annotation {
	return Annotation{Key: key, Value: value.OfDuration(val)}
}

func OfFloat(key string, val float64) Annotation {
	return Annotation{Key: key, Value: value.OfFloat(val)}
}

func OfBool(key string, val bool) Annotation {
	return Annotation{Key: key, Value: value.OfBool(val)}
}

func OfTimestamp(key string, val time.Time) Annotation {
	return Annotation{Key: key, Value: value.OfTimestamp(val)}
}

func OfAny(key string, val any) Annotation {
	return Annotation{Key: key, Value: value.OfAny(val)}
}
