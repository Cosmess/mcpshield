# Specification Workspace

This directory contains the durable artifacts used by MCPShield's spec-driven workflow.

```text
.specs/
├── STATE.md
├── LESSONS.md
├── lessons.json
└── features/
    └── <feature>/
        ├── spec.md
        ├── context.md       # only when gray-area decisions need recording
        ├── design.md        # only when architecture needs explicit treatment
        ├── tasks.md         # only for complex work
        └── verification.md
```

Artifacts are created lazily. Do not create empty phase files. Feature slugs use lowercase
kebab case and remain stable for the life of the feature.

The upstream workflow references are documented in `.agents/skills/mcpshield-sdd/SKILL.md`.