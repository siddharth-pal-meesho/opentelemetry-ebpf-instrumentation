<!-- m-wiki: type=concept slug=tracer-interface topic=ebpf base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# The Tracer interface

`pkg/ebpf/tracer.go:Tracer` is the contract every eBPF program bundle implements. It is deliberately declarative: a tracer does not attach anything itself, it *declares* which kernel attachment points it wants, and the loader in `pkg/ebpf/instrumenter.go` performs the attachment.

## The interface family

| Interface | Responsibility |
|---|---|
| `pkg/ebpf/tracer.go:CommonTracer` | Load compiled specs, register closers, populate tail-call arrays |
| `pkg/ebpf/tracer.go:KprobesTracer` | Declare kernel probes and tracepoints |
| `pkg/ebpf/tracer.go:Tracer` | The full contract — everything above plus user-space probes, socket programs, iterators, and the span-producing `Run` loop |
| `pkg/ebpf/tracer.go:UtilityTracer` | Background programs that never produce spans; run via `pkg/ebpf/tracer_linux.go:RunUtilityTracer` |
| `pkg/ebpf/tracer.go:PIDsAccounter` | Gate which processes a loaded program reports on |

Concrete implementations are one package each under `pkg/internal/ebpf/`, all constructed through a `New` function: `pkg/internal/ebpf/gotracer/gotracer.go:New`, `pkg/internal/ebpf/generictracer/generictracer.go:New`, `pkg/internal/ebpf/tpinjector/tpinjector.go:New`, `pkg/internal/ebpf/logenricher/logenricher.go:New`, `pkg/internal/ebpf/gpuevent/gpuevent.go:New`.

## What a tracer declares

Each method returns a collection the loader iterates. A tracer that does not use a mechanism returns an empty map or slice.

- `GoProbes()` — uprobes on Go functions, keyed by symbol, resolved through DWARF offsets
- `UProbes()` — uprobes on shared libraries, nested by library then symbol
- `USDTProbes()` — USDT marker probes
- `KProbes()` / `Tracepoints()` — kernel functions and static tracepoints
- `SocketFilters()`, `SockMsgs()`, `SockOps()` — socket-layer programs
- `Iters()`, `Tracing()` — BPF iterators and tracing programs

Three methods are metadata rather than attachment: `Required()` states whether failure to load should be fatal, `Capabilities()` reports what the tracer can observe, and `RegisterOffsets` supplies the Go offsets that `GoProbes` entries resolve against.

The library-reference methods (`RecordInstrumentedLib`, `AddInstrumentedLibRef`, `AlreadyInstrumentedLib`, `UnlinkInstrumentedLib`) exist because a shared library is instrumented once but referenced by many processes; they are a refcount, keyed by inode.

## Why this design

**One loader, many observers.** Attachment is genuinely hard — versioned library matching, offset resolution, endianness, verifier failure reporting. Centralizing it means a new tracer implements accessor methods and inherits all of that. `pkg/ebpf/instrumenter.go:instrumenter.uprobes` and its siblings are written once.

**Load once, attach per executable.** `pkg/ebpf/tracer_linux.go:ProcessTracer.loadTracers` loads programs into the kernel a single time for a tracer group; `pkg/ebpf/tracer_linux.go:ProcessTracer.NewExecutable` attaches per binary. Programs are therefore *shared* across processes, and the only thing separating one service's events from another's is PID filtering through `pkg/ebpf/tracer.go:ProcessTracer.AllowPID` and `pkg/ebpf/tracer.go:ProcessTracer.BlockPID`.

That last point is the sharpest consequence of the design: **PID accounting is the tenancy boundary**, so an accounting bug leaks spans between services rather than merely dropping them.

**The width is the cost.** `Tracer` has well over a dozen methods. "Implements Tracer" tells you almost nothing about what a tracer observes — read its `New` constructor and the non-empty methods instead.

## Related

- [Probe attachment](probe-attachment.md) — how each declared mechanism is attached
- [Context propagation](context-propagation.md) — what `tpinjector` declares and why
- [eBPF layer](../04-EBPF-LAYER.md)
- [Discovery](../03-DISCOVERY.md) — where tracer groups are selected

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
