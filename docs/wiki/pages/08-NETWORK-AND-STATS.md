<!-- m-wiki: type=top-level slug=08-network-and-stats topic=null base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: top-level. 0 sources.

# Network and stat pipelines

Two of OBI's three observability modes have nothing to do with application spans. **NetO11y** produces network flow metrics — who talked to whom, how much. **StatsO11y** produces socket-level statistics. Both bypass discovery entirely: they do not care which processes match a selector, because they observe traffic, not programs.

They are structurally simpler than AppO11y and are documented together for that reason.

## TL;DR

- NetO11y is built by `pkg/netolly/agent/agent.go:FlowsAgent` and run by `pkg/netolly/agent/agent.go:Flows.Run`.
- StatsO11y is built by `pkg/statsolly/agent/agent.go:StatsAgent` and run by `pkg/statsolly/agent/agent.go:Stats.Run`.
- Both reuse the same [swarm](pipeline/swarm-instancer.md) and [queue](pipeline/msg-queue.md) substrate as AppO11y, and both end in the same OTLP and Prometheus exporters.
- Neither runs a `ProcessTracer`. NetO11y attaches through traffic control (TC) or a socket filter; StatsO11y reads a ring buffer directly.
- Both share the Kubernetes decorator, reverse DNS, GeoIP and CIDR stages — the decoration chain is where they overlap most with AppO11y.

## Mental model

**Same conveyor, different raw material.** AppO11y's belt carries `request.Span`. NetO11y's carries flow records; StatsO11y's carries stat records. The stations differ in what they attach — a flow gets a CIDR classification and a GeoIP country, a span gets a route and a peer name — but the belt itself, the way stations are registered and shut down, is identical.

The consequence worth internalizing: understanding `swarm` and `msg.Queue` transfers completely between all three modes. What does not transfer is the acquisition layer.

## Structure / data flow

Both flowcharts are maintained upstream in [`devdocs/pipeline-map.md`](../../../devdocs/pipeline-map.md); this is the code-anchored view.

```
NETWORK FLOWS
   eBPF map tracer  ─┐
   eBPF ringbuf tracer ┴─► Flows.buildPipeline
                              │
                    internet protocol filter   (optional)
                              ▼
                        flow deduper           (optional)
                              ▼
                    Kubernetes decorator       (optional)
                              ▼
                        reverse DNS            (optional)
                              ▼
                          GeoIP                (optional)
                              ▼
                     CIDR redecorator          (optional)
                              ▼
                     attributes filter
                       ├─► OTLP metrics
                       ├─► Prometheus
                       └─► flow printer

SOCKET STATS
   eBPF ringbuf tracer ──► statsAgent pipeline
                              ▼
                    (same decoration chain)
                              ▼
                     attributes filter
                       ├─► OTLP metrics
                       ├─► Prometheus
                       └─► stat printer
```

## Key code locations

| What | Where |
|---|---|
| Flows agent construction | `pkg/netolly/agent/agent.go:FlowsAgent` |
| Flows agent run loop | `pkg/netolly/agent/agent.go:Flows.Run` |
| Flows shutdown | `pkg/netolly/agent/agent.go:Flows.stop` |
| Flows pipeline assembly | `pkg/netolly/agent/pipeline.go:Flows.buildPipeline` |
| eBPF fetcher selection (TC vs socket filter) | `pkg/netolly/agent/agent.go:newFetcher` |
| TC monitor mode resolution | `pkg/netolly/agent/agent.go:monitorMode` |
| Ingress/egress direction resolution | `pkg/netolly/agent/agent.go:flowDirections` |
| Agent lifecycle status | `pkg/netolly/agent/agent.go:Flows.Status` |
| Flow record definitions | `pkg/netolly/flowdef` |
| Stats agent construction | `pkg/statsolly/agent/agent.go:StatsAgent` |
| Stats agent run loop | `pkg/statsolly/agent/agent.go:Stats.Run` |
| Stats fetcher | `pkg/statsolly/agent/agent.go:newFetcher` |
| Mode launch points | `pkg/instrumenter/instrumenter.go:setupNetO11y`, `pkg/instrumenter/instrumenter.go:setupStatsO11y` |
| Network config block | `pkg/obi/network_cfg.go` |
| Stats config block | `pkg/obi/stats_cfg.go` |
| Shared decoration stages | `pkg/internal/pipe/cidr`, `pkg/internal/pipe/geoip`, `pkg/internal/pipe/rdns` |

## Traffic control is the sharp difference

NetO11y's acquisition path is the one genuinely novel piece. `newFetcher` chooses between a TC-based fetcher and a socket-filter fetcher, and `monitorMode` decides how TC programs are attached. This brings in `pkg/ebpf/tcmanager` and an interface manager, because TC programs attach to *network interfaces* rather than processes — interfaces that appear and disappear as containers start and stop.

Nothing in AppO11y has an equivalent: there is no per-interface lifecycle to manage when you attach to a binary.

## Sharp edges

- **These modes ignore discovery entirely.** Configuring `discovery > instrument` has no effect on flow or stat metrics. Operators regularly expect selection criteria to scope network metrics; they do not.
- **`Deduper == DeduperNone` changes exported attributes.** `attributeGroups` adds `GroupNetIfaceDirection` only in that case, so turning deduplication off silently widens metric cardinality.
- **CIDR and GeoIP are attribute-group gated too.** Both add attribute groups at boot based on config, so enabling them later in a config reload is not sufficient.
- **Each agent has its own `Status` enum and its own `stop`.** They do not share a lifecycle type with AppO11y despite looking parallel, so shutdown semantics must be read per agent.
- **TC attachment can outlive the process.** Traffic control programs attached to interfaces need explicit cleanup; that is what `Flows.stop` is for, and why a hard kill can leave state behind.

## Related concepts

- [The swarm instancer](pipeline/swarm-instancer.md) — shared with these pipelines
- [Typed message queues](pipeline/msg-queue.md)
- [Attribute groups](observability/attribute-groups.md) — where the network-specific groups are decided
- [Architecture](01-ARCHITECTURE.md) — how the three modes relate
- [Kubernetes metadata](07-KUBERNETES-METADATA.md) — the shared decorator

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, incident references, dates, decisions that synthesis missed. -->

---

[← Previous](07-KUBERNETES-METADATA.md) · [Index](../index.md) · [Next →](09-BUILD-AND-VALIDATION.md)
