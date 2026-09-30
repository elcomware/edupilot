# EduPilot Master Blueprint — v0.1

## 1. Vision

EduPilot will be a **modular school operating system**. Finance is the first major commercial domain, but Finance must sit on a reusable **EduPilot Core** rather than becoming the foundation of the whole system.

This allows EduPilot Finance to be built and sold now while Admissions, Academics, HR, Attendance, Library, Transport, Parent Portal, LMS, Communication and other school-management capabilities can later be added without replacing the architecture.

> **Finance is EduPilot's first commercial domain—not EduPilot's architecture.**

---

# 2. Product Architecture

```text
EDUPILOT
│
├── PLATFORM CORE
│   ├── Organisations
│   ├── Campuses
│   ├── Academic structures
│   ├── People / identities
│   ├── Students & households
│   ├── Staff
│   ├── Users / authentication
│   ├── Roles & permissions
│   ├── Workflows / approvals
│   ├── Documents
│   ├── Notifications
│   ├── Audit
│   ├── Configuration
│   ├── Reporting framework
│   ├── Import / export
│   └── Integration framework
│
├── FINANCE SUITE — BUILD NOW
│   ├── Student billing
│   ├── Fees
│   ├── Collections
│   ├── Receivables
│   ├── Sales / uniforms / books
│   ├── Cash management
│   ├── Banking
│   ├── Expenses
│   ├── Procurement
│   ├── Suppliers / payables
│   ├── Payroll
│   ├── Accounting / GL
│   ├── Budgeting
│   ├── Assets
│   └── Financial reporting
│
├── FUTURE EDUPILOT DOMAINS
│   ├── Admissions
│   ├── SIS
│   ├── Academics
│   ├── Assessment
│   ├── Attendance
│   ├── HR
│   ├── Timetabling
│   ├── Behaviour / pastoral
│   ├── Health
│   ├── Library
│   ├── Transport
│   ├── Meals
│   ├── Communications
│   ├── Parent portal
│   ├── Student portal
│   ├── Staff portal
│   └── LMS
│
└── EDUPILOT CLOUD — LATER
    ├── Multi-device sync
    ├── Mobile / web APIs
    ├── Online payments
    ├── Parent services
    ├── Consolidated group reporting
    └── Integrations
```

---

# 3. Technology Foundation

| Layer | Decision |
|---|---|
| Desktop host | **Wails 3** |
| Desktop frontend | **React + TypeScript** |
| Backend language | **Go** |
| Architecture | **Modular monolith** |
| Domain architecture | **Hexagonal / Clean Architecture** |
| Standalone database | **SQLite** |
| Server / cloud database | **PostgreSQL** |
| IDs | **UUIDv7** |
| Money | **Integer minor units + currency** |
| Financial records | **Append / reversal model** |
| Integration | **Domain events + durable outbox** |
| Multi-school | **Organisation-scoped from day one** |
| Frontend/backend bridge | **Gateway abstraction** |
| Desktop API | **Wails adapter** |
| Server API | **HTTP adapter later** |
| Files | **Storage abstraction** |
| UI | **Shared EduPilot design system** |
| Languages | **English + French from the beginning** |
| Deployment now | **Windows desktop / offline** |
| Multi-user later | **EduPilot Site Server** |
| Cloud later | **Go + PostgreSQL + object storage** |
| Repository | **Monorepo initially** |

---

# 4. Where Frontend, Backend and Desktop Host Live

For the first EduPilot desktop installation:

```text
SCHOOL WINDOWS COMPUTER
│
├── EduPilot.exe
│   ├── Wails desktop host
│   ├── Go backend
│   └── React frontend
│
└── EduPilot Data
    ├── edupilot.db
    ├── documents/
    ├── attachments/
    ├── backups/
    ├── logs/
    └── configuration/
```

## Frontend

```text
React
TypeScript
↓
Rendered inside Windows WebView2
```

## Backend

```text
Go
↓
Compiled inside EduPilot.exe
```

## Database

```text
SQLite
↓
Stored separately from the executable
```

## Desktop Host

```text
Wails
↓
Creates native window
Runs Go
Loads React
Handles OS integration
```

No external server is required for the first standalone edition.

---

# 5. Do Not Make Wails the Architecture

Wrong:

```text
React
  ↓
Wails functions
  ↓
random database code
```

Correct:

```text
React
  ↓
Frontend Gateway
  ↓
Wails Transport
  ↓
Application Services
  ↓
Domain
  ↓
Repository Interfaces
  ↓
SQLite Adapter
```

Later:

```text
React Web
  ↓
HTTP API
  ↓
SAME Application Services
  ↓
SAME Domain
  ↓
PostgreSQL
```

Wails is only one delivery adapter.

---

# 6. Backend Architecture

Use a **modular monolith** with Clean / Hexagonal Architecture principles.

```text
              PRESENTATION
                   │
          ┌────────┴────────┐
          │                 │
       Wails             HTTP API
          │                 │
          └────────┬────────┘
                   ▼
             APPLICATION
                   │
                   ▼
                DOMAIN
                   │
                   ▼
                 PORTS
                   │
          ┌────────┴────────┐
          ▼                 ▼
       SQLite           PostgreSQL
       Files             Cloud
       Printer           Payments
```

Dependencies always point inward.

The domain must not know about React, Wails, SQLite, PostgreSQL or HTTP.

---

# 7. Deployment Modes

## Mode A — Standalone Desktop

Build first.

```text
┌───────────────────────────┐
│ SCHOOL FINANCE COMPUTER   │
│                           │
│ EduPilot Desktop          │
│ ├ React                   │
│ ├ Wails                   │
│ ├ Go                      │
│ └ SQLite                  │
└───────────────────────────┘
```

Best for:

- one bursar;
- small school;
- one finance office computer;
- pilot deployments;
- unstable internet environments.

---

## Mode B — School Site Server

For multi-user schools:

```text
                SCHOOL LAN

                 ┌───────────────┐
                 │ EduPilot Site │
                 │    Server     │
                 │               │
                 │ Go backend    │
                 │ DB            │
                 └───────┬───────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
          ▼              ▼              ▼
       Cashier        Accountant      Director
       Desktop         Desktop         Desktop
```

Do not place one SQLite database file on a shared network drive for multiple computers to open directly.

Desktop clients must communicate with the EduPilot Site Server.

---

## Mode C — EduPilot Cloud

Later:

```text
              ┌────────────────────────┐
              │    EDUPILOT CLOUD      │
              │                        │
              │ Go API                 │
              │ PostgreSQL             │
              │ Object Storage         │
              │ Authentication         │
              │ Sync                   │
              └───────────┬────────────┘
                          │
             ┌────────────┼────────────┐
             │            │            │
        School A      School B      School C
```

Cloud capabilities can include:

- parent portal;
- mobile applications;
- web access;
- online payments;
- group reporting;
- notifications;
- cross-campus analytics;
- cloud backup.

---

# 8. EduPilot Node Concept

An **EduPilot Node** owns a school's local operational state.

Small school:

```text
EduPilot Node
=
inside EduPilot Desktop
```

Large school:

```text
EduPilot Node
=
school server
```

This gives EduPilot a clean migration path from standalone desktop to multi-user site deployment to cloud-connected deployment.

---

# 9. Repository Structure

Use one Git monorepository initially.

```text
edupilot/
│
├── cmd/
│   ├── desktop/
│   └── server/
│
├── internal/
│   ├── platform/
│   │   ├── organisation/
│   │   ├── campus/
│   │   ├── identity/
│   │   ├── people/
│   │   ├── academic/
│   │   ├── authorization/
│   │   ├── workflow/
│   │   ├── numbering/
│   │   ├── documents/
│   │   ├── notifications/
│   │   ├── audit/
│   │   ├── settings/
│   │   └── importexport/
│   │
│   ├── finance/
│   │   ├── catalogue/
│   │   ├── fees/
│   │   ├── billing/
│   │   ├── receivables/
│   │   ├── collections/
│   │   ├── cashier/
│   │   ├── sales/
│   │   ├── inventory/
│   │   ├── procurement/
│   │   ├── payables/
│   │   ├── expenses/
│   │   ├── payroll/
│   │   ├── banking/
│   │   ├── accounting/
│   │   ├── budgeting/
│   │   ├── assets/
│   │   └── reporting/
│   │
│   └── infrastructure/
│       ├── database/
│       ├── filesystem/
│       ├── documents/
│       ├── crypto/
│       ├── printing/
│       ├── backup/
│       ├── sync/
│       └── integrations/
│
├── frontend/
│   └── src/
│
├── database/
│   ├── migrations/
│   ├── seeds/
│   ├── factories/
│   └── fixtures/
│
├── api/
│   ├── contracts/
│   └── openapi/
│
├── docs/
│   ├── architecture/
│   ├── adr/
│   ├── domains/
│   ├── security/
│   └── deployment/
│
├── tests/
├── build/
├── scripts/
└── Taskfile.yml
```

---

# 10. Standard Backend Module Structure

Every module should follow a predictable pattern.

Example:

```text
finance/billing/
│
├── domain/
│   ├── invoice.go
│   ├── invoice_line.go
│   ├── rules.go
│   └── events.go
│
├── application/
│   ├── create_invoice.go
│   ├── issue_invoice.go
│   ├── cancel_invoice.go
│   └── queries.go
│
├── ports/
│   └── repository.go
│
├── adapters/
│   ├── sqlite/
│   └── postgres/
│
└── contracts/
```

---

# 11. Dependency Rule

```text
React
    ↓
Transport
    ↓
Application
    ↓
Domain

Infrastructure
    ↓
implements Domain/Application ports
```

Forbidden:

```text
Domain → React          ❌
Domain → Wails          ❌
Domain → SQLite         ❌
Domain → PostgreSQL     ❌
Domain → HTTP           ❌
```

---

# 12. Frontend Architecture

```text
frontend/src/
│
├── app/
│   ├── router/
│   ├── shell/
│   ├── providers/
│   └── bootstrap/
│
├── design-system/
│   ├── buttons/
│   ├── forms/
│   ├── tables/
│   ├── dialogs/
│   ├── typography/
│   ├── navigation/
│   └── tokens/
│
├── platform/
│   ├── organisations/
│   ├── campuses/
│   ├── people/
│   ├── users/
│   ├── settings/
│   └── workflows/
│
├── finance/
│   ├── dashboard/
│   ├── students/
│   ├── billing/
│   ├── collections/
│   ├── accounting/
│   ├── payroll/
│   ├── inventory/
│   ├── procurement/
│   ├── banking/
│   ├── budgeting/
│   ├── assets/
│   └── reports/
│
├── gateway/
│   ├── desktop/
│   ├── http/
│   └── types/
│
├── shared/
└── i18n/
```

---

# 13. Frontend Gateway Abstraction

React components should not directly depend on Wails bindings.

Desktop:

```text
React Component
      ↓
FinanceGateway
      ↓
DesktopFinanceGateway
      ↓
Wails generated binding
```

Future web:

```text
React Component
      ↓
FinanceGateway
      ↓
HttpFinanceGateway
      ↓
EduPilot Server
```

This allows the React UI to evolve into a web application without rebuilding the business layer.

---

# 14. EduPilot Core

Build once:

```text
Organisation
Campus
Academic Year
Academic Structure
People
Students
Households
Employees
Users
Roles
Permissions
Workflows
Approvals
Audit
Documents
Numbering
Notifications
Settings
Import/Export
Files
Search
Backups
```

Finance consumes the Core.

Finance must not own these shared concepts.

---

# 15. Organisation Model

```text
Organisation
│
├── Campuses
├── Divisions
├── Departments
├── Academic Years
├── People
├── Students
├── Employees
├── Finance
└── Configuration
```

EduPilot must support:

- nursery only;
- primary school;
- secondary school;
- K–12 school;
- multi-campus school;
- school group;
- French school;
- British school;
- Cambridge;
- IB;
- bilingual school.

---

# 16. Academic Core

Finance initially needs only the structural concepts:

```text
AcademicYear
Term
Programme
Division
Level
Grade
Class
Enrollment
```

Teaching, assessment and curriculum come later.

---

# 17. Person Model

```text
Person
│
├── StudentProfile
├── GuardianProfile
├── EmployeeProfile
├── SupplierContact
└── UserAccount
```

A teacher who is also a parent remains one person with multiple profiles.

---

# 18. Finance Suite

```text
FINANCE
│
├── Catalogue
├── Fee Structures
├── Billing
├── Accounts Receivable
├── Collections
├── Cashier
├── Student Accounts
├── Household Accounts
├── Sponsors
├── Discounts
├── Scholarships
├── Payment Plans
├── Refunds
│
├── Sales
├── Products
├── Uniforms
├── Inventory
├── Stores
│
├── Suppliers
├── Procurement
├── Accounts Payable
├── Expenses
├── Petty Cash
│
├── Payroll
├── Loans
├── Advances
│
├── Banking
├── Reconciliation
│
├── Accounting
├── General Ledger
├── Fiscal Periods
│
├── Budgets
├── Assets
│
└── Financial Reporting
```

---

# 19. Student and Household Finance

```text
Household
├── Guardians
├── Students
├── Billing contacts
├── Addresses
├── Communication preferences
└── Financial account
```

Each student has a proper financial sub-ledger:

```text
Student Account
├── Charges
├── Invoices
├── Discounts
├── Credits
├── Payments
├── Refunds
├── Adjustments
├── Allocations
├── Carry forwards
└── Balance
```

Never store only a mutable `student.balance`.

The balance must be derivable from transactions.

---

# 20. Fee Catalogue

```text
ACADEMIC FEES
├── Application
├── Registration
├── Re-registration
├── Tuition
├── Development levy
├── Examination
├── Graduation
└── Certification

SERVICES
├── Transport
├── Meals
├── Boarding
├── Daycare
├── Clubs
├── Trips
└── Activities

PRODUCTS
├── Uniform
├── PE kit
├── Books
├── Stationery
├── ID cards
└── Bags

OTHER
└── Configurable items
```

Each item can define:

```text
Revenue account
Tax treatment
Cost centre
Refundability
Required / optional
Recurring / non-recurring
Inventory-backed yes/no
```

---

# 21. Fee Structures

Fee structures can depend on:

- campus;
- academic year;
- programme;
- grade;
- residency;
- boarding/day;
- transport zone;
- student category;
- scholarship status;
- custom groups.

Fee structures must be effective-dated and versioned.

Never overwrite previous-year fees.

---

# 22. Billing Plans

Support:

```text
Annual
Termly
Semester
Monthly
Custom instalments
Percentage instalments
Fixed instalments
Milestone-based billing
Individual payment plan
```

---

# 23. Discounts, Scholarships and Waivers

Support:

```text
Sibling discount
Scholarship
Staff-child discount
Early-payment discount
Promotional discount
Financial aid
Sponsor support
Negotiated discount
Manual waiver
Full scholarship
Percentage waiver
Fixed-value waiver
```

Every adjustment needs:

```text
Reason
Authority
Approver
Amount
Supporting document
Audit history
```

No silent fee editing.

---

# 24. Sponsorship

Support multiple payers for one student:

```text
Parent       40%
Employer     30%
Scholarship  30%
```

Third-party sponsors should have proper accounts.

---

# 25. Invoicing

Statuses:

```text
Draft
Approved
Issued
Partially paid
Paid
Overdue
Cancelled
Credited
```

Documents:

```text
Invoice
Proforma invoice
Debit note
Credit note
Account statement
Fee schedule
Balance confirmation
```

---

# 26. Receipting

Payment methods:

```text
Cash
Bank transfer
Cheque
Mobile money
Card
Online payment
Sponsor
Payroll deduction
Credit balance
```

Flow:

```text
Payment received
      ↓
Payment recorded
      ↓
Receipt generated
      ↓
Payment allocated
      ↓
Accounting entry posted
      ↓
Cash/bank position updated
```

---

# 27. Payment Allocation

One payment may cover multiple charges.

Support:

- automatic allocation;
- oldest first;
- selected invoice;
- selected charge;
- manual allocation;
- unallocated credit.

---

# 28. Cashier and Cash Desk

```text
Cash Desk
├── Opening balance
├── Receipts
├── Refunds
├── Cash expenses
├── Deposits
├── Cash transfers
└── Closing
```

Daily close:

```text
Expected cash
Actual cash
Difference
Reason
Supervisor approval
```

---

# 29. Arrears and Collections

```text
Total billed
Total collected
Outstanding
Current
1–30 days
31–60
61–90
90+
```

Per student:

```text
Balance
Due dates
Promises to pay
Payment plan
Collection notes
Contacts
Last reminder
```

---

# 30. Statements

Generate:

- student statement;
- household statement;
- sponsor statement;
- class debt report;
- grade debt report;
- aged receivables;
- payment history;
- fee reconciliation.

---

# 31. Refunds and Reversals

Refund workflow:

```text
Refund requested
      ↓
Reason recorded
      ↓
Supporting evidence
      ↓
Finance review
      ↓
Approval
      ↓
Payment
      ↓
Accounting reversal/posting
```

Never directly delete posted transactions.

Corrections use:

```text
Original transaction
+
Reversal
+
Replacement transaction
```

---

# 32. School Shop / Uniforms

```text
Products
├── Uniform
│   ├── Shirt
│   ├── Skirt
│   ├── Trousers
│   └── Blazer
├── PE
├── Books
├── Stationery
└── Accessories
```

Variants:

```text
Item
├── Size
├── Colour
├── Style
├── Brand
└── Academic division
```

---

# 33. Inventory

Inventory movements:

```text
Opening stock
Purchase
Receipt
Sale
Issue
Transfer
Return
Adjustment
Damage
Loss
Write-off
Stock count
```

Stock quantity must derive from movement history.

---

# 34. Stores

Support:

```text
Central store
Primary shop
Secondary shop
Uniform store
Library stock
Kitchen
```

and transfers between stores.

---

# 35. Point of Sale

```text
Search item
Select variant
Select student/customer
Sell
Receive payment
Print receipt
Reduce stock
Post revenue
```

---

# 36. Suppliers and Procurement

Supplier record:

```text
Vendor
Contacts
Payment details
Tax details
Products/services
Purchase history
Outstanding balance
Documents
```

Procurement flow:

```text
Purchase Request
      ↓
Approval
      ↓
RFQ (optional)
      ↓
Purchase Order
      ↓
Goods Received
      ↓
Supplier Invoice
      ↓
Payment
      ↓
Accounting
```

---

# 37. Accounts Payable

```text
Supplier Invoice
Credit Note
Payment
Advance
Allocation
Balance
Ageing
```

---

# 38. Expenses and Petty Cash

Expenses may originate from:

- supplier invoices;
- petty cash;
- staff reimbursement;
- bank payment;
- recurring expense;
- expense claim.

Classification:

```text
Account
Department
Campus
Cost centre
Project
Funding source
```

Petty cash:

```text
Fund
Opening
Disbursement
Voucher
Supporting receipt
Replenishment
Closing
Reconciliation
```

---

# 39. Payroll

```text
Employee
Contract
Compensation
Payroll group
Pay calendar
Earnings
Deductions
Benefits
Loans
Advances
Payroll run
Payslip
Payment
Accounting
```

---

# 40. Salary Structures

Reusable salary structures may include:

```text
Basic
Housing
Transport
Responsibility
Teaching hours
Overtime
Bonus
Other allowance
```

Deductions:

```text
Tax
Social security
Pension
Insurance
Loan repayment
Salary advance
Absence
Other
```

Country-specific payroll logic must be implemented through configurable rule packs rather than hard-coded into the core.

---

# 41. Payroll Runs

```text
Prepare
Calculate
Validate
Review
Approve
Lock
Post
Pay
Issue payslips
```

Posted payroll should be immutable except through controlled reversal or amendment workflows.

---

# 42. Payslips

Professional payslip structure:

```text
School
Employee
Period

EARNINGS
Basic
Allowances
Overtime

DEDUCTIONS
Taxes
Social
Loans

Gross
Total deductions
Net pay

Payment account
Employer contributions
Year-to-date values
```

Support PDF generation, batch printing and secure export.

---

# 43. Employee Loans and Salary Advances

Track:

```text
Principal
Interest if applicable
Disbursement
Repayment schedule
Payroll deduction
Balance
Settlement
```

---

# 44. Banking

Bank accounts:

```text
Operating
Tuition
Payroll
Savings
Petty cash
Mobile money wallet
```

Transactions:

```text
Deposit
Withdrawal
Transfer
Bank charges
Interest
Payment
Receipt
```

---

# 45. Bank Reconciliation

```text
Bank statement
vs
EduPilot ledger
```

Statuses:

```text
Matched
Suggested match
Unmatched
Excluded
Reconciled
```

Support CSV/XLSX statement imports later.

---

# 46. Accounting Core

EduPilot Finance must use real double-entry accounting.

Example chart:

```text
1000 Assets
  1100 Cash
  1200 Banks
  1300 Receivables
  1400 Inventory

2000 Liabilities

3000 Equity / Funds

4000 Revenue
  4100 Tuition
  4200 Registration
  4300 Exams
  4400 Transport
  4500 Uniform sales

5000 Expenses
  5100 Salaries
  5200 Utilities
  ...
```

---

# 47. General Ledger

Every posted financial event creates:

```text
Journal Entry
├── Header
├── Date
├── Reference
├── Source
├── Description
└── Lines
    ├── Debit
    └── Credit
```

Fundamental invariant:

> **Debits must always equal credits.**

---

# 48. Accounting Integration

```text
                 FINANCIAL EVENT
                        │
              ┌─────────┴─────────┐
              │                   │
           Business            Accounting
           operation             posting
              │                   │
              └─────────┬─────────┘
                        ▼
                 GENERAL LEDGER
```

Example:

```text
Student invoice issued

Dr Accounts Receivable
Cr Tuition Revenue
```

Payment:

```text
Dr Bank
Cr Accounts Receivable
```

Payroll:

```text
Dr Salary Expense
Cr Payroll Liability
```

Uniform sale:

```text
Dr Cash
Cr Uniform Revenue

Dr Cost of Goods Sold
Cr Inventory
```

---

# 49. Accounting Periods and Journals

Fiscal year:

```text
Fiscal Year
├── Period 1
├── Period 2
...
└── Closing period
```

States:

```text
Open
Soft close
Hard close
Reopen with permission
```

Journals:

```text
Sales journal
Cash journal
Bank journal
Purchases journal
Payroll journal
Inventory journal
General journal
```

---

# 50. Cost Centres and Funds

Examples:

```text
Primary
Secondary
Nursery
Administration
Transport
Kitchen
Boarding
ICT
Sports
```

Funds/projects:

```text
Government grants
Donor funding
Construction projects
Scholarships
Clubs
Restricted funds
Capital campaigns
```

---

# 51. Budgeting

```text
Annual budget
Department budget
Monthly distribution
Revision/version
Budget transfers
Approvals
```

Reports:

```text
Budget
Actual
Variance
Variance %
Forecast
```

---

# 52. Fixed Assets

Register:

```text
Buildings
Vehicles
Computers
Projectors
Furniture
Generators
Equipment
```

Fields:

```text
Purchase
Location
Custodian
Serial number
Useful life
Depreciation
Current value
Disposal
```

---

# 53. Financial Statements

Native reports:

```text
Trial Balance
General Ledger
Income Statement
Balance Sheet
Cash Flow
Receivables
Payables
Revenue analysis
Expense analysis
Budget vs Actual
Payroll summaries
Bank reconciliation
```

---

# 54. Dashboards

Executive dashboard:

```text
Enrollment
Total expected fees
Total billed
Collections
Collection %
Outstanding
Overdue
Payroll
Operating expenses
Net cash
Bank balances
```

Drilldown:

```text
Organisation
→ Campus
→ Division
→ Grade
→ Class
→ Student
```

Role-specific dashboards:

## Cashier

```text
Today's receipts
Cash drawer
Recent payments
Unallocated payments
Close session
```

## Accountant

```text
Bank balance
Reconciliation
AR/AP
Journals
Period status
```

## Finance Director

```text
Collections
Outstanding
Payroll
Expenses
Cash
Budget
Revenue
```

## Director

```text
Executive KPIs
Campus comparisons
Revenue
Collection
Cost
Payroll ratio
Cash position
```

---

# 55. Role-Based Access Control

Example roles:

```text
Super Administrator
Organisation Owner
Director
Finance Director
Accountant
Bursar
Cashier
Payroll Officer
HR
Storekeeper
Procurement Officer
Auditor
Read-only Executive
Teacher
Parent
```

Granular permissions:

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

Permissions must include scope and business policy, not only role names.

---

# 56. Maker-Checker Controls

Critical operations may require two people.

```text
User A
creates transaction

User B
approves transaction
```

Examples:

- refunds;
- large discounts;
- supplier payments;
- payroll;
- manual journals;
- budget transfers.

---

# 57. Audit

Every audit record should capture:

```text
Who
What
When
Organisation
Campus
Device/session
Entity
Entity ID
Operation
Before
After
Reason
Correlation ID
```

Financial audit should be append-only.

---

# 58. Workflow Engine

Reusable workflow model:

```text
Workflow
├── Trigger
├── Conditions
├── Steps
├── Approvers
├── Thresholds
└── Outcome
```

Example:

```text
Expense < 100,000
→ Finance Manager

Expense ≥ 100,000
→ Finance Manager
→ Director
```

The same engine can later serve HR, Admissions and other domains.

---

# 59. Documents

Attach evidence to:

```text
Payment
Expense
Supplier
Purchase order
Employee
Payroll adjustment
Refund
Scholarship
Student account
Asset
```

Store metadata in the database and file bytes in a filesystem/object-storage abstraction.

---

# 60. Document Engine

One reusable EduPilot document engine should generate:

```text
Invoice
Receipt
Credit Note
Statement
Fee Schedule
Payment Plan
Purchase Order
Goods Receipt
Expense Voucher
Payment Voucher
Payslip
Payroll Summary
Financial Statements
Audit Reports
```

Architecture:

```text
Document Data
+
Document Template
+
Organisation Branding
+
Renderer
```

---

# 61. Numbering Engine

Configurable sequences:

```text
INV/2026/000001
REC/2026/000001
PAY/2026/000001
PO/2026/000001
JV/2026/000001
```

Example template:

```text
{CAMPUS}/{TYPE}/{YEAR}/{SEQ:6}
```

---

# 62. Search

Global search should index:

```text
Students
Guardians
Staff
Invoice
Receipt
Payment
Supplier
Purchase Order
Asset
Payroll
Journal
```

Modules register searchable entities.

---

# 63. Import Framework

Reusable pipeline:

```text
CSV/XLSX
    ↓
Upload
    ↓
Column mapping
    ↓
Validation
    ↓
Preview
    ↓
Import
    ↓
Results
```

Initial imports:

- students;
- guardians;
- opening balances;
- fees;
- products;
- employees;
- salary structures;
- suppliers;
- chart of accounts;
- bank statements.

---

# 64. Export Framework

Support:

```text
PDF
XLSX
CSV
```

with permissions on sensitive exports.

---

# 65. Backup Architecture

Backup contents:

```text
Database
+
Documents
+
Configuration
+
Backup Manifest
+
Checksums
```

Flow:

```text
Prepare
↓
Consistent database backup
↓
Collect files
↓
Generate manifest
↓
Checksums
↓
Encrypt
↓
Store
↓
Verify
```

Before major migrations:

```text
automatic pre-migration backup
```

---

# 66. Restore

```text
Select backup
↓
Validate checksum
↓
Validate version
↓
Validate organisation
↓
Create safety backup
↓
Restore
↓
Run database integrity validation
↓
Start application
```

---

# 67. Offline-First Architecture

```text
React
  ↓
Go application services
  ↓
Domain
  ↓
Repositories
  ↓
SQLite
```

Local transaction:

```text
Local transaction
      ↓
SQLite commit
      ↓
Outbox event
      ↓
Sync later
```

---

# 68. Durable Outbox

Build now:

```text
outbox_events
├── id
├── organisation_id
├── site_id
├── event_type
├── aggregate_type
├── aggregate_id
├── payload
├── created_at
├── processed_at
└── attempt_count
```

At first it can power local internal processes.

Later it becomes the foundation for EduPilot Cloud synchronization.

---

# 69. Domain Events

Examples:

```text
StudentEnrolled
InvoiceIssued
InvoiceCredited
PaymentReceived
PaymentReversed
RefundApproved
StockReceived
StockSold
PayrollPosted
JournalPosted
PeriodClosed
```

---

# 70. Do Not Use Full Event Sourcing

Use:

```text
Relational database
+
immutable financial transactions
+
domain events
+
audit log
```

Do not reconstruct the entire system from event streams.

---

# 71. Database Strategy

Standalone:

```text
SQLite
```

Future server/cloud:

```text
PostgreSQL
```

Site server may use SQLite for smaller deployments where only the server owns the file, or PostgreSQL for larger deployments.

Frontend never accesses SQL directly.

```text
React
↓
Application command
↓
Service
↓
Repository
↓
Database
```

---

# 72. Database Folder

```text
database/
├── migrations/
│   ├── sqlite/
│   └── postgres/
├── seeds/
│   ├── core/
│   ├── finance/
│   └── demo/
├── factories/
└── fixtures/
```

Released migrations are immutable. Add new migrations rather than editing old ones.

---

# 73. Production Data Location

Application and school data must remain separate.

```text
EduPilot Application
        ≠
EduPilot Data
```

Conceptual production data root:

```text
EduPilot/
├── database/
│   └── edupilot.db
├── files/
├── backups/
├── logs/
└── runtime/
```

The operating-system-specific path must be resolved through a platform storage provider.

---

# 74. Money Representation

Never use floating-point values for money.

```text
Money {
    amount_minor: int64
    currency: XOF
}
```

Example:

```text
250000 XOF

amount_minor = 250000
currency = XOF
```

Currencies with subunits use their defined scaling.

Exchange rates should use exact decimal/rational representation.

---

# 75. IDs

Use globally unique sortable IDs such as UUIDv7.

Do not use auto-increment database IDs as business identifiers.

This prepares EduPilot for:

```text
School A
School B
Offline Node C
Cloud
```

creating objects independently.

---

# 76. Tenant Boundaries

Relevant records carry:

```text
organisation_id
```

and where appropriate:

```text
campus_id
site_id
```

Enforce organisation boundaries in repositories, permissions, services and tests.

Never rely only on frontend filtering.

---

# 77. Financial Immutability

Once posted:

```text
DELETE      ❌
EDIT        ❌
```

Instead use:

```text
Reverse
Credit
Void
Correct
Repost
```

Financial history must remain traceable.

---

# 78. Core Financial Invariants

Automated tests must enforce:

```text
Debits = Credits.

Posted journals cannot be edited.

A closed accounting period cannot receive postings.

Receipt numbers are unique within their sequence.

Invoice numbers are unique.

Posted payroll runs are immutable.

Payment allocation cannot exceed payment value.

Payment allocation cannot exceed permitted invoice balance.

Stock movements determine stock quantity.

Every reversal references its original transaction.

Every posted transaction identifies its actor.

Every sensitive change produces an audit record.
```

A production release should fail if these invariants fail.

---

# 79. Internationalisation

Build English and French from day one.

Do not hard-code user-facing text directly inside components.

Use an i18n key system such as:

```text
payments.actions.pay
```

Academic terminology must be configurable:

```text
Grade
Class
Year
Niveau
Classe
Form
```

depending on context.

---

# 80. Regionalisation

Support:

```text
Language
Currency
Number formatting
Date formatting
Academic terminology
Tax rules
Payroll rules
```

Country-specific tax/payroll logic should live in configurable rule packs rather than the core.

---

# 81. Configuration Hierarchy

```text
System defaults
    ↓
Organisation
    ↓
Campus
    ↓
Module
    ↓
User
```

Avoid duplicated settings systems across modules.

---

# 82. Desktop Navigation

```text
DASHBOARD

PEOPLE
  Students
  Households

BILLING
  Fee Catalogue
  Fee Structures
  Billing
  Invoices
  Discounts
  Scholarships
  Payment Plans

COLLECTIONS
  Payments
  Receipts
  Cash Desks
  Arrears
  Statements

SALES & STOCK
  Products
  Sales
  Inventory
  Stores

PURCHASING
  Suppliers
  Purchase Requests
  Purchase Orders
  Supplier Invoices

EXPENSES
  Expenses
  Claims
  Petty Cash

PAYROLL
  Employees
  Contracts
  Salary Structures
  Payroll Runs
  Payslips
  Loans
  Advances

BANKING
  Bank Accounts
  Transactions
  Reconciliation

ACCOUNTING
  Chart of Accounts
  Journals
  General Ledger
  Fiscal Periods

BUDGETS

ASSETS

REPORTS

ADMINISTRATION
  Organisation
  Campuses
  Academic Years
  Users
  Roles
  Permissions
  Workflows
  Numbering
  Documents
  Imports
  Backups
  Audit
  Settings
```

---

# 83. Design System

Build once:

```text
Typography
Colours
Spacing
Grid
Inputs
Tables
Forms
Modals
Drawers
Navigation
Badges
Status indicators
Charts
Financial number treatment
Print styles
Dark/light themes if desired
```

Finance should look mature, calm, professional and information-dense.

---

# 84. Data Table System

One strong reusable table system should support:

- sorting;
- filtering;
- search;
- saved views;
- column chooser;
- resizing;
- pinning;
- grouping;
- totals;
- pagination;
- bulk selection;
- export;
- keyboard navigation.

Used by:

```text
Students
Invoices
Payments
Products
Suppliers
Employees
Payroll
Transactions
Journals
```

---

# 85. Form Architecture

Financial forms should support:

```text
Validation
Keyboard navigation
Unsaved-state warning
Draft state
Permissions
Audit
Errors
Loading
Safe commit handling
```

Critical financial operations should wait for backend confirmation before showing a posted state.

---

# 86. Structured Errors

Use domain errors such as:

```text
INSUFFICIENT_PERMISSION
PERIOD_CLOSED
INVOICE_ALREADY_POSTED
PAYMENT_ALREADY_REVERSED
INVALID_ALLOCATION
UNBALANCED_JOURNAL
STOCK_INSUFFICIENT
PAYROLL_LOCKED
```

Avoid random backend error strings.

---

# 87. Sync Architecture

Outbound:

```text
LOCAL TRANSACTION
       ↓
SQLite transaction
       ↓
Domain event
       ↓
Outbox
       ↓
Sync worker
       ↓
Cloud API
       ↓
PostgreSQL
```

Inbound:

```text
Cloud
↓
Sync
↓
Inbox
↓
Idempotency check
↓
Application service
↓
Local database
```

---

# 88. Conflict Rules

Finance must not use naïve “last write wins.”

Posted financial records are immutable.

For editable master data, use version information such as:

```text
version
updated_at
origin_site
```

with controlled conflict resolution.

---

# 89. Multi-Device Business Numbering

Internal ID:

```text
UUIDv7
```

Business-visible number:

```text
REC/LOME/2026/004521
```

The two are different.

Later, numbering can support:

- site-specific sequences;
- cashier sequences;
- preallocated number ranges;
- centralized sequences.

---

# 90. Cloud Backend

Future cloud architecture:

```text
React Web
      ↓
HTTPS API
      ↓
Go application
      ↓
EduPilot Domain
      ↓
PostgreSQL
```

Deployment:

```text
Linux
Container
Reverse proxy / load balancer
TLS
Go application
PostgreSQL
Object storage
Backup system
```

---

# 91. Module Rules

Every new module must provide:

```text
Domain
Application Commands
Application Queries
Permissions
Repository Port
Database Migration
Events
Audit Rules
Frontend Feature
Transport Contract
Tests
Documentation
```

---

# 92. Adding Future Modules

## Library

```text
internal/library/
├── catalogue/
├── circulation/
├── members/
└── reporting/

frontend/src/library/
database/migrations/...
```

Library uses Core People, Students, Academic Structure, Permissions, Audit and Notifications.

It does not create its own student table.

## Admissions

```text
Admissions
├── Applicants
├── Applications
├── Documents
├── Decisions
└── Enrollment
```

Flow:

```text
Admissions
   ↓
Person Core
   ↓
Student Profile
   ↓
Enrollment
   ↓
Finance event
   ↓
Registration Invoice
```

## HR

HR owns:

```text
Employee
Contract
Leave
Performance
Attendance
Personnel Documents
```

Payroll consumes employee/contract/compensation data through defined contracts.

Payroll does not become HR.

## Academics

Academics owns:

```text
Curriculum
Courses
Subjects
Teaching
Assessment
Grades
Reports
```

Academics reuses Students, People and Academic Structure from Core.

---

# 93. Testing Architecture

```text
Unit
  Domain rules

Repository
  SQLite integration

Application
  Commands / queries

Financial invariants
  Debit = Credit etc.

Migration
  old database → new database

Feature
  complete workflows

Frontend
  component / interaction

E2E
  Wails desktop

Backup
  backup → restore → verify

Sync
  offline → reconnect → reconcile

Security
  permission boundaries
```

---

# 94. Build Pipeline

```text
Lint
↓
Unit tests
↓
Backend integration tests
↓
Frontend tests
↓
Migration tests
↓
Financial invariant tests
↓
Production frontend build
↓
Go build
↓
Wails build
↓
E2E
↓
Installer
↓
Smoke test
↓
Release
```

Initial commercial target:

```text
Windows installer
```

Later:

```text
macOS
Linux
```

---

# 95. Versioning

Use separate versions:

```text
EduPilot Application
v1.4.0

Database Schema
v37

Sync Protocol
v3
```

Do not treat them as one version number.

---

# 96. Architecture Decision Records

Keep major decisions in:

```text
docs/adr/
```

Examples:

```text
ADR-001 Wails desktop host
ADR-002 Modular monolith
ADR-003 SQLite standalone
ADR-004 UUIDv7 identifiers
ADR-005 Integer money representation
ADR-006 Append-only posted transactions
ADR-007 Outbox pattern
ADR-008 Frontend gateway abstraction
ADR-009 Organisation tenancy
ADR-010 Go cloud backend
```

---

# 97. What We Explicitly Do Not Do

```text
❌ Finance logic inside React

❌ SQL inside React

❌ Direct SQLite access from multiple network PCs

❌ Generic "utils" dumping ground

❌ Finance owning students

❌ Academics owning students

❌ Separate person records per module

❌ Floating-point money

❌ Editing posted journals

❌ Deleting receipts

❌ Hard-coded fee categories

❌ Hard-coded country tax logic in core payroll

❌ Hard-coded Grade 1 / Form 1 / CP structures

❌ Module-specific user systems

❌ Module-specific audit systems

❌ Module-specific file managers

❌ Module-specific workflow engines

❌ Premature microservices
```

---

# 98. Development Phases

## Phase 0 — EduPilot Foundation

Build:

```text
Repository
Wails desktop bootstrap
React shell
Go application bootstrap
SQLite
Migration engine
Logging
Configuration
Error model
UUIDs
Money
Organisation
Campus
Academic year
People
Users
Authentication
Roles / permissions
Audit
Events
Outbox
Files
Document architecture
Backup framework
Design system
Testing framework
```

This proves the architecture.

---

## Phase 1 — Student Finance

First complete commercial vertical slice:

```text
Student
Household
Fee catalogue
Fee structure
Invoice
Payment
Receipt
Student statement
Arrears
Cash desk
Audit
PDF
Backup
```

Complete flow:

```text
Create Student
↓
Enroll
↓
Assign Fees
↓
Generate Invoice
↓
Receive Payment
↓
Issue Receipt
↓
Allocate Payment
↓
Post Accounting
↓
Print Statement
↓
View Dashboard
```

---

## Phase 2 — Accounting Core

```text
Chart of Accounts
Journals
General Ledger
Fiscal periods
Automatic posting rules
Trial Balance
Income Statement
Balance Sheet
Cash Flow
```

---

## Phase 3 — Complete Receivables

```text
Scholarships
Discounts
Sponsors
Credits
Refunds
Payment plans
Advanced arrears
Reminders
Household billing
```

---

## Phase 4 — Sales & Inventory

```text
Products
Uniforms
Sizes
Variants
Stores
Stock
POS
Sales
Returns
Stock count
Stock transfers
```

---

## Phase 5 — Procurement & Payables

```text
Suppliers
Requisitions
Approvals
Purchase Orders
Receiving
Supplier Invoices
Payables
Supplier Payments
```

---

## Phase 6 — Expenses

```text
Expenses
Claims
Petty cash
Recurring expenses
Vouchers
Approvals
```

---

## Phase 7 — Payroll

```text
Employees
Contracts
Salary structures
Earnings
Allowances
Deductions
Taxes / rules
Benefits
Loans
Advances
Payroll runs
Approvals
Payments
Payslips
GL posting
```

---

## Phase 8 — Banking

```text
Banks
Accounts
Transfers
Bank statement imports
Reconciliation
Mobile money
```

---

## Phase 9 — Budgeting & Assets

```text
Budgeting
Forecasts
Budget revisions
Cost centres
Funds
Projects

Fixed assets
Depreciation
Transfers
Disposals
```

---

## Phase 10 — Executive Reporting

```text
Collections
Outstanding debt
Revenue
Expenses
Payroll ratio
Cash
Budget variance
Cost/student
Campus comparisons
Financial statements
Audit reports
```

---

## Phase 11 — EduPilot Site Server

```text
Standalone Mode

      becomes

Desktop Clients
      ↓
EduPilot Site Server
      ↓
School Database
```

---

## Phase 12 — Cloud Foundation

```text
Cloud API
PostgreSQL adapter
Sync engine
Inbox / outbox
Device registration
Site registration
Cloud authentication
Object storage
Central reporting
```

---

## Phase 13 — Full EduPilot Ecosystem

```text
Admissions
SIS
HR
Attendance
Timetable
Academics
Assessment
Pastoral
Communication
Health
Library
Transport
Meals
LMS
Parent portal
Student portal
Mobile
```

All sharing **EduPilot Core**.

---

# 99. First Commercial Release Boundary

EduPilot Finance 1.0 should include at minimum:

1. EduPilot Core.
2. Students and households.
3. Fee catalogue.
4. Fee structures.
5. Billing and invoices.
6. Scholarships and discounts.
7. Payments and allocations.
8. Receipts.
9. Statements.
10. Arrears.
11. Cash desk.
12. Chart of accounts.
13. Double-entry GL.
14. Bank/cash accounts.
15. Expenses.
16. Basic payroll + mature payslips.
17. Audit.
18. Roles and permissions.
19. PDFs, printing and export.
20. Backup and restore.
21. Management and financial reports.

---

# 100. First Implementation Milestone

Do **not** begin with the Finance dashboard.

Start with:

```text
EDUPILOT FOUNDATION 0.1

✓ repository architecture
✓ Wails desktop bootstrap
✓ React shell
✓ Go application bootstrap
✓ SQLite connection
✓ migration engine
✓ organisation
✓ campus
✓ academic year
✓ person
✓ user
✓ authentication
✓ roles / permissions
✓ audit
✓ UUIDs
✓ Money
✓ application error model
✓ domain events
✓ durable outbox
✓ file storage
✓ configuration
✓ backup skeleton
✓ design tokens
✓ navigation shell
✓ test infrastructure
```

Then build the first finance vertical slice:

```text
Student
→ Fee
→ Invoice
→ Payment
→ Receipt
→ Ledger
→ Statement
```

This proves the complete path:

```text
React
→ Wails
→ Go Application
→ Finance Domain
→ SQLite
→ Accounting
→ Document Output
→ Backup
```

---

# 101. Governing Architectural Principle

Place this at the top of `ARCHITECTURE.md`:

> **EduPilot Core owns institutional concepts. Each business domain owns its own business rules. Interfaces own communication between domains. Infrastructure is replaceable. User interfaces are clients of the application, never owners of business truth.**

And below it:

> **Finance is EduPilot's first commercial domain—not EduPilot's architecture.**

This is the foundation upon which the entire EduPilot ecosystem should be built.
