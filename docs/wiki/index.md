# obi — Wiki

> Synthesized view of this codebase. Code is the prior; this wiki is downstream.
> Owned by `m-wiki`. Edit via skills, not directly.

This repository is [OpenTelemetry eBPF Instrumentation](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation) (OBI) — a Linux agent that instruments running applications from the outside using eBPF, with no SDK, no recompilation and no restart.

The project's own developer documentation lives in [`devdocs/`](../../devdocs/README.md) and remains authoritative for protocol details, configuration reference and design write-ups. This wiki complements it with symbol-anchored navigation: start from a question, land on the function.

## Top-level pages

| #   | Page | What you'll learn |
|-----|------|-------------------|
| 01 | [Architecture](pages/01-ARCHITECTURE.md) | The three observability modes, and why AppO11y is split into a discovery pipeline and an export pipeline |
| 02 | [Entry point and boot sequence](pages/02-ENTRYPOINT.md) | What happens between process start and the first span, and the four gates that can refuse to continue |
| 03 | [Discovery](pages/03-DISCOVERY.md) | How a running process becomes an instrumented target, stage by stage |
| 04 | [eBPF layer](pages/04-EBPF-LAYER.md) | The tracer contract, the load/attach/run/unlink lifecycle, and what is generated versus written |
| 05 | [Pipeline and export](pages/05-PIPELINE-AND-EXPORT.md) | The decoration chain, the metrics fork, and every exporter |
| 06 | [Configuration](pages/06-CONFIGURATION.md) | The typed config root, its YAML and environment surfaces, and how validation is layered |
| 07 | [Kubernetes metadata](pages/07-KUBERNETES-METADATA.md) | Local informers versus the shared `k8s-cache` service, and what happens when either fails |
| 08 | [Network and stat pipelines](pages/08-NETWORK-AND-STATS.md) | The two modes that bypass discovery entirely |
| 09 | [Build and validation](pages/09-BUILD-AND-VALIDATION.md) | Which `make` target to run, and why generated code is committed |

## Topics

### discovery (2 pages)

- [Process environment capture](pages/discovery/process-env-capture.md) — why only an allowlist of environment variables is retained per process, and the memory bound behind it
- [Selection and typing](pages/discovery/selection-and-typing.md) — how selection criteria are resolved, and how a binary's classification picks its tracer group

### ebpf (4 pages)

- [Context propagation](pages/ebpf/context-propagation.md) — injecting trace context from the kernel, via HTTP headers or TCP options
- [Pin-internal maps and scratch memory](pages/ebpf/pin-internal-and-scratch-mem.md) — the two C conventions every eBPF program here follows, and the kernel limits behind them
- [Probe attachment](pages/ebpf/probe-attachment.md) — how each declared attachment point is resolved and attached, and how each one fails
- [The Tracer interface](pages/ebpf/tracer-interface.md) — the contract every eBPF program bundle implements

### observability (2 pages)

- [Attribute groups](pages/observability/attribute-groups.md) — how metric attributes are enabled in groups, driven by what metadata is actually available
- [Internal metrics](pages/observability/internal-metrics.md) — how the agent instruments itself, and the boot ordering that makes it correct

### pipeline (2 pages)

- [The swarm instancer](pages/pipeline/swarm-instancer.md) — the two-phase construct-then-run pattern every pipeline is built from
- [Typed message queues](pages/pipeline/msg-queue.md) — the multi-subscriber transport that connects pipeline nodes

## How to read this wiki

**If you're brand new:**

1. [Architecture](pages/01-ARCHITECTURE.md) — get the three-mode picture and the discovery/export split.
2. [The swarm instancer](pages/pipeline/swarm-instancer.md) and [typed message queues](pages/pipeline/msg-queue.md) — every pipeline in the repo is these two ideas repeated.
3. [Discovery](pages/03-DISCOVERY.md), then [pipeline and export](pages/05-PIPELINE-AND-EXPORT.md) — follow one span end to end.
4. [`devdocs/pipeline-map.md`](../../devdocs/pipeline-map.md) — the upstream flowcharts, which will now read as familiar.

**If you're debugging:**

- *No spans at all for a service* → [selection and typing](pages/discovery/selection-and-typing.md), then [discovery](pages/03-DISCOVERY.md).
- *Spans exist but are shallow* → typing fell back to the generic tracer; see [selection and typing](pages/discovery/selection-and-typing.md).
- *Traces break between services* → [context propagation](pages/ebpf/context-propagation.md); it is off by default.
- *Missing pod or container attributes* → [Kubernetes metadata](pages/07-KUBERNETES-METADATA.md) and [attribute groups](pages/observability/attribute-groups.md).
- *Agent won't start* → the four ordered gates in [entry point](pages/02-ENTRYPOINT.md).
- *Metrics and traces disagree about span names* → the span-name limiter in [pipeline and export](pages/05-PIPELINE-AND-EXPORT.md).
- *Agent memory grows without bound* → [process environment capture](pages/discovery/process-env-capture.md) documents one such class of bug.

**If you're adding a feature:**

- Adding a probe or protocol → [the Tracer interface](pages/ebpf/tracer-interface.md), [probe attachment](pages/ebpf/probe-attachment.md), [pin-internal and scratch memory](pages/ebpf/pin-internal-and-scratch-mem.md), then regenerate per [build and validation](pages/09-BUILD-AND-VALIDATION.md).
- Adding a pipeline stage → [the swarm instancer](pages/pipeline/swarm-instancer.md), [typed message queues](pages/pipeline/msg-queue.md), [pipeline and export](pages/05-PIPELINE-AND-EXPORT.md).
- Adding a config option → [configuration](pages/06-CONFIGURATION.md); remember the struct field, generated schema, generated docs and parity checks must all agree.
- Adding a metric or attribute → [attribute groups](pages/observability/attribute-groups.md) and [`devdocs/metrics.md`](../../devdocs/metrics.md).

Before opening a PR, read [`AGENTS.md`](../../AGENTS.md) and [`AI-POLICY.md`](../../AI-POLICY.md). This is a Linux Foundation project: changes must be minimal and scoped, and generative-AI assistance above trivial autocomplete must be disclosed.

## Unexplored topics

Candidates the bootstrap identified but did not generate pages for. Good ingest targets:

- **Protocol parsing** — HTTP/1, HTTP/2 and gRPC, SQL, Redis, Kafka, AMQP, MQTT, Couchbase and Aerospike detection all live under `pkg/ebpf/common/` and `bpf/`; [`devdocs/protocols/`](../../devdocs/protocols/README.md) is the starting point.
- **Language runtimes** — Go, JVM, Node.js and Python each have runtime-specific handling (`pkg/internal/{java,jvmtools,nodejs}`, `devdocs/runtimes/`); the Python asyncio/uvloop propagation design is a substantial topic on its own.
- **TLS and encrypted traffic** — SSL detection and the Java TLS `ioctl` hardening described in [`devdocs/java-tls-ioctl-security.md`](../../devdocs/java-tls-ioctl-security.md).
- **Trace-log and trace-profile correlation** — the log enricher and the `traces_ctx_v1` map.
- **Runtime metrics** — the `application_runtime` feature and per-runtime coverage.
- **Testing infrastructure** — the integration suite and the OATS harness under `internal/test/`.
- **Release engineering** — `RELEASING.md`, `VERSIONING.md`, `SUPPORT_MATRIX.md` and the offsets-tracker workflow.

## Operating this wiki

| Action | How |
|---|---|
| Re-sync after code changes | `/m-wiki:wiki-init` (enters update mode) |
| Add a source | drop a file in `raw/<topic>/`, then re-run `/m-wiki:wiki-init` |
| Ask a question | `qmd query "<question>" --collection obi-wiki` |
| Health-check | `/m-wiki:wiki-verify` |

> **This wiki is not yet queryable.** The qmd collection could not be registered on this run — the installed `qmd` binary fails to load its native SQLite module (`better-sqlite3` built for a different Node.js ABI). Repair qmd via `/m-wiki:wiki-setup`, then re-run `/m-wiki:wiki-init`. See [`log.md`](log.md) for the recorded error.

---

<!-- m-wiki: index-version=1 generated-at=2026-08-13T14:38:34Z base-sha=724e96d5baf0 -->
