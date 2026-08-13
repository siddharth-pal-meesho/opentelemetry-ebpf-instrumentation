<!-- m-wiki: type=concept slug=attribute-groups topic=observability base-sha=724e96d5baf0 generated-at=2026-08-13T14:38:34Z sources=[] -->

> Generated 2026-08-13 at base-sha 724e96d5baf0. Type: concept. 0 sources.

# Attribute groups

Metric cardinality is the dominant cost in any metrics backend, and attributes are what create it. Rather than exporting every attribute it could compute, OBI enables *groups* of related attributes, and only when the data behind them is actually available.

## The groups

Declared as a bitmask in `pkg/export/attributes/attr_defs.go:21-38`:

| Group | Covers |
|---|---|
| `GroupKubernetes` | pod, namespace, workload owner |
| `GroupContainer` | container identity, when Kubernetes is absent |
| `GroupHTTPRoutes` | the grouped URL route |
| `GroupNetIfaceDirection` | network interface and traffic direction |
| `GroupNetCIDR` | CIDR classification of peers |
| `GroupNetGeoIP` | GeoIP-derived location |

## Enablement happens once, at boot

`pkg/instrumenter/instrumenter.go:attributeGroups` decides the default set during `BuildCommonContextInfo`, before any pipeline exists:

- `GroupKubernetes` if the informer reports enabled; **else** `GroupContainer` if Docker metadata is enabled — the two are mutually exclusive by construction.
- `GroupHTTPRoutes` if `config.Routes` is non-nil.
- `GroupNetIfaceDirection` if the network deduper is `DeduperNone`.
- `GroupNetCIDR` if CIDRs are configured; `GroupNetGeoIP` if GeoIP is.

Each is *added* to `ctxInfo.MetricAttributeGroups`; nothing is ever removed.

## Why availability drives enablement

The pairing of `GroupKubernetes` and `GroupContainer` shows the principle. Both answer "where is this running", from different sources. Enabling both would double cardinality to express one fact twice, so exactly one is chosen based on which metadata source exists.

`GroupNetIfaceDirection` follows the same logic inverted. Deduplication collapses the two directional observations of one flow into a single record; with deduplication on, interface and direction are no longer meaningful distinguishing attributes, so the group is only added when the deduper is off. **Turning deduplication off therefore widens metric cardinality**, which is not obvious from a setting named "deduper".

## Group defaults versus user selection

Groups are the *default* attribute set. Users override per metric through `attributes.select`, carried as `SelectionCfg` in `attributes.SelectorConfig` and assembled in `pkg/appolly/instrumenter.go:newGraphBuilder` alongside `ExtraGroupAttributesCfg` and `SensitiveQueryParamsCfg`.

`pkg/export/attributes/attr_defs.go:57-62` reads the group flags back out to build the concrete attribute definitions, and `extraGroupAttributes` allows adding attributes to a group without changing the group's own definition.

## Sharp edges

- **The decision is not revisited.** If Kubernetes later force-disables — see `pkg/internal/appolly/appolly.go:setupKubernetes` — `GroupKubernetes` remains in the set. The group is enabled; the values simply resolve empty.
- **A config reload cannot change groups.** They are computed once in `BuildCommonContextInfo`; enabling GeoIP later requires a restart.
- **Sensitive query parameters are configured separately.** `SensitiveQueryParamsCfg` is not a group — it is a redaction list, and enabling `GroupHTTPRoutes` does not imply any redaction.
- **The groups are network-heavy.** Three of six exist for the [network and stat pipelines](../08-NETWORK-AND-STATS.md), so reading this list as "span attributes" is misleading.

## Related

- [Internal metrics](internal-metrics.md) — the other boot-time observability decision
- [Pipeline and export](../05-PIPELINE-AND-EXPORT.md) — where `SelectorConfig` is used
- [Kubernetes metadata](../07-KUBERNETES-METADATA.md) — the availability signal for `GroupKubernetes`
- [Network and stat pipelines](../08-NETWORK-AND-STATS.md)

## Sources

No `raw/` sources yet — this page is synthesized from code at HEAD.

## Notes

<!-- Anything below is human-owned. wiki-init never reads or modifies content under this heading.
     Use this for tribal knowledge, decisions, dates, incident references the synthesis missed. -->

---

[← Wiki index](../../index.md)

<!-- atomic: keep this page ≤600 words. New scope → new concept page that builds on this one. Do not append paragraphs here. -->
