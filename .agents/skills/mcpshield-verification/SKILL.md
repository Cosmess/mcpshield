---
name: mcpshield-verification
description: Use to verify MCPShield work against its specification, tests, security constraints, and operational gates. Require evidence per acceptance criterion and keep author and verifier reasoning separate when possible.
metadata:
  source: https://github.com/tech-leads-club/agent-skills
  related_skills: tlc-spec-driven, spec-driven-eval, the-judge
---

# MCPShield Verification

Read the active feature `spec.md` and the implementation diff. Do not infer completion
from file presence or from a passing build alone.

For every acceptance criterion, record:

- the expected outcome from the spec;
- the test, command, or inspection that proves it;
- the observed result;
- any gap or unexecuted check.

Run the narrowest relevant check first, then the repository gates. For Go changes, the
baseline gates are `go test ./...`, `go test -race ./...`, and `go vet ./...` once the
module exists. Security-sensitive changes also require abuse-case tests and explicit
review of logs, errors, timeouts, and secret handling.

Write the result to the active feature's `verification.md`. Use PASS only when every
required criterion has evidence; otherwise use FAIL or PASS WITH GAPS and list the gaps.