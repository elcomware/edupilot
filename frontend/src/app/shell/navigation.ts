import type { NavSection } from './types'

/**
 * Desktop navigation, mirroring the EduPilot information architecture. Labels
 * are i18n keys so that academic and financial terminology stays configurable.
 */
export const NAVIGATION: readonly NavSection[] = [
  {
    key: 'nav.dashboard',
    path: '/dashboard',
    items: [],
  },
  {
    key: 'nav.people',
    path: '/people',
    items: [
      { key: 'nav.people.all', path: '/people' },
      { key: 'nav.people.students', path: '/people/students' },
      { key: 'nav.people.households', path: '/people/households' },
    ],
  },
  {
    key: 'nav.billing',
    path: '/billing',
    items: [
      { key: 'nav.billing.feeCatalogue', path: '/billing/fee-catalogue' },
      { key: 'nav.billing.feeStructures', path: '/billing/fee-structures' },
      { key: 'nav.billing.invoices', path: '/billing/invoices' },
      { key: 'nav.billing.discounts', path: '/billing/discounts' },
      { key: 'nav.billing.scholarships', path: '/billing/scholarships' },
      { key: 'nav.billing.paymentPlans', path: '/billing/payment-plans' },
    ],
  },
  {
    key: 'nav.collections',
    path: '/collections',
    items: [
      { key: 'nav.collections.payments', path: '/collections/payments' },
      { key: 'nav.collections.receipts', path: '/collections/receipts' },
      { key: 'nav.collections.cashDesks', path: '/collections/cash-desks' },
      { key: 'nav.collections.arrears', path: '/collections/arrears' },
      { key: 'nav.collections.statements', path: '/collections/statements' },
    ],
  },
  {
    key: 'nav.salesAndStock',
    path: '/sales',
    items: [
      { key: 'nav.salesAndStock.products', path: '/sales/products' },
      { key: 'nav.salesAndStock.sales', path: '/sales/sales' },
      { key: 'nav.salesAndStock.inventory', path: '/sales/inventory' },
      { key: 'nav.salesAndStock.stores', path: '/sales/stores' },
    ],
  },
  {
    key: 'nav.purchasing',
    path: '/purchasing',
    items: [
      { key: 'nav.purchasing.suppliers', path: '/purchasing/suppliers' },
      { key: 'nav.purchasing.purchaseRequests', path: '/purchasing/requests' },
      { key: 'nav.purchasing.purchaseOrders', path: '/purchasing/orders' },
      { key: 'nav.purchasing.supplierInvoices', path: '/purchasing/supplier-invoices' },
    ],
  },
  {
    key: 'nav.expenses',
    path: '/expenses',
    items: [
      { key: 'nav.expenses.claims', path: '/expenses/claims' },
      { key: 'nav.expenses.pettyCash', path: '/expenses/petty-cash' },
    ],
  },
  {
    key: 'nav.payroll',
    path: '/payroll',
    items: [
      { key: 'nav.payroll.employees', path: '/payroll/employees' },
      { key: 'nav.payroll.contracts', path: '/payroll/contracts' },
      { key: 'nav.payroll.salaryStructures', path: '/payroll/salary-structures' },
      { key: 'nav.payroll.payrollRuns', path: '/payroll/runs' },
      { key: 'nav.payroll.payslips', path: '/payroll/payslips' },
      { key: 'nav.payroll.loans', path: '/payroll/loans' },
      { key: 'nav.payroll.advances', path: '/payroll/advances' },
    ],
  },
  {
    key: 'nav.banking',
    path: '/banking',
    items: [
      { key: 'nav.banking.accounts', path: '/banking/accounts' },
      { key: 'nav.banking.transactions', path: '/banking/transactions' },
      { key: 'nav.banking.reconciliation', path: '/banking/reconciliation' },
    ],
  },
  {
    key: 'nav.accounting',
    path: '/accounting',
    items: [
      { key: 'nav.accounting.chartOfAccounts', path: '/accounting/chart-of-accounts' },
      { key: 'nav.accounting.journals', path: '/accounting/journals' },
      { key: 'nav.accounting.generalLedger', path: '/accounting/general-ledger' },
      { key: 'nav.accounting.fiscalPeriods', path: '/accounting/periods' },
    ],
  },
  { key: 'nav.budgets', path: '/budgets', items: [] },
  { key: 'nav.assets', path: '/assets', items: [] },
  { key: 'nav.reports', path: '/reports', items: [] },
  {
    key: 'nav.administration',
    path: '/administration',
    items: [
      { key: 'nav.administration.organisation', path: '/administration/organisation' },
      { key: 'nav.administration.campuses', path: '/administration/campuses' },
      { key: 'nav.administration.academicYears', path: '/administration/academic-years' },
      { key: 'nav.administration.users', path: '/administration/users' },
      { key: 'nav.administration.roles', path: '/administration/roles' },
      { key: 'nav.administration.permissions', path: '/administration/permissions' },
      { key: 'nav.administration.workflows', path: '/administration/workflows' },
      { key: 'nav.administration.numbering', path: '/administration/numbering' },
      { key: 'nav.administration.documents', path: '/administration/documents' },
      { key: 'nav.administration.imports', path: '/administration/imports' },
      { key: 'nav.administration.backups', path: '/administration/backups' },
      { key: 'nav.administration.audit', path: '/administration/audit' },
      { key: 'nav.administration.settings', path: '/administration/settings' },
    ],
  },
]
