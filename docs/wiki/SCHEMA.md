# Wiki schema for obi

This file is the contract for `docs/wiki/`. The single skill `m-wiki:wiki-init` reads this on entry. Do not edit `index.md`, `log.md`, or `SCHEMA.md` directly — the PreToolUse hook will block.

`raw/` is the team's input layer — humans drop sources there freely; **the hook does NOT block writes to `raw/`**. Run `wiki-init` whenever the team adds files to `raw/` or after meaningful code changes.

## Code precedence

For any wiki claim with a `path:Symbol` or `path:LINE` citation, the code at HEAD is authoritative. If code disagrees, the claim is wrong. `wiki-init` (update mode) auto-updates without human gate. Source-vs-source contradictions get a `> ⚠ contradicts: …` callout in the body.

## Boundary with the repository's own documentation

This repository is upstream OpenTelemetry eBPF Instrumentation (OBI). Its canonical developer documentation lives in [`devdocs/`](../../devdocs/README.md) — notably [`devdocs/pipeline-map.md`](../../devdocs/pipeline-map.md), [`devdocs/context-propagation.md`](../../devdocs/context-propagation.md) and [`devdocs/k8s-cache.md`](../../devdocs/k8s-cache.md). Contributor rules live in [`AGENTS.md`](../../AGENTS.md) and [`CONTRIBUTING.md`](../../CONTRIBUTING.md).

**The wiki does not duplicate those files.** Its value is:

- Atomic concept pages — cross-cutting patterns referenced from many places in the tree
- Symbol-anchored navigation from a question ("where does a PID become a tracer?") to code
- Synthesized knowledge from sources dropped in `raw/`
- Compounding knowledge that grows over time as `raw/` accumulates

Note: the m-wiki default template names `meesho-init` as the owner of `docs/architecture.md` and `CLAUDE.md`. That skill has not been run here and `docs/architecture.md` does not exist; `devdocs/` plays that role in this repository.

## Page types

| Type | Path | Constraints |
|---|---|---|
| `top-level` | `pages/NN-NAME.md` | Narrative — focus on slices NOT covered by `devdocs/` |
| `concept` | `pages/<topic>/<slug>.md` | Atomic — soft target ≤600 words; lint warns at >1500 |

`raw/<topic>/<file>.md` is immutable team-curated storage. Humans drop files there; `wiki-init` reads them.

## Hierarchy

`max-hierarchy-depth: 1`. Single root `index.md` (no per-topic index files in v1).

The stack heuristic detects multiple `go.mod` files outside `internal/tools/` and would ordinarily select depth 2. Those extra modules are throwaway probe binaries under `configs/offsets/`, OATS test harnesses under `internal/test/oats/`, and vendored demos under `examples/` — not modules of a monorepo. The product is a single Go module (`go.opentelemetry.io/obi`), so topics stay flat at depth 1.

## Topics declared at init

- `pipeline` — the swarm/queue substrate every OBI pipeline is assembled from
- `ebpf` — the tracer contract, probe attachment, and kernel-side context propagation
- `discovery` — how a running process becomes an instrumented target
- `observability` — how OBI measures and attributes itself

New topics are created by `wiki-init` (update mode) when a `raw/` file fits no existing topic.

## Page metadata

Every `top-level` and `concept` page begins with:

```
<!-- m-wiki: type=<type> slug=<slug> topic=<topic-or-null> base-sha=<12-char> generated-at=<ISO> sources=[...] -->

> Generated <date> at base-sha <12-char>. Type: <type>. <N> sources.
```

`sources=[...]` lists `raw/<topic>/<file>.md` paths the page draws from. `wiki-init` validates each path exists.

## Naming

- Top-level: `pages/NN-NAME.md` — `NN` two-digit zero-padded; `NAME` `KEBAB-UPPERCASE`.
- Concept: `pages/<topic>/<slug>.md` — `topic` and `slug` `kebab-lowercase`.
- Raw: `raw/<topic>/<filename>.md` — convention: `<YYYY-MM-DD>-<slug>.md` so chronology is sortable; not enforced.

## Link format

- Internal: `[<title>](<relative-path>)`. Markdown only — Obsidian wikilinks `[[...]]` are not used.
- Code (preferred): `path/to/file.go:FunctionName` — function names are stable across formatters/refactors. Lint greps for `^(func|type) FunctionName` at HEAD.
- Go methods: `path/to/file.go:ReceiverType.MethodName` — the verifier has a receiver-aware pattern for `func (r *ReceiverType) MethodName(`.
- Code (fallback): `path/to/file.go:LINE` or `path/to/file.go:LINE-LINE` for spans without a single function anchor.

**Go specifics that force the LINE fallback in this repository:**

- Generic receiver methods (`func (q *Queue[T]) Send(...)`) — the receiver pattern does not accept the `[T]` type parameter list.
- Package-level `var` and `const` declarations (`var DefaultConfig`, `GroupKubernetes`) — the Go pattern only anchors `func` and `type`.

`.c` and `.h` files have no verifier pattern at all, so eBPF C anchors are cited as `path:LINE` and are reported as `unsupported_ext` LINE residual, never as drift.

## Update-mode decision tree

When `wiki-init` runs in update mode and finds an uncited file in `raw/`:

1. **Run code-truth on candidate pages first** so decisions are made against fresh state.
2. **Source extends an existing concept** → append the raw path to the existing page's `## Sources` and provenance `sources=[]`. Create a NEW concept page for genuinely new material that builds on the original (open with "Builds on [<existing-slug>](...)"). Do not bloat the existing page.
3. **Source contradicts an existing claim, code-arbitrated** → code precedence. Add a `> ⚠ note: <raw-path> disagrees with claim at <file:Symbol>` callout. Do not modify the claim.
4. **Source contradicts, non-code-arbitrated** → add `> ⚠ contradicts: …` callout. Human resolves later.
5. **Novel cross-cutting idea** → new concept page under appropriate topic (or new topic dir).
6. **Doesn't fit anywhere** → ask the user where to place it. No silent stubs.

## qmd integration

Registered as qmd collection: **`obi-wiki`**. Scope is `docs/wiki/` — the synthesized answer surface. `raw/` (at repo root, team-curated input) is **not** indexed; concept pages cite the raw paths and agents read raws via those citations.

The collection is named for the repository (`obi`), not for `basename $REPO_ROOT`. This wiki was bootstrapped inside an ephemeral Context Maintainer worktree whose directory name is a throwaway identifier; naming the collection after it would have produced a permanent qmd entry pointing at a path that gets deleted.

- Bootstrap: `wiki-init` runs `qmd collection add docs/wiki/ --name obi-wiki` after writing files.
- Update: `wiki-init` runs `qmd update obi-wiki` after each sync.
- The PostToolUse hook fires `qmd update` whenever `Edit`/`Write` modifies `docs/wiki/pages/**`.
- Retrieval: agents query via the qmd MCP server (`qmd query "<question>" --collection obi-wiki`). No dedicated wiki-query skill — qmd's MCP gives every agent direct access.

**Optional second collection for `raw/`**: if a team wants raw sources directly searchable (e.g., before they're synthesized into pages), run once: `qmd collection add raw/ --name obi-wiki-raw && qmd update obi-wiki-raw`. Not auto-registered.

## Machine-managed artifacts (v0.6+)

Two hidden artifacts live alongside the page tree under `docs/wiki/`. Both are committed to git so they're auditable in code review; neither is human-edited.

| Artifact | Owner | Lifecycle |
|---|---|---|
| `docs/wiki/.citation-index.json` | wiki-init Phase 2 | Reverse index of every `path:Symbol` citation in the wiki → which page(s) cite it. Bootstrap builds it from scratch (step 9.5); update mode incrementally rewrites entries for regenerated pages only (step 2.7). Schema versioned (`v0.6.0`). Consumed by the pre-commit shim (below) to detect drift. |
| `docs/wiki/.drift-queue/<ISO-TIMESTAMP>.yml` | pre-commit shim | One YAML file per commit that touches any source file cited by the wiki. The shim writes these; `wiki-init` update mode reads them (Phase 1 step 5.5) and drains the resolved ones after Phase 2 regeneration (step 5.5). Only `affected_pages` is load-bearing; everything else is human-readable metadata. |

**Pre-commit framework hooks.** m-wiki normally registers `m-wiki-drift-queue` (pre-commit) and `m-wiki-sha-backfill` (post-commit) as `repo: local` entries in a `.pre-commit-config.yaml`. They look up staged files in `.citation-index.json` and write a `.drift-queue/` entry when wiki-cited sources change. Never blocks a commit. `git commit --no-verify` bypasses them via git's own mechanism.

Note for this repository: there is **no** `.pre-commit-config.yaml`. This project uses a plain git hook — `hooks/pre-commit`, copied into `.git/hooks/pre-commit` by the `install-hooks` Makefile target, which `prereqs` (and therefore `make verify`) depends on. m-wiki's hooks were **not** installed by this bootstrap: wiring them would mean either editing the upstream-owned `hooks/pre-commit` or introducing a pre-commit framework this project does not use. That is a maintainer's decision, not a side effect of generating a wiki. Until it is made, `.drift-queue/` stays empty and update-mode runs rely on the code-truth precheck alone.

Power users can hand-write a `.drift-queue/<timestamp>.yml` to force regeneration of specific pages on the next wiki-init run.

---

<!-- m-wiki: schema-version=2 generated-at=2026-08-13T14:38:34Z base-sha=724e96d5baf0 -->
