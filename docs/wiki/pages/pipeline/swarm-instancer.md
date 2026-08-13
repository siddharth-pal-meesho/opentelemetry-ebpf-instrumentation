<!-- m-wiki: type=concept slug=swarm-instancer topic=pipeline base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# The swarm instancer

A *swarm* is OBI's answer to dependency injection and lifecycle management. Rather than a framework, it is roughly ninety lines of Go that separate **constructing** a set of long-running services from **running** them, so that a construction failure in any one of them aborts the whole group before a single goroutine starts.

Every pipeline in the repository — application discovery, span decoration and export, network flows, socket stats, process-event decoration — is built this way.

## Where it applies in this repo

| Piece | Where |
|---|---|
| Construction coordinator | `pkg/pipe/swarm/instancer.go:Instancer` |
| A node's constructor signature | `pkg/pipe/swarm/instancer.go:InstanceFunc` |
| Registering a node | `pkg/pipe/swarm/instancer.go:Instancer.Add` |
| Building the group | `pkg/pipe/swarm/instancer.go:Instancer.Instance` |
| Shortcut for infallible nodes | `pkg/pipe/swarm/instancer.go:DirectInstance` |
| Naming a node for diagnostics | `pkg/pipe/swarm/instancer.go:WithID` |
| The runnable group | `pkg/pipe/swarm/runner.go:Runner` |
| A node's run loop signature | `pkg/pipe/swarm/runner.go:RunFunc` |
| Starting every node | `pkg/pipe/swarm/runner.go:Runner.Start` |
| Completion signal | `pkg/pipe/swarm/runner.go:Runner.Done` |
| Bounded shutdown | `pkg/pipe/swarm/runner.go:WithCancelTimeout` |
| Pass-through node | `pkg/pipe/swarm/runner.go:Bypass` |

Representative uses: `pkg/appolly/discover/finder.go:ProcessFinder.Start` builds the eight-node discovery pipeline, and `pkg/appolly/instrumenter.go:newGraphBuilder` builds the export pipeline.

## The two-phase contract

An `InstanceFunc` takes a context and returns a `RunFunc` or an error. Phase one calls every registered `InstanceFunc`; phase two runs every returned `RunFunc`.

The split is the whole point. Nodes open sockets, load eBPF programs, and connect to the Kubernetes API during construction — all operations that can fail. If the fourth of eight nodes fails, `Instance` cancels the build context and returns the error, and nodes one through three never run. Without the split, a partially-started pipeline would need to be torn down.

`Instance` also hands each constructor a *build context* that it cancels on failure, so a node that started background work during construction learns to stop.

## Why this design

Three properties fall out of it:

- **Fail-fast, all-or-nothing startup.** No half-built pipelines, so shutdown logic never has to handle a partially-initialized graph.
- **Order-independent wiring.** Nodes do not reference each other; they reference [queues](msg-queue.md). `Add` order therefore does not encode data flow — reading it as a topology is a mistake, and the actual flow is only visible in which queue each node reads and writes.
- **Diagnosable liveness.** `WithID` names a node, and the runner uses that name to report a node that exits unexpectedly. Without an explicit ID, a node is labelled by its registration index (`#3`), which is why every production `Add` call in this repository passes one.

`DirectInstance` covers nodes whose construction cannot fail, avoiding a wrapper closure that only ever returns `nil` error.

## Shutdown

`Runner.Start` accepts `WithCancelTimeout`. On context cancellation each node is expected to return from its `RunFunc`; a node that does not is reported as a `CancelTimeoutError` naming the offending ID. `Runner.Done` yields a channel carrying the first error, which callers such as `pkg/internal/appolly/appolly.go:Instrumenter.WaitUntilFinished` select on alongside their own timeout.

## Related

- [Typed message queues](msg-queue.md) — what actually connects the nodes
- [Architecture](../01-ARCHITECTURE.md)
- [Pipeline and export](../05-PIPELINE-AND-EXPORT.md)
- [Discovery](../03-DISCOVERY.md)

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
