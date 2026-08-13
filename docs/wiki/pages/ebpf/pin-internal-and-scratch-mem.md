<!-- m-wiki: type=concept slug=pin-internal-and-scratch-mem topic=ebpf base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# Pin-internal maps and scratch memory

Two small C conventions in `bpf/common/` that every eBPF program in this repository is expected to follow. Both exist to work around hard kernel limits, and both are enforced by review rather than by a compiler.

## Pin-internal maps

```c
enum { OBI_PIN_INTERNAL = 100 };
```

Defined in `bpf/common/pin_internal.h:6`.

BPF maps can be *pinned* into the BPF filesystem so they outlive the loading process and can be shared with external tooling. That is occasionally what you want and usually not: a pinned map is global state with a name that must not collide, and it survives agent restarts carrying whatever was in it.

The repository rule is therefore inverted from the kernel default — **maps that are not explicitly pinned for external use must default to `OBI_PIN_INTERNAL`** ([`AGENTS.md`](../../../../AGENTS.md)). The sentinel value marks a map as OBI-internal so the loader knows it should not be exposed. Pinning is the exception that must be justified, not the default.

The Go side handles the actual filesystem path: `pkg/ebpf/tracer_linux.go:ProcessTracer.makeOtelBPFFSPath` and `pkg/ebpf/tracer_linux.go:ProcessTracer.setupOtelBPFFSPath` construct the BPF-FS location, and `pkg/ebpf/tracer_linux.go:unloadInternalMaps` tears internal maps down.

## Scratch memory

```c
#define SCRATCH_MEM_SIZED(NAME, SIZE)  /* per-CPU array of one element */
#define SCRATCH_MEM_TYPED(NAME, TYPE)  SCRATCH_MEM_SIZED(NAME, sizeof(TYPE))
#define SCRATCH_MEM(NAME)              SCRATCH_MEM_TYPED(NAME, NAME)
```

Defined in `bpf/common/scratch_mem.h:9-23`.

A BPF program gets a 512-byte stack. Anything larger — an HTTP buffer, a parsed protocol structure — cannot be a local variable. The standard workaround is a per-CPU array map with exactly one element, used as a scratch buffer: per-CPU means no locking, one element means no key management.

The macros generate that map plus a `NAME_mem()` accessor returning a pointer to it. `SCRATCH_MEM(NAME)` is the common form, where the buffer's type shares the map's name.

Using them is a repository rule: **do not introduce ad-hoc temporary buffers** when these macros apply.

## Why uniformity matters here

Both conventions are about making eBPF's constraints *visible in the code*. A hand-rolled per-CPU array works identically but tells a reviewer nothing about intent; `SCRATCH_MEM_TYPED(http_buf, http_info_t)` says "this is a stack-limit workaround" at a glance. A map without `OBI_PIN_INTERNAL` reads as a deliberate decision to expose state.

Uniformity also matters for the verifier. Scratch access through a single accessor produces a consistent pointer-provenance pattern that the verifier has already accepted everywhere else in the codebase — one of the reasons the repository asks contributors to avoid constructs that increase verifier complexity.

## Related C conventions

From [`AGENTS.md`](../../../../AGENTS.md), enforced by `make clang-tidy` and by review:

- Buffers and raw memory use `unsigned char *`, never `u8 *`.
- Prefer `const` correctness and the narrowest appropriate integer type; unsigned for sizes, counts and indexes.
- Prefer enums over macros for constants — which is why `OBI_PIN_INTERNAL` is an `enum`, not a `#define`.
- Prefer `sizeof(*ptr)` and derive sizes from objects rather than introducing separate size constants — which is what `SCRATCH_MEM_TYPED` does.
- Use `bpf_probe_read_kernel` / `bpf_probe_read_user` explicitly; plain `bpf_probe_read` only in genuinely generic code.
- Prefer `bpf_tail_call_static(...)` and define tail-call program arrays in the C code.

## Related

- [The Tracer interface](tracer-interface.md) — the Go side of these programs
- [Probe attachment](probe-attachment.md)
- [Build and validation](../09-BUILD-AND-VALIDATION.md) — `clang-format`, `clang-tidy`, and regeneration
- [eBPF layer](../04-EBPF-LAYER.md)

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD plus `AGENTS.md`.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
