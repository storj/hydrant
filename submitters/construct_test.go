package submitters

import (
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/hydrant/config"
	"storj.io/hydrant/filter"
	"storj.io/hydrant/process"
)

func TestConstruct(t *testing.T) {
	var cfg config.Config
	assert.NoError(t, json.Unmarshal(exampleData, &cfg))

	sub, err := Environment{
		Filter:  filter.NewBuiltinEnvionment(),
		Process: process.DefaultStore,
	}.New(cfg)
	assert.NoError(t, err)

	srv := httptest.NewServer(sub.Handler())
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/tree")
	assert.NoError(t, err)
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	data, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	(*jsontext.Value)(&data).Indent()
	fmt.Println(string(data))
}

var exampleData = []byte(`{
	"refresh_interval": "10m0s",
	"submitter": "default",
	"submitters": {
		"collectora": {
			"kind": "http",
			"process_fields": [
				"foo"
			],
			"endpoint": "http://example.com",
			"flush_interval": "10m0s",
			"max_batch_size": 10000
		},
		"default": [
			{
				"kind": "filter",
				"filter": "eq(name, 'storj.io/storj/storagenode/piece/(*Piece).Commit') && has(span_id)",
				"submitter": {
					"kind": "grouper",
					"flush_interval": "1m0s",
					"group_by": [
						"success"
					],
					"submitter": [
						"collectora",
						"null",
						"hyd",
						"jaeger"
					]
				}
			},
			{
				"kind": "filter",
				"filter": "eq(name, 'storj.io/storj/storagenode')",
				"submitter": {
					"kind": "grouper",
					"flush_interval": "10m0s",
					"group_by": [
						"name"
					],
					"submitter": [
						"collectora",
						"hyd"
					]
				}
			},
			{
				"kind": "filter",
				"filter": "has(message) && eq(name, 'storj.io/storj/storagenode')",
				"submitter": "null"
			}
		],
		"hyd": {
			"kind": "hydrator"
		},
		"jaeger": {
			"kind": "otel",
			"endpoint": "http://localhost:4318",
			"flush_interval": "5s",
			"max_batch_size": 1000
		},
		"null": {
			"kind": "null"
		}
	}
}`)
