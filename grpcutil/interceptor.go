package grpcutil

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"storj.io/hydrant"
)

// UnaryInterceptor returns a grpc.UnaryServerInterceptor that creates a span
// for each unary RPC.
//
// If name is non-nil it controls the span name. Otherwise the full method
// string is used (e.g. "/package.Service/Method").
func UnaryInterceptor(name func(ctx context.Context, info *grpc.UnaryServerInfo) string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		n := info.FullMethod
		if name != nil {
			n = name(ctx, info)
		}

		ctx, span := hydrant.StartSpanNamed(ctx, n,
			hydrant.String("grpc.method", info.FullMethod),
		)

		defer func() {
			defer span.Done(&err)

			span.Annotate(
				hydrant.String("grpc.code", status.Code(err).String()),
			)

			if v := recover(); v != nil {
				span.Annotate(hydrant.Bool("grpc.panic", true))
				panic(v)
			}
		}()

		return handler(ctx, req)
	}
}

// StreamInterceptor returns a grpc.StreamServerInterceptor that creates a span
// for each streaming RPC.
//
// If name is non-nil it controls the span name. Otherwise the full method
// string is used.
func StreamInterceptor(name func(ctx context.Context, info *grpc.StreamServerInfo) string) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		ctx := ss.Context()

		n := info.FullMethod
		if name != nil {
			n = name(ctx, info)
		}

		ctx, span := hydrant.StartSpanNamed(ctx, n,
			hydrant.String("grpc.method", info.FullMethod),
		)

		defer func() {
			defer span.Done(&err)

			span.Annotate(
				hydrant.String("grpc.code", status.Code(err).String()),
			)

			if v := recover(); v != nil {
				span.Annotate(hydrant.Bool("grpc.panic", true))
				panic(v)
			}
		}()

		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

// wrappedStream overrides Context() to carry the span.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
