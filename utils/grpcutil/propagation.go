package grpcutil

import (
	"context"
	"encoding/hex"

	"google.golang.org/grpc/metadata"

	"storj.io/hydrant"
)

const traceparentKey = "traceparent"

// InjectTraceparent adds a W3C traceparent value to outgoing gRPC metadata
// from the current span in ctx. If there is no active span the context is
// returned unchanged. Use this when making outgoing gRPC calls.
func InjectTraceparent(ctx context.Context, span *hydrant.Span) context.Context {
	if span == nil {
		return ctx
	}

	traceId := span.TraceId()
	spanId := span.SpanId()

	var buf [55]byte
	buf[0] = '0'
	buf[1] = '0'
	buf[2] = '-'
	hex.Encode(buf[3:35], traceId[:])
	buf[35] = '-'
	hex.Encode(buf[36:52], spanId[:])
	buf[52] = '-'
	buf[53] = '0'
	buf[54] = '1'

	return metadata.AppendToOutgoingContext(ctx, traceparentKey, string(buf[:]))
}

// ExtractTraceparent parses the W3C traceparent value from incoming gRPC
// metadata and returns the trace ID and parent span ID. If the metadata is
// missing or malformed, zero values are returned.
func ExtractTraceparent(ctx context.Context) (traceId [16]byte, parentId [8]byte) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return traceId, parentId
	}

	vals := md.Get(traceparentKey)
	if len(vals) == 0 {
		return traceId, parentId
	}

	tp := vals[0]
	if len(tp) != 55 || tp[2] != '-' || tp[35] != '-' || tp[52] != '-' {
		return traceId, parentId
	}

	t, err := hex.DecodeString(tp[3:35])
	if err != nil {
		return traceId, parentId
	}
	s, err := hex.DecodeString(tp[36:52])
	if err != nil {
		return traceId, parentId
	}

	copy(traceId[:], t)
	copy(parentId[:], s)
	return traceId, parentId
}
