<!-- m-wiki: type=top-level slug=07-kubernetes-metadata topic=null base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: top-level. 0 sources.

# Kubernetes metadata

eBPF sees PIDs, inodes and sockets. Dashboards want pod names, namespaces and workload owners. Bridging that gap is what the Kubernetes metadata subsystem does, and it participates in every pipeline: discovery uses it to enrich process events before matching, and the export pipelines use it to decorate spans, flows and stats.

There are two ways to get that metadata, and choosing between them is a deployment decision with real cost implications.

## TL;DR

- `pkg/kube/informer_provider.go:MetadataProvider` is the single entry point; everything else asks it for a `Store`.
- It runs in one of two modes: **local informers** watching the Kubernetes API directly, or a **remote client** subscribing to the standalone `k8s-cache` service over gRPC.
- The mode is chosen by whether `meta_cache_address` is configured — see `initLocalInformers` versus `initRemoteInformerCacheClient`.
- The provider is constructed early, in `pkg/instrumenter/instrumenter.go:BuildCommonContextInfo`, before any pipeline exists, because node metadata resolution depends on it.
- If it fails to start, `pkg/internal/appolly/appolly.go:setupKubernetes` force-disables Kubernetes and falls back to Docker container metadata.

## Mental model

**One watcher, many readers.** Kubernetes informers are expensive: each one holds an open watch against the API server and a full in-memory copy of the objects it tracks. A cluster running OBI as a DaemonSet has one agent per node, so naive local informers means N watch connections and N copies of the same pod list.

`k8s-cache` exists to make that 1 instead of N. It runs the informers once, and agents subscribe to a gRPC stream that first replays a snapshot, then pushes deltas. An explicit `SYNC_FINISHED` event tells each agent when the replay is complete and it is safe to start decorating — without it, early telemetry would be emitted with missing labels.

## Structure / data flow

```
   ┌── local mode ────────────────────────────────┐
   │  MetadataProvider.initLocalInformers          │
   │        └─ watches Pod / Node / Service        │
   │           against the Kubernetes API          │
   └───────────────────────────────────────────────┘
                    OR
   ┌── remote mode (meta_cache_address set) ───────┐
   │  MetadataProvider.initRemoteInformerCacheClient│
   │        └─ cacheSvcClient.connect               │
   │             gRPC informer.EventStreamService/  │
   │             Subscribe → snapshot, then deltas  │
   │             reconnects with backoff            │
   └───────────────────────────────────────────────┘
                          │
                          ▼
                   kube.Store  (indexed pod/owner view)
                          │
        ┌─────────────────┼──────────────────────────┐
        ▼                 ▼                          ▼
  WatcherKubeEnricher  KubeDecorator          netolly / statsolly
  (discovery)          (span decoration)      Kubernetes decorators
```

## Key code locations

| What | Where |
|---|---|
| Provider construction | `pkg/kube/informer_provider.go:NewMetadataProvider` |
| Provider configuration struct | `pkg/kube/informer_provider.go:MetadataConfig` |
| Enablement check | `pkg/kube/informer_provider.go:MetadataProvider.IsKubeEnabled` |
| Store access (triggers init) | `pkg/kube/informer_provider.go:MetadataProvider.Get` |
| Local informer setup | `pkg/kube/informer_provider.go:MetadataProvider.initLocalInformers` |
| Remote cache client setup | `pkg/kube/informer_provider.go:MetadataProvider.initRemoteInformerCacheClient` |
| Disable on failure | `pkg/kube/informer_provider.go:MetadataProvider.ForceDisable` |
| Node name resolution | `pkg/kube/informer_provider.go:MetadataProvider.CurrentNodeName` |
| Cluster name resolution | `pkg/kube/informer_provider.go:MetadataProvider.ClusterName` |
| Informer opt-out list | `pkg/kube/informer_provider.go:disabledInformerOpts` |
| gRPC subscription client | `pkg/kube/cache_svc_client.go:cacheSvcClient.connect` |
| Event handling | `pkg/kube/cache_svc_client.go:cacheSvcClient.On` |
| Reconnect backoff floor | `pkg/kube/cache_svc_client.go:normalizeReconnectInitialInterval` |
| Provider creation at boot | `pkg/instrumenter/instrumenter.go:BuildCommonContextInfo` |
| Failure fallback to Docker | `pkg/internal/appolly/appolly.go:setupKubernetes` |
| Span decoration node | `pkg/transform/k8s.go:KubeDecoratorProvider` |
| k8s-cache entry point | `cmd/k8s-cache/main.go` |
| k8s-cache wire protocol | `proto/informer.proto` |

## The k8s-cache service

Deployment guidance, RBAC minimums, configuration reference and internal metrics for the standalone service are documented in [`devdocs/k8s-cache.md`](../../../devdocs/k8s-cache.md). In short: a small Go binary, gRPC on port `50055` by default, published as `otel/opentelemetry-ebpf-k8s-cache`. Its source is split across `pkg/kube/kubecache/{service,meta,informer,instrument}`.

`proto/informer.proto` defines the only genuine service contract in this repository — a server-streaming `Subscribe` RPC between `k8s-cache` and agents. It is an internal cluster protocol, not a public API.

## Sharp edges

- **`Get` is the initializer.** The provider does lazy setup, so the first caller pays the informer sync cost. `BuildCommonContextInfo` and `refreshK8sInformerCache` deliberately force this early rather than letting it happen mid-pipeline.
- **Kubernetes failure is not fatal, it is a silent reshape.** The agent keeps running with Docker metadata; spans simply lose their pod attributes. The only signal is one error log at startup.
- **Enabling Kubernetes disables the Docker watcher.** `BuildCommonContextInfo` only starts the Docker store when Kubernetes is not enabled, and `setupKubernetes` starts it late if Kubernetes turns out to be unavailable.
- **`meta_restrict_local_node` changes what the informers see.** A DaemonSet restricted to its own node cannot resolve metadata for peers on other nodes, which shows up as unresolved names in service-graph metrics rather than as an error.
- **Attribute availability is decided at boot.** `attributeGroups` adds `GroupKubernetes` only if the informer reports enabled at that moment; a later force-disable does not remove the group.
- **Informers can be individually disabled.** `disabledInformerOpts` lets an operator drop, for example, the Service informer — reducing memory but removing the attributes derived from it.

## Related concepts

- [Attribute groups](observability/attribute-groups.md) — how Kubernetes availability selects exported attributes
- [Discovery](03-DISCOVERY.md) — the `WatcherKubeEnricher` stage
- [Pipeline and export](05-PIPELINE-AND-EXPORT.md) — the `KubeDecorator` stage
- [Entry point and boot sequence](02-ENTRYPOINT.md) — where the provider is built

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, incident references, dates, decisions that synthesis missed. -->

---

[← Previous](06-CONFIGURATION.md) · [Index](../index.md) · [Next →](08-NETWORK-AND-STATS.md)
