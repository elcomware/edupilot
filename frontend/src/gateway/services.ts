import type {
  AcademicYear,
  Account,
  AssignRoleRequest,
  Campus,
  CreateHouseholdRequest,
  CreateYearRequest,
  DashboardSummary,
  FiscalPeriod,
  Household,
  Invoice,
  LinkRelationshipRequest,
  ListPeopleWithRoleRequest,
  ListPeopleRequest,
  ListRelationshipsRequest,
  Money,
  OperationParams,
  Organisation,
  Page,
  PageRequest,
  Payment,
  Person,
  PersonId,
  PersonRole,
  RegisterPersonRequest,
  Relationship,
  Result,
  Statement,
  StudentId,
  Term,
  Transport,
  TrialBalanceRow,
} from './types'

export interface OrganisationGateway {
  getCurrent(): Promise<Result<Organisation>>
  listCampuses(request: PageRequest): Promise<Result<Page<Campus>>>
}

export interface AcademicGateway {
  listYears(request: PageRequest): Promise<Result<Page<AcademicYear>>>
  getCurrentYear(): Promise<Result<AcademicYear>>
  listTerms(academicYearId: string): Promise<Result<readonly Term[]>>
  createYear(request: CreateYearRequest): Promise<Result<AcademicYear>>
}

export interface PeopleGateway {
  list(request: ListPeopleRequest): Promise<Result<Page<Person>>>
  get(id: PersonId): Promise<Result<Person>>
  register(request: RegisterPersonRequest): Promise<Result<Person>>
  assignRole(request: AssignRoleRequest): Promise<Result<PersonRole>>
  listWithRole(request: ListPeopleWithRoleRequest): Promise<Result<Page<Person>>>
  listRelationships(request: ListRelationshipsRequest): Promise<Result<readonly Relationship[]>>
  link(request: LinkRelationshipRequest): Promise<Result<Relationship>>
  unlink(id: string): Promise<Result<Relationship>>
  listHouseholds(academicYearId?: string): Promise<Result<readonly Household[]>>
  createHousehold(request: CreateHouseholdRequest): Promise<Result<Household>>
}

export interface StudentGateway {
  list(request: PageRequest): Promise<Result<Page<Person>>>
  getStatement(studentId: StudentId, asAt?: string): Promise<Result<Statement>>
  getBalance(studentId: StudentId): Promise<Result<Money>>
}

export interface BillingGateway {
  listInvoices(request: PageRequest): Promise<Result<Page<Invoice>>>
  getInvoice(id: string): Promise<Result<Invoice>>
  createInvoice(payload: Record<string, unknown>): Promise<Result<Invoice>>
  issueInvoice(id: string): Promise<Result<Invoice>>
  cancelInvoice(id: string): Promise<Result<Invoice>>
}

export interface CollectionsGateway {
  listPayments(request: PageRequest): Promise<Result<Page<Payment>>>
  recordPayment(payload: Record<string, unknown>): Promise<Result<Payment>>
  allocatePayment(paymentId: string, allocations: readonly Record<string, unknown>[]): Promise<Result<Payment>>
  reversePayment(paymentId: string, reason: string): Promise<Result<Payment>>
}

export interface AccountingGateway {
  listAccounts(request: PageRequest): Promise<Result<Page<Account>>>
  listPeriods(request: PageRequest): Promise<Result<Page<FiscalPeriod>>>
  getTrialBalance(asAt: string): Promise<Result<readonly TrialBalanceRow[]>>
  closePeriod(id: string): Promise<Result<FiscalPeriod>>
}

export interface PayrollGateway {
  listRuns(request: PageRequest): Promise<Result<Page<Record<string, unknown>>>>
  prepareRun(periodId: string): Promise<Result<Record<string, unknown>>>
  approveRun(runId: string): Promise<Result<Record<string, unknown>>>
  postRun(runId: string): Promise<Result<Record<string, unknown>>>
}

export interface InventoryGateway {
  listProducts(request: PageRequest): Promise<Result<Page<Record<string, unknown>>>>
  listStockMovements(request: PageRequest): Promise<Result<Page<Record<string, unknown>>>>
  receiveStock(payload: Record<string, unknown>): Promise<Result<Record<string, unknown>>>
}

export interface ProcurementGateway {
  listSuppliers(request: PageRequest): Promise<Result<Page<Record<string, unknown>>>>
  listPurchaseOrders(request: PageRequest): Promise<Result<Page<Record<string, unknown>>>>
  createPurchaseOrder(payload: Record<string, unknown>): Promise<Result<Record<string, unknown>>>
}

export interface BankingGateway {
  listAccounts(request: PageRequest): Promise<Result<Page<Record<string, unknown>>>>
  listTransactions(request: PageRequest): Promise<Result<Page<Record<string, unknown>>>>
  reconcile(payload: Record<string, unknown>): Promise<Result<Record<string, unknown>>>
}

export interface BudgetingGateway {
  getVariance(fiscalYearId: string): Promise<Result<Record<string, unknown>>>
}

export interface AssetsGateway {
  listAssets(request: PageRequest): Promise<Result<Page<Record<string, unknown>>>>
}

export interface ReportingGateway {
  getDashboard(): Promise<Result<DashboardSummary>>
  runReport(reportCode: string, params: Record<string, unknown>): Promise<Result<Record<string, unknown>>>
}

export interface FinanceGateway {
  organisation: OrganisationGateway
  academic: AcademicGateway
  people: PeopleGateway
  students: StudentGateway
  billing: BillingGateway
  collections: CollectionsGateway
  accounting: AccountingGateway
  payroll: PayrollGateway
  inventory: InventoryGateway
  procurement: ProcurementGateway
  banking: BankingGateway
  budgeting: BudgetingGateway
  assets: AssetsGateway
  reporting: ReportingGateway
}

type Op = <T>(operation: string, params?: OperationParams) => Promise<Result<T>>

export function createGatewayServices(transport: Transport): FinanceGateway {
  const op: Op = <T,>(operation: string, params?: OperationParams) => transport.call<T>(operation, params)

  return {
    organisation: {
      getCurrent: () => op<Organisation>('platform.organisation.App.GetCurrent'),
      listCampuses: (request) => op<Page<Campus>>('platform.campus.App.ListCampuses', request),
    },
    academic: {
      listYears: (request) => op<Page<AcademicYear>>('platform.academic.App.ListYears', request),
      getCurrentYear: () => op<AcademicYear>('platform.academic.App.GetCurrentYear'),
      listTerms: (academicYearId) => op<readonly Term[]>('platform.academic.App.ListTerms', { academicYearId }),
      createYear: (request) => op<AcademicYear>('platform.academic.App.CreateYear', request),
    },
    people: {
      list: (request) => op<Page<Person>>('platform.people.App.ListPeople', request),
      get: (id) => op<Person>('platform.people.App.GetPerson', { id }),
      register: (request) => op<Person>('platform.people.App.RegisterPerson', request),
      assignRole: (request) => op<PersonRole>('platform.people.App.AssignRole', request),
      listWithRole: (request) => op<Page<Person>>('platform.people.App.ListPeopleWithRole', request),
      listRelationships: (request) => op<readonly Relationship[]>('platform.people.App.ListRelationships', request),
      link: (request) => op<Relationship>('platform.people.App.Link', request),
      unlink: (id) => op<Relationship>('platform.people.App.Unlink', { id }),
      listHouseholds: (academicYearId) =>
        op<readonly Household[]>('platform.people.App.ListHouseholds', { academicYearId: academicYearId ?? '' }),
      createHousehold: (request) => op<Household>('platform.people.App.CreateHousehold', request),
    },
    students: {
      list: (request) => op<Page<Person>>('platform.people.App.ListPeopleWithRole', { ...request, role: 'STUDENT' }),
      getStatement: (studentId, asAt) => op<Statement>('finance.receivables.App.GetStatement', { studentId, asAt }),
      getBalance: (studentId) => op<Money>('finance.receivables.App.GetBalance', { studentId }),
    },
    billing: {
      listInvoices: (request) => op<Page<Invoice>>('finance.billing.App.ListInvoices', { request }),
      getInvoice: (id) => op<Invoice>('finance.billing.App.GetInvoice', { id }),
      createInvoice: (payload) => op<Invoice>('finance.billing.App.CreateInvoice', payload),
      issueInvoice: (id) => op<Invoice>('finance.billing.App.IssueInvoice', { id }),
      cancelInvoice: (id) => op<Invoice>('finance.billing.App.CancelInvoice', { id }),
    },
    collections: {
      listPayments: (request) => op<Page<Payment>>('finance.collections.App.ListPayments', { request }),
      recordPayment: (payload) => op<Payment>('finance.collections.App.RecordPayment', payload),
      allocatePayment: (paymentId, allocations) =>
        op<Payment>('finance.collections.App.AllocatePayment', { paymentId, allocations }),
      reversePayment: (paymentId, reason) => op<Payment>('finance.collections.App.ReversePayment', { paymentId, reason }),
    },
    accounting: {
      listAccounts: (request) => op<Page<Account>>('finance.accounting.App.ListAccounts', { request }),
      listPeriods: (request) => op<Page<FiscalPeriod>>('finance.accounting.App.ListPeriods', { request }),
      getTrialBalance: (asAt) => op<readonly TrialBalanceRow[]>('finance.accounting.App.GetTrialBalance', { asAt }),
      closePeriod: (id) => op<FiscalPeriod>('finance.accounting.App.ClosePeriod', { id }),
    },
    payroll: {
      listRuns: (request) => op<Page<Record<string, unknown>>>('finance.payroll.App.ListRuns', { request }),
      prepareRun: (periodId) => op<Record<string, unknown>>('finance.payroll.App.PrepareRun', { periodId }),
      approveRun: (runId) => op<Record<string, unknown>>('finance.payroll.App.ApproveRun', { runId }),
      postRun: (runId) => op<Record<string, unknown>>('finance.payroll.App.PostRun', { runId }),
    },
    inventory: {
      listProducts: (request) => op<Page<Record<string, unknown>>>('finance.inventory.App.ListProducts', { request }),
      listStockMovements: (request) =>
        op<Page<Record<string, unknown>>>('finance.inventory.App.ListStockMovements', { request }),
      receiveStock: (payload) => op<Record<string, unknown>>('finance.inventory.App.ReceiveStock', payload),
    },
    procurement: {
      listSuppliers: (request) => op<Page<Record<string, unknown>>>('finance.procurement.App.ListSuppliers', { request }),
      listPurchaseOrders: (request) =>
        op<Page<Record<string, unknown>>>('finance.procurement.App.ListPurchaseOrders', { request }),
      createPurchaseOrder: (payload) =>
        op<Record<string, unknown>>('finance.procurement.App.CreatePurchaseOrder', payload),
    },
    banking: {
      listAccounts: (request) => op<Page<Record<string, unknown>>>('finance.banking.App.ListAccounts', { request }),
      listTransactions: (request) => op<Page<Record<string, unknown>>>('finance.banking.App.ListTransactions', { request }),
      reconcile: (payload) => op<Record<string, unknown>>('finance.banking.App.Reconcile', payload),
    },
    budgeting: {
      getVariance: (fiscalYearId) => op<Record<string, unknown>>('finance.budgeting.App.GetVariance', { fiscalYearId }),
    },
    assets: {
      listAssets: (request) => op<Page<Record<string, unknown>>>('finance.assets.App.ListAssets', { request }),
    },
    reporting: {
      getDashboard: () => op<DashboardSummary>('finance.reporting.App.GetDashboard'),
      runReport: (reportCode, params) => op<Record<string, unknown>>('finance.reporting.App.RunReport', { reportCode, params }),
    },
  }
}
