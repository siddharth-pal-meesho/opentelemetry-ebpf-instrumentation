<!-- m-wiki: type=top-level slug=06-configuration topic=null base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: top-level. 0 sources.

# Configuration

`pkg/obi/config.go:Config` is the single typed root for everything the agent can be told to do. Every subsystem — eBPF tracer options, discovery criteria, exporters, filters, Kubernetes metadata, health checks — hangs off it as a nested struct. There is no second configuration path and no global mutable settings object.

Two surfaces populate it: a YAML document and environment variables. Both are declared on the same struct fields, side by side.

## TL;DR

- One root struct, `pkg/obi/config.go:Config`, with `yaml:"…"` and `env:"…"` tags on the same fields.
- Defaults live in a package-level `DefaultConfig` value (`pkg/obi/config.go:118`), not scattered through constructors.
- `pkg/obi/config.go:LoadConfig` accepts a `nil` reader — configuration purely from the environment is a supported mode.
- Validation is layered: `Validate`, `ValidateStatic`, `ValidateForReceiver`, and `ValidateStaticForReceiver` share one `validate` implementation parameterized by a `validationContext`.
- Feature enablement is a bitmask queried through `pkg/obi/config.go:Config.Enabled`, not a set of booleans.
- The user-facing reference is [`devdocs/config/CONFIG.md`](../../../devdocs/config/CONFIG.md) with a generated JSON schema alongside it; this page covers the mechanism, not the option list.

## Mental model

**A struct with two front doors.** Each field can be reached by a YAML key or an environment variable, and the tags on the field are the only place that mapping is declared. Adding an option means editing one struct field; the YAML surface, the env surface, the JSON schema, and the documentation all derive from it.

That derivation is enforced rather than trusted: `cmd/obi-schema`, `cmd/config-docs`, `cmd/check-config-v2-parity` and `cmd/check-config-v2-artifacts` exist to regenerate and verify those downstream artifacts. A field added without regenerating them fails CI rather than silently drifting.

## Structure / data flow

```
   -config <path>  or  OTEL_EBPF_CONFIG_PATH        (env wins)
            │
            ▼
   obi.LoadConfig(io.Reader | nil)
            │   start from DefaultConfig
            │   overlay YAML document
            │   overlay environment variables
            │   normalize()
            ▼
   Config  ──► Validate()                        ← fatal in main()
            └► Enabled(FeatureAppO11y | …)       ← consulted by RunWithContextInfo
```

Decoding uses `confmap` with custom hooks. Two are worth knowing about because they change what YAML shapes are legal: `stringSliceToTextUnmarshalerHookFunc` lets a list of strings populate a field whose type implements `TextUnmarshaler`, and `inlineMetadataHookFunc` handles inline metadata blocks.

## Key code locations

| What | Where |
|---|---|
| Root configuration struct | `pkg/obi/config.go:Config` |
| Defaults | `pkg/obi/config.go:118` |
| Load and merge | `pkg/obi/config.go:LoadConfig` |
| Confmap unmarshal entry | `pkg/obi/config.go:Config.Unmarshal` |
| Validation (all four variants) | `pkg/obi/config.go:Config.Validate` |
| Feature bitmask type | `pkg/obi/config.go:Feature` |
| Feature query | `pkg/obi/config.go:Config.Enabled` |
| Merged base + per-app metrics config | `pkg/obi/config.go:Config.JoinMetricsConfig` |
| Startup config echo | `pkg/obi/config.go:Config.Log` |
| Health check block | `pkg/obi/config.go:HealthCheckConfig` |
| eBPF tracer options | `pkg/config/ebpf_tracer.go` |
| Discovery selection block | `pkg/obi/config.go:Config` → `Discovery services.DiscoveryConfig` |
| Generated JSON schema | `devdocs/config/config-schema.json` |

## Deprecated surfaces still in the struct

Several fields are retained purely for backward compatibility and are marked `Deprecated:` in place:

| Field | Superseded by |
|---|---|
| `Exec` (`executable_path`) | `AutoTargetExe` / `discovery > instrument` |
| `ServiceName`, `ServiceNamespace` | metadata on the instrumentation target itself |
| `Attributes.Kubernetes.MetaSourceLabels` | `resource_labels` |
| `discovery > services` | `discovery > instrument` |

The deprecated Kubernetes label mapping is merged forward at runtime in `pkg/instrumenter/instrumenter.go:BuildCommonContextInfo`, which prepends the legacy values ahead of `resource_labels` and emits a one-shot deprecation warning.

## Sharp edges

- **`OTEL_SERVICE_NAME` and `OTEL_EBPF_SERVICE_NAME` are chained through `envDefault`.** The struct tag uses one as the default expansion of the other; reading either variable in isolation gives the wrong answer about precedence.
- **Some fields are YAML-only, some env-only.** `AutoTargetExe` and `AutoTargetLanguage` carry no `yaml:` tag — they are set through `discovery > instrument` in YAML, or directly by env var. The struct field name is not the YAML key.
- **A block of fields is intentionally undocumented.** Everything after the "will remain undocumented" comment (channel buffer lengths, send timeouts, profile port) is developer/support tooling, not user API.
- **`Enabled` is not a plain flag read.** It consults the bitmask *and* related exporter configuration, so a feature can be enabled by the presence of an endpoint rather than by an explicit toggle.
- **Validation has four entry points.** They differ in whether the agent is standalone or embedded as a collector receiver; picking the wrong one when embedding produces validation errors for fields the host is responsible for.
- **Adding a config key is a multi-artifact change.** Struct field, generated schema, generated docs, and the parity checks under `cmd/` all have to agree.

## Related concepts

- [Entry point and boot sequence](02-ENTRYPOINT.md) — where the config is loaded and validated
- [Architecture](01-ARCHITECTURE.md) — what the feature bitmask actually gates
- [Internal metrics](observability/internal-metrics.md) — a subsystem selected entirely by config shape
- [Build and validation](09-BUILD-AND-VALIDATION.md) — regenerating schema and docs

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, incident references, dates, decisions that synthesis missed. -->

---

[← Previous](05-PIPELINE-AND-EXPORT.md) · [Index](../index.md) · [Next →](07-KUBERNETES-METADATA.md)
