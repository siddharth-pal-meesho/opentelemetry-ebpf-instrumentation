# Acronyms

> Domain-specific acronyms used in this repo's code and docs. Generic programming
> acronyms (HTTP, JSON, gRPC, URL, API, TLS, ELF-adjacent basics) are only listed
> when this repo uses them in a narrower sense than a reader would assume.
>
> Every definition below is grounded in a cited file. If a definition and the code
> disagree, the code wins — fix this table.

| Acronym | Definition | Used in |
|---------|------------|---------|
| AMQP | Advanced Message Queuing Protocol — broker protocol parsed for messaging spans | [`pkg/internal/ebpf/amqpparser/`](../pkg/internal/ebpf/amqpparser/), [`devdocs/protocols/tcp/amqp.md`](protocols/tcp/amqp.md) |
| ASN | Autonomous System Number — MaxMind GeoIP lookup that decorates network flows | [`internal/config/schema/network.go`](../internal/config/schema/network.go), [`pkg/internal/pipe/geoip/`](../pkg/internal/pipe/geoip/) |
| BPF / eBPF | (extended) Berkeley Packet Filter — the in-kernel VM every probe in `bpf/` runs on | [`bpf/`](../bpf/), [`pkg/internal/ebpf/`](../pkg/internal/ebpf/) |
| BTF | BPF Type Format — kernel type metadata OBI requires (`/sys/kernel/btf/vmlinux`) | [`CONTRIBUTING.md`](../CONTRIBUTING.md), [`SUPPORT_MATRIX.md`](../SUPPORT_MATRIX.md) |
| CGI | Common Gateway Interface — FastCGI framing detected to attribute PHP-FPM spans | [`pkg/ebpf/common/fast_cgi_detect_transform.go`](../pkg/ebpf/common/fast_cgi_detect_transform.go) |
| CIDR | Classless Inter-Domain Routing — prefix notation for network-flow grouping rules | [`pkg/internal/pipe/cidr/`](../pkg/internal/pipe/cidr/) |
| CP | Context Propagation — injecting/extracting `traceparent` across a process boundary | [`devdocs/context-propagation.md`](context-propagation.md), [`devdocs/grpc-context-propagation.md`](grpc-context-propagation.md) |
| CUDA | NVIDIA's GPU compute API — kernel launches and allocations traced as GPU events | [`pkg/internal/ebpf/gpuevent/`](../pkg/internal/ebpf/gpuevent/), [`bpf/gpuevent/`](../bpf/gpuevent/) |
| ELF | Executable and Linkable Format — binary layout parsed to locate uprobe targets | [`pkg/internal/fastelf/`](../pkg/internal/fastelf/), [`pkg/internal/goexec/`](../pkg/internal/goexec/) |
| GenAI | Generative AI — LLM/vector-store protocol instrumentation and token-usage metrics | [`devdocs/features.md`](features.md), [`SUPPORT_MATRIX.md`](../SUPPORT_MATRIX.md) |
| H2 | HTTP/2 — shorthand throughout the H2 sniffing and HPACK injection paths | [`bpf/common/h2_defs.h`](../bpf/common/h2_defs.h), [`devdocs/grpc-context-propagation.md`](grpc-context-propagation.md) |
| HPACK | HTTP/2 header compression (RFC 7541) — the only network mechanism for gRPC/H2 CP | [`devdocs/grpc-context-propagation.md`](grpc-context-propagation.md), [`bpf/tpinjector/`](../bpf/tpinjector/) |
| JVM | Java Virtual Machine — runtime detected for TLS uprobes and HotSpot USDT metrics | [`pkg/appolly/app/runtime/jvm.go`](../pkg/appolly/app/runtime/jvm.go), [`devdocs/runtimes/jvm.md`](runtimes/jvm.md) |
| LRU | Least Recently Used — eviction policy for both BPF and userspace correlation caches | [`pkg/internal/helpers/cache/expirable_lru.go`](../pkg/internal/helpers/cache/expirable_lru.go), [`bpf/gotracer/maps/`](../bpf/gotracer/maps/) |
| MCP | Model Context Protocol — an instrumented HTTP payload protocol (not Claude's MCP wiring) | [`pkg/ebpf/common/http/mcp.go`](../pkg/ebpf/common/http/mcp.go) |
| MQTT | Message Queuing Telemetry Transport — pub/sub protocol parsed for messaging spans | [`pkg/internal/ebpf/mqttparser/`](../pkg/internal/ebpf/mqttparser/) |
| MSSQL | Microsoft SQL Server — TDS wire protocol detected for database spans | [`pkg/ebpf/common/sql_detect_mssql.go`](../pkg/ebpf/common/sql_detect_mssql.go) |
| O11y | Observability (numeronym) — names the three feature families: `AppO11y`, `NetO11y`, `StatsO11y` | [`cmd/obi/internal/configcmd/configcmd.go`](../cmd/obi/internal/configcmd/configcmd.go), [`devdocs/metrics.md`](metrics.md) |
| OBI | OpenTelemetry eBPF Instrumentation — this project | [`README.md`](../README.md) |
| OTLP | OpenTelemetry Protocol — the wire format for exported traces and metrics | [`pkg/export/otel/otelcfg/`](../pkg/export/otel/otelcfg/) |
| `PT_*` | ELF Program header Type (e.g. `PT_LOAD`) — segment classification during ELF parsing | [`pkg/internal/fastelf/fastelf.go`](../pkg/internal/fastelf/fastelf.go) |
| RDNS | Reverse DNS — optional pipeline stage resolving IPs, fed by XDP DNS-response inspection | [`bpf/rdns/rdns_xdp.c`](../bpf/rdns/rdns_xdp.c), [`devdocs/pipeline-map.md`](pipeline-map.md) |
| RED | Rate, Errors, Duration — the standard service-level metric triad OBI exports | [`devdocs/exclude-otel-instrumented-services.md`](exclude-otel-instrumented-services.md), [`pkg/export/otel/metrics.go`](../pkg/export/otel/metrics.go) |
| SASL | Simple Authentication and Security Layer — auth frames skipped when parsing AMQP/Kafka | [`pkg/internal/ebpf/amqpparser/frame.go`](../pkg/internal/ebpf/amqpparser/frame.go) |
| `SHT_*` | ELF Section Header Type (e.g. `SHT_SYMTAB`, `SHT_RELA`) — section classification | [`pkg/internal/fastelf/fastelf.go`](../pkg/internal/fastelf/fastelf.go), [`pkg/internal/goexec/instructions.go`](../pkg/internal/goexec/instructions.go) |
| SQL++ / SQLPP | Couchbase's SQL-for-JSON query language — optional HTTP payload extraction | [`pkg/ebpf/common/http/sqlpp.go`](../pkg/ebpf/common/http/sqlpp.go), [`devdocs/protocols/tcp/couchbase.md`](protocols/tcp/couchbase.md) |
| TC | Traffic Control — kernel ingress/egress hook used for network-flow probes | [`bpf/netolly/flows.c`](../bpf/netolly/flows.c), [`devdocs/metrics.md`](metrics.md) |
| TCX | TC eXpress — the newer, link-based TC attachment backend (alternative to netlink `tc`) | [`pkg/config/tcbackend.go`](../pkg/config/tcbackend.go), [`pkg/ebpf/tcmanager/`](../pkg/ebpf/tcmanager/) |
| TP | `traceparent` — the W3C Trace Context header; the payload `tpinjector` writes | [`bpf/tpinjector/tpinjector.c`](../bpf/tpinjector/tpinjector.c), [`devdocs/context-propagation.md`](context-propagation.md) |
| USDT | Userland Statically Defined Tracing — probe markers (e.g. HotSpot `hotspot:mem__pool__*`) | [`pkg/ebpf/usdt.go`](../pkg/ebpf/usdt.go), [`devdocs/runtimes/jvm.md`](runtimes/jvm.md) |
| VMA | Virtual Memory Address — load-time address of an ELF section (`.text`, `.gopclntab`) | [`pkg/internal/goexec/moduledata_test.go`](../pkg/internal/goexec/moduledata_test.go) |
| XDP | eXpress Data Path — earliest ingress hook; inspects DNS responses for RDNS | [`bpf/rdns/rdns_xdp.c`](../bpf/rdns/rdns_xdp.c) |
