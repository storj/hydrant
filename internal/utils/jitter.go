package utils

import (
	"cmp"
	"math/rand/v2"
	"time"
)

func Bound[T cmp.Ordered](value T, bounds [2]T) T {
	return min(max(value, min(bounds[0], bounds[1])), max(bounds[0], bounds[1]))
}

func Jitter(v time.Duration) time.Duration {
	nanos := rand.NormFloat64()*float64(v/4) + float64(v)
	if nanos <= 0 {
		nanos = 1
	}
	return time.Duration(nanos)
}
