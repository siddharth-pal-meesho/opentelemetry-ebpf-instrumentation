<!-- m-wiki: type=concept slug=context-propagation topic=ebpf base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# Context propagation

Distributed tracing needs a trace ID to survive a network hop. With an SDK the application writes a `traceparent` header. OBI has no SDK, so it must inject that context from the kernel, into traffic the application already sent.

The full design — execution order, mutual exclusion, and the map schemas — is documented upstream in [`devdocs/context-propagation.md`](../../../../devdocs/context-propagation.md) and, for HTTP/2, [`devdocs/grpc-context-propagation.md`](../../../../devdocs/grpc-context-propagation.md). This page is the code-anchored orientation.

## Two injection layers

| Layer | Mechanism | Applies to |
|---|---|---|
| L7 — HTTP headers | `sk_msg` program rewrites the payload to add `Traceparent:` | Plaintext HTTP |
| L4 — TCP options | Custom TCP option, kind 25 | Any TCP traffic, including encrypted |

Both are configured through one comma-separated setting, `OTEL_EBPF_BPF_CONTEXT_PROPAGATION`, accepting `headers`, `tcp`, `all`, or `disabled` (the default).

## Where it applies in this repo

The injector is a tracer like any other, enabled in the common tracer group:

- Enablement decision — `pkg/appolly/discover/finder.go:newCommonTracersGroup`, gated on `cfg.EBPF.ContextPropagation.HasHeaders()` or `HasTCP()`
- Constructor — `pkg/internal/ebpf/tpinjector/tpinjector.go:New`
- eBPF C sources — `bpf/tpinjector/tpinjector.c`, `bpf/tpinjector/h2_parse.h`, `bpf/tpinjector/inject_policy.h`, `bpf/tpinjector/sock_iter.c`
- Shared trace-context headers — `bpf/common/tp_info.h`, `bpf/common/trace_parent.h`

Because `tpinjector` sits in the *common* group rather than the Go or generic group, it loads once and serves every instrumented process, regardless of language.

## Why a mutual-exclusion mechanism exists

Several BPF programs can observe the same outgoing request: a Go uprobe on the HTTP client, the `sk_msg` injector, and a kprobe on `tcp_sendmsg`. Each is capable of writing trace context. Without coordination, the same request gets context injected more than once, or two layers disagree about which trace ID won.

The `outgoing_trace_map` carries two flags that arbitrate this, both described in the upstream document:

- `valid` — whether the recorded context should be used (set to 0 for SSL, where the uprobe sees plaintext but the socket sees ciphertext)
- `written` — whether injection already succeeded, so later layers stand down

On the ingress side the strategy is inverted: **last one wins**. A request may be observed at several layers on the way in, and the most specific observation — the one closest to the application — is the one that should define the parent span.

## Sharp edges

- **Disabled by default.** Context propagation is opt-in. A deployment reporting broken trace continuity between services very often simply has not set it.
- **TCP option kind 25 is not universally survivable.** Middleboxes, load balancers and some NICs strip unknown TCP options. Header injection survives L7 proxies; TCP options survive encryption. Neither survives everything, which is why `headers,tcp` is the common configuration.
- **Header injection changes packet length.** Adding `Traceparent:` to a payload in flight is the reason this path needs `sk_msg` rather than a passive observer, and why it only applies to plaintext.
- **Enabling it loads extra programs for every process.** The common group is not scoped to matched services, so the cost is per host, not per selected service.
- **These C anchors are line-cited only.** `.c`/`.h` files have no symbol pattern in the citation checker, so references here are file-level and will not be auto-corrected if code moves.

## Related

- [The Tracer interface](tracer-interface.md) — the contract `tpinjector` implements
- [Probe attachment](probe-attachment.md) — how `sk_msg` and `sock_ops` programs get attached
- [Discovery](../03-DISCOVERY.md) — where the common tracer group is assembled
- [eBPF layer](../04-EBPF-LAYER.md)

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD plus the repository's own `devdocs/`.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
