package submitters

import (
	"bytes"
	"context"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeebo/errs/v2"
	"github.com/zeebo/hmux"
	colllogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	"storj.io/hydrant"
	"storj.io/hydrant/internal/utils"
	"storj.io/hydrant/utils/otelutil"
	"storj.io/hydrant/value"
)

// OTelSubmitter exports hydrant events as OTLP protobuf to an OpenTelemetry
// collector. Span events go to /v1/traces and log events go to /v1/logs.
type OTelSubmitter struct {
	tracesURL string
	logsURL   string
	interval  time.Duration
	resource  *resourcepb.Resource
	live      liveBuffer

	stats struct {
		received     atomic.Uint64
		spansDropped atomic.Uint64
		logsDropped  atomic.Uint64
		flushes      atomic.Uint64
		flushErrors  atomic.Uint64
	}

	mu      sync.Mutex
	spans   []hydrant.Event
	logs    []hydrant.Event
	trigger chan struct{}
}

func NewOTelSubmitter(
	endpoint string,
	process []hydrant.Annotation,
	interval time.Duration,
	batchSize int,
) *OTelSubmitter {
	endpoint = strings.TrimRight(endpoint, "/")

	var attrs []*commonpb.KeyValue
	for _, a := range process {
		if kv := otelutil.AnnotationToAttribute(a); kv != nil {
			attrs = append(attrs, kv)
		}
	}

	return &OTelSubmitter{
		tracesURL: endpoint + "/v1/traces",
		logsURL:   endpoint + "/v1/logs",
		interval:  interval,
		resource:  &resourcepb.Resource{Attributes: attrs},
		live:      newLiveBuffer(),
		spans:     make([]hydrant.Event, 0, batchSize),
		logs:      make([]hydrant.Event, 0, batchSize),
		trigger:   make(chan struct{}, 1),
	}
}

func (o *OTelSubmitter) Children() []Submitter {
	return []Submitter{}
}

func (o *OTelSubmitter) ExtraData() any {
	return map[string]string{"endpoint": o.tracesURL}
}

func (o *OTelSubmitter) Submit(ctx context.Context, ev hydrant.Event) {
	o.live.Record(ev)
	o.stats.received.Add(1)

	o.mu.Lock()
	if otelutil.IsSpanEvent(ev) {
		if len(o.spans) < cap(o.spans) {
			o.spans = append(o.spans, ev)
		} else {
			o.stats.spansDropped.Add(1)
		}
	} else {
		if len(o.logs) < cap(o.logs) {
			o.logs = append(o.logs, ev)
		} else {
			o.stats.logsDropped.Add(1)
		}
	}
	full := len(o.spans) >= cap(o.spans)*2/3 || len(o.logs) >= cap(o.logs)*2/3
	o.mu.Unlock()

	if full {
		select {
		case o.trigger <- struct{}{}:
		default:
		}
	}
}

func (o *OTelSubmitter) Run(ctx context.Context) {
	nextTick := time.After(utils.Jitter(o.interval))
	for {
		select {
		case <-ctx.Done():
			o.flush(context.WithoutCancel(ctx))
			return
		case <-o.trigger:
		case <-nextTick:
		}
		nextTick = time.After(utils.Jitter(o.interval))
		o.flush(ctx)
	}
}

func (o *OTelSubmitter) flush(ctx context.Context) {
	o.mu.Lock()
	spans := slices.Clone(o.spans)
	o.spans = o.spans[:0]
	logs := slices.Clone(o.logs)
	o.logs = o.logs[:0]
	o.mu.Unlock()

	if len(spans) > 0 {
		o.flushSpans(ctx, spans)
	}
	if len(logs) > 0 {
		o.flushLogs(ctx, logs)
	}
}

func (o *OTelSubmitter) flushSpans(ctx context.Context, events []hydrant.Event) {
	otelSpans := make([]*tracepb.Span, 0, len(events))
	for _, ev := range events {
		otelSpans = append(otelSpans, eventToOTelSpan(ev))
	}

	req := &colltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{{
			Resource:   o.resource,
			ScopeSpans: []*tracepb.ScopeSpans{{Spans: otelSpans}},
		}},
	}

	data, err := proto.Marshal(req)
	if err != nil {
		o.stats.flushErrors.Add(1)
		return
	}
	if err := otelPost(ctx, o.tracesURL, data); err != nil {
		o.stats.flushErrors.Add(1)
		return
	}
	o.stats.flushes.Add(1)
}

func (o *OTelSubmitter) flushLogs(ctx context.Context, events []hydrant.Event) {
	records := make([]*logspb.LogRecord, 0, len(events))
	for _, ev := range events {
		records = append(records, eventToOTelLogRecord(ev))
	}

	req := &colllogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource:  o.resource,
			ScopeLogs: []*logspb.ScopeLogs{{LogRecords: records}},
		}},
	}

	data, err := proto.Marshal(req)
	if err != nil {
		o.stats.flushErrors.Add(1)
		return
	}
	if err := otelPost(ctx, o.logsURL, data); err != nil {
		o.stats.flushErrors.Add(1)
		return
	}
	o.stats.flushes.Add(1)
}

func (o *OTelSubmitter) Handler() http.Handler {
	return hmux.Dir{
		"/tree": constJSONHandler(treeify(o)),
		"/live": o.live.Handler(),
		"/stats": statsHandler(func() []stat {
			return []stat{
				{"received", o.stats.received.Load()},
				{"spans_dropped", o.stats.spansDropped.Load()},
				{"logs_dropped", o.stats.logsDropped.Load()},
				{"flushes", o.stats.flushes.Load()},
				{"flush_errors", o.stats.flushErrors.Load()},
			}
		}),
	}
}

func eventToOTelSpan(ev hydrant.Event) *tracepb.Span {
	s := &tracepb.Span{}
	var attrs []*commonpb.KeyValue

	for _, a := range ev {
		switch a.Key {
		case "name":
			x, _ := a.Value.String()
			s.Name = x
		case "span_id":
			x, _ := a.Value.SpanId()
			s.SpanId = x[:]
		case "parent_id":
			x, _ := a.Value.SpanId()
			s.ParentSpanId = x[:]
		case "trace_id":
			x, _ := a.Value.TraceId()
			s.TraceId = x[:]
		case "start":
			x, _ := a.Value.Timestamp()
			s.StartTimeUnixNano = uint64(x.UnixNano())
		case "timestamp":
			x, _ := a.Value.Timestamp()
			s.EndTimeUnixNano = uint64(x.UnixNano())
		case "duration":
			// derived from start/end, skip
		case "success":
			x, _ := a.Value.Bool()
			if x {
				s.Status = &tracepb.Status{Code: tracepb.Status_STATUS_CODE_OK}
			} else {
				s.Status = &tracepb.Status{Code: tracepb.Status_STATUS_CODE_ERROR}
			}
		default:
			if kv := otelutil.AnnotationToAttribute(a); kv != nil {
				attrs = append(attrs, kv)
			}
		}
	}

	s.Attributes = attrs
	return s
}

func eventToOTelLogRecord(ev hydrant.Event) *logspb.LogRecord {
	lr := &logspb.LogRecord{}
	var attrs []*commonpb.KeyValue

	for _, a := range ev {
		switch a.Key {
		case "message":
			x, _ := a.Value.String()
			lr.Body = &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: x}}
		case "timestamp":
			if a.Value.Kind() == value.KindTimestamp {
				x, _ := a.Value.Timestamp()
				lr.TimeUnixNano = uint64(x.UnixNano())
			}
		case "span_id":
			x, _ := a.Value.SpanId()
			lr.SpanId = x[:]
		case "trace_id":
			x, _ := a.Value.TraceId()
			lr.TraceId = x[:]
		default:
			if kv := otelutil.AnnotationToAttribute(a); kv != nil {
				attrs = append(attrs, kv)
			}
		}
	}

	lr.Attributes = attrs
	return lr
}

func otelPost(ctx context.Context, url string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return errs.Errorf("otel collector returned %d", resp.StatusCode)
	}
	return nil
}
