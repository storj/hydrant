package submitters

import (
	"github.com/zeebo/errs/v2"

	"storj.io/hydrant/config"
)

type constructor struct {
	env      Environment
	named    map[string]*lateSubmitter
	runnable []runnable
}

func newConstructor(env Environment, named map[string]*lateSubmitter) *constructor {
	return &constructor{
		env:   env,
		named: named,
	}
}

func (c *constructor) Runnable() []runnable {
	return c.runnable
}

func (c *constructor) Construct(cfg config.Submitter) (Submitter, error) {
	switch cfg := cfg.(type) {
	case config.MultiSubmitter:
		subs := make([]Submitter, 0, len(cfg))
		for _, scfg := range cfg {
			sub, err := c.Construct(scfg)
			if err != nil {
				return nil, err
			}
			subs = append(subs, sub)
		}
		return NewMultiSubmitter(
			subs...,
		), nil

	case config.NamedSubmitter:
		sub, exists := c.named[string(cfg)]
		if !exists {
			return nil, errs.Errorf("unknown submitter name %q", string(cfg))
		}
		return sub, nil

	case config.FilterSubmitter:
		fil, err := c.env.Filter.Parse(cfg.Filter)
		if err != nil {
			return nil, err
		}
		sub, err := c.Construct(cfg.Submitter)
		if err != nil {
			return nil, err
		}
		return NewFilterSubmitter(
			fil,
			sub,
		), nil

	case config.GrouperSubmitter:
		sub, err := c.Construct(cfg.Submitter)
		if err != nil {
			return nil, err
		}

		gs := NewGrouperSubmitter(
			cfg.GroupBy,
			cfg.FlushInterval,
			sub,
		)
		c.runnable = append(c.runnable, gs)

		return gs, nil

	case config.HTTPSubmitter:
		hs := NewHTTPSubmitter(
			cfg.Endpoint,
			c.env.Process.Select(cfg.ProcessFields),
			cfg.FlushInterval,
			cfg.MaxBatchSize,
		)
		c.runnable = append(c.runnable, hs)

		return hs, nil

	case config.OTelSubmitter:
		os := NewOTelSubmitter(
			cfg.Endpoint,
			c.env.Process.Select(cfg.ProcessFields),
			cfg.FlushInterval,
			cfg.MaxBatchSize,
		)
		c.runnable = append(c.runnable, os)

		return os, nil

	case config.PrometheusSubmitter:
		return NewPrometheusSubmitter(
			cfg.Namespace,
			cfg.Buckets,
		), nil

	case config.HydratorSubmitter:
		return NewHydratorSubmitter(), nil

	case config.NullSubmitter:
		return NewNullSubmitter(), nil

	default:
		return nil, errs.Errorf("unknown submitter type %T", cfg)
	}
}
