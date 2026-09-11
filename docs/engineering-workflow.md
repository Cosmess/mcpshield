# Engineering Workflow

MCPShield uses a lightweight, repository-local harness inspired by the Tech Leads Club
spec-driven skills. The upstream skills remain the methodology reference; this repository
stores only project-specific rules and artifacts.

## SDD lifecycle

```text
Specify -> Design when needed -> Tasks when needed -> Execute -> Verify -> Pull Request -> Merge
```

### Specify

Create `.specs/features/<feature>/spec.md` with a concrete problem, scope, assumptions,
EARS-shaped acceptance criteria, requirement IDs, and observable outcomes. Do not write
implementation code before the acceptance criteria are reviewable.

### Design

For changes with new boundaries, persistence, protocols, security behavior, or one-way
doors, add `design.md`. Capture interfaces, data flow, risks, reuse, and project-level
decisions. Record durable architectural decisions in an ADR and `.specs/STATE.md`.

### Tasks

For complex work, add `tasks.md` with atomic tasks, dependencies, touched paths, tests,
and a gate command. A task must produce a demonstrable outcome rather than a technical
layer with no user-visible or operational proof.

### Execute

Implement one task at a time. Derive tests from the acceptance criteria, run the narrowest
useful gate first, then the required project gates. Commit only verified work.

### Verify

Verification is independent of authorship where possible. Record per-criterion evidence,
commands and results in `verification.md`. A missing or unexecuted check is a gap, not a pass.

## Pull Request and merge policy

Use one focused branch per slice. Open a Pull Request after local verification, request
review, address findings, and merge only after required checks pass. Update `docs/progress.md`
after merge. Never use a direct push to `main` as a substitute for review.

## Harness maintenance

Run a harness evaluation when instruction paths, skills, routing rules, or validation scripts
change materially. Check for broken paths, duplicated rules, and instructions that do not
change agent behavior. Keep the harness smaller than the product documentation it protects.