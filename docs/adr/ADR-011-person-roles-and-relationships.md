# ADR-011 — Person, roles and relationships

- **Status:** Accepted
- **Date:** 2026-09-29
- **Supersedes:** the `people.person_type` column introduced by `0001_core_init.sql`
- **Affects:** `internal/platform/people`, `internal/platform/academic`, `internal/finance/*`

## Context

Blueprint §17 requires that "a teacher who is also a parent remains one person with multiple profiles". Real schools need more than that:

- an employee can also be a guardian of a child at the same school;
- a guardian can be the emergency contact of a *different* employee;
- one guardian can have many students, each in many classes;
- roles change year to year — a child is a student in some years and an alumnus later;
- a school group spans several organisations, so a person may be related to records in more than one tenant.

`0001_core_init.sql` modelled this as a single `people.person_type` column constrained to one value of `STUDENT | GUARDIAN | EMPLOYEE | SUPPLIER_CONTACT`. That column cannot represent two simultaneous roles, so the moment a teacher is also a parent, the data becomes contradictory. This ADR replaces it.

Separately, the platform is school-year dependent: enrolment, class membership, employment and guardianship are all facts *about a year*, not about a person. Finance has its own fiscal calendar, which frequently does not align with the school year.

## Decision

### 1. A person is an identity, not a role

`people` holds who somebody is: names, date of birth, nationality, contacts, address. It is **not** year-scoped — a person exists before and after they attend.

Roles are separate records. A person has zero or more `person_roles`, and each role is scoped to an academic year.

```text
Person  (identity, organisation-scoped, not year-scoped)
  │
  ├── PersonRole (STUDENT | EMPLOYEE | GUARDIAN | SUPPLIER_CONTACT)
  │     │         scoped to (organisation, academic_year)
  │     │
  │     ├── StudentProfile    admission date, student number, status
  │     ├── EmployeeProfile   employee number, job title, hired on, payroll group
  │     └── GuardianProfile   pick-up rights, billing responsibility
  │
  └── Relationship (directed, typed, year-scoped)
```

`person_type` is removed. A person's roles are a query, never a column, so the contradictory state is unrepresentable.

### 2. Roles are year-scoped

`person_roles` carries `academic_year_id` plus `starts_on` / `ends_on`. A role may be repeated across years, which is how a student progresses and how an alumnus is represented without destroying history.

A person may hold several roles **in the same year** — that is the teacher-who-is-also-a-parent case, and it is not a special case, it is just two rows.

### 3. Relationships are one directed typed table

All human relationships are rows in `person_relationships`:

```text
from_person_id  --[type]-->  to_person_id
```

`relationship_type` carries the semantics, `relationship_role` the specific label:

| type | example role | meaning |
|---|---|---|
| `GUARDIAN_OF` | `MOTHER`, `FATHER`, `LEGAL_GUARDIAN`, `GRANDPARENT` | from is guardian of to |
| `EMERGENCY_CONTACT_OF` | `SIBLING`, `UNCLE`, `FRIEND` | from is emergency contact for to |
| `SPONSOR_OF` | `EMPLOYER`, `TRUST`, `DONOR` | from sponsors to's fees |
| `CAREGIVER_OF` | `NANNY`, `HOUSEHOLD_HELP` | from cares for to |
| `NEXT_OF_KIN_OF` | `PARENT`, `SIBLING` | from is next of kin for to |
| `SIBLING_OF`, `SPOUSE_OF`, `STEP_PARENT_OF` | — | symmetric families |

Three consequences fall out of one table:

- an employee who is a guardian is `Person(employee) --GUARDIAN_OF--> Person(student)`;
- a guardian who is an emergency contact for a colleague is `Person(guardian) --EMERGENCY_CONTACT_OF--> Person(employee)`;
- the same schema serves HR, transport, boarding and payroll later, so no domain owns "the family tree".

`is_primary` marks the billing-relevant guardian, but the billing rules themselves live in Finance, which reads the relationship rather than owning it.

A relationship may not link a person to themselves, and a symmetric type is stored once, in a canonical direction, by the application layer.

### 4. Class membership belongs to Academics, not to People

A guardian's link is to a *student*, never to a class. A student's class membership is an enrolment owned by `platform/academic`, so "many classes, many years" is expressed by enrolment rows. People holds no class columns; this is what keeps Finance from re-inventing enrolment.

### 5. Households group billing parties, not relatives

`households` and `household_members` exist so Finance has a billing party that is not a person (a couple paying for three children, or a separated family). Membership is derived from `GUARDIAN_OF` relationships at read time; the table holds only the grouping and the billing name. Households are year-scoped, because a family can be re-grouped between years.

### 6. The school year is the platform's time axis; the fiscal year is Finance's

`academic_years` is the platform time axis and every year-scoped platform record references it. Finance owns `fiscal_years` and `fiscal_periods` separately (ADR to be written with the accounting module), because a school with a September–June year books a July–June fiscal year.

Nothing in `platform/` may store a fiscal year. Finance maps between the two where a report needs both. The platform never calls the fiscal year "the year".

### 7. Cross-organisation relationships

`person_relationships` is organisation-scoped, so a school group cannot accidentally read another tenant's data through a relationship. A person who genuinely spans two organisations is two `people` rows, linked by a shared national identifier or by the sync/outbox convergence described in ADR-007. Denormalising relationships across tenants to make group reporting easy would break the tenancy rule in ADR-009, so group reporting is a query across tenants with explicit authorisation instead.

### 8. ID cards are a separate module

Configurable ID cards for students, employees and guardians are a document concern with its own template model (field selection, order, photo, expiry, campus branding, print batches). They are specified in ADR-012 and implemented as `platform/identitycards`, which reads Person data and hands a render request to the document engine. ID cards are not a column on a person or a profile.

## Consequences

- `people.person_type` is removed from `0001_core_init.sql` in both dialects. `0001` had not been released when this decision was taken, so it was corrected in place rather than by a follow-up migration; the released-migration immutability rule in `database/README.md` continues to apply from here on.
- Removing the column needed no SQLite table rebuild. A rebuild inside the migrator's transaction was rejected because `PRAGMA foreign_keys` is a silent no-op inside a transaction, so `DROP TABLE people` would have fired `user_accounts.person_id ON DELETE SET NULL` and quietly unlinked user accounts.
- `documents` is declared before `people` in both dialects, because a person references their photograph and PostgreSQL resolves foreign key targets at DDL time.
- The `AcademicYearScope` port lets People validate a year without depending on Academics' internals.
- Queries for "all employees this year" are an indexed join on `person_roles`; role history is free.
- Adding a role type later (alumnus, donor, driver) is one enum value plus one profile table, not a schema change to `people`.
- The single `person_type` column is gone, so the "which role is this person" question is always answered by a query that can return several rows.

## Alternatives rejected

- **Keep `person_type` and add a second column.** Two mutually exclusive columns cannot express three roles and still let a query answer "is this person a guardian".
- **Junction table per relationship type** (`student_guardians`, `employee_emergency_contacts`, …). Each new real-world case multiplies tables, and no domain can see the others, so the ID card and HR features would each re-implement a family view.
- **Make relationships not year-scoped.** A guardianship ends; a year-less edge either has to be deleted (losing history, which the append-only rules forbid) or needs a manual end date that disagrees with the school calendar.
- **Put enrolment on the student profile.** It would make Academics a consumer of People instead of an owner, and would put a school structure on a core identity record.
