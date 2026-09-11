# MCPShield Agent Instructions

## Source of truth

- Read `README.md` and `docs/progress.md` before starting work.
- Read only the active feature artifacts under `.specs/features/<feature>/`.
- Project-level decisions live in `.specs/STATE.md` and `docs/adr/`.
- Do not treat chat memory as a requirement; record durable decisions in the repository.

## Engineering rules

- Implement the smallest coherent vertical slice.
- Security decisions are deterministic and explicit. AI output is advisory only.
- Keep request paths cancellable, bounded, observable, and auditable.
- Never commit secrets, bearer tokens, credentials, or unsanitized sensitive payloads.
- Prefer the standard library and focused maintained dependencies.
- Do not add empty placeholder packages or speculative abstractions.

## SDD workflow

Use the local skills in `.agents/skills/` as the project routing layer:

1. `mcpshield-sdd` for specification and task execution.
2. `mcpshield-verification` for evidence-based validation.
3. `mcpshield-harness` when auditing or changing the agent harness itself.

Feature artifacts are created lazily. A skipped SDD phase must not leave an empty file.

## Completion gate

A task is not complete until its acceptance criteria are covered by tests or an explicit
verification record, the relevant checks pass, documentation is updated, and the branch
has a reviewed Pull Request merged into `main`.