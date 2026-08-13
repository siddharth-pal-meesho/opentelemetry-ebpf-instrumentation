<!-- m-wiki: type=top-level slug=02-entrypoint topic=null base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: top-level. 0 sources.

# Entry point and boot sequence

There are two shipped binaries in this repository. `cmd/obi` is the agent itself — the thing that loads eBPF programs and exports telemetry. `cmd/k8s-cache` is an optional sidecar service that centralizes Kubernetes metadata for many agents. Everything else under `cmd/` is build-time tooling (config schema generation, parity checks, port-lookup table generation) rather than a deployable.

This page traces what happens between process start and the first span, and where startup can refuse to continue.

## TL;DR

- `cmd/obi/main.go:main` intercepts config subcommands first, then loads configuration, then hands off to `pkg/instrumenter/instrumenter.go:Run`.
- Configuration comes from a YAML file (`-config` flag or `OTEL_EBPF_CONFIG_PATH`) merged with environment variables; the file is optional and a nil reader is a valid input to `pkg/obi/config.go:LoadConfig`.
- Startup has four ordered gates: OS support, config validation, system capabilities, and per-mode construction. Only the capability gate is soft, and only when `enforce_sys_caps` is false.
- Shutdown is signal-driven: `SIGINT`/`SIGTERM` cancel a context that unwinds every pipeline, bounded by `shutdown_timeout`.

## Mental model

Boot is a **sequence of refusals**. Each stage tries to prove the agent cannot possibly work and exits early if it succeeds. Only after all of them decline does the agent build any pipelines. This matters operationally: almost every "OBI won't start" report resolves to one of four log lines emitted in `main`, in a fixed order, and knowing the order tells you which check to look at.

## Structure / data flow

```
main()
  │
  ├─ configcmd.MaybeRun(os.Args[1:])        ← subcommands (config docs/validate) exit here
  │
  ├─ flag: -config <path>   ⊕   env: OTEL_EBPF_CONFIG_PATH   (env wins)
  │     └─ loadConfig → obi.LoadConfig(reader)      ← YAML + env merge; nil reader = env only
  │
  ├─ log level + format applied  (text | json)      ← invalid level is fatal, invalid format warns
  │
  ├─ obi.CheckOSSupport()                            ← FATAL if unsupported
  ├─ config.Validate()                               ← FATAL on bad config
  ├─ obi.CheckOSCapabilities(config)                 ← FATAL only if enforce_sys_caps, else WARN
  │
  ├─ optional pprof listener on profile_port
  ├─ config.Log()                                    ← controlled by log_config
  │
  ├─ signal.NotifyContext(SIGINT, SIGTERM)
  └─ instrumenter.Run(ctx, config)                   ← blocks until every mode finishes
```

Inside `Run`, `pkg/instrumenter/instrumenter.go:BuildCommonContextInfo` assembles the shared state that all three modes need — the Kubernetes metadata provider, node metadata, the internal-metrics reporter, the Docker container store, and the default attribute groups — before `RunWithContextInfo` starts anything.

## Key code locations

| What | Where |
|---|---|
| `main` and the ordered startup gates | `cmd/obi/main.go:main` |
| Config file opening and load | `cmd/obi/main.go:loadConfig` |
| YAML + environment merge | `pkg/obi/config.go:LoadConfig` |
| Config validation | `pkg/obi/config.go:Config.Validate` |
| Startup banner and config echo | `pkg/obi/config.go:Config.Log` |
| Blocking run entry | `pkg/instrumenter/instrumenter.go:Run` |
| Shared context assembly | `pkg/instrumenter/instrumenter.go:BuildCommonContextInfo` |
| Mode gating and errgroup | `pkg/instrumenter/instrumenter.go:RunWithContextInfo` |
| Health check listener startup | `pkg/instrumenter/instrumenter.go:startHealthCheck` |
| Health endpoints (UDS / TCP) | `pkg/health/health.go:ListenAndServeUDS`, `pkg/health/health.go:ListenAndServeTCP` |
| AppO11y startup sequence | `pkg/instrumenter/instrumenter.go:setupAppO11y` |
| k8s-cache service entry point | `cmd/k8s-cache/main.go` |

## The health check is not an API

`startHealthCheck` binds either a Unix domain socket or a TCP port, chosen by `pkg/obi/config.go:HealthCheckConfig`. It exists for container liveness/readiness probes. It is not an application API surface and exposes no telemetry — telemetry leaves through OTLP exporters or the Prometheus scrape endpoint, never here.

This distinction matters when automated tooling scans the repository for HTTP routes: the route literals it finds under `configs/offsets/`, `internal/test/`, and `examples/` belong to DWARF-offset probe fixtures, integration-test servers, and vendored demo applications — none of them are served by OBI.

## Sharp edges

- **`OTEL_EBPF_CONFIG_PATH` silently overrides `-config`.** The env var is applied after flag parsing and wins unconditionally.
- **An unreadable config file is fatal, a missing one is not.** No `-config` and no env var means "configure entirely from the environment", which is a supported deployment mode and not an error.
- **The capability check runs after validation.** A config that is invalid will mask the capability warning you actually needed to see.
- **`instrumenter.Run` blocks for the process lifetime.** Anything that needs to run alongside it — the pprof listener, health check — is launched as a goroutine before the call.
- **Graceful shutdown is bounded.** `shutdown_timeout` caps how long the agent waits for eBPF probes to unload; exceeding it surfaces as `graceful shutdown has timed out while waiting for eBPF tracers to finish` from `pkg/internal/appolly/appolly.go:Instrumenter.WaitUntilFinished`.

## Related concepts

- [Configuration](06-CONFIGURATION.md) — the shape of `obi.Config` and how it is populated
- [Architecture](01-ARCHITECTURE.md) — what `Run` actually starts
- [Internal metrics](observability/internal-metrics.md) — the reporter chosen during `BuildCommonContextInfo`
- [Attribute groups](observability/attribute-groups.md) — defaults decided at boot
- [Kubernetes metadata](07-KUBERNETES-METADATA.md) — the informer built before any pipeline

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, incident references, dates, decisions that synthesis missed. -->

---

[← Previous](01-ARCHITECTURE.md) · [Index](../index.md) · [Next →](03-DISCOVERY.md)
