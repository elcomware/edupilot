# ADR-005 — Integer minor units for money

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

EduPilot is a double-entry accounting system. A rounding error of one minor unit is a ledger error, and a float cannot represent `0.10` exactly. Schools also transact in currencies with and without subunits (XOF has none; EUR, GBP and USD have two).

## Decision

Represent money as a value object:

```text
Money {
    amount_minor: int64
    currency:     Currency
}
```

- **No floating point anywhere** in the money path — not in Go, not in the database, not in TypeScript, not in JSON.
- Currencies declare their own scaling, so zero-decimal currencies store whole units and two-decimal currencies store cents.
- Exchange rates use exact decimal/rational representation, never `float64`.
- Percentages used in discounts, tax and instalment plans use integer basis points.
- Money is never multiplied by a quantity implicitly; a `Multiply` operation takes an explicit, rounding-aware strategy.

## Consequences

- Totals are exact and reproducible; trial balance always balances to zero.
- The database stores integers, which both SQLite and PostgreSQL handle natively.
- Formatting and parsing become explicit responsibilities rather than free `toFixed` calls.
- Every arithmetic operation must state its rounding behaviour. This is deliberate friction.

## Alternatives considered

- **`float64`** — rejected: non-deterministic rounding breaks financial invariants.
- **Decimal stored as a scaled integer with a global scale** — rejected: cannot represent both zero-decimal and two-decimal currencies correctly.
- **String amounts** — rejected: arithmetic becomes string manipulation and comparisons become lexical.
