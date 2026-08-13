<!-- m-wiki: type=top-level slug=03-discovery topic=null base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: top-level. 0 sources.

# Discovery

Discovery answers one question repeatedly, for the life of the agent: *which processes on this host should be instrumented, and with what?* It is a pipeline, not a scan — processes appear and disappear continuously, and every stage is a long-running node reacting to events rather than a function returning a list.

The output of discovery is a stream of `ebpf.Instrumentable` values, each carrying a file, its detected runtime, and the tracer that was attached to it.

## TL;DR

- Assembled in `pkg/appolly/discover/finder.go:ProcessFinder.Start` as a swarm of eight named nodes connected by queues.
- The first node watches the process table; every later node either enriches, filters, classifies, or attaches.
- Selection criteria are computed once at startup by `pkg/appolly/discover/matcher.go:FindingCriteria` and support three mutually-exclusive styles: explicit PIDs, the deprecated `discovery > services` section, and the current `discovery > instrument` section.
- Classification happens in `pkg/appolly/discover/typer.go:typer.asInstrumentable`, which decides Go-with-offsets versus generic, and that decision selects the tracer group.
- Attachment and detachment are symmetric: the same event loop that runs a tracer on `EventCreated` unlinks it on `EventDeleted`.

## Mental model

A **conveyor with gates**. Every process event enters at one end as a bare PID. Each station either bolts on more information (Kubernetes pod, Docker container, detected language), or decides the item does not belong on the belt at all. By the time an item reaches the far end it has accumulated enough metadata to answer "which eBPF programs does this deserve?", and the last station physically attaches them.

Two properties follow from this being a pipeline rather than a scan:

1. **Enrichment is optional and ordered.** If Kubernetes is disabled, that node passes events through untouched — the pipeline shape does not change, only what each node does.
2. **Nothing is retroactive.** A process that starts before its Kubernetes metadata is available is enriched with whatever was known at the moment it flowed through.

## Structure / data flow

Node IDs below are the literal `swarm.WithID(...)` strings, which is what appears in log output.

```
ProcessWatcher                 poll /proc + dynamic PID additions
      │ []Event[ProcessAttrs]
      ▼
WatcherKubeEnricher            attach pod / owner metadata      (no-op if kube disabled)
      ▼
DockerDiscoveryDecoratorProvider   attach container metadata    (no-op if kube enabled)
      ▼
LanguageDecoratorProvider      detect runtime from the ELF
      ▼
CriteriaMatcher ─┬─ DynamicMatcher      config criteria │ runtime-added PIDs
      │ []Event[ProcessMatch]
      ▼
ExecTyper                      read ELF, inspect Go offsets, build svc.Attrs
      │ []Event[ebpf.Instrumentable]
      ▼
ContainerStoreUpdater          update container DB *before* later stages see the event
      ▼
traceAttacher                  create + attach the ProcessTracer
      │ Event[*ebpf.Instrumentable]
      ▼
  instrumentedEventLoop        run the tracer; spans flow to the export pipeline
```

`ContainerStoreUpdater` sitting between `ExecTyper` and `traceAttacher` is deliberate. The code comment in `ProcessFinder.Start` records the reason: the updater could subscribe to `executableTypes` directly with no output queue, but forcing downstream stages to wait on a separate `storedExecutableTypes` queue guarantees the container database is populated before anything reads it, eliminating a race.

## Key code locations

| What | Where |
|---|---|
| Pipeline assembly, all node registrations | `pkg/appolly/discover/finder.go:ProcessFinder.Start` |
| Constructor and shared inputs | `pkg/appolly/discover/finder.go:NewProcessFinder` |
| Selection criteria resolution | `pkg/appolly/discover/matcher.go:FindingCriteria` |
| ELF inspection and classification | `pkg/appolly/discover/typer.go:typer.asInstrumentable` |
| Filtering and event classification | `pkg/appolly/discover/typer.go:typer.FilterClassify` |
| Node provider for the typer | `pkg/appolly/discover/typer.go:ExecTyperProvider` |
| Runtime-mutable PID set | `pkg/appolly/discover/dynamic_pid_selector.go:DynamicPIDSelector` |
| Tracer group: always-on programs | `pkg/appolly/discover/finder.go:newCommonTracersGroup` |
| Tracer group: Go binaries | `pkg/appolly/discover/finder.go:newGoTracersGroup` |
| Tracer group: everything else | `pkg/appolly/discover/finder.go:newGenericTracersGroup` |
| Attach / detach event loop | `pkg/internal/appolly/appolly.go:Instrumenter.instrumentedEventLoop` |
| Per-process environment capture | `pkg/internal/procs/procenv.go:EnvVars` |

## The three tracer groups

Which eBPF programs a process gets is decided entirely by these three constructors:

- **Common** (`newCommonTracersGroup`) — loaded once for any tracer group. Contains the context-propagation injector when `context_propagation` requests headers or TCP, the log enricher when enabled, and the CUDA/GPU tracer when CUDA instrumentation is on. All three are conditional; the group is frequently empty.
- **Go** (`newGoTracersGroup`) — a single `gotracer`, used when the ELF is a Go binary whose function offsets could be resolved.
- **Generic** (`newGenericTracersGroup`) — a single `generictracer`, used for everything else. It observes syscalls and sockets rather than language-level functions.

## Sharp edges

- **Kube and Docker decorators are wired in series but only one is active.** Both nodes always exist; each bypasses its queues when the other owns metadata. Reading the graph suggests double decoration that does not happen.
- **Criteria are computed once.** `FindingCriteria` runs at `Start`, so configuration-driven selection is fixed for the process lifetime. Only the `DynamicPIDSelector` path adds targets afterwards.
- **`target_pids` short-circuits everything.** If set, `FindingCriteria` returns immediately with a PID-only selector and every other selection mechanism is ignored.
- **Deprecated and current selection styles are mutually exclusive, not merged.** `OnlyDefinesDeprecatedServiceSelection` decides which branch runs; setting both does not combine them.
- **Detachment is by executable, not by process.** On `EventDeleted` the loop calls `UnlinkExecutable`; a tracer serving several instances of the same binary survives until the last one exits, which is why `EventInstanceDeleted` exists as a separate case.
- **Discovery holds memory for every matched process.** Anything captured here lives as long as the process does — see [process environment capture](discovery/process-env-capture.md) for a case where this caused unbounded heap growth.

## Related concepts

- [Selection and typing](discovery/selection-and-typing.md) — how criteria are expressed and how a binary is classified
- [Process environment capture](discovery/process-env-capture.md) — why only an allowlist of env vars is retained
- [The Tracer interface](ebpf/tracer-interface.md) — what `traceAttacher` ultimately constructs
- [Probe attachment](ebpf/probe-attachment.md) — what happens after a tracer is selected
- [eBPF layer](04-EBPF-LAYER.md)

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, incident references, dates, decisions that synthesis missed. -->

---

[← Previous](02-ENTRYPOINT.md) · [Index](../index.md) · [Next →](04-EBPF-LAYER.md)
