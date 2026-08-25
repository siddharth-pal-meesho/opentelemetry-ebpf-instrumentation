<!-- This file captures service-specific code review knowledge that only your team knows.
     Write concrete, actionable rules under each heading — the AI reviewer will use these
     as mandatory context for every PR in this repo.
     Keep this file under 200 lines — be specific, not exhaustive. -->

## How to Review in This Service

<!-- List the data rules that must ALWAYS hold in this service — things that would break
     the business or corrupt data if violated.
     For each invariant, state the entity, the constraint, and what to flag in a diff. -->
**Domain invariants**
- TODO

<!-- List the high-value request paths a reviewer should trace end-to-end.
     For each workflow, state the entry point (route/controller), the handler chain, and
     why it matters (latency-sensitive, money path, etc.). -->
**Critical workflows**
- TODO

<!-- List bugs, outage patterns, or recurring mistakes this service has hit before.
     For each pattern, describe the failure, root cause, and what to watch for in diffs. -->
**Known failure patterns**
- TODO

<!-- List hot paths, SLA-sensitive operations, resource constraints, and performance
     conventions specific to this service. Mention concrete thresholds where they exist. -->
**Performance considerations**
- TODO

<!-- List external systems this service talks to and the contracts/quirks a reviewer
     should know. Include auth, messaging, config backends, and their failure modes. -->
**Integration boundaries**
- TODO
