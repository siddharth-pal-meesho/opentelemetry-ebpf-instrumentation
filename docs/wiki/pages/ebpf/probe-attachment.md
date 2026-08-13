<!-- m-wiki: type=concept slug=probe-attachment topic=ebpf base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# Probe attachment

Builds on [the Tracer interface](tracer-interface.md). A tracer *declares* attachment points; `pkg/ebpf/instrumenter.go` is what actually attaches them. This page covers what that code does per mechanism and the failure modes each one has.

## The attachment sweep

`pkg/ebpf/tracer_linux.go:ProcessTracer.NewExecutable` walks the mechanisms in turn. Each has its own method:

| Mechanism | Attaching function |
|---|---|
| Go function uprobes | `pkg/ebpf/instrumenter.go:instrumenter.goprobes` |
| Shared-library uprobes | `pkg/ebpf/instrumenter.go:instrumenter.uprobes` |
| USDT markers | `pkg/ebpf/instrumenter.go:instrumenter.usdtProbes` |
| Kernel probes | `pkg/ebpf/instrumenter.go:instrumenter.kprobes` |
| Static tracepoints | `pkg/ebpf/instrumenter.go:instrumenter.tracepoints` |
| Socket filters | `pkg/ebpf/instrumenter.go:instrumenter.sockfilters` |
| `sk_msg` programs | `pkg/ebpf/instrumenter.go:instrumenter.sockmsgs` |
| `sock_ops` programs | `pkg/ebpf/instrumenter.go:instrumenter.sockops` |
| BPF iterators | `pkg/ebpf/instrumenter.go:instrumenter.iters` |
| Tracing programs | `pkg/ebpf/instrumenter.go:instrumenter.tracing` |

Each returns closers that are registered on the tracer, so detachment in `pkg/ebpf/tracer_linux.go:ProcessTracer.UnlinkExecutable` is just closing what was opened.

## Resolving where to attach

Uprobes need a file offset, not a symbol name. Three helpers do that work:

- `pkg/ebpf/instrumenter.go:resolveExePath` maps a PID to its executable path and inode.
- `pkg/ebpf/instrumenter.go:processMaps` reads the process's memory mappings so shared libraries can be located.
- `pkg/ebpf/instrumenter.go:gatherOffsets` reads the ELF and fills in offsets for the declared probes.

Deduplication is by **inode**: `pkg/ebpf/instrumenter.go:instrumenter.hasModule` and `pkg/ebpf/instrumenter.go:instrumenter.addModule` ensure a shared library mapped by fifty processes is instrumented once.

## Versioned library matching

`pkg/ebpf/instrumenter.go:matchVersionedUprobeLibrary` and `pkg/ebpf/instrumenter.go:parseVersionAnnotation` let a probe declare a semantic-version constraint against the library it targets, with `pkg/ebpf/instrumenter.go:versionFromPath` extracting a version from the library's filename.

This exists because internal function signatures change between library releases — attaching an offset-based probe to the wrong version reads garbage rather than failing loudly.

**The sharp edge:** a version outside the constraint means the probe silently does not attach. There is no span-level symptom, only a debug log. "Instrumentation works on one host and not another" frequently resolves here.

## Failure modes worth knowing

- **Verifier rejection** happens at load, not attach. `pkg/ebpf/tracer_linux.go:printVerifierErrorInfo` exists because the raw kernel error is close to unreadable. Keeping programs simple and avoiding constructs that inflate verifier complexity is an explicit repository rule ([`AGENTS.md`](../../../../AGENTS.md)).
- **Socket filter attachment is special-cased.** `pkg/ebpf/instrumenter.go:instrumenter.handleSockFilterErr` wraps the error path, and `pkg/ebpf/instrumenter.go:attachSocketFilter` performs the raw attach — socket filters bind to a socket rather than to code.
- **USDT probes carry extra cleanup.** `usdtIPMapCleanup` and `usdtLinkCloser` exist so that USDT instruction-pointer map entries are removed on detach; `pkg/ebpf/instrumenter.go:usdtIPMapPIDs` enumerates the PIDs involved.
- **Byte order is handled explicitly.** `pkg/ebpf/instrumenter.go:isLittleEndian` and `pkg/ebpf/instrumenter.go:htons` appear because socket-filter configuration crosses into network byte order.

## Related

- [The Tracer interface](tracer-interface.md) — what declares these attachment points
- [Context propagation](context-propagation.md) — a tracer using `sk_msg` and `sock_ops`
- [eBPF layer](../04-EBPF-LAYER.md)
- [Build and validation](../09-BUILD-AND-VALIDATION.md) — regenerating the programs being attached

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
