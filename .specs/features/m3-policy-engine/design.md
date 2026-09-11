# M3 Native Policy Engine Design

## Context

M3 is the first authorization slice. It must be pure, deterministic, and usable by the MCP proxy without provider-specific policy types. The active policy set is compiled once at startup into immutable rules.

## Core types

```text
Decision
PolicyInput
OperationClass
PolicyRule
EvaluationResult
```

`EvaluationResult` contains the decision, matched policy IDs, reason, and optional limits. It does not contain HTTP responses, OPA values, or AI output.

## Evaluation order

1. Validate policy configuration at startup.
2. Build typed input from authenticated principal and MCP metadata.
3. Classify the operation using registry metadata and conservative fallback.
4. Find all matching rules.
5. Select highest priority.
6. Resolve equal-priority conflicts with deny-wins.
7. Return a complete result for enforcement and audit.

## Decision precedence

Priority is explicit and higher values win. For equal priority, the restrictive ordering is:

```text
DENY > REQUIRE_APPROVAL > ALLOW_WITH_LIMITS > ALLOW_WITH_REDACTION > ALLOW
```

This is a deterministic safety rule, not a substitute for policy review.

## Matching

Rules may constrain:

- subject and role;
- tenant;
- upstream ID;
- MCP method;
- tool glob/pattern;
- operation class;
- argument paths with exact, enum, or bounded predicates.

An omitted dimension means any value. A rule never matches an unknown operation unless it explicitly names `UNKNOWN`.

## Enforcement boundary

The evaluator runs before `session.CallTool`. `DENY` returns a generic MCP protocol error and records an audit event. `REQUIRE_APPROVAL`, redaction, and limits are returned as decisions in M3 but remain non-executing until later phases implement them.

## Simulation

Simulation receives a candidate immutable policy set and a typed input. It returns the same `EvaluationResult` as live evaluation and cannot access an upstream session or active mutable state.

## Testing strategy

- Table-driven matcher and precedence tests.
- Classifier tests for discovery, read, write, execution, admin, unknown.
- Proxy integration tests with an outbound call counter.
- Simulation tests proving no outbound call and no active policy mutation.
- Fuzz tests for tool patterns and argument path extraction where parsing is introduced.

## Risks and mitigations

| Risk | Mitigation |
| --- | --- |
| Glob patterns overmatch tools | Validate and test exact boundary cases. |
| Policy order becomes accidental precedence | Compile explicit priorities and sort deterministically. |
| Unknown classification is too permissive | Unknown defaults to deny. |
| Future OPA semantics diverge | Keep the internal decision contract stable and adapter-specific. |
