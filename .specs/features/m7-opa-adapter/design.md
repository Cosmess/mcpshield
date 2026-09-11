# M7 OPA/Rego Adapter Design

## Adapter boundary

Define an internal evaluator boundary conceptually equivalent to:

```text
Evaluate(ctx, policy.Input) -> policy.Result
```

Native and OPA implementations return the same internal result. The proxy does not import OPA SDK types, Rego values, or HTTP client details.

## Sanitized OPA input

```json
{
  "principal": {
    "subject": "user-1",
    "tenant_id": "tenant-a",
    "roles": ["developer"],
    "scopes": ["mcp:call"]
  },
  "mcp": {
    "upstream_id": "github",
    "method": "tools/call",
    "tool": "github.merge_pr",
    "operation": "WRITE"
  },
  "risk": {
    "score": 65,
    "severity": "HIGH",
    "signals": ["write_operation", "production_environment"]
  },
  "arguments": {
    "repository": "example/repo",
    "pull_request": "42"
  }
}
```

Arguments are bounded and passed through DLP sanitization before adapter input. Tokens,
private keys, matched secret values, and raw payloads are excluded.

## Strict output

Accept only:

```json
{
  "decision": "ALLOW|DENY|REQUIRE_APPROVAL|ALLOW_WITH_REDACTION|ALLOW_WITH_LIMITS",
  "reason": "short sanitized reason",
  "policy_ids": ["policy-id"],
  "policy_version": "bundle-v12"
}
```

Unknown decisions, excessive reason length, invalid policy IDs, or malformed JSON fail safely.

## Availability

OPA calls have context timeout and bounded response bytes. Configuration chooses one mode:

```text
opa_required       -> timeout/unavailable fails closed
opa_with_native_fallback -> timeout/unavailable invokes native evaluator and audits fallback
native_only        -> no OPA client is created
```

Fallback is never implicit.

## Decision precedence

The adapter cannot weaken a hard native safety deny. If future composition evaluates native
and OPA together, the restrictive decision wins unless a reviewed ADR defines another rule.

## Observability

Fixed metrics:

```text
mcpshield_opa_evaluations_total
mcpshield_opa_denies_total
mcpshield_opa_timeouts_total
mcpshield_opa_fallbacks_total
mcpshield_opa_errors_total
```

Audit metadata includes evaluator, policy version, decision, duration, and outcome. It does
not include raw input or credentials.

## Testing strategy

- Fake OPA HTTP server for valid, deny, malformed, slow, and unavailable responses.
- Contract tests shared by native and OPA evaluators.
- Sanitization tests with synthetic secrets and nested arguments.
- Timeout and cancellation tests.
- Simulation test proving no upstream call.
- Race tests for immutable configuration and concurrent evaluations.
