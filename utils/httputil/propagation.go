package httputil

import (
	"encoding/hex"
	"net/http"

	"storj.io/hydrant"
)

const traceparentHeader = "traceparent"

// InjectTraceparent sets the W3C traceparent header on an outgoing HTTP
// request from the current span in ctx. If there is no active span the
// request is left unchanged. This is intended for use in HTTP clients.
func InjectTraceparent(req *http.Request, span *hydrant.Span) {
	if span == nil {
		return
	}

	traceId := span.TraceId()
	spanId := span.SpanId()

	// format: 00-<32 hex trace_id>-<16 hex span_id>-01
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

	req.Header.Set(traceparentHeader, string(buf[:]))
}

// ExtractTraceparent parses the W3C traceparent header from an incoming HTTP
// request and returns the trace ID and parent span ID. If the header is
// missing or malformed, zero values are returned and StartSpanNamed will
// generate fresh IDs.
func ExtractTraceparent(req *http.Request) (traceId [16]byte, parentId [8]byte) {
	tp := req.Header.Get(traceparentHeader)
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
