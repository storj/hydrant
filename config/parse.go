package config

import (
	"context"
	"encoding/json"
	"io"
)

type jsonConfig struct {
	ConfigSourceConfig
	GlobalFields []Expression  `json:"global_fields"`
	Destinations []Destination `json:"destinations"`
}

func Parse(ctx context.Context, data io.Reader) (ConfigSourceConfig, []Destination, error) {
	var cfg jsonConfig
	if err := json.NewDecoder(data).Decode(&cfg); err != nil {
		return ConfigSourceConfig{}, nil, err
	}

	for i := range cfg.Destinations {
		cfg.Destinations[i].GlobalFields = append(cfg.Destinations[i].GlobalFields, cfg.GlobalFields...)
	}

	return cfg.ConfigSourceConfig, cfg.Destinations, nil
}
