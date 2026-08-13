<!-- m-wiki: type=concept slug=internal-metrics topic=observability base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# Internal metrics

OBI instruments other processes; internal metrics are how it instruments *itself*. They answer operational questions the exported telemetry cannot: is the agent dropping spans, are eBPF probes failing to attach, is the OTLP exporter erroring, how many processes are currently instrumented.

## The reporter interface

`pkg/export/imetrics/imetrics.go:Reporter` is the contract. Its methods name the events worth counting, which makes the interface a readable inventory of the agent's own failure modes:

- `TracerFlush` — spans flushed out of eBPF ring buffers
- `OTELMetricExport` / `OTELMetricExportError`, `OTELTraceExport` / `OTELTraceExportError` — exporter throughput and failures
- `PrometheusRequest` — scrapes served
- `InstrumentProcess` / `UninstrumentProcess` — attach and detach events
- `InstrumentationError` — an attach that failed
- `AvoidInstrumentationMetrics` / `AvoidInstrumentationTraces` — a process deliberately skipped
- `BpfProbeStats` — per-probe statistics

`pkg/export/imetrics/imetrics.go:NoopReporter` implements every method as an empty body, so "internal metrics disabled" costs an inlinable no-op call rather than a nil check at each site.

## Selection is by configuration shape

`pkg/instrumenter/instrumenter.go:internalMetrics` picks the implementation with a four-way switch, in order:

1. `exporter: otel` → an OTLP reporter via `otel.NewInternalMetricsReporter`
2. `exporter: prometheus`, **or** `internal_metrics.prometheus.port` set to anything non-zero → a Prometheus reporter
3. `config.Prometheus.Registry != nil` → a Prometheus reporter writing into a caller-supplied registry (the vendored-as-a-library path)
4. otherwise → `NoopReporter`

Branch 2 is the one that surprises people: **setting a port enables internal metrics**, whether or not the exporter was named explicitly.

## An acknowledged dependency cycle

Branch 2 contains a wrinkle the code marks with a `TODO`. The Prometheus *manager* has internal metrics of its own, so after building the reporter the manager is handed it back:

```go
metrics := imetrics.NewPrometheusReporter(&config.InternalMetrics, promMgr, nil)
promMgr.InstrumentWith(metrics)
```

The comment states the intended fix — let `prommgr` create and return the reporter itself — so the back-reference is known debt, not an accident.

`pkg/export/imetrics/imetrics.go:IsBuiltinNoopReporter` exists for the same family of reasons: callers occasionally need to know whether reporting is genuinely active before doing expensive work to produce a value nobody will record.

## Ordering constraints at boot

`pkg/instrumenter/instrumenter.go:BuildCommonContextInfo` builds these in a specific, commented order:

1. The Kubernetes metadata provider is created with a `NoopReporter` placeholder.
2. Node metadata resolves — it reads the informer but not its metrics reporter.
3. The real reporter is created, so its OTLP resource carries host metadata such as `host.id`.
4. `ctxInfo.K8sInformer.SetInternalMetrics(...)` wires the real reporter back into the informer.

Reordering these silently produces internal metrics missing host attributes — a failure that is invisible until someone tries to group agent metrics by node.

## Related

- [Attribute groups](attribute-groups.md) — the other boot-time observability decision
- [Entry point and boot sequence](../02-ENTRYPOINT.md) — where this ordering happens
- [Configuration](../06-CONFIGURATION.md) — the `internal_metrics` block
- [Kubernetes metadata](../07-KUBERNETES-METADATA.md) — the informer being wired

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
