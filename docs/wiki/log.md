# Operation log

Append-only chronological record. Format: `## [<ISO-date>] <op> | <subject> | <one-line>`.
Operations: bootstrap · sync · code-truth-update.

---

## [2026-08-13T14:38:34Z] bootstrap | obi | base-sha=724e96d5baf0

Pages: 19 (9 top-level, 10 concepts). Topics: 4. Raw files at init: 0.

Invoked by Context Maintainer with `mode=reconcile`, but the invocation carried no
`update_wiki` work item and `docs/wiki/SCHEMA.md` did not exist. Phase 1's reconcile
fast path escalated to full mode on both documented triggers
(`ctxm.reconcile.escalate reason=no-wiki-schema`, `reason=empty-target-keys`), so this
ran as a bootstrap.

qmd collection `obi-wiki` was NOT registered — the installed qmd 2.1.0 fails to load
`better-sqlite3` (built for NODE_MODULE_VERSION 131, runtime requires 147). Wiki pages,
MANIFEST.md and .citation-index.json were written regardless; retrieval is unavailable
until qmd is repaired and `/m-wiki:wiki-init` is re-run.
