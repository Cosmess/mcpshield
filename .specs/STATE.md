# Project State

## Decisions

| ID | Decision | Status | Source |
| --- | --- | --- | --- |
| AD-001 | Use a repository-local, lightweight SDD and verification harness while referencing the upstream Tech Leads Club methodology. | Accepted | `docs/engineering-workflow.md` |
| AD-002 | Keep the root README in Portuguese and engineering artifacts in English. | Accepted | `CONTRIBUTING.md` |
| AD-003 | Require reviewed Pull Requests and a progress update after every merged slice. | Accepted | `CONTRIBUTING.md` |

## Handoff

- Current phase: M2 foundation merged with verification gaps.
- Active work: close authentication observability and JWKS verification gaps.
- Next action: add auth audit/metrics and edge-case tests before starting M3 policy work.
- Known blockers: GitHub status checks and branch protection are not configured yet.