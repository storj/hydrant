package config

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"testing"
)

var (
	//go:embed "example.json"
	embedExampleJson []byte
)

func TestParse(t *testing.T) {
	sourceConfig, dests, err := Parse(context.Background(), bytes.NewReader(embedExampleJson))
	if err != nil {
		t.Fatalf("parse failure: %+v", err)
	}
	v, err := json.MarshalIndent(dests, "", "  ")
	if err != nil {
		t.Fatalf("serialization failure: %+v", err)
	}
	t.Logf("source config: %#v", sourceConfig)
	t.Logf("destinations: %s", v)
}
