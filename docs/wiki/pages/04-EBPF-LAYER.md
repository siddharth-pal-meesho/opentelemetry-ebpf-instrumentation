<!-- m-wiki: type=top-level slug=04-ebpf-layer topic=null base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: top-level. 0 sources.

# eBPF layer

This is the boundary between Go and the kernel. Above it, OBI deals in `request.Span` values and pipeline nodes. Below it, there are C programs compiled to BPF bytecode, maps holding kernel-side state, and probes attached to functions, syscalls, tracepoints and sockets.

The layer's job is to make that boundary uniform: every eBPF program, no matter what it observes, is presented to the rest of the agent as an implementation of one interface.

## TL;DR

- `pkg/ebpf/tracer.go:Tracer` is the contract. A tracer declares which probe kinds it wants; the loader attaches them.
- `pkg/ebpf/tracer.go:ProcessTracer` owns a group of tracers loaded for one process type, and its lifecycle is load → attach → run → unlink.
- The C sources live under `bpf/`, one directory per program. The Go bindings (`*_bpfel.go`) and bytecode (`*.o`) are **generated** by `bpf2go` and must never be hand-edited.
- PID filtering is how a shared probe stays scoped: `AllowPID` and `BlockPID` gate which processes a loaded program reports on.
- Minimum supported kernel is 5.8 with BTF, plus RHEL-family 4.18 with backported patches — see [`AGENTS.md`](../../../AGENTS.md).

## Mental model

**A socket adapter for the kernel.** The kernel offers many different attachment mechanisms — uprobes on user-space functions, kprobes on kernel functions, tracepoints, socket filters, `sk_msg` and `sock_ops` hooks, BPF iterators, USDT markers. Each has different attachment rules and different failure modes.

Rather than let every tracer deal with that, `Tracer` exposes one method per mechanism. A tracer implements the ones it needs and returns empty maps for the rest. The instrumenter walks those methods in turn and does the attaching. Adding a new observation point means implementing one more method, not writing new attachment logic.

## Structure / data flow

```
   discovery decides a tracer group
              │
              ▼
   NewProcessTracer(type, []Tracer, cfg, metrics)
              │
              ├─ loadTracers        LoadSpecs() → rewrite constants → load into kernel
              │                     SetupTailCalls() → populate program arrays
              │
              ├─ NewExecutable      per-executable attachment:
              │     goprobes  ──►  uprobes on Go functions at resolved offsets
              │     uprobes   ──►  uprobes on shared libraries (versioned matching)
              │     usdtProbes ─►  USDT markers
              │     kprobes   ──►  kernel functions
              │     tracepoints ─► static tracepoints
              │     sockfilters ─► socket filter programs
              │     sockmsgs / sockops / iters / tracing
              │
              ├─ Run(ctx, eventContext, out)   read ring buffers → []request.Span
              │
              └─ UnlinkExecutable   detach when the last instance exits
```

Programs communicate upward through ring buffers and maps held in `pkg/ebpf/common/common.go:EBPFEventContext`, a process-global structure shared by every tracer so that internal maps are loaded once rather than per-tracer.

## Key code locations

| What | Where |
|---|---|
| The tracer contract | `pkg/ebpf/tracer.go:Tracer` |
| Shared load/tail-call behavior | `pkg/ebpf/tracer.go:CommonTracer` |
| Kernel-probe subset | `pkg/ebpf/tracer.go:KprobesTracer` |
| Non-span background programs | `pkg/ebpf/tracer.go:UtilityTracer` |
| PID gating contract | `pkg/ebpf/tracer.go:PIDsAccounter` |
| A discovered, classified executable | `pkg/ebpf/tracer.go:Instrumentable` |
| Per-process tracer group | `pkg/ebpf/tracer.go:ProcessTracer` |
| Construction | `pkg/ebpf/tracer_linux.go:NewProcessTracer` |
| Loading specs into the kernel | `pkg/ebpf/tracer_linux.go:ProcessTracer.loadTracers` |
| Attaching to one executable | `pkg/ebpf/tracer_linux.go:ProcessTracer.NewExecutable` |
| Event loop producing spans | `pkg/ebpf/tracer_linux.go:ProcessTracer.Run` |
| Detachment | `pkg/ebpf/tracer_linux.go:ProcessTracer.UnlinkExecutable` |
| Utility (non-span) tracer runner | `pkg/ebpf/tracer_linux.go:RunUtilityTracer` |
| Go uprobe attachment | `pkg/ebpf/instrumenter.go:instrumenter.goprobes` |
| Library uprobe attachment | `pkg/ebpf/instrumenter.go:instrumenter.uprobes` |
| Kprobe attachment | `pkg/ebpf/instrumenter.go:instrumenter.kprobes` |
| USDT attachment | `pkg/ebpf/instrumenter.go:instrumenter.usdtProbes` |
| Socket filter attachment | `pkg/ebpf/instrumenter.go:instrumenter.sockfilters` |
| Shared kernel-side state | `pkg/ebpf/common/common.go:EBPFEventContext` |

## Generated artifacts

Four file patterns under `pkg/internal/ebpf/` and `pkg/ebpf/common/` are machine-generated and listed as such in [`AGENTS.md`](../../../AGENTS.md):

| Pattern | Produced by |
|---|---|
| `*_bpfel.go`, `*_bpfeb.go` | `bpf2go`, from the C sources in `bpf/` |
| `*_bpfel.o`, `*_bpfeb.o` | clang, compiled BPF bytecode |
| `bpf/bpfcore/` | copied/generated CO-RE headers including `vmlinux.h` |

Editing any of them by hand is discarded on the next `make generate`. Changing a `.c` file under `bpf/` obliges you to regenerate — see [build and validation](09-BUILD-AND-VALIDATION.md).

## Sharp edges

- **`Tracer` is a wide interface.** Implementations return empty maps for mechanisms they do not use, so "implements Tracer" says very little about what a tracer actually attaches to. Read the constructor, not the interface.
- **Loading is per tracer group, attachment is per executable.** `loadTracers` runs once; `NewExecutable` runs for every binary. Confusing the two leads to expecting per-process isolation that does not exist — programs are shared, and only PID filtering separates their output.
- **PID filtering is the only tenancy boundary.** A bug in `AllowPID`/`BlockPID` accounting leaks spans between services rather than merely losing them.
- **Verifier rejections surface late.** They appear at load time with kernel-generated messages; `printVerifierErrorInfo` exists specifically because the raw error is close to unreadable.
- **Library uprobes do version matching.** `matchVersionedUprobeLibrary` and `parseVersionAnnotation` mean a probe can silently not attach because the target library's version fell outside a constraint, with no span-level symptom.
- **eBPF C anchors cannot be verified by this wiki.** `.c`/`.h` files have no symbol pattern in the citation checker, so C references are cited by line number and will not be auto-corrected when they move.

## Related concepts

- [The Tracer interface](ebpf/tracer-interface.md) — the contract in detail
- [Probe attachment](ebpf/probe-attachment.md) — how each mechanism is attached
- [Context propagation](ebpf/context-propagation.md) — the kernel-side trace-context injection
- [Discovery](03-DISCOVERY.md) — where tracer groups are chosen
- [Build and validation](09-BUILD-AND-VALIDATION.md) — regenerating bindings

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, incident references, dates, decisions that synthesis missed. -->

---

[← Previous](03-DISCOVERY.md) · [Index](../index.md) · [Next →](05-PIPELINE-AND-EXPORT.md)
