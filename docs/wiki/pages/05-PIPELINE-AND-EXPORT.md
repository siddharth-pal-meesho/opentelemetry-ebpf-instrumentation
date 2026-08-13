<!-- m-wiki: type=top-level slug=05-pipeline-and-export topic=null base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: top-level. 0 sources.

# Pipeline and export

Once eBPF programs are attached, spans arrive as `[]request.Span` batches and stop being a kernel concern. From here to the wire, OBI runs them through a chain of decorators, filters and exporters — all optional, all assembled at startup, none of them aware of PIDs or probes.

This is the half of AppO11y that a future release plans to run as a separate lower-privilege process.

## TL;DR

- Assembled in `pkg/appolly/instrumenter.go:Build`, which delegates to `pkg/appolly/instrumenter.go:newGraphBuilder`.
- One linear trace path: read → routes → Kubernetes → Docker → name resolution → attribute filter → signal gate → exporters.
- A **separate metrics sub-pipeline** branches off the same gated span queue and is only constructed when a metrics exporter is actually configured.
- Exporters are additive: OTLP traces, OTLP metrics, OTLP service-graph metrics, a Prometheus scrape endpoint, and a debug printer can all be active simultaneously.
- Every stage is a swarm node with an explicit ID, connected by queues whose buffer size and send timeout come from configuration.

## Mental model

**A conveyor belt with optional stations and a fork.** Spans enter at one end. Each station either adds attributes or removes spans. Nothing reorders or aggregates them on the trace path.

The fork matters: metrics are not a different pipeline fed by different data, they are a second consumer of the same decorated spans. `spanNameAggregatedMetrics` exists because metric cardinality — unlike trace volume — must be bounded, so the span-name limiter sits on the metrics branch only. A span that appears verbatim in a trace may be counted under a collapsed name in a metric.

## Structure / data flow

```
    tracesInput  (from every ProcessTracer)
         │
   ReadFromChannel        traces.ReadDecorator — instance ID, hostname
         ▼
   Routes                 URL path grouping                       (skipped if routes unset)
         ▼
   KubeDecorator          pod / namespace / owner attributes      (mutually exclusive
         ▼                                                          with Docker)
   DockerDecorator        container attributes
         ▼
   NameResolution         resolve peer names
         ▼
   AttributesFilter       drop spans by attribute rules
         ▼
   DynamicSignalSpanGate  per-PID signal gating
         │
         ├──────────────► OTELTracesReceiver     OTLP traces
         ├──────────────► PrinterNode            debug output
         │
         └── metrics sub-pipeline (only when a metrics exporter is enabled)
                   │
             SpanNameLimiter          cap distinct span names
                   ├──► OTELMetricsExport          RED metrics
                   ├──► OTELSvcGraphMetricsExport  service-graph metrics
                   ├──► PrometheusEndpoint         scrape surface
                   └──► OTELRuntimeMetricsExport   runtime metrics
```

Process events travel a parallel three-node chain — host, Kubernetes, then Docker decoration — built in `pkg/internal/appolly/appolly.go:New` and consumed by the metrics exporters so they can emit and expire per-process series.

## Key code locations

| What | Where |
|---|---|
| Public pipeline constructor | `pkg/appolly/instrumenter.go:Build` |
| Node and queue registration | `pkg/appolly/instrumenter.go:newGraphBuilder` |
| Metrics branch construction | `pkg/appolly/instrumenter.go:setupMetricsSubPipeline` |
| Merged base + per-app metrics config | `pkg/appolly/instrumenter.go:JoinMetricsConfig` |
| Span reader / first decorator | `pkg/internal/traces/read_decorator.go:ReadFromChannel` |
| Reader configuration | `pkg/internal/traces/read_decorator.go:ReadDecorator` |
| URL route grouping | `pkg/transform/routes.go:RoutesProvider` |
| Kubernetes span decoration | `pkg/transform/k8s.go:KubeDecoratorProvider` |
| Attribute-based filtering | `pkg/filter/attribute.go:ByAttribute` |
| OTLP trace export | `pkg/export/otel/traces.go:TracesReceiver` |
| OTLP metric export | `pkg/export/otel/metrics.go:ReportMetrics` |
| Prometheus scrape endpoint | `pkg/export/prom/prom.go:PrometheusEndpoint` |
| Pipeline start / done | `pkg/appolly/instrumenter.go:Build` → `pkg/pipe/swarm/runner.go:Runner.Start` |

## Vendoring hooks

Two seams exist for embedding OBI inside another collector rather than running it standalone:

- `ctxInfo.OverrideAppExportQueue` replaces the `exportableSpans` queue, letting the host process attach its own exporters instead of OBI's.
- `pkg/appolly/discover/finder.go:ProcessFinder.Start` accepts `WithEnrichedProcessEvents`, exposing already-enriched process events to the embedder.

Both are why `newGraphBuilder` carries a `TODO` about moving the queues into a public structure — today the wiring is internal and only these two overrides are supported.

## Sharp edges

- **The metrics sub-pipeline may not exist.** `exportingMetrics` requires both a metrics feature and a configured OTLP or Prometheus endpoint. If it is false, `SpanNameLimiter` and every metrics exporter are never constructed — not merely idle.
- **`spanNameAggregatedMetrics` can be nil.** It is only created when app/span or service-graph metrics are on, and it is then passed into `PrometheusEndpoint` regardless, so runtime-metrics-only configurations hand a nil queue downstream.
- **The Kubernetes queue has a longer send timeout than the rest.** `routerToKubeDecorator` uses `max(InformersSyncTimeout, ChannelSendTimeout)` so a slow informer sync does not fail the pipeline. Other queues do not get this allowance.
- **Span-name limiting changes metrics but not traces.** Two spans that look identical in a metric may be distinct in a trace. This is intended and regularly surprises people comparing the two.
- **`spanPtrPromGetters` copies each span by value.** It exists only to bridge `request.Span` and `*request.Span` getters — the comment marks it explicitly as a convenience to avoid rewriting the pipeline's slice types.

## Related concepts

- [The swarm instancer](pipeline/swarm-instancer.md) — how these nodes are built and stopped
- [Typed message queues](pipeline/msg-queue.md) — the queues joining them
- [Attribute groups](observability/attribute-groups.md) — what decorators are allowed to emit
- [Architecture](01-ARCHITECTURE.md)
- [Network and stat pipelines](08-NETWORK-AND-STATS.md)

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, incident references, dates, decisions that synthesis missed. -->

---

[← Previous](04-EBPF-LAYER.md) · [Index](../index.md) · [Next →](06-CONFIGURATION.md)
