package aggregator

import (
	"time"

	"storj.io/hydrant"
)

type Query struct {
	Filters       Filter
	Aggregates    []Aggregate
	GroupBy       []string
	AggregateOver []string
}

type Destination struct {
	Address             string
	AggregationInterval time.Duration
	MaxMemoryUsage      any // TODO
	Queries             []Query
}

type Config struct {
	RefreshInterval time.Duration
	GlobalFields    []string
	Destinations    []Destination
}

type ConfigSource interface {
	Load() (Config, error)
}

type Aggregator struct {
	cfgs []ConfigSource
}

var _ hydrant.Submitter = (*Aggregator)(nil)

func NewAggregator(cfgs []ConfigSource) *Aggregator {
	return &Aggregator{
		cfgs: cfgs,
	}
}

func (a *Aggregator) Submit(ev hydrant.Event) {
}
