package hydrant

type nullSubmitter struct{}

func (ns nullSubmitter) Submit(ev Event) {}

type bufferSubmitter []Event

func (bs *bufferSubmitter) Submit(ev Event) { *bs = append(*bs, ev) }
