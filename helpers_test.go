package hydrant

import "context"

type nullSubmitter struct{}

func (ns nullSubmitter) Submit(ctx context.Context, ev Event) {}

type bufferSubmitter []Event

func (bs *bufferSubmitter) Submit(ctx context.Context, ev Event) {
	*bs = append(*bs, ev)
}
