package destination

import (
	"math/rand/v2"
	"time"
)

func jitter(v time.Duration) time.Duration {
	nanos := rand.NormFloat64()*float64(v/4) + float64(v)
	if nanos <= 0 {
		nanos = 1
	}
	return time.Duration(nanos)
}
