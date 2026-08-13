<!-- m-wiki: type=concept slug=selection-and-typing topic=discovery base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# Selection and typing

Two decisions turn a running process into an instrumented target. **Selection** asks whether the operator wants it instrumented. **Typing** asks what kind of binary it is, which determines the eBPF programs it receives. They happen in adjacent pipeline stages and are frequently confused.

## Selection

`pkg/appolly/discover/matcher.go:FindingCriteria` resolves configuration into a list of `services.Selector` values, once, at pipeline start. It has three mutually exclusive branches, evaluated in this order:

1. **Explicit PIDs.** If `target_pids` is non-empty, it returns immediately with a PID-only selector. Every other mechanism is ignored.
2. **Deprecated service selection.** If `pkg/appolly/discover/matcher.go:OnlyDefinesDeprecatedServiceSelection` is true, the legacy `discovery > services` section is used, merged with the single-service `executable_path` / `open_port` fields when set. Criteria are regex-based and go through `normalizeRegexCriteria`.
3. **Current instrument selection.** Otherwise `discovery > instrument` is used, merged with `AutoTargetExe`, `AutoTargetLanguage` and `open_port` when set. Criteria are glob-based.

The regex-versus-glob split is the practical difference between the two styles: `discovery > services` matches with regular expressions, `discovery > instrument` with globs.

Matching itself runs in two nodes registered side by side — `criteriaMatcherProvider` for configuration criteria and `dynamicMatcherProvider` for PIDs added at runtime through `pkg/appolly/discover/dynamic_pid_selector.go:DynamicPIDSelector`. Both write to the same output queue, so a process can arrive from either path.

## Typing

`pkg/appolly/discover/typer.go:typer.asInstrumentable` opens the ELF and decides what the binary is. The key question is whether Go function offsets can be resolved:

- `pkg/appolly/discover/typer.go:typer.inspectOffsets` attempts DWARF/symbol-based offset resolution.
- If it succeeds, the executable is a Go target and gets `pkg/appolly/discover/finder.go:newGoTracersGroup` — uprobes on actual Go functions, which yields the richest spans.
- If it does not, the executable gets `pkg/appolly/discover/finder.go:newGenericTracersGroup` — syscall- and socket-level observation, which works for any language but sees less.
- `pkg/appolly/discover/typer.go:isGoProxy` identifies Go binaries acting as proxies, which need different treatment.

`pkg/appolly/discover/typer.go:typer.makeServiceAttrs` builds the `svc.Attrs` carried by every span from this process — service name, namespace, and sampler. `pkg/appolly/discover/typer.go:samplerFromConfig` resolves the per-service sampler here, meaning **sampling is decided at discovery time, not at export time**.

`pkg/appolly/discover/typer.go:typer.loadAllGoFunctionNames` and `pkg/appolly/discover/typer.go:typer.addGoFunctionName` build the set of Go symbols worth probing.

## Sharp edges

- **Selection is frozen at startup.** `FindingCriteria` runs once in `pkg/appolly/discover/finder.go:ProcessFinder.Start`. Changing configuration requires a restart; only the dynamic PID selector adds targets afterwards.
- **The two config styles do not merge.** Setting both `discovery > services` and `discovery > instrument` does not combine them — one branch wins.
- **`target_pids` silently disables everything else.** No warning is emitted about the criteria it bypassed.
- **Typing failure is not an error.** A Go binary stripped of symbols quietly becomes a generic target. Spans still appear, with less detail — which reads as "instrumentation is worse in production" when the difference is actually a build flag.
- **Selection and typing are separately cached.** The typer holds a `cacheKey`-keyed cache of `instrumentedExecutable` values, so re-running the same binary reuses the earlier classification.

## Related

- [Discovery](../03-DISCOVERY.md) — the pipeline these stages sit in
- [Process environment capture](process-env-capture.md) — what else is retained per process
- [The Tracer interface](../ebpf/tracer-interface.md) — what the chosen group implements
- [Configuration](../06-CONFIGURATION.md) — the config blocks being read

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
