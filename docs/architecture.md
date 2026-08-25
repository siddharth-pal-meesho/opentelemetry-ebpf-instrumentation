# Architecture

> OBI (OpenTelemetry eBPF Instrumentation) — a Linux agent that instruments running
> processes from the outside using eBPF: no SDK, no recompilation, no restart.
>
> This document is the code-grounded architecture reference: module boundaries,
> request lifecycle, entry points, and load-bearing invariants with file citations.
> For the narrative, symbol-anchored walkthrough see [wiki/index.md](wiki/index.md);
> for upstream protocol/design write-ups see [`devdocs/`](../devdocs/README.md).

## Section 1 — High-level design

### Service purpose

OBI is a single Go binary (`bin/obi`) that runs as a privileged agent on a Linux host or
as a Kubernetes DaemonSet. It discovers running processes, decides which ones match the
user's selection criteria, attaches eBPF programs to them, reads the resulting kernel
events, decorates them with process/container/Kubernetes metadata, and exports them as
OpenTelemetry traces and metrics (or a Prometheus scrape endpoint).

This repository is a **fork** of
[open-telemetry/opentelemetry-ebpf-instrumentation](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation).
`main` tracks upstream; feature branches carry Meesho-local patches on top. See
[Fork context](#fork-context) below.

### System context

There is no inbound API in the usual sense — OBI is an agent, not a service.

| Direction | Surface | Where |
| --- | --- | --- |
| Inbound (data) | Kernel eBPF events (ring buffers, perf maps, socket filters) | `pkg/internal/ebpf/*` |
| Inbound (control) | YAML config file + `OTEL_EBPF_*` environment variables | `pkg/obi/config.go` |
| Inbound (ops) | Health check (TCP port or Unix socket), pprof port | `pkg/health/`, `cmd/obi/main.go` |
| Outbound | OTLP traces/metrics/logs (gRPC + HTTP), Prometheus scrape endpoint, stdout debug printers | `pkg/export/otel/`, `pkg/export/prom/`, `pkg/export/debug/` |
| Sidecar service | `k8s-cache` — a shared Kubernetes metadata informer, so N agents don't each hammer the API server | `cmd/k8s-cache/main.go`, `pkg/kube/` |

### Module boundaries

| Path | Owns |
| --- | --- |
| `bpf/` | eBPF C programs, maps, shared headers. One directory per program bundle (`generictracer`, `gotracer`, `tpinjector`, `netolly`, `statsolly`, …). |
| `pkg/internal/ebpf/<name>/` | The Go loader for each `bpf/<name>/` bundle, plus its `bpf2go`-generated bindings (`*_bpfel.go`, `*_bpfel.o`). |
| `pkg/ebpf/` | The `Tracer` contract every bundle implements, and the attach/load/unlink lifecycle. |
| `pkg/appolly/`, `pkg/internal/appolly/` | Application observability: discovery pipeline and span decoration. `pkg/appolly/app/request` defines `request.Span`, the pipeline's currency. |
| `pkg/netolly/`, `pkg/statsolly/` | The two modes that bypass discovery entirely — network flows and TCP stat metrics. |
| `pkg/pipe/` | The pipeline substrate: `swarm` (construct-then-run DAG) and `msg` (typed multi-subscriber queues). |
| `pkg/export/` | Everything that leaves the process: OTel exporters, Prometheus, attribute selection, feature gating. |
| `pkg/obi/` | The typed configuration root and OS/capability preflight checks. |
| `pkg/kube/`, `pkg/docker/` | Metadata enrichment sources. |
| `cmd/` | Binary entry points, plus small codegen/validation tools (`obi-schema`, `config-docs`, `check-config-v2-*`). |
| `internal/test/integration/` | Integration-test harness (Docker Compose driven). |

### Architecture philosophy

Optimized for **low steady-state overhead in the kernel and bounded memory in userspace**,
traded against generality. Two consequences show up everywhere in the code:

1. **Everything optional is bypassable, not conditional.** Pipeline stages that a config
   doesn't enable are not branched around at runtime — they are wired out at construction
   time via `swarm.Bypass` (`pkg/pipe/swarm/runner.go:31`), so the hot path has no checks.
2. **Every buffer is explicitly capped.** Unbounded growth is treated as a defect, not a
   tuning problem. eBPF maps are fixed-size by necessity; userspace aggregation carries
   the same discipline (see `pkg/export/otel/api_dependency.go:41-44`).

<!-- assumption: philosophy inferred from code (bypass wiring, explicit caps, deadlock
     panic in msg.Queue); Phase 5/6 interviews did not run in this non-interactive session -->

### Data flow (request lifecycle)

OBI runs up to three independent pipelines, gated by
`cfg.Enabled(FeatureAppO11y | FeatureNetO11y | FeatureStatsO11y)` in
`pkg/instrumenter/instrumenter.go` (`RunWithContextInfo`). Each runs in its own
`errgroup` goroutine; if one fails, the others are cancelled.

The **application pipeline** — the primary path — is two connected pipelines:

1. `ProcessWatcher` observes process start/stop.
2. Optional `WatcherKubeEnricher` / `DockerEnricher` attach pod and container metadata.
3. `CriteriaMatcher` drops processes that don't match the user's selection criteria.
4. `ExecTyper` classifies the ELF (Go binary vs. generic) and picks the tracer group.
5. `TraceAttacher` builds one `ebpf.Tracer` per executable and attaches its probes.
6. Tracers emit `[]request.Span` batches into a `msg.Queue`.
7. `traces.ReadDecorator` → routes decorator → Kubernetes/Docker decorators → name
   resolver → attribute filter.
8. The stream forks: OTel trace exporter on one side; unknown-IP dropper → span-name
   cardinality limiter → RED metrics / service-graph metrics / Prometheus endpoint on
   the other.

The upstream flowcharts for all three pipelines (including the optional stages) are in
[`devdocs/pipeline-map.md`](../devdocs/pipeline-map.md) — that file is authoritative for
stage ordering; do not restate it here.

### Cross-cutting concerns

| Concern | Implementation | Entry point |
| --- | --- | --- |
| Pipeline construction | Two-phase: every node registers an `InstanceFunc`; **no node runs until all constructors succeed** | `pkg/pipe/swarm/instancer.go:42` |
| Inter-node transport | Typed, multi-subscriber `msg.Queue[T]`; subscription and bypass are wire-up-time only | `pkg/pipe/msg/queue.go` |
| Deadlock detection | A `Send` blocked longer than `defaultSendTimeout` (1 min) panics rather than hanging silently | `pkg/pipe/msg/queue.go:20` |
| Configuration | Struct tags drive YAML *and* env *and* JSON-schema *and* generated docs — one struct, four surfaces | `pkg/obi/config.go` |
| Feature gating | Bitmask over `export.Features`; every exporter asks the bitmask, never the raw config | `pkg/export/feature.go` |
| Self-instrumentation | Internal metrics reporter injected into queues via `utilizationGauge` | `pkg/export/imetrics/` |
| Logging | stdlib `log/slog`, text or JSON, level from `OTEL_EBPF_LOG_LEVEL` | `cmd/obi/main.go` |
| Preflight | OS support, config validation, and Linux capability checks — in that order, before any pipeline is built | `cmd/obi/main.go` → `pkg/obi/os.go` |

## Section 2 — Low-level details

### Entry points

| Binary | File | Wires up |
| --- | --- | --- |
| `obi` | `cmd/obi/main.go` | `configcmd.MaybeRun` subcommand dispatch → slog handler → `obi.LoadConfig` → `CheckOSSupport` → `Validate` → `CheckOSCapabilities` → optional pprof listener → `instrumenter.Run` |
| `k8s-cache` | `cmd/k8s-cache/main.go` | Shared Kubernetes informer service consumed by OBI agents |
| `obi-schema` | `cmd/obi-schema/main.go` | Emits the config JSON schema (see the schema gate below) |
| `config-docs` | `cmd/config-docs/main.go` | Renders the configuration reference doc from that schema |

`instrumenter.Run` (`pkg/instrumenter/instrumenter.go`) is the real top of the tree:
it builds a `global.ContextInfo`, then fans out to `setupAppO11y` / `setupNetO11y` /
`setupStatsO11y`.

### Layering & key types

| Layer | Type | File |
| --- | --- | --- |
| Config root | `obi.Config` | `pkg/obi/config.go` |
| Pipeline node factory | `swarm.InstanceFunc` → `swarm.RunFunc` | `pkg/pipe/swarm/instancer.go:16` |
| Pipeline transport | `msg.Queue[T]` | `pkg/pipe/msg/queue.go` |
| eBPF program bundle | `ebpf.Tracer` | `pkg/ebpf/tracer.go:75` |
| Pipeline currency | `request.Span` | `pkg/appolly/app/request/span.go` |
| Export gating | `export.Features` (bitmask) | `pkg/export/feature.go` |

`ebpf.Tracer` is the widest contract in the repo. A bundle declares its attachment
points — `GoProbes`, `UProbes`, `USDTProbes`, `KProbes`, `Tracepoints`, `SocketFilters`,
`SockMsgs`, `SockOps`, `Iters`, `Tracing` — and the loader resolves and attaches each.
It also owns shared-library dedup (`RecordInstrumentedLib` / `AlreadyInstrumentedLib` /
`UnlinkInstrumentedLib`) so two processes sharing a `.so` are instrumented once.
`UtilityTracer` is the narrower variant for programs not tied to a service.

### eBPF ↔ Go binding

Each `bpf/<name>/<name>.c` has a matching loader carrying a `go:generate` directive:

```go
//go:generate $BPF2GO -cc $BPF_CLANG -cflags $BPF_CFLAGS -target amd64,arm64 Bpf ../../../../bpf/generictracer/generictracer.c -- -I../../../../bpf
```

— `pkg/internal/ebpf/generictracer/generictracer.go:39`

**Generated artifacts are committed** (`*_bpfel.go`, `*_bpfeb.go`, `*.o`). Editing a
`.c` file without regenerating leaves the committed `.o` stale and CI fails on
`check-clean-work-tree`. Use `make docker-generate` — it runs the generator in the
pinned image so output is byte-identical to CI.

### Configuration touch points

Config is one struct with four derived surfaces. A new key is a multi-place change:

| Surface | Source |
| --- | --- |
| YAML | `yaml:"…"` struct tag |
| Environment | `env:"OTEL_EBPF_…"` struct tag (`caarlos0/env/v11`) |
| JSON schema | generated by `cmd/obi-schema`, committed |
| Reference docs | generated by `cmd/config-docs` from that schema, committed |

`obi.LoadConfig` (`pkg/obi/config.go:904`) deep-copies pointer fields out of
`DefaultConfig` before unmarshalling — a shallow copy would let YAML mutate the package-level
default and leak across calls. Precedence is: defaults → YAML (with `${ENV}` expansion) →
environment → `normalize()` → `Validate()`.

Load-bearing keys:

| Key / env | Drives |
| --- | --- |
| `OTEL_EBPF_CONFIG_PATH` | Config file location (overrides `-config`) |
| `otel_metrics_export.features` | Which exporters instantiate at all (`export.Features` bitmask) |
| `OTEL_EBPF_CHANNEL_BUFFER_LEN` / `_SEND_TIMEOUT` / `_SEND_TIMEOUT_PANIC` | `msg.Queue` sizing and the deadlock panic |
| `OTEL_EBPF_ENFORCE_SYS_CAPS` | Whether missing Linux capabilities are fatal or a warning |
| `OTEL_EBPF_TRACE_PRINTER` | Debug span printer instead of (or alongside) real export |

### Metric feature gating

Adding a metric means adding a bit to `export.Features`, a name to `FeatureMapper`, and an
accessor — all in `pkg/export/feature.go`. `FeatureMapper` is deliberately a public `var`
so extension packages can add or remove entries before load. Two constraints encoded there:

- `application_jvm` is a deprecated alias for `application_runtime`, accepted at parse time
  but excluded from generated docs (`Features.JSONSchema`).
- Legacy and OTel span metrics are mutually exclusive when named explicitly, but
  auto-resolved in favour of OTel when selected via `all` / `*`
  (`InvalidSpanMetricsConfig` / `ResolveSpanMetricsConflict`).

### Critical invariants & gotchas

| Invariant | Where | Why |
| --- | --- | --- |
| No pipeline node starts until **every** constructor in the swarm succeeds | `pkg/pipe/swarm/instancer.go:42-60` | A half-built pipeline would emit partial, silently-wrong telemetry. Constructors get a context that is cancelled on any sibling's failure. |
| `Queue.Bypass` / `Subscribe` are wire-up-time only, serialized by one global mutex | `pkg/pipe/msg/queue.go:29` | Keeps the `Send` hot path lock-free. Mutating routing after start is unsupported. |
| A `Send` blocked > 1 min panics | `pkg/pipe/msg/queue.go:20` | A stalled subscriber deadlocks the whole pipeline; failing loudly beats silently dropping telemetry. |
| `LoadConfig` deep-copies `DefaultConfig`'s pointer fields | `pkg/obi/config.go:906-913` | Otherwise YAML unmarshal mutates package-level defaults. |
| Preflight order: OS support → config validate → capabilities | `cmd/obi/main.go` | Capability checks are meaningless against an unvalidated config. |
| Shared libraries are instrumented once, refcounted | `ebpf.Tracer.RecordInstrumentedLib` etc., `pkg/ebpf/tracer.go` | Duplicate probes on one `.so` double-count spans. |
| Committed eBPF artifacts must match their `.c` source | CI `check-clean-work-tree` | See eBPF ↔ Go binding above. |
| Client spans complete **before** their entry span | `pkg/export/otel/api_dependency.go:9-13` | Any egress↔entry join must buffer egress and flush on entry arrival — never the reverse. |

### Fork context

`main` tracks upstream OTel. Meesho-local work lives on feature branches and is written to
stay rebaseable:

| Patch | Files | What |
| --- | --- | --- |
| Trace-graph / traceparent adoption | `pkg/internal/ebpf/generictracer/`, `bpf/generictracer/` | Adopt app-written traceparent on HTTP/2 client requests; re-parent sockmsg-adopted client traceparents |
| P5 — `api_dependency_total` | `pkg/export/otel/api_dependency.go`, `pkg/export/feature.go` | Per-request join of a service instance's egress CLIENT spans to their entry SERVER/CONSUMER span, aggregated at the edge into `api_dependency_total{entry_api,dst_api,server_address,parented}` |

The P5 tracker is bounded on three axes, and those bounds are the design, not tuning knobs
(`pkg/export/otel/api_dependency.go:41-44`): pending traces per instance (capped FIFO,
evict oldest), buffered egress calls per trace (capped, drop excess), and distinct
`(entry_api, dst_api)` pairs per instance (capped; excess increments
`api_dependency_overflow_total`). Traces whose entry span never arrives are **dropped, never
guessed** — the service-graph metric still covers them at service granularity.

Local patches carry a `Meesho fork patch` marker comment so a rebase conflict is
attributable. Keep that convention.

<!-- meesho-init: plugin-version=2.1.3 generated-at=2026-08-25T06:26:04Z base-sha=86c6d7d52dee763fd1e22399aa0d143599acdf59+dirty -->
