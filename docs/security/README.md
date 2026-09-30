# Security

## Permission model

Permissions are granular, scoped and policy-bound. Role names are a convenience, not the
authorisation mechanism.

```text
finance.invoice.create
finance.invoice.approve
finance.payment.receive
finance.payment.reverse
finance.refund.request
finance.refund.approve
payroll.run.create
payroll.run.approve
accounting.period.close
```

A permission is evaluated against:

- the actor and their roles;
- the `organisation_id` and, where relevant, the `campus_id`;
- the business policy for the operation (amount thresholds, period state, self-approval limits).

Enforcement happens in the application service and the repository. The frontend hides controls it
knows about; it is never the enforcement point.

## Maker-checker

Critical operations require two people:

- refunds;
- large discounts and waivers;
- supplier payments;
- payroll approval and posting;
- manual journal entries;
- budget transfers;
- accounting period closure.

The actor who creates a transaction may not be the actor who approves it. The workflow engine in
`internal/platform/workflow` decides the approval chain from the configured thresholds.

## Financial immutability

Posted financial records cannot be edited or deleted. Corrections are reversal plus replacement,
and every reversal references its original. Attempting a mutation on a posted record is rejected by
the domain, not by convention.

## Audit

Every sensitive change produces an append-only audit record capturing who, what, when,
organisation, campus, device and session, entity and entity ID, operation, before, after, reason
and correlation ID. Financial audit history is never rewritten.

## Data protection

- Passwords are hashed with a memory-hard function (Argon2id) and a per-user salt.
- Backup archives are encrypted before they leave the node.
- Sensitive exports are permission-controlled and audited.
- Secrets are never committed. Configuration comes from the settings hierarchy or the environment.

## Tenant isolation

Every relevant record carries `organisation_id`. Repositories, services, permissions and tests all
enforce the boundary. Cross-organisation access is an explicit, permission-gated, audited operation.

## Offline nodes

A school node holds a complete copy of its school's data. Node-to-cloud transport is TLS. Inbound
cloud data passes an idempotency check before any application service runs, and posted financial
records are immutable across the sync boundary, so a cloud payload can never silently rewrite
school books.
