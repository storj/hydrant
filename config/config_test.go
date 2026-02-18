package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/pp"
)

func TestConfig(t *testing.T) {
	var cfg Config
	assert.NoError(t, json.Unmarshal(exampleData, &cfg))
	pp.Println(cfg)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	assert.NoError(t, enc.Encode(cfg))
	fmt.Println(buf.String())

	assert.Equal(t, exampleData, buf.Bytes())
}

func BenchmarkFindKind(b *testing.B) {
	data := []byte(`{
		"kind": "filter",
		"filter": "equal(name, 'storj.io/storj/storagenode/piece/(*Piece).Commit') && exists(span_id)",
		"submitter": {
			"kind": "grouper",
			"flush_interval": "1m",
			"group_by": ["success"],
			"submitter": ["collectora", "mem"]
		}
	}`)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = findKind(data)
	}
}

func BenchmarkUnmarshalConfig(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		var cfg Config
		_ = json.Unmarshal(exampleData, &cfg)
	}
}

func BenchmarkMarshalConfig(b *testing.B) {
	var cfg Config
	_ = json.Unmarshal(exampleData, &cfg)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = json.Marshal(cfg)
	}
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
						"hyd",
						"prom"
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
		},
		"prom": {
			"kind": "prometheus",
			"namespace": "myapp",
			"buckets": [
				0.01,
				0.05,
				0.1,
				0.5,
				1,
				5
			]
		}
	}
}
`)
