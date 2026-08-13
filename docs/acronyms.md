# Acronyms

> Domain-specific acronyms used in this repo's code and docs. Generic programming
> acronyms (HTTP, JSON, gRPC, URL, API, JWT) are NOT listed here — only domain terms
> that a new engineer wouldn't infer from context.

| Acronym | Definition | Used in |
|---------|------------|---------|
| AMQP | Advanced Message Queuing Protocol — broker wire protocol parsed into messaging spans | [`devdocs/protocols/tcp/amqp.md`](../devdocs/protocols/tcp/amqp.md), `bpf/generictracer/` |
| BPFFS | BPF filesystem — kernel pseudo-filesystem (default `/sys/fs/bpf`) where OBI pins maps for sharing across processes | [`devdocs/trace-profile-correlation.md`](../devdocs/trace-profile-correlation.md), config `ebpf.bpf_fs_path` |
| BTF | BPF Type Format — kernel type metadata OBI requires; checked via `/sys/kernel/btf/vmlinux` | [`CONTRIBUTING.md`](../CONTRIBUTING.md), [`SUPPORT_MATRIX.md`](../SUPPORT_MATRIX.md) |
| DCP | Database Change Protocol — Couchbase streaming/replication opcodes decoded by the KV parser | `pkg/internal/ebpf/couchbasekv/` |
| ELF | Executable and Linkable Format — binary format parsed to locate symbols and offsets for uprobe attachment | `pkg/internal/goexec/` |
| HPACK | HTTP/2 header compression — the only network mechanism used to inject `traceparent` into gRPC/H2 calls | [`devdocs/grpc-context-propagation.md`](../devdocs/grpc-context-propagation.md) |
| LGTM | Loki, Grafana, Tempo, Mimir — Grafana's single-container OTLP backend used by the examples | [`examples/nginx/`](../examples/nginx/), [`examples/store-demo/`](../examples/store-demo/) |
| MQTT | Message Queuing Telemetry Transport — pub/sub protocol decoded by the generic tracer | `bpf/generictracer/`, `devdocs/protocols/` |
| OATS | OpenTelemetry Acceptance Test Suite — Grafana's YAML-driven end-to-end test harness | [`internal/test/oats/`](../internal/test/oats/), [`RELEASING.md`](../RELEASING.md) |
| OBI | OpenTelemetry eBPF Instrumentation — this project | [`README.md`](../README.md) |
| OTLP | OpenTelemetry Protocol — wire protocol OBI exports traces and metrics over | `pkg/export/otel/` |
| RED | Rate, Errors, Duration — the per-service request metric set OBI emits, distinct from span metrics | [`devdocs/exclude-otel-instrumented-services.md`](../devdocs/exclude-otel-instrumented-services.md), [`devdocs/pipeline-map.md`](../devdocs/pipeline-map.md) |
| SPID | Server Process ID — MSSQL/TDS session identifier carried in the packet header | `bpf/generictracer/protocol_mssql.h`, `pkg/internal/sqlprune/` |
| SQL++ | Couchbase's SQL-for-JSON query language; extracted from HTTP payloads when payload extraction is enabled | [`devdocs/config/CONFIG.md`](../devdocs/config/CONFIG.md) (`ebpf.payload_extraction.http.sqlpp`) |
| TC | Traffic Control — Linux kernel subsystem OBI attaches network programs to | `internal/config/` (`TCBackend`) |
| TCX | TC eXpress — the newer TC BPF attach API; selectable as an alternative TC backend | `internal/config/` (`TCBackendTCX`) |
| TDS | Tabular Data Stream — Microsoft SQL Server wire protocol | `bpf/generictracer/protocol_mssql.h` |
| TP | traceparent — the W3C Trace Context header OBI reads and injects (`tp_info_t`, `tpinjector`) | `bpf/tpinjector/`, `bpf/common/tp_info.h`, [`devdocs/context-propagation.md`](../devdocs/context-propagation.md) |
| USDT | Userland Statically Defined Tracing — user-space probes, e.g. HotSpot's `hotspot:mem__pool__gc__*` | [`devdocs/runtimes/jvm.md`](../devdocs/runtimes/jvm.md) |
| VMA | Virtual Memory Address — segment addresses used when resolving Go module data in a mapped binary | `pkg/internal/goexec/` |
| XDP | eXpress Data Path — earliest kernel packet-processing hook; used for reverse-DNS packet inspection | `bpf/rdns/rdns_xdp.c` |
