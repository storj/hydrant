package hydrant

import (
	"context"
	"testing"
)

func TestSetDefaultSubmitter(t *testing.T) {
	defer SetDefaultSubmitter(GetSubmitter(context.Background()))

	type t1 struct{ Submitter }
	type t2 struct{ Submitter }

	SetDefaultSubmitter(new(t1))
	_ = GetDefaultSubmitter().(*t1)

	SetDefaultSubmitter(new(t2))
	_ = GetDefaultSubmitter().(*t2)
}
