<!-- m-wiki: type=top-level slug=01-architecture topic=null base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: top-level. 0 sources.

# Architecture

OBI is a single Linux agent that watches the processes on a host, decides which ones to instrument, loads eBPF programs into the kernel to observe them, and turns the resulting kernel events into OpenTelemetry traces and metrics. It never asks the observed application to cooperate: there is no SDK to import, no code to recompile, and no restart. Everything is attached from the outside, at runtime.

The agent runs **three independent observability modes** in one process. They share configuration, Kubernetes metadata and exporters, but each owns its own eBPF programs and its own processing graph.

## TL;DR

- One binary, `cmd/obi`, hosts up to three concurrent modes: application observability (AppO11y), network flows (NetO11y), and socket/stat metrics (StatsO11y).
- Each mode is gated independently by configuration and launched in its own goroutine by `pkg/instrumenter/instrumenter.go:RunWithContextInfo`; a failure in one cancels the others.
- AppO11y is itself two pipelines joined by a queue: a **discovery** pipeline that finds and attaches to processes, and a **decoration/export** pipeline that reads spans and ships them.
- Both pipelines are built from the same substrate — a [swarm instancer](pipeline/swarm-instancer.md) of independently-running nodes wired together by [typed queues](pipeline/msg-queue.md).
- The split exists because the project plans to eventually separate a high-privilege finder/instrumenter process from a lower-privilege reader/decorator process (see [`devdocs/pipeline-map.md`](../../../devdocs/pipeline-map.md)).

## Mental model

Think of OBI as a **factory floor with an attachment crew**.

The attachment crew (discovery) walks the host looking for processes. For each one it asks: does this match what the operator asked to instrument? What language is it written in? Is it a Go binary with symbols we can hook, or an opaque binary we can only observe through syscalls? Having decided, the crew bolts the right set of eBPF probes onto it.

The factory floor (the decoration and export pipeline) never sees processes at all. It sees a stream of `request.Span` values arriving from the kernel, and it moves them down a conveyor belt of optional stations — add Kubernetes labels, resolve a hostname, drop spans nobody asked for, limit span-name cardinality — until they reach one or more exporters.

The two halves are deliberately decoupled. Discovery hands spans to the pipeline through a queue and otherwise knows nothing about exporters; the pipeline knows nothing about PIDs, ELF headers, or probe attachment.

## Structure / data flow

```
                       cmd/obi/main.go:main
                                │  loads obi.Config, checks OS support + capabilities
                                ▼
              pkg/instrumenter/instrumenter.go:Run
                                │  builds shared ContextInfo (kube informer, docker
                                │  store, internal metrics, attribute groups)
                                ▼
                  RunWithContextInfo  ── errgroup ──┬────────────┬─────────────┐
                                                    │            │             │
                                            setupAppO11y   setupNetO11y  setupStatsO11y
                                                    │            │             │
                                                    ▼            ▼             ▼
                                          ┌─────────────┐   flows agent   stats agent
                                          │ DISCOVERY   │   (netolly)     (statsolly)
                                          │ ProcessFinder│
                                          │  watcher →  │
                                          │  k8s/docker →│
                                          │  language → │
                                          │  matcher →  │
                                          │  ExecTyper →│
                                          │  attacher   │
                                          └──────┬──────┘
                                                 │ ebpf.ProcessTracer per executable
                                                 │
                                                 ▼  msg.Queue[[]request.Span]
                                          ┌─────────────────────────┐
                                          │ DECORATE + EXPORT       │
                                          │  ReadDecorator →        │
                                          │  Routes → Kube →        │
                                          │  Docker → NameResolver →│
                                          │  AttributesFilter →     │
                                          │  OTEL traces / metrics  │
                                          │  Prometheus endpoint    │
                                          └─────────────────────────┘
```

## Key code locations

| What | Where |
|---|---|
| Process entry point, config load, signal handling | `cmd/obi/main.go:main` |
| Mode gating and top-level errgroup | `pkg/instrumenter/instrumenter.go:RunWithContextInfo` |
| Shared cross-cutting state (kube, docker, metrics) | `pkg/instrumenter/instrumenter.go:BuildCommonContextInfo` |
| AppO11y orchestration object | `pkg/internal/appolly/appolly.go:Instrumenter` |
| Discovery pipeline assembly | `pkg/appolly/discover/finder.go:ProcessFinder.Start` |
| Decoration + export pipeline assembly | `pkg/appolly/instrumenter.go:Build` |
| Network flows agent | `pkg/netolly/agent/agent.go:FlowsAgent` |
| Socket/stat metrics agent | `pkg/statsolly/agent/agent.go:StatsAgent` |
| Node construction + lifecycle primitive | `pkg/pipe/swarm/instancer.go:Instancer` |
| Inter-node transport | `pkg/pipe/msg/queue.go:Queue` |

## Sharp edges

- **The three modes are not equal citizens.** AppO11y is the deepest pipeline by far; NetO11y and StatsO11y are comparatively thin agents over a flow/stat record type. Reading `instrumenter.go` gives a symmetrical impression that the packages do not bear out.
- **A dynamic PID selector silently enables everything.** `RunWithContextInfo` turns on all three modes if `ctxInfo.DynamicPIDSelector` is non-nil, regardless of the feature flags — this is the "vendored as a library" path, and it surprises people reading only the config.
- **Failure in any mode kills the process.** The modes run under one `errgroup`, so a NetO11y startup error tears down a perfectly healthy AppO11y pipeline.
- **`ContextInfo` is a mutable god-object** passed to nearly every provider. Ordering inside `BuildCommonContextInfo` is load-bearing — the code comments there explain that node metadata must resolve before the internal-metrics reporter is created, and the Kubernetes informer is constructed before the reporter is wired back into it.
- **Kubernetes failure silently reshapes the pipeline.** If the informer cannot start, `pkg/internal/appolly/appolly.go:setupKubernetes` force-disables Kubernetes and starts the Docker metadata watcher as a fallback, changing which decorators actually do work.

## Related concepts

- [The swarm instancer](pipeline/swarm-instancer.md) — how every pipeline in this repo is constructed and shut down
- [Typed message queues](pipeline/msg-queue.md) — the transport between pipeline nodes
- [The Tracer interface](ebpf/tracer-interface.md) — what an eBPF program must implement to join the pipeline
- [Entry point and boot sequence](02-ENTRYPOINT.md)
- [Discovery](03-DISCOVERY.md)
- [Pipeline and export](05-PIPELINE-AND-EXPORT.md)

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, incident references, dates, decisions that synthesis missed. -->

---

[← Previous](../index.md) · [Index](../index.md) · [Next →](02-ENTRYPOINT.md)
