---
name: mcpshield-harness
description: Use when creating, changing, or auditing MCPShield agent instructions, skills, SDD artifacts, validation rules, or workflow automation. Look for broken paths, duplicated mandates, stale commands, and instructions that do not affect agent behavior.
metadata:
  source: https://github.com/tech-leads-club/agent-skills
  related_skills: harness-eval, skill-architect, docs-writer
---

# MCPShield Harness Maintenance

Treat `AGENTS.md`, `.agents/skills/`, `.specs/`, `docs/engineering-workflow.md`, and
`.github/` workflow templates as the harness surface.

Before changing it:

1. inventory referenced paths and commands;
2. confirm each path exists or is intentionally created by the change;
3. identify duplicated or conflicting rules;
4. test the smallest affected workflow;
5. update the progress checkpoint if behavior changes.

Keep project-specific instructions here. Reference upstream skills instead of copying them
unless MCPShield needs a concrete adaptation that can be maintained locally.