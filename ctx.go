package hydrant

import (
	"context"
	"hydrant/event"
)

type submitterKeyType struct{}

type Submitter interface {
	Submit(event.Event)
}

func GetSubmitter(ctx context.Context) Submitter {
	s, _ := ctx.Value(submitterKeyType{}).(Submitter)
	return s
}

func WithSubmitter(ctx context.Context, s Submitter) context.Context {
	return context.WithValue(ctx, submitterKeyType{}, s)
}
