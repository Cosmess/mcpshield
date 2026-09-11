---
name: mcpshield-sdd
description: Use for MCPShield feature discovery, specification, design, task breakdown, implementation, and progress handoff. Keep requirements traceable and tests derived from observable outcomes. Do not use for unrelated repository work.
metadata:
  source: https://github.com/tech-leads-club/agent-skills
  related_skills: tlc-spec-driven, tlc-plan, tlc-implement
---

# MCPShield Spec-Driven Development

Read `AGENTS.md`, `docs/progress.md`, `.specs/STATE.md`, and only the active feature
artifacts before acting.

Use the upstream `tlc-spec-driven` lifecycle as the reference:

```text
SPECIFY -> DESIGN when needed -> TASKS when needed -> EXECUTE -> VERIFY
```

Project-specific requirements:

- Acceptance criteria must describe externally observable outcomes and use stable IDs.
- Security behavior must state the default and the failure behavior.
- Each task names its touched paths, tests, gate, and done-when evidence.
- Create artifacts lazily; never scaffold empty SDD phase files.
- Keep one feature branch and one focused Pull Request per coherent slice.
- Update `docs/progress.md` only with verified facts after merge.

For complex features, use `.specs/features/<feature>/spec.md`, `design.md`, `tasks.md`,
and `verification.md` as needed. Use ADRs for project-level decisions, not feature task lists.