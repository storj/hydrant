package otelutil

import (
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"

	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

// IsSpanEvent returns true if the event looks like it came from a hydrant span
// meaning it has a parent_id field of the appropriate type. logs created under
// spans have span_id and trace_id fields, so we can't use those.
func IsSpanEvent(ev hydrant.Event) bool {
	for _, a := range ev {
		if a.Key == "parent_id" && a.Value.Kind() == value.KindSpanId {
			return true
		}
	}
	return false
}

func AnnotationToAttribute(a hydrant.Annotation) *commonpb.KeyValue {
	kv := &commonpb.KeyValue{Key: a.Key}

	switch a.Value.Kind() {
	case value.KindString:
		x, _ := a.Value.String()
		kv.Value = &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: x}}

	case value.KindBool:
		x, _ := a.Value.Bool()
		kv.Value = &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: x}}

	case value.KindInt:
		x, _ := a.Value.Int()
		kv.Value = &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: x}}

	case value.KindUint:
		x, _ := a.Value.Uint()
		kv.Value = &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}

	case value.KindFloat:
		x, _ := a.Value.Float()
		kv.Value = &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: x}}

	case value.KindDuration:
		x, _ := a.Value.Duration()
		kv.Value = &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: x.String()}}

	case value.KindTimestamp:
		x, _ := a.Value.Timestamp()
		kv.Value = &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: x.Format(time.RFC3339Nano)}}

	case value.KindBytes:
		x, _ := a.Value.Bytes()
		kv.Value = &commonpb.AnyValue{Value: &commonpb.AnyValue_BytesValue{BytesValue: x}}

	default:
		// KindHistogram, KindEmpty: skip
		return nil
	}

	return kv
}

func AttributeToAnnotation(kv *commonpb.KeyValue) hydrant.Annotation {
	if kv.Value == nil {
		return hydrant.String(kv.Key, "")
	}
	switch v := kv.Value.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return hydrant.String(kv.Key, v.StringValue)
	case *commonpb.AnyValue_BoolValue:
		return hydrant.Bool(kv.Key, v.BoolValue)
	case *commonpb.AnyValue_IntValue:
		return hydrant.Int(kv.Key, v.IntValue)
	case *commonpb.AnyValue_DoubleValue:
		return hydrant.Float(kv.Key, v.DoubleValue)
	case *commonpb.AnyValue_BytesValue:
		return hydrant.Bytes(kv.Key, v.BytesValue)
	default:
		// ArrayValue, KvlistValue: stringify
		return hydrant.String(kv.Key, kv.Value.String())
	}
}
