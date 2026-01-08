package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zeebo/hmux"
	"github.com/zeebo/mwc"

	"storj.io/hydrant"
	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/process"
	"storj.io/hydrant/receiver"
	"storj.io/hydrant/submitters"
)

func main() {
	sub, err := submitters.Environment{
		Filter:  filter.NewBuiltinEnvionment(),
		Process: process.DefaultStore,
	}.New(config.Config{
		RefreshInterval: 10 * time.Minute,
		Submitter:       config.NamedSubmitter("default"),
		Submitters: map[string]config.Submitter{
			"collectora": config.HTTPSubmitter{
				ProcessFields: []string{
					"proc.starttime",
					"go.os",
					"go.arch",
				},
				Endpoint:      "http://localhost:9912/receive",
				FlushInterval: 1 * time.Minute,
				MaxBatchSize:  10000,
			},

			"hydrator": config.HydratorSubmitter{},

			"sink": config.MultiSubmitter{
				config.NamedSubmitter("collectora"),
				config.NamedSubmitter("hydrator"),
			},

			"default": config.MultiSubmitter{
				config.FilterSubmitter{
					Filter: "eq(key(name), test_one)",
					Submitter: config.GrouperSubmitter{
						FlushInterval: 10 * time.Second,
						GroupBy:       []string{"name", "group_a", "group_b"},
						Submitter:     config.NamedSubmitter("sink"),
					},
				},

				config.FilterSubmitter{
					Filter: "eq(key(name), test_two)",
					Submitter: config.GrouperSubmitter{
						FlushInterval: 10 * time.Second,
						GroupBy:       []string{"name", "group_a"},
						Submitter:     config.NamedSubmitter("sink"),
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("config:", func() string {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "\t")
		if err := enc.Encode(sub.Config()); err != nil {
			panic(err)
		}
		return buf.String()
	}())

	go func() {
		for range time.NewTicker(100 * time.Millisecond).C {
			sub.Submit(context.Background(), hydrant.Event{
				hydrant.String("name", "test_one"),
				hydrant.String("group_a", "foo"),
				hydrant.String("group_b", "bar"),
				hydrant.Float("value", mwc.Float64()),
			})

			sub.Submit(context.Background(), hydrant.Event{
				hydrant.String("name", "test_one"),
				hydrant.String("group_a", "foo"),
				hydrant.String("group_b", "baz"),
				hydrant.Float("value", mwc.Float64()),
			})

			sub.Submit(context.Background(), hydrant.Event{
				hydrant.String("name", "test_two"),
				hydrant.String("group_a", "foo"),
				hydrant.String("group_b", "bar"),
				hydrant.Float("value", mwc.Float64()),
			})

			sub.Submit(context.Background(), hydrant.Event{
				hydrant.String("name", "test_two"),
				hydrant.String("group_a", "foo"),
				hydrant.String("group_b", "baz"),
				hydrant.Float("value", mwc.Float64()),
			})
		}
	}()

	go sub.Run(context.Background())

	panic(http.ListenAndServe(":9912", hmux.Dir{
		"*":        sub.Handler(),
		"/receive": receiver.NewHTTPHandler(new(loggingSubmitter)),
	}))
}

type loggingSubmitter struct{}

func (l *loggingSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	fmt.Println("event received:", ev)
}
