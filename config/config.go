package config

import (
	"encoding/json"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	dur, err := time.ParseDuration(v)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

type Expression string

func (e Expression) String() string { return string(e) }

type Query struct {
	Filter        Expression   `json:"filter"`
	GroupBy       []Expression `json:"group_by"`
	AggregateOver []Expression `json:"aggregate_over"`
	Aggregates    []Expression `json:"aggregates"`
}

type Destination struct {
	URL                 string       `json:"url"`
	AggregationInterval Duration     `json:"aggregation_interval"`
	GlobalFields        []Expression `json:"global_fields"`
	Queries             []Query      `json:"queries"`
}
