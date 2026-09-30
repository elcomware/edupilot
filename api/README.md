# API

```text
api/
├── contracts/
└── openapi/
```

## Contracts

`api/contracts/` holds the transport-facing request and response shapes for every module. They are
defined by the owning module under `internal/**/contracts/`, and mirrored here for consumers
outside the repository.

The desktop edition and the HTTP edition satisfy the same contracts. That is what makes the desktop
app a fully functional node rather than a thin client.

## OpenAPI

`api/openapi/` holds the generated OpenAPI description of the HTTP adapter, used for the Site Server
and the Cloud API.

Rules:

- Generated from the contracts, never hand-edited. CI fails if the checked-in document is stale.
- Money is always `{ amountMinor, currency }`, never a bare number.
- Every error response uses the structured error model with an `AppErrorCode`, never a free-text
  message: `INSUFFICIENT_PERMISSION`, `PERIOD_CLOSED`, `INVOICE_ALREADY_POSTED`,
  `PAYMENT_ALREADY_REVERSED`, `INVALID_ALLOCATION`, `UNBALANCED_JOURNAL`, `STOCK_INSUFFICIENT`,
  `PAYROLL_LOCKED`.
- Every operation requires an organisation scope; the scope is never an optional filter.
