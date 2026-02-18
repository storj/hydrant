package otelutil

import (
	"io"
	"net/http"
	"time"

	colllogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	"storj.io/hydrant"
)

// NewTraceReceiver returns an http.Handler that accepts OTLP trace exports
// (ExportTraceServiceRequest as application/x-protobuf) and submits them
// as hydrant events.
func NewTraceReceiver(sub hydrant.Submitter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		buf, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		var req colltracepb.ExportTraceServiceRequest
		if err := proto.Unmarshal(buf, &req); err != nil {
			http.Error(w, "invalid protobuf", http.StatusBadRequest)
			return
		}

		for _, rs := range req.ResourceSpans {
			resourceAttrs := resourceAnnotations(rs.GetResource().GetAttributes())
			for _, ss := range rs.ScopeSpans {
				for _, span := range ss.Spans {
					ev := spanToEvent(span, resourceAttrs)
					sub.Submit(r.Context(), ev)
				}
			}
		}

		resp, _ := proto.Marshal(&colltracepb.ExportTraceServiceResponse{})
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Write(resp)
	})
}

// NewLogReceiver returns an http.Handler that accepts OTLP log exports
// (ExportLogsServiceRequest as application/x-protobuf) and submits them
// as hydrant events.
func NewLogReceiver(sub hydrant.Submitter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		buf, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		var req colllogspb.ExportLogsServiceRequest
		if err := proto.Unmarshal(buf, &req); err != nil {
			http.Error(w, "invalid protobuf", http.StatusBadRequest)
			return
		}

		for _, rl := range req.ResourceLogs {
			resourceAttrs := resourceAnnotations(rl.GetResource().GetAttributes())
			for _, sl := range rl.ScopeLogs {
				for _, lr := range sl.LogRecords {
					ev := logRecordToEvent(lr, resourceAttrs)
					sub.Submit(r.Context(), ev)
				}
			}
		}

		resp, _ := proto.Marshal(&colllogspb.ExportLogsServiceResponse{})
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Write(resp)
	})
}

func spanToEvent(span *tracepb.Span, resourceAttrs []hydrant.Annotation) hydrant.Event {
	startNano := int64(span.StartTimeUnixNano)
	endNano := int64(span.EndTimeUnixNano)
	startTime := time.Unix(0, startNano)
	endTime := time.Unix(0, endNano)

	success := true
	if span.Status != nil && span.Status.Code == tracepb.Status_STATUS_CODE_ERROR {
		success = false
	}

	ev := make(hydrant.Event, 0, 8+len(span.Attributes)+len(resourceAttrs))
	ev = append(ev,
		hydrant.String("name", span.Name),
		hydrant.Timestamp("start", startTime),
		hydrant.SpanId("span_id", [8]byte(span.SpanId)),         // TODO: panic safety
		hydrant.SpanId("parent_id", [8]byte(span.ParentSpanId)), // TODO: panic safety
		hydrant.TraceId("trace_id", [16]byte(span.TraceId)),     // TODO: panic safety
		hydrant.Duration("duration", endTime.Sub(startTime)),
		hydrant.Bool("success", success),
		hydrant.Timestamp("timestamp", endTime),
	)

	ev = append(ev, resourceAttrs...)

	for _, kv := range span.Attributes {
		ev = append(ev, AttributeToAnnotation(kv))
	}

	return ev
}

func logRecordToEvent(lr *logspb.LogRecord, resourceAttrs []hydrant.Annotation) hydrant.Event {
	ev := make(hydrant.Event, 0, 4+len(lr.Attributes)+len(resourceAttrs))

	if lr.Body != nil {
		if sv, ok := lr.Body.Value.(*commonpb.AnyValue_StringValue); ok {
			ev = append(ev, hydrant.String("message", sv.StringValue))
		} else {
			ev = append(ev, hydrant.String("message", lr.Body.String()))
		}
	}

	if lr.TimeUnixNano != 0 {
		ev = append(ev, hydrant.Timestamp("timestamp", time.Unix(0, int64(lr.TimeUnixNano))))
	}

	if id := lr.SpanId; len(id) == 8 {
		ev = append(ev, hydrant.SpanId("span_id", [8]byte(id)))
	}
	if id := lr.TraceId; len(id) == 16 {
		ev = append(ev, hydrant.TraceId("trace_id", [16]byte(id)))
	}

	ev = append(ev, resourceAttrs...)

	for _, kv := range lr.Attributes {
		ev = append(ev, AttributeToAnnotation(kv))
	}

	return ev
}

func resourceAnnotations(attrs []*commonpb.KeyValue) []hydrant.Annotation {
	if len(attrs) == 0 {
		return nil
	}
	out := make([]hydrant.Annotation, 0, len(attrs))
	for _, kv := range attrs {
		a := AttributeToAnnotation(kv)
		a.Key = "resource." + a.Key
		out = append(out, a)
	}
	return out
}
