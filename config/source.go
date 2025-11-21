package config

import (
	"context"
	"net/http"

	"github.com/zeebo/errs/v2"
)

type SourceConfig struct {
	RefreshInterval Duration `json:"refresh_interval"`
}

type Source struct {
	url string
}

func NewSource(url string) *Source {
	return &Source{
		url: url,
	}
}

func (cs *Source) Load(ctx context.Context) (cfg SourceConfig, dsts []Destination, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cs.url, nil)
	if err != nil {
		return cfg, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return cfg, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return cfg, nil, errs.Errorf("unexpected http response %d", resp.StatusCode)
	}

	cfg, dsts, err = Parse(ctx, resp.Body)
	if err != nil {
		return cfg, nil, err
	}

	return cfg, dsts, resp.Body.Close()
}
