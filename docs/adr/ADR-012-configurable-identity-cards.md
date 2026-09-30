# ADR-012 — Configurable identity cards

- **Status:** Accepted
- **Date:** 2026-09-29
- **Affects:** `internal/platform/identitycards`, `internal/platform/documents`, `internal/infrastructure/documents`

## Context

Every school issues identity cards, and no two schools issue the same card. A card carries a photo, a name, a role, a card number, an expiry, campus branding, and — increasingly — a barcode or QR for gate and canteen use. The card for a student, the card for a staff member and the card for a guardian differ in layout, in which fields appear, and in what the number means.

The blueprint (§59, §60) requires evidence documents to attach to many entities and requires one reusable document engine. It does not say identity cards are documents, and treating them as ordinary documents produces a worse result: a card is a fixed, printable, batch-oriented, re-issuable layout with a lifecycle of its own, not a rendered page of text.

Requirements gathered from the domain:

- cards for students, employees and guardians, from configurable templates rather than hard-coded layouts;
- a card number per template with its own numbering sequence, e.g. `STU/2026/000123`, distinct from the person's own student or employee number;
- a validity window tied to the academic year, so a card expires when the enrolment does and is bulk-reissued at the start of the next year;
- a photo, stored through the storage abstraction, not as a column in SQL;
- a machine-readable element (barcode or QR) carrying the card number, for gate scanning later;
- campus branding — logo, colours, wording — which is a rendering concern;
- batch issue, re-issue, and revocation, with every card number issued recorded permanently, because a lost card must never be reissued under the same number.

## Decision

`platform/identitycards` is a platform module that owns the card *lifecycle* and the *template model*. It does not own rendering. It depends on People for person data and on the document engine for bytes.

### Template is data, not code

A template is a persisted, organisation-scoped, effective-dated record: the card kind (student, employee, guardian), the academic year it applies to, the ordered list of fields to render, the card-number pattern, the validity rule, and the machine-readable settings. Field references are stable names such as `person.full_name` or `person.student_number`, resolved by the module against a read-only projection of the person.

A template is versioned and effective-dated exactly like fee structures (blueprint §21): last year's template is never overwritten, because a card issued last year must still be explainable.

### Card numbers come from the numbering engine

Card numbers use the existing numbering engine (blueprint §61) with a `{TYPE}/{YEAR}/{SEQ:6}` template, not a column on the person. A card number is issued once and never reused, even after revocation.

### The card is an issued record, not a document

```text
idcard_templates        the configurable layout, effective-dated and versioned
identity_cards          one row per card ever issued: number, person, template, issued_on, expires_on, status, photo_document_id
```

Status moves `ACTIVE → SUSPENDED → REVOKED` or `ACTIVE → EXPIRED`. Revocation and expiry are status changes, never deletion, so "card 000123 was issued to this person in 2026 and revoked in 2027" stays answerable.

### Rendering is delegated

The module produces a render request — resolved field values, template, branding, layout — and hands it to the document engine. The renderer is an infrastructure concern (`infrastructure/documents`), so replacing a PDF library, adding a label printer, or rendering in the cloud later touches no domain code. Bytes are stored through the storage abstraction and referenced by `photo_document_id`, consistent with ADR documents metadata/bytes split.

### One card number per person per template

A person holds at most one active card per template, which the database enforces with a partial unique index. Re-issuing revokes the previous card and issues a new number in the same transaction, so a lost card can never be duplicated.

## Consequences

- Issuing a card for a student requires a Student role for the current academic year, so the module validates the role exists rather than trusting the caller. The same check applies to employees and guardians.
- The annual re-issue is a bulk use case over the people who hold a qualifying role, and it is idempotent per person and academic year.
- Templates are configuration, so a school can add a field or change the layout without a release. The set of resolvable field names is a closed, documented set — a template may not invent field paths.
- Campus branding is resolved at render time from the person's campus, so one template serves a multi-campus group.
- Gate scanning is a later consumer of the card number; the module exposes a lookup by number from the first release so the later feature needs no schema change.

## Alternatives rejected

- **Model cards as ordinary documents.** Documents are attached evidence with no lifecycle and no number. A card needs both, and forcing it into the document engine would give it either a meaningless number or an append-only history it cannot have.
- **Hard-code one card layout in the frontend or in Go.** Directly contradicts the requirement that the card be configurable per school, and would need a release to change.
- **Put the card number on the person.** A person has one student number for life, but cards are re-issued and expire, so the number belongs to the card.
- **Store the photo in the card row.** Breaks the metadata/bytes split that the rest of the system depends on and makes a photo export a SQL problem.
