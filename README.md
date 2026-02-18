# hydrant

Observability library for Go with centrally managed configuration,
full-resolution distributions, and composable event pipelines.

## Quick Start

Set up a pipeline, instrument your code, and open the web UI:

```go
sub, err := submitters.Environment{
    Filter:  filter.NewBuiltinEnvionment(),
    Process: process.DefaultStore,
}.New(config.Config{
    Submitter: config.GrouperSubmitter{
        FlushInterval: 30 * time.Second,
        GroupBy:       []string{"name", "endpoint"},
        Submitter:     config.HydratorSubmitter{},
    },
})
if err != nil {
    panic(err)
}

hydrant.SetDefaultSubmitter(sub)
go sub.Run(context.Background())

http.Handle("/ui/", http.StripPrefix("/ui/", sub.Handler()))
panic(http.ListenAndServe(":9912", nil))
```

Instrument functions with spans:

```go
func HandleRequest(ctx context.Context) (err error) {
    ctx, span := hydrant.StartSpan(ctx,
        hydrant.String("endpoint", "/api/users"),
    )
    defer span.Done(&err)

    // child spans automatically link to parent
    ctx, child := hydrant.StartSpanNamed(ctx, "db_query")
    defer child.Done(&err)

    // standalone log events
    hydrant.Log(ctx, "fetched 42 rows", hydrant.Int("rows", 42))

    return nil
}
```

Open `http://localhost:9912/ui/` to see the pipeline tree and query histogram
data. See [examples/basic](examples/basic/main.go) for a complete runnable
version.

## Why hydrant?

**Full-resolution histograms.** Every observation is preserved in the
histogram, not bucketed or sketched away. You can query p99.9 or p99.99 after
the fact without going back and re-instrumenting.

**Centrally configured pipelines.** Servers poll for configuration and
hot-swap their event pipelines without restarting. Change what you collect, how
you filter, and where you send it all from one place.

**Composable submitters.** Event pipelines are built by chaining filters,
groupers, and outputs together. Route different events to different
destinations. The whole thing is defined in JSON.

## Examples

Runnable examples live in the [examples/](examples/) directory:

- **[basic](examples/basic/main.go)** - Minimal pipeline with spans, grouper,
  and the web UI. Good starting point.

- **[httpserver](examples/httpserver/main.go)** - Real HTTP server wrapped with
  the `httputil` middleware. Shows automatic span creation per request with
  method, path, status code, and response size.

- **[otelbridge](examples/otelbridge/main.go)** - Bidirectional OTel
  integration. Exports hydrant spans to Jaeger/any OTLP collector while also
  accepting incoming OTLP traces and logs.

- **[prometheus](examples/prometheus/main.go)** - Prometheus metrics export.
  Grouped events are exposed at `/metrics` with duration histograms, event
  counts, error counts, and active span gauge.

- **[slog](examples/slog/main.go)** - Bridge Go's `log/slog` into hydrant.
  All slog output becomes hydrant events linked to the active span.

- **[remoteconfig](examples/remoteconfig/main.go)** - Central configuration
  with `RemoteSubmitter`. A config server serves pipeline JSON over HTTP and
  the client hot-swaps its pipeline on changes without restarting.

Run any example with:

```
go run ./examples/basic
```

## slog Integration

The `utils/slogutil` package bridges Go's `log/slog` into hydrant. All slog
attributes map to hydrant annotations with full type fidelity (integers stay
integers, durations stay durations, etc). If a span is active on the context,
log events are linked to it via span_id and trace_id.

```go
// Replace the default slog logger.
slogutil.SetDefault(nil)

// Or create a logger directly.
logger := slogutil.Logger(&slogutil.HandlerOptions{
    Level: slog.LevelDebug,
})

// Logs inside a span are linked automatically.
ctx, span := hydrant.StartSpanNamed(ctx, "handle_request")
defer span.Done(&err)
slog.InfoContext(ctx, "processing", slog.String("user", "alice"))
```

See [examples/slog](examples/slog/main.go) for a full working example.

## HTTP and gRPC Middleware

The `utils/httputil` and `utils/grpcutil` packages provide zero-effort
instrumentation for HTTP handlers and gRPC services.

### HTTP

```go
// Wrap any http.Handler. Creates a span per request with http.method,
// http.path, http.status_code, http.response_size annotations.
http.Handle("/api/", httputil.Wrap(apiHandler))

// Control span naming to avoid high-cardinality paths.
http.Handle("/api/", &httputil.Handler{
    Name:    func(r *http.Request) string { return r.Method },
    Handler: apiHandler,
})
```

See [examples/httpserver](examples/httpserver/main.go) for a full working
example.

### gRPC

```go
grpc.NewServer(
    grpc.UnaryInterceptor(grpcutil.UnaryInterceptor(nil)),
    grpc.StreamInterceptor(grpcutil.StreamInterceptor(nil)),
)
```

Pass `nil` for default span names (the full method path). Pass a function to
control naming.

## OpenTelemetry Bridge

Hydrant integrates with the OpenTelemetry ecosystem in both directions.

### Export to an OTLP collector

Add an `OTelSubmitter` to your pipeline config. Span events go to `/v1/traces`,
log events go to `/v1/logs`.

```go
config.OTelSubmitter{
    Endpoint:      "http://localhost:4318",
    FlushInterval: 5 * time.Second,
    MaxBatchSize:  1000,
}
```

Or in JSON:

```json
{
    "kind": "otel",
    "endpoint": "http://localhost:4318",
    "flush_interval": "5s",
    "max_batch_size": 1000
}
```

### Receive OTLP data

Accept traces and logs from OTel-instrumented services:

```go
http.Handle("/v1/traces", otelutil.NewTraceReceiver(submitter))
http.Handle("/v1/logs", otelutil.NewLogReceiver(submitter))
```

Incoming OTLP spans are converted to hydrant events with proper system fields
(name, span_id, trace_id, duration, success, etc). Resource attributes are
prefixed with `resource.`.

See [examples/otelbridge](examples/otelbridge/main.go) for both directions
working together.

## Prometheus

The `PrometheusSubmitter` exposes grouped metrics in Prometheus text format.
Place it after a `GrouperSubmitter` in your pipeline so it receives aggregated
histograms.

```json
{
    "kind": "prometheus",
    "namespace": "myapp",
    "buckets": [0.01, 0.05, 0.1, 0.5, 1, 5]
}
```

Both `namespace` and `buckets` are optional. The default namespace is `hydrant`
and the default buckets are the standard Prometheus latency defaults
(.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10).

The `/metrics` endpoint on the submitter's handler exposes:

- `{namespace}_duration_seconds` - histogram of event durations
- `{namespace}_events_total` - counter of total events
- `{namespace}_errors_total` - counter of events where `success` was false
- `{namespace}_active_spans` - gauge of currently active spans

## Core Concepts

### Events and Annotations

An **Event** is a list of typed key-value pairs called **Annotations**.
Supported types: `String`, `Int`, `Uint`, `Float`, `Bool`, `Duration`,
`Timestamp`, `Identifier`, `Bytes`, and `Histogram`.

### Spans

A **Span** is a timed, named unit of work. Spans automatically capture:
- `name` - function name (auto-detected) or explicit
- `span_id` - unique identifier
- `parent_id` / `trace_id` - trace hierarchy
- `start` / `duration` / `timestamp` - timing
- `success` - derived from the error pointer passed to `Done`

Spans nest through `context.Context`. Child spans inherit the parent's trace
and span IDs.

`hydrant.IterateSpans` walks all currently active spans. Useful for building
`/debug/spans` endpoints to diagnose hung requests.

`hydrant.ActiveSpanCount` returns the number of currently active spans more
cheaply then walking them all. Useful for checking for task leaks.

### Submitters

A **Submitter** receives events and does something with them. They compose
into pipelines:

| Submitter              | Purpose                                                |
|------------------------|--------------------------------------------------------|
| **MultiSubmitter**     | Fan-out to multiple destinations                       |
| **FilterSubmitter**    | Conditional routing based on filter expressions        |
| **GrouperSubmitter**   | Time-windowed aggregation with histogram merging       |
| **HTTPSubmitter**      | Batch and send events to a remote collector over HTTP  |
| **OTelSubmitter**      | Export events as OTLP protobuf to an OTel collector    |
| **PrometheusSubmitter**| Expose grouped metrics as Prometheus /metrics endpoint |
| **HydratorSubmitter**  | In-memory histogram storage with query API             |
| **NullSubmitter**      | Discard events                                         |
| **NamedSubmitter**     | Reference another submitter by name (enables reuse)    |

### Grouper and Histograms

The **GrouperSubmitter** aggregates events over a configurable time window.
Numeric values (`Int`, `Uint`, `Float`, `Duration`, `Timestamp`) are observed
into full-resolution histograms. Existing histograms are merged directly. On
flush, the grouped event carries the full histogram for each field, plus
aggregation metadata (`agg:start_time`, `agg:end_time`, `agg:duration`).

The **HydratorSubmitter** indexes these histograms in memory. You can query
any quantile at any precision through the web UI or the `/query` API.

## Configuration

Pipelines are defined in JSON. Submitter type is determined by a `kind` field
for objects, or by shape: strings are named references, arrays are
multi-submitters.

```json
{
    "refresh_interval": "5m",
    "submitter": "default",
    "submitters": {
        "collector": {
            "kind": "http",
            "endpoint": "http://collector:9090/receive",
            "flush_interval": "1m",
            "max_batch_size": 10000,
            "process_fields": ["proc.starttime", "os.hostname"]
        },
        "jaeger": {
            "kind": "otel",
            "endpoint": "http://jaeger:4318",
            "flush_interval": "5s",
            "max_batch_size": 1000
        },
        "hydrator": {
            "kind": "hydrator"
        },
        "prom": {
            "kind": "prometheus",
            "namespace": "myapp"
        },
        "default": {
            "kind": "grouper",
            "flush_interval": "30s",
            "group_by": ["name", "endpoint"],
            "submitter": ["collector", "jaeger", "hydrator", "prom"]
        }
    }
}
```

### Remote Configuration

`RemoteSubmitter` polls a config endpoint and hot-swaps the pipeline on
changes. No dropped events, no restarts.

```go
remote := submitters.NewRemoteSubmitter(env, "http://config-server:8080/config")
hydrant.SetDefaultSubmitter(remote)
go remote.Run(ctx)
```

## Filter Language

FilterSubmitter uses an expression language for routing events:

```
eq(key(name), api_request)           # field equality
has(span_id)                         # field existence
lt(key(duration), 1.0)               # numeric comparison (also: lte, gt, gte)
not(eq(key(status), error))          # negation
since(key(start))                    # time since a timestamp
rand()                               # random float [0, 1), useful for sampling
```

Combine expressions with `&&` and `||`. A few common patterns:

```
# 10% sampling
lt(rand(), 0.1)

# only spans slower than 500ms
has(span_id) && gt(key(duration), 0.5)

# drop health checks, keep everything else
not(eq(key(name), health_check))
```

The filter environment is extensible. Register custom functions with
`env.SetFunction(name, fn)`.

## Web UI

The built-in web UI (served by `sub.Handler()`) provides:

- **Names view** - browse all named submitters
- **Tree view** - visualize the full pipeline hierarchy
- **Histogram query** - filter and query metrics stored in HydratorSubmitters
  with configurable quantile resolution and linear/exponential spacing
- **Distribution charts** - SVG quantile visualization with linear and log
  scale modes

## Architecture

```
  ┌──────────────────────────────────────────────────────────┐
  │                    Your Application                      │
  │                                                          │
  │   ctx, span := hydrant.StartSpan(ctx)                    │
  │   defer span.Done(&err)                                  │
  │   hydrant.Log(ctx, "processed request", ...)             │
  │                                                          │
  └──────────────────────┬───────────────────────────────────┘
                         │ Events ([]Annotation)
                         ▼
  ┌────────────────────────────────────────────────────────────┐
  │              Composable Submitter Pipeline                 │
  │                                                            │
  │   ┌───────────────────────────────────────────────┐        │
  │   │ ConfiguredSubmitter (root)                    │        │
  │   │                                               │        │
  │   │  ┌─ FilterSubmitter ───────────────────────┐  │        │
  │   │  │  filter: eq(key(name), api_request)     │  │        │
  │   │  │                                         │  │        │
  │   │  │  ┌─ GrouperSubmitter ────────────────┐  │  │        │
  │   │  │  │  group_by: [name, endpoint]       │  │  │        │
  │   │  │  │  flush_interval: 30s              │  │  │        │
  │   │  │  │                                   │  │  │        │
  │   │  │  │  ┌─ MultiSubmitter ───────────┐   │  │  │        │
  │   │  │  │  │                            │   │  │  │        │
  │   │  │  │  │  ┌─ HTTPSubmitter ──────┐  │   │  │  │        │
  │   │  │  │  │  │  → collector:9090    │  │   │  │  │        │
  │   │  │  │  │  └──────────────────────┘  │   │  │  │        │
  │   │  │  │  │  ┌─ OTelSubmitter ──────┐  │   │  │  │        │
  │   │  │  │  │  │  → jaeger:4318       │  │   │  │  │        │
  │   │  │  │  │  └──────────────────────┘  │   │  │  │        │
  │   │  │  │  │  ┌─ HydratorSubmitter ──┐  │   │  │  │        │
  │   │  │  │  │  │  (in-memory query)   │  │   │  │  │        │
  │   │  │  │  │  └──────────────────────┘  │   │  │  │        │
  │   │  │  │  └────────────────────────────┘   │  │  │        │
  │   │  │  └───────────────────────────────────┘  │  │        │
  │   │  └─────────────────────────────────────────┘  │        │
  │   └───────────────────────────────────────────────┘        │
  └────────────────────────────────────────────────────────────┘
           │               │                    │
           ▼               ▼                    ▼
   Remote Collector   OTel/Jaeger          Web UI &
   (zstd/HTTP)        (OTLP/HTTP)         Histogram Query


  ┌──────────────────────────────────────────────────────────┐
  │                Central Config Server                     │
  │                                                          │
  │  Serves JSON config over HTTP. RemoteSubmitter polls     │
  │  periodically and hot-swaps the pipeline on changes.     │
  └──────────────────────────────────────────────────────────┘
```

## Process Metadata

Hydrant automatically collects process-level metadata that can be included in
HTTP batches via `process_fields`:

| Field                                                | Description        |
|------------------------------------------------------|--------------------|
| `proc.starttime`                                     | Process start time |
| `os.hostname`                                        | Machine hostname   |
| `os.ip`                                              | Outbound IP        |
| `go.os` / `go.arch`                                  | Runtime platform   |
| `go.version`                                         | Go version         |
| `go.main.path` / `go.main.version`                   | Module info        |
| `go.vcs.time` / `go.vcs.rev` / `go.vcs.modified`     | VCS metadata       |

## Packages

| Package                  | Description                                                      |
|--------------------------|------------------------------------------------------------------|
| `hydrant`                | Core API: `StartSpan`, `Log`, `Event`, `Annotation`, `Submitter` |
| `hydrant/config`         | JSON-serializable pipeline configuration types                   |
| `hydrant/submitters`     | Built-in submitter implementations and web UI                    |
| `hydrant/filter`         | Expression parser, compiler, and built-in functions              |
| `hydrant/receiver`       | HTTP handler for receiving zstd-compressed event batches         |
| `hydrant/utils/httputil` | HTTP middleware for automatic span instrumentation               |
| `hydrant/utils/grpcutil` | gRPC interceptors for automatic span instrumentation             |
| `hydrant/utils/otelutil` | OTLP trace/log receivers and conversion utilities                |
| `hydrant/utils/slogutil` | Bridge Go's log/slog into hydrant events                         |
| `hydrant/process`        | Automatic process metadata collection                            |
| `hydrant/value`          | Type-safe tagged union for annotation values                     |
