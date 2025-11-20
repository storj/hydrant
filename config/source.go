package config

import (
	"context"
	"net/http"
)

type ConfigSourceConfig struct {
	RefreshInterval Duration `json:"refresh_interval"`
}

type ConfigSource struct {
	url string
}

func NewConfigSource(url string) *ConfigSource {
	return &ConfigSource{
		url: url,
	}
}

func (cs *ConfigSource) Load(ctx context.Context) (cfg ConfigSourceConfig, dsts []Destination, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cs.url, nil)
	if err != nil {
		return cfg, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return cfg, nil, err
	}
	defer resp.Body.Close()

	cfg, dsts, err = Parse(ctx, resp.Body)
	if err != nil {
		return cfg, nil, err
	}

	return cfg, dsts, resp.Body.Close()
}
