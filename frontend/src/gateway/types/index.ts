export type Uuid = string
export type OrganisationId = Uuid
export type CampusId = Uuid
export type PersonId = Uuid
export type StudentId = Uuid
export type IsoDate = string
export type IsoDateTime = string
export type CurrencyCode = string

/**
 * Money is always integer minor units plus a currency. It is never a float,
 * on this side of the gateway or the other.
 */
export interface Money {
  readonly amountMinor: number
  readonly currency: CurrencyCode
}

export type AppErrorCode =
  | 'INSUFFICIENT_PERMISSION'
  | 'UNAUTHENTICATED'
  | 'PERIOD_CLOSED'
  | 'INVOICE_ALREADY_POSTED'
  | 'PAYMENT_ALREADY_REVERSED'
  | 'INVALID_ALLOCATION'
  | 'UNBALANCED_JOURNAL'
  | 'STOCK_INSUFFICIENT'
  | 'PAYROLL_LOCKED'
  | 'VALIDATION_FAILED'
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'ALREADY_EXISTS'
  | 'NETWORK'
  | 'INTERNAL'

export interface AppError {
  readonly code: AppErrorCode
  readonly message: string
  readonly details?: Readonly<Record<string, unknown>>
}

export type Result<T> = { readonly ok: true; readonly value: T } | { readonly ok: false; readonly error: AppError }

export function ok<T>(value: T): Result<T> {
  return { ok: true, value }
}

export function err<T = never>(code: AppErrorCode, message: string, details?: Record<string, unknown>): Result<T> {
  return details === undefined
    ? { ok: false, error: { code, message } }
    : { ok: false, error: { code, message, details } }
}

export function isOk<T>(result: Result<T>): result is { ok: true; value: T } {
  return result.ok
}

export interface PageRequest {
  readonly limit: number
  readonly offset: number
  readonly search?: string
  readonly sort?: string
}

export interface Page<T> {
  readonly items: readonly T[]
  readonly total: number
}

/**
 * A request body for one operation. It is looser than a dictionary on purpose:
 * the request shapes are declared as interfaces, and an interface is not
 * assignable to `Record<string, unknown>` even when every field is a plain
 * value. Typing the body as an object keeps the call sites strongly typed
 * without forcing every request to be reshaped to satisfy a dictionary.
 */
export type OperationParams = object

/**
 * Transport is the only thing that differs between the desktop edition and the
 * Site Server / Cloud editions. Gateways are defined once on top of it.
 */
export interface Transport {
  call<T>(operation: string, params?: OperationParams): Promise<Result<T>>
}

export interface Organisation {
  readonly id: OrganisationId
  readonly name: string
  readonly legalName: string
  readonly currency: CurrencyCode
  readonly country: string
  readonly defaultLocale: string
}

export interface Campus {
  readonly id: CampusId
  readonly organisationId: OrganisationId
  readonly name: string
  readonly code: string
  readonly address: string
}

export interface AcademicYear {
  readonly id: Uuid
  readonly organisationId: OrganisationId
  readonly name: string
  readonly startsOn: IsoDate
  readonly endsOn: IsoDate
  readonly isCurrent: boolean
}

export interface Term {
  readonly id: Uuid
  readonly academicYearId: Uuid
  readonly name: string
  readonly sequence: number
  readonly startsOn: IsoDate
  readonly endsOn: IsoDate
}

/**
 * A postal address travels as a value object, so the interface never has to know
 * which columns of which table hold which part of it.
 *
 * Calendar dates cross the gateway as `YYYY-MM-DD` and an absent date as an
 * empty string, never as a zero date, so "no date" needs no special parsing.
 */
export interface Address {
  readonly line1: string
  readonly line2: string
  readonly city: string
  readonly postalCode: string
  readonly country: string
}

export const PERSON_ROLES = ['STUDENT', 'EMPLOYEE', 'GUARDIAN', 'SUPPLIER_CONTACT'] as const
export type PersonRoleType = (typeof PERSON_ROLES)[number]

export type StudentStatus = 'ENROLLED' | 'PROBATION' | 'SUSPENDED' | 'GRADUATED' | 'WITHDRAWN'

export type ContractType = 'PERMANENT' | 'FIXED_TERM' | 'PART_TIME' | 'CONTRACT' | 'INTERN'

export interface StudentRoleDetails {
  readonly studentNumber: string
  readonly status: StudentStatus
  readonly admissionDate: IsoDate
  readonly previousSchool: string
  readonly isBoarding: boolean
}

export interface EmployeeRoleDetails {
  readonly employeeNumber: string
  readonly jobTitle: string
  readonly department: string
  readonly hiredOn: IsoDate
  readonly endedOn: IsoDate
  readonly contractType: ContractType
  readonly payrollGroup: string
}

export interface GuardianRoleDetails {
  readonly isEmergencyContact: boolean
  readonly mayCollectStudent: boolean
  readonly isBillingContact: boolean
  readonly canAuthoriseMedical: boolean
}

/**
 * One role a person holds in one academic year, with the profile that belongs to
 * that role. The three profile blocks are separate and optional rather than one
 * shared shape, so the labels can come from the role instead of being guessed
 * from whichever field happens to be filled in.
 */
export interface PersonRole {
  readonly id: Uuid
  readonly personId: PersonId
  readonly academicYearId: Uuid
  readonly role: PersonRoleType
  readonly startsOn: IsoDate
  readonly endsOn: IsoDate
  readonly isCurrent: boolean
  readonly student?: StudentRoleDetails
  readonly employee?: EmployeeRoleDetails
  readonly guardian?: GuardianRoleDetails
}

/**
 * A person is a person, not a type. The same person may be an employee and a
 * parent in the same academic year, and the interface has to be able to say so.
 */
export interface Person {
  readonly id: PersonId
  readonly organisationId: OrganisationId
  readonly firstName: string
  readonly middleName: string
  readonly lastName: string
  readonly preferredName: string
  readonly displayName: string
  readonly email: string
  readonly phone: string
  readonly secondaryPhone: string
  readonly dateOfBirth: IsoDate
  readonly gender: string
  readonly nationality: string
  readonly nationalId: string
  readonly address: Address
  readonly photoDocumentId: Uuid
  readonly isActive: boolean
  readonly roles: readonly PersonRole[]
  readonly guardiansOf: readonly PersonId[]
  readonly guardianOf: readonly PersonId[]
}

export const RELATIONSHIP_TYPES = [
  'GUARDIAN_OF',
  'EMERGENCY_CONTACT_OF',
  'SPONSOR_OF',
  'CAREGIVER_OF',
  'NEXT_OF_KIN_OF',
  'SIBLING_OF',
  'SPOUSE_OF',
  'STEP_PARENT_OF',
] as const
export type RelationshipType = (typeof RELATIONSHIP_TYPES)[number]

export const RELATIONSHIP_ROLES = [
  'MOTHER',
  'FATHER',
  'PARENT',
  'LEGAL_GUARDIAN',
  'FOSTER_PARENT',
  'GRANDPARENT',
  'SIBLING',
  'AUNT',
  'UNCLE',
  'SPOUSE',
  'EMPLOYER',
  'FRIEND',
  'NANNY',
] as const
/**
 * An edge may be recorded without naming how the two people are related — an
 * emergency contact is somebody's contact, not necessarily a relative. The empty
 * role is therefore part of the contract, not a missing value.
 */
export type RelationshipRole = (typeof RELATIONSHIP_ROLES)[number] | ''

/**
 * A directed edge between two people for one academic year. Unlinking
 * deactivates the edge rather than deleting it, so a document printed last year
 * still resolves the guardian who was on it.
 */
export interface Relationship {
  readonly id: Uuid
  readonly academicYearId: Uuid
  readonly fromPersonId: PersonId
  readonly toPersonId: PersonId
  readonly type: RelationshipType
  readonly role: RelationshipRole
  readonly isPrimary: boolean
  readonly isActive: boolean
}

export const HOUSEHOLD_MEMBER_ROLES = ['HEAD', 'PARTNER', 'DEPENDENT', 'OTHER'] as const
export type HouseholdMemberRole = (typeof HOUSEHOLD_MEMBER_ROLES)[number]

export interface HouseholdMember {
  readonly personId: PersonId
  readonly role: HouseholdMemberRole
  readonly isBillingPart: boolean
}

/** A household is a billing party: the group an invoice is addressed to. */
export interface Household {
  readonly id: Uuid
  readonly academicYearId: Uuid
  readonly name: string
  readonly billingEmail: string
  readonly billingPhone: string
  readonly billingAddress: Address
  readonly members: readonly HouseholdMember[]
}

export interface ListPeopleRequest extends PageRequest {
  readonly includeInactive?: boolean
  readonly search?: string
  /**
   * Scopes the roles shown beside each person. Empty means the current year,
   * which is right for a roster: a role held last year says nothing about this
   * one, and a directory mixing the two would be misleading.
   */
  readonly academicYearId?: string
}

/**
 * A roster request: everyone holding one role in a year. This is how the
 * students, employees and guardians lists are built, rather than loading every
 * person and filtering in the browser, which cannot see a year it did not load.
 */
export interface ListPeopleWithRoleRequest extends PageRequest {
  readonly role: PersonRole
  readonly academicYearId?: string
}

export interface RegisterPersonRequest {
  readonly firstName: string
  readonly middleName?: string
  readonly lastName: string
  readonly preferredName?: string
  readonly email?: string
  readonly phone?: string
  readonly secondaryPhone?: string
  readonly dateOfBirth?: IsoDate
  readonly gender?: string
  readonly nationality?: string
  readonly nationalId?: string
  readonly address?: Partial<Address>
}

export interface AssignRoleRequest {
  readonly personId: PersonId
  /**
   * Left empty to mean the school's current academic year, which is what a
   * bursar almost always wants. Named when back-filling an earlier year.
   */
  readonly academicYearId?: Uuid
  readonly role: PersonRoleType
  readonly startsOn?: IsoDate
  readonly endsOn?: IsoDate
  readonly studentNumber?: string
  readonly admissionDate?: IsoDate
  readonly studentStatus?: StudentStatus
  readonly previousSchool?: string
  readonly isBoarding?: boolean
  readonly employeeNumber?: string
  readonly jobTitle?: string
  readonly department?: string
  readonly hiredOn?: IsoDate
  readonly endedOn?: IsoDate
  readonly contractType?: ContractType
  readonly payrollGroup?: string
  readonly isEmergencyContact?: boolean
  readonly mayCollectStudent?: boolean
  readonly isBillingContact?: boolean
  readonly canAuthoriseMedical?: boolean
}

export interface LinkRelationshipRequest {
  readonly fromPersonId: PersonId
  readonly toPersonId: PersonId
  readonly academicYearId?: Uuid
  readonly type: RelationshipType
  readonly role?: RelationshipRole
  readonly makePrimary?: boolean
}

export interface ListRelationshipsRequest {
  readonly personId: PersonId
  readonly academicYearId?: Uuid
  readonly type?: RelationshipType
  /** Incoming reads the edges arriving at the person: their guardians. */
  readonly incoming?: boolean
}

export interface CreateYearRequest {
  readonly name: string
  readonly startsOn: IsoDate
  readonly endsOn: IsoDate
  readonly makeCurrent?: boolean
}

export interface HouseholdMemberInput {
  readonly personId: PersonId
  readonly role: HouseholdMemberRole
  readonly isBillingPart?: boolean
}

export interface CreateHouseholdRequest {
  readonly academicYearId?: Uuid
  readonly name: string
  readonly billingEmail?: string
  readonly billingPhone?: string
  readonly billingAddress?: Partial<Address>
  readonly members: readonly HouseholdMemberInput[]
}

export type InvoiceStatus =
  | 'DRAFT'
  | 'APPROVED'
  | 'ISSUED'
  | 'PARTIALLY_PAID'
  | 'PAID'
  | 'OVERDUE'
  | 'CANCELLED'
  | 'CREDITED'

export interface Invoice {
  readonly id: Uuid
  readonly number: string
  readonly studentId: StudentId
  readonly issueDate: IsoDate
  readonly dueDate: IsoDate
  readonly status: InvoiceStatus
  readonly total: Money
  readonly balance: Money
}

export interface Payment {
  readonly id: Uuid
  readonly receiptNumber: string
  readonly studentId: StudentId
  readonly receivedOn: IsoDate
  readonly method: string
  readonly amount: Money
  readonly allocated: Money
  readonly unallocated: Money
}

export interface Statement {
  readonly studentId: StudentId
  readonly asAt: IsoDate
  readonly charges: Money
  readonly payments: Money
  readonly balance: Money
  readonly overdue: Money
  readonly lines: readonly StatementLine[]
}

export interface StatementLine {
  readonly date: IsoDate
  readonly reference: string
  readonly description: string
  readonly debit: Money | null
  readonly credit: Money | null
  readonly runningBalance: Money
}

export interface Account {
  readonly id: Uuid
  readonly code: string
  readonly name: string
  readonly type: 'ASSET' | 'LIABILITY' | 'EQUITY' | 'REVENUE' | 'EXPENSE'
  readonly parentId: Uuid | null
}

export interface FiscalPeriod {
  readonly id: Uuid
  readonly name: string
  readonly startsOn: IsoDate
  readonly endsOn: IsoDate
  readonly state: 'OPEN' | 'SOFT_CLOSED' | 'HARD_CLOSED'
}

export interface TrialBalanceRow {
  readonly accountCode: string
  readonly accountName: string
  readonly debit: Money
  readonly credit: Money
}

export interface AgedBucket {
  readonly label: string
  readonly amount: Money
}

export interface DashboardSummary {
  readonly totalBilled: Money
  readonly totalCollected: Money
  readonly outstanding: Money
  readonly overdue: Money
  readonly collectionRatePercent: number
  readonly ageing: readonly AgedBucket[]
}
