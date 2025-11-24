package hydrant_test

import (
	"context"
	"testing"

	"storj.io/hydrant"
	"storj.io/hydrant/process"
	"storj.io/hydrant/protocol"
)

func TestDefaultSubmitter(t *testing.T) {
	originalDefault := hydrant.DefaultSubmitter
	defer func() { hydrant.DefaultSubmitter = originalDefault }()

	hydrant.DefaultSubmitter = protocol.NewHTTPSubmitter("http://localhost:1/",
		process.NewSelected(process.DefaultStore, nil))
	submitter2 := protocol.NewHTTPSubmitter("http://localhost:2/",
		process.NewSelected(process.DefaultStore, nil))

	// make sure by default the submitter is the default submitter
	if hydrant.GetSubmitter(context.Background()) != hydrant.DefaultSubmitter {
		t.Fatalf("unexpected submitter")
	}
	// make sure by submitter is right when ctx is a *contextSpan
	if hydrant.GetSubmitter(hydrant.WithSubmitter(context.Background(), submitter2)) != submitter2 {
		t.Fatalf("unexpected submitter")
	}
	// make sure by submitter is right when ctx is not a *contextSpan
	if hydrant.GetSubmitter(context.WithValue(hydrant.WithSubmitter(context.Background(), submitter2), "key", "value")) != submitter2 {
		t.Fatalf("unexpected submitter")
	}
	// make sure the default submitter can be overridden with nil when ctx is a *contextSpan
	if hydrant.GetSubmitter(hydrant.WithSubmitter(context.Background(), nil)) != nil {
		t.Fatalf("unexpected submitter")
	}
	// make sure the default submitter can be overridden with nil when ctx is not a *contextSpan
	if hydrant.GetSubmitter(context.WithValue(hydrant.WithSubmitter(context.Background(), nil), "key", "value")) != nil {
		t.Fatalf("unexpected submitter")
	}
}
