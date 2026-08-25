# schemas/

Machine-readable schema definitions for OBI. Everything under this directory is
a contract consumed by tooling, not prose — each subdirectory carries its own
README explaining how to change it safely.

## Contents

| Path | What it is | Consumed by |
|------|------------|-------------|
| [`obi/`](obi/README.md) | OpenTelemetry semantic-convention registry: the signals and attributes OBI emits on top of (or as overrides of) upstream semconv. `manifest.yaml` pins the upstream semconv dependency; `groups/` holds one YAML per attribute namespace. | [weaver](https://github.com/open-telemetry/weaver) — `make lint-schema` (`registry check`) and the weaver-validated integration suites (`live-check`) |

## Working on the registry

Read [`obi/README.md`](obi/README.md) first. Two rules bite hardest:

- Overrides of upstream definitions must use the `x.obi.<namespace>` group-id
  prefix — weaver resolves duplicate attribute ids by lexicographic group order,
  so a group that sorts before the upstream one silently loses.
- An override **replaces** the upstream attribute wholesale. Extending an enum
  means restating the full upstream member list alongside OBI's additions, and
  re-syncing it whenever the semconv dependency is bumped.

The `manifest.yaml` dependency version and the upstream semconv version OBI is
built against must move together; a CI guard asserts they match.

## No database schemas here

OBI is an eBPF instrumentation agent and owns no datastore, so this tree holds
no MySQL / Mongo / Scylla / Elasticsearch / Bigtable schema documentation. The
database drivers in `go.mod` (for example `go.mongodb.org/mongo-driver`) are
instrumentation *targets* — used by the integration-test workloads under
`internal/test/integration/components/`, by the struct-offset inspectors in
`configs/offsets/`, and by the wire-protocol decoders in `pkg/ebpf/common/` —
never as storage for OBI itself.

*Last audited by `/m-skills:db-schema-docs` on 2026-08-25.*
