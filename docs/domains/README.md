# Domains

Business documentation for each EduPilot domain. Finance is documented first; the other domains
follow the same template as they are built.

## Finance Suite — `internal/finance`

| Module | Owns |
|---|---|
| `catalogue` | Fee items and products, with revenue account, tax treatment, cost centre, refundability |
| `fees` | Effective-dated, versioned fee structures per campus, year, programme, grade, residency |
| `billing` | Billing plans, invoice lifecycle, invoice lines, credit and debit notes |
| `receivables` | Student and household sub-ledgers, ageing, promises to pay, collection notes |
| `collections` | Payment receipting, allocation across charges, unallocated credit, reversal |
| `cashier` | Cash desk sessions, opening balance, receipts, refunds, deposits, daily close |
| `sales` | Point of sale, sales returns, revenue recognition for the school shop |
| `inventory` | Products, variants, stock movements, stock counts, stores |
| `procurement` | Suppliers, purchase requests, RFQ, purchase orders, goods receipt |
| `payables` | Supplier invoices, advances, payment allocation, balances, AP ageing |
| `expenses` | Expenses, claims, recurring expenses, petty cash funds, vouchers |
| `payroll` | Payroll groups, pay calendars, runs, earnings, deductions, loans, advances, payslips |
| `banking` | Bank and cash accounts, transfers, statements, reconciliation |
| `accounting` | Chart of accounts, journals, general ledger, fiscal periods, posting rules |
| `budgeting` | Budgets, distribution, revisions, transfers, variance |
| `assets` | Fixed asset register, depreciation, transfers, disposals |
| `reporting` | Financial statements, ageing, statements, collections and payroll summaries |

Finance does not own students, households, employees, campuses, users, roles, permissions,
workflows, documents, numbering, notifications, audit or settings. It consumes all of them from
EduPilot Core.

## Future domains

Each of these consumes EduPilot Core and adds only its own business rules:

```text
Admissions   Applicants, applications, decisions, enrolment
SIS          Enrolment, promotion, transfer, graduation
Academics    Curriculum, courses, subjects, teaching, assessment, grades
HR           Employee, contract, leave, performance, attendance, personnel documents
Library      Catalogue, circulation, members, reporting
Transport    Routes, vehicles, passengers, billing
```

None of them creates its own person or student table.

## Template for a domain document

1. Purpose and scope.
2. Aggregate roots and entities.
3. Business rules, expressed as invariants.
4. Application commands and queries.
5. Permissions required.
6. Domain events published and consumed.
7. Documents generated.
8. Accounting impact, if any.
9. Open questions.
