<!-- m-wiki: type=concept slug=process-env-capture topic=discovery base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# Process environment capture

Discovery reads each matched process's environment from `/proc/<pid>/environ` because several downstream components need values from it — the service name, OTLP resource attributes, exporter endpoints. What it keeps is an **allowlist**, not the whole environment, and the reason is a memory bound.

## Where it applies in this repo

| Piece | Where |
|---|---|
| Public read entry point | `pkg/internal/procs/procenv.go:EnvVars` |
| Parse and filter | `pkg/internal/procs/procenv.go:envStrsToMap` |
| The allowlist itself | `pkg/internal/procs/procenv.go:20-31` |

`EnvVars` reads via `procfs.NewProc(pid).Environ()` and hands the raw `KEY=VALUE` strings to `envStrsToMap`, which splits on the first `=`, trims both sides, drops entries with an empty key or value, and keeps only allowlisted keys.

## The invariant

> The captured map is retained per process, referenced from every span, for the process lifetime.

That single sentence is why the allowlist exists. The map is not a transient parse result — it is held for as long as the process runs and is reachable from span data. Capturing the full environment therefore grows the live heap proportional to **number of processes × environment size**.

On nodes dense with large-environment processes this is not a marginal cost. A JVM commonly carries a multi-kilobyte environment (a long `CLASSPATH`, container-injected service-discovery variables); multiply that by every process on a busy node and the agent's heap grows without bound until it is OOM-killed. That failure mode is what the allowlist prevents, and it is recorded in the comment above the map.

## What is on the list, and why each entry is there

Every entry is annotated in place with its consumer, so the list can be audited rather than guessed at:

| Variable | Consumed by |
|---|---|
| `OTEL_SERVICE_NAME` | service naming in `discover/exec` |
| `OTEL_RESOURCE_ATTRIBUTES` | `discover/exec` and OTLP resource attributes |
| `OTEL_EXPORTER_OTLP_PROTOCOL` and the traces/metrics variants | detecting that the application already exports OTLP |
| `OTEL_EXPORTER_OTLP_ENDPOINT` and the traces/metrics variants | same detection |
| `TMPDIR` | Java agent injector temp directory |
| `CLASSPATH` | Java route harvester |

The `OTEL_EXPORTER_OTLP_*` group matters for a specific behaviour: OBI uses it to notice that a process is *already* instrumented with an OpenTelemetry SDK, so it can avoid double-instrumenting it.

## Maintaining the list

The allowlist is a coupling point that the compiler cannot check. Reading a new environment variable somewhere in the codebase without adding it here yields an empty string at runtime, with no error.

The discipline the code establishes: **add the key here in the same change that adds the reader, with a comment naming the consumer.** Conversely, when a consumer is deleted, its entry should go too — a stale entry silently reintroduces part of the memory cost the list exists to avoid.

## Related

- [Selection and typing](selection-and-typing.md) — the stage that triggers this read
- [Discovery](../03-DISCOVERY.md) — where per-process state accumulates
- [Configuration](../06-CONFIGURATION.md) — OBI's own environment surface, which is unrelated to this one

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
