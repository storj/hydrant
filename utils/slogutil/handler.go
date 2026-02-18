// Package slogutil bridges Go's log/slog into hydrant. Log records become
// hydrant events with message, timestamp, level, and source annotations.
// If a span is active on the context, span_id and trace_id are included
// automatically.
package slogutil

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"time"

	"storj.io/hydrant"
)

type HandlerOptions struct {
	// Level is the minimum log level to handle. Default is slog.LevelInfo.
	Level slog.Leveler
}

// NewHandler returns an slog.Handler that submits log records as hydrant events.
func NewHandler(opts *HandlerOptions) slog.Handler {
	h := &handler{}
	if opts != nil && opts.Level != nil {
		h.level = opts.Level
	}
	return h
}

type handler struct {
	level  slog.Leveler
	attrs  []hydrant.Annotation
	groups []string
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.level != nil {
		minLevel = h.level.Level()
	}
	return level >= minLevel
}

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	ev := make(hydrant.Event, 0, 5+len(h.attrs)+r.NumAttrs())

	ev = append(ev,
		hydrant.String("message", r.Message),
		hydrant.Timestamp("timestamp", r.Time),
		hydrant.String("level", r.Level.String()),
	)

	if r.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		ev = append(ev,
			hydrant.String("file", frame.File),
			hydrant.Int("line", int64(frame.Line)),
		)
	}

	// Add span context if present.
	if span := hydrant.GetSpan(ctx); span != nil {
		ev = append(ev,
			hydrant.SpanId("span_id", span.SpanId()),
			hydrant.TraceId("trace_id", span.TraceId()),
		)
	}

	// Pre-attrs from WithAttrs.
	ev = append(ev, h.attrs...)

	// Record attrs.
	prefix := h.groupPrefix()
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(&ev, prefix, a)
		return true
	})

	hydrant.GetSubmitter(ctx).Submit(ctx, ev)
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()
	prefix := h.groupPrefix()
	ev := hydrant.Event(h2.attrs)
	for _, a := range attrs {
		h2.appendAttr(&ev, prefix, a)
	}
	h2.attrs = []hydrant.Annotation(ev)
	return h2
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *handler) clone() *handler {
	return &handler{
		level:  h.level,
		attrs:  slices.Clone(h.attrs),
		groups: slices.Clone(h.groups),
	}
}

func (h *handler) groupPrefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	var prefix string
	for _, g := range h.groups {
		prefix += g + "."
	}
	return prefix
}

func (h *handler) appendAttr(ev *hydrant.Event, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()

	if a.Equal(slog.Attr{}) {
		return
	}

	key := prefix + a.Key

	switch a.Value.Kind() {
	case slog.KindString:
		*ev = append(*ev, hydrant.String(key, a.Value.String()))
	case slog.KindInt64:
		*ev = append(*ev, hydrant.Int(key, a.Value.Int64()))
	case slog.KindUint64:
		*ev = append(*ev, hydrant.Uint(key, a.Value.Uint64()))
	case slog.KindFloat64:
		*ev = append(*ev, hydrant.Float(key, a.Value.Float64()))
	case slog.KindBool:
		*ev = append(*ev, hydrant.Bool(key, a.Value.Bool()))
	case slog.KindDuration:
		*ev = append(*ev, hydrant.Duration(key, a.Value.Duration()))
	case slog.KindTime:
		*ev = append(*ev, hydrant.Timestamp(key, a.Value.Time()))
	case slog.KindGroup:
		for _, ga := range a.Value.Group() {
			h.appendAttr(ev, key+".", ga)
		}
	default:
		*ev = append(*ev, hydrant.String(key, fmt.Sprint(a.Value.Any())))
	}
}

// Logger is a convenience that returns an *slog.Logger backed by hydrant.
func Logger(opts *HandlerOptions) *slog.Logger {
	return slog.New(NewHandler(opts))
}

// SetDefault sets the default slog logger to one backed by hydrant.
func SetDefault(opts *HandlerOptions) {
	slog.SetDefault(Logger(opts))
}

// TimeValue returns a slog.Attr that uses slog.KindTime (which maps to
// hydrant.Timestamp) instead of formatting the time as a string. Equivalent
// to slog.Time but provided for discoverability.
func TimeValue(key string, t time.Time) slog.Attr {
	return slog.Time(key, t)
}
