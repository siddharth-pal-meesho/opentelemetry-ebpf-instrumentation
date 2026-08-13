<!-- m-wiki: type=concept slug=msg-queue topic=pipeline base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# Typed message queues

`msg.Queue[T]` is the only transport between pipeline nodes in OBI. It is a generic multi-subscriber fan-out channel: one producer sends a value, every subscriber receives a copy. Nodes never hold references to each other — they hold references to queues, which is what makes the [swarm](swarm-instancer.md) wiring order-independent.

## Where it applies in this repo

| Piece | Where |
|---|---|
| The queue type | `pkg/pipe/msg/queue.go:Queue` |
| Construction | `pkg/pipe/msg/queue.go:NewQueue` |
| Sending | `pkg/pipe/msg/queue.go:195` (`Send`), `pkg/pipe/msg/queue.go:185` (`SendCtx`) |
| Subscribing | `pkg/pipe/msg/queue.go:297` (`Subscribe`) |
| Short-circuiting a disabled node | `pkg/pipe/msg/queue.go:325` (`Bypass`) |
| Signalling no more senders | `pkg/pipe/msg/queue.go:387` (`MarkCloseable`) |
| Buffer sizing option | `pkg/pipe/msg/queue.go:ChannelBufferLen` |
| Send timeout option | `pkg/pipe/msg/queue.go:SendTimeout` |
| Naming a subscriber | `pkg/pipe/msg/queue.go:SubscriberName` |

Queues are almost never built with `NewQueue` directly in pipeline code. They come from a config-aware helper, `pkg/internal/helpers/msg/queue_from_cfg.go:QueueFromConfig`, which takes the config plus a queue name — so buffer length, send timeout and the panic-on-timeout behaviour all derive from `pkg/obi/config.go:Config` rather than being hard-coded per call site.

> Method citations here use line numbers rather than symbols. `Send`, `Subscribe`, `Bypass` and friends are methods on a *generic* receiver (`func (q *Queue[T]) Send(...)`), and the wiki's Go symbol matcher cannot anchor the `[T]` parameter list. See [`SCHEMA.md`](../../SCHEMA.md).

## Why this design

**Fan-out, not hand-off.** Several exporters routinely consume the same decorated span stream — OTLP traces, the debug printer, and the metrics branch all subscribe to `exportableSpans`. A plain Go channel would give the value to exactly one of them.

**Bypass makes optional stages free.** Most decorators are conditional: no Kubernetes, no `KubeDecorator` work. Rather than branch at every call site or pay for a pass-through goroutine, a disabled node calls `Bypass` so its input queue forwards straight to its output. The graph keeps its shape; the disabled stage costs nothing. This is why `pkg/appolly/instrumenter.go:newGraphBuilder` can wire Kube and Docker decorators in series even though only one is ever active.

**Backpressure is explicit and configurable.** A send blocks when a subscriber's buffer is full. `SendTimeout` bounds that wait, and `PanicOnSendTimeout` turns exceeding it into a crash rather than silent stalling — a deliberate development aid. The Kubernetes-facing queue in the export pipeline raises its own timeout to `max(InformersSyncTimeout, ChannelSendTimeout)` so a slow informer sync cannot fail the pipeline.

**Named subscribers make stalls diagnosable.** `SubscriberName` and the internal `sendPath` helper mean a blocked send can name the route and the subscriber, instead of reporting an anonymous full channel.

## Sharp edges

- **Subscribe before start.** Subscribers must register during the swarm's construction phase; subscribing after sends have begun misses everything already delivered.
- **Closing is subscriber-driven.** `MarkCloseable` records that no further sends will occur; the queue actually closes once its accounting agrees. Calling `Send` after close trips `assertNotClosed` and panics.
- **A nil queue is a valid pipeline state.** `spanNameAggregatedMetrics` is left nil when neither app/span nor service-graph metrics are enabled, and is still passed downstream — see [pipeline and export](../05-PIPELINE-AND-EXPORT.md).

## Related

- [The swarm instancer](swarm-instancer.md) — the nodes these queues connect
- [Pipeline and export](../05-PIPELINE-AND-EXPORT.md)
- [Configuration](../06-CONFIGURATION.md) — where buffer length and timeouts come from

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
