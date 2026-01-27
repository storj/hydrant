package hydrant

import "testing"

func TestSetDefaultSubmitter(t *testing.T) {
	type t1 struct{ Submitter }
	type t2 struct{ Submitter }

	SetDefaultSubmitter(new(t1))
	_ = GetDefaultSubmitter().(*t1)

	SetDefaultSubmitter(new(t2))
	_ = GetDefaultSubmitter().(*t2)
}
