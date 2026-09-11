# M6 Human Approval Design

## Context

Approval is a security-sensitive state machine, not a controller flag. The domain service creates and transitions records; repositories provide atomic operations; HTTP handlers and future AI explanations cannot mutate state directly.

## Core types

```text
ApprovalStatus: PENDING | APPROVED | DENIED | EXPIRED | CONSUMED
ApprovalRequest
ApprovalRecord
Reviewer
ApprovalDecision
ApprovalRepository
ApprovalService
```

## Fingerprint

Canonicalize only sanitized fields:

```text
principal subject
tenant ID
upstream ID
MCP method
tool name
canonical argument hash
policy IDs
risk score/severity/signal IDs
```

Hash the canonical representation with a standard cryptographic hash. Never include raw bearer tokens, credentials, or raw DLP matches.

## Transition rules

| From | Event | To | Guard |
| --- | --- | --- | --- |
| none | create | PENDING | policy decision is REQUIRE_APPROVAL |
| PENDING | approve | APPROVED | reviewer scope, not self-review by default, not expired |
| PENDING | deny | DENIED | reviewer scope |
| PENDING | clock/lookup | EXPIRED | expiry reached |
| APPROVED | consume | CONSUMED | fingerprint match, not expired, atomic compare-and-consume |
| APPROVED | clock/lookup | EXPIRED | expiry reached before consume |

No transition reopens DENIED, EXPIRED, or CONSUMED.

## Concurrency

`Consume` must be an atomic repository operation equivalent to compare status/fingerprint/expiry and update APPROVED -> CONSUMED in one transaction. Exactly one concurrent caller may succeed.

## Sensitive data handling

The record stores sanitized arguments or a summary plus hash, never raw credentials. DLP match metadata may include detector ID and field path, not values. Audit stores transition metadata only.

## API/use-case boundary

The first implementation should expose domain methods such as:

```text
CreatePending(ctx, request)
Approve(ctx, approvalID, reviewer)
Deny(ctx, approvalID, reviewer, reason)
Consume(ctx, approvalID, boundRequest)
Expire(ctx, now)
```

No method accepts an AI provider as an authority. An explanation adapter may read a sanitized record but has no transition capability.

## Testing strategy

- Table-driven state transition tests.
- Sanitization and fingerprint mutation tests.
- Reviewer scope and self-approval tests.
- Expiry boundary tests with a controllable clock.
- Race test with concurrent consumption.
- Fake repository tests proving atomic compare-and-consume semantics.
- Proxy integration test proving no upstream call without successful consume.

## Risks and mitigations

| Risk | Mitigation |
| --- | --- |
| Self-approval privilege escalation | Reject same subject by default and require explicit configuration. |
| Approval replay | Fingerprint binding, TTL, and one-time consumption. |
| Audit leakage | Metadata-only sanitized events. |
| Repository race | Atomic compare-and-consume contract and race tests. |
| Stale approval after policy change | Include policy IDs/version/context in fingerprint. |
