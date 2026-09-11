# Contributing to MCPShield

## Branches

`main` is protected conceptually: work starts from an up-to-date `main` branch and lands
through a Pull Request.

Use a focused branch name:

```text
feature/<short-name>
fix/<short-name>
docs/<short-name>
chore/<short-name>
```

One coherent change belongs in one branch. Do not mix unrelated refactors into a feature.

## Pull Requests

Every Pull Request must state the requirement or decision it implements, the verification
performed, known gaps, and the next progress checkpoint. Review is required before merge.

The normal lifecycle is:

```text
main -> focused branch -> atomic commits -> Pull Request -> review -> merge -> progress update
```

## Commits

Use Conventional Commits and keep commits atomic:

```text
feat: add upstream registry
fix: reject expired bearer tokens
docs: record MCP protocol baseline
test: cover approval fingerprint replay
chore: update development harness
```

Do not claim a check passed unless it was executed. Record unavailable checks explicitly.

## Documentation language

The root README is Portuguese for the portfolio audience. Engineering documentation,
ADRs, specs, skills, and code comments are written in English.