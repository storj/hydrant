package config

import (
	"context"
	"encoding/json"
	"io"
)

type jsonConfig struct {
	SourceConfig
	GlobalFields []Expression  `json:"global_fields"`
	Destinations []Destination `json:"destinations"`
}

func Parse(ctx context.Context, data io.Reader) (SourceConfig, []Destination, error) {
	var cfg jsonConfig
	if err := json.NewDecoder(data).Decode(&cfg); err != nil {
		return SourceConfig{}, nil, err
	}

	for i := range cfg.Destinations {
		cfg.Destinations[i].GlobalFields = append(cfg.Destinations[i].GlobalFields, cfg.GlobalFields...)
	}

	return cfg.SourceConfig, cfg.Destinations, nil
}
