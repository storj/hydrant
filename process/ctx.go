package process

import (
	"context"

	"storj.io/hydrant"
)

func Annotations(ctx context.Context) []hydrant.Annotation {
	if s := GetStore(ctx); s != nil {
		return s.Annotations()
	}
	return nil
}

type storeKeyType struct{}

func GetStore(ctx context.Context) *Store {
	if s, ok := ctx.Value(storeKeyType{}).(*Store); ok {
		return s
	}
	return DefaultStore
}

func WithAnnotations(ctx context.Context, s *Store) context.Context {
	return context.WithValue(ctx, storeKeyType{}, s)
}
