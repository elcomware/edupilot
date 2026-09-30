package domain

import (
	"strings"
	"time"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
)

// Role is what a person does at the school during an academic year.
type Role string

// The roles EduPilot Core recognises. The set is open in the sense that a new
// value is a deliberate act, because each one implies a profile table and a set
// of permissions.
const (
	RoleStudent         Role = "STUDENT"
	RoleEmployee        Role = "EMPLOYEE"
	RoleGuardian        Role = "GUARDIAN"
	RoleSupplierContact Role = "SUPPLIER_CONTACT"
)

// Valid reports whether the role is one EduPilot knows.
func (r Role) Valid() bool {
	switch r {
	case RoleStudent, RoleEmployee, RoleGuardian, RoleSupplierContact:
		return true
	default:
		return false
	}
}

// ParseRole turns a role received from a client into a Role, reporting an
// unknown name as a validation failure rather than letting it through to become
// a role nobody can query.
func ParseRole(value string) (Role, error) {
	role := Role(value)
	if !role.Valid() {
		return "", apperr.ValidationFailed("unknown role").WithDetail("field", "role").WithDetail("value", value)
	}
	return role, nil
}

// PersonRole is one role a person holds at an organisation in one academic year.
//
// Roles are year-scoped and separate from the person, which is what allows a
// teacher to also be a guardian in the same year, and a child to be a student in
// one year and an alumnus in the next. A person may hold several roles at once.
type PersonRole struct {
	ID             ids.UUID
	OrganisationID ids.UUID
	PersonID       ids.UUID
	AcademicYearID ids.UUID
	Role           Role
	StartsOn       *time.Time
	EndsOn         *time.Time
	IsCurrent      bool
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewPersonRole creates a person role with a generated identifier.
func NewPersonRole(organisationID, personID, academicYearID ids.UUID, role Role) (*PersonRole, error) {
	return NewPersonRoleWithID(ids.New(), organisationID, personID, academicYearID, role)
}

// NewPersonRoleWithID creates a person role with a caller-supplied identifier.
func NewPersonRoleWithID(id, organisationID, personID, academicYearID ids.UUID, role Role) (*PersonRole, error) {
	if organisationID.IsNil() {
		return nil, apperr.ValidationFailed("a role must belong to an organisation").
			WithDetail("field", "person_role.organisation_id")
	}
	if personID.IsNil() {
		return nil, apperr.ValidationFailed("a role must belong to a person").
			WithDetail("field", "person_role.person_id")
	}
	if academicYearID.IsNil() {
		return nil, apperr.ValidationFailed("a role must belong to an academic year").
			WithDetail("field", "person_role.academic_year_id")
	}
	if !role.Valid() {
		return nil, apperr.ValidationFailed("unknown role").
			WithDetail("field", "person_role.role").
			WithDetail("value", string(role))
	}

	return &PersonRole{
		ID:             id,
		OrganisationID: organisationID,
		PersonID:       personID,
		AcademicYearID: academicYearID,
		Role:           role,
		IsCurrent:      true,
		Version:        1,
	}, nil
}

// SetPeriod sets the dates the role applies to. Both are optional: a role with
// no dates is open-ended. When both are given the end must not precede the
// start.
func (r *PersonRole) SetPeriod(startsOn, endsOn *time.Time) error {
	if startsOn != nil && endsOn != nil && endsOn.Before(*startsOn) {
		return apperr.ValidationFailed("a role must not end before it starts").
			WithDetail("field", "person_role.period")
	}

	if startsOn == nil {
		r.StartsOn = nil
	} else {
		start := dateOnly(*startsOn)
		r.StartsOn = &start
	}
	if endsOn == nil {
		r.EndsOn = nil
	} else {
		end := dateOnly(*endsOn)
		r.EndsOn = &end
	}
	return nil
}

// Covers reports whether the role applies on a given date. A role with no dates
// is open-ended and always covers.
func (r *PersonRole) Covers(day time.Time) bool {
	if !r.IsCurrent {
		return false
	}
	date := dateOnly(day)
	if r.StartsOn != nil && date.Before(*r.StartsOn) {
		return false
	}
	if r.EndsOn != nil && date.After(*r.EndsOn) {
		return false
	}
	return true
}

// End closes the role at a date, keeping the row so last year's roles stay
// readable. A role is never deleted.
func (r *PersonRole) End(on time.Time) error {
	end := dateOnly(on)
	if r.StartsOn != nil && end.Before(*r.StartsOn) {
		return apperr.ValidationFailed("a role must not end before it starts").
			WithDetail("field", "person_role.ends_on")
	}
	r.EndsOn = &end
	r.IsCurrent = false
	return nil
}

// StudentStatus is where a student stands in the current year.
type StudentStatus string

// Student lifecycle states. WITHDRAWN and GRADUATED are terminal for a year but
// the enrolment row is kept, because fees may still be owed against it.
const (
	StudentApplicant StudentStatus = "APPLICANT"
	StudentEnrolled  StudentStatus = "ENROLLED"
	StudentSuspended StudentStatus = "SUSPENDED"
	StudentWithdrawn StudentStatus = "WITHDRAWN"
	StudentGraduated StudentStatus = "GRADUATED"
)

// StudentProfile holds the facts that only make sense for a student. It is keyed
// by the role, so a person has exactly one student profile per year they are a
// student in.
type StudentProfile struct {
	PersonRoleID   ids.UUID
	StudentNumber  string
	AdmissionDate  *time.Time
	Status         StudentStatus
	PreviousSchool string
	IsBoarding     bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewStudentProfile creates a student profile for a student role.
func NewStudentProfile(personRoleID ids.UUID, studentNumber string) *StudentProfile {
	return &StudentProfile{
		PersonRoleID:  personRoleID,
		StudentNumber: strings.TrimSpace(studentNumber),
		Status:        StudentEnrolled,
	}
}

// SetStatus changes the student's status for the year.
func (p *StudentProfile) SetStatus(status StudentStatus) error {
	switch status {
	case StudentApplicant, StudentEnrolled, StudentSuspended, StudentWithdrawn, StudentGraduated:
	default:
		return apperr.ValidationFailed("unknown student status").
			WithDetail("field", "student_profile.status").
			WithDetail("value", string(status))
	}
	p.Status = status
	return nil
}

// SetStudentNumber sets the school-issued student number, which is distinct
// from the person identifier and from any card number.
func (p *StudentProfile) SetStudentNumber(number string) error {
	trimmed := strings.TrimSpace(number)
	if err := checkLength(trimmed, 64, "student_profile.student_number"); err != nil {
		return err
	}
	p.StudentNumber = trimmed
	return nil
}

// SetAdmissionDate records when the student joined, date-only.
func (p *StudentProfile) SetAdmissionDate(day time.Time) {
	if day.IsZero() {
		p.AdmissionDate = nil
		return
	}
	date := dateOnly(day)
	p.AdmissionDate = &date
}

// ContractType is the kind of employment contract.
type ContractType string

// Employment contract types. Payroll reads these; country-specific rules live in
// configurable rule packs, never hard-coded here (blueprint section 40).
const (
	ContractPermanent ContractType = "PERMANENT"
	ContractFixedTerm ContractType = "FIXED_TERM"
	ContractPartTime  ContractType = "PART_TIME"
	ContractTemporary ContractType = "CONTRACT"
	ContractIntern    ContractType = "INTERN"
)

// EmployeeProfile holds the facts that only make sense for an employee.
type EmployeeProfile struct {
	PersonRoleID   ids.UUID
	EmployeeNumber string
	JobTitle       string
	Department     string
	HiredOn        *time.Time
	EndedOn        *time.Time
	PayrollGroup   string
	ContractType   ContractType
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewEmployeeProfile creates an employee profile for an employee role.
func NewEmployeeProfile(personRoleID ids.UUID, employeeNumber string) *EmployeeProfile {
	return &EmployeeProfile{
		PersonRoleID:   personRoleID,
		EmployeeNumber: strings.TrimSpace(employeeNumber),
		ContractType:   ContractPermanent,
	}
}

// SetContractType sets the employment contract type.
func (p *EmployeeProfile) SetContractType(contractType ContractType) error {
	switch contractType {
	case ContractPermanent, ContractFixedTerm, ContractPartTime, ContractTemporary, ContractIntern:
	default:
		return apperr.ValidationFailed("unknown contract type").
			WithDetail("field", "employee_profile.contract_type").
			WithDetail("value", string(contractType))
	}
	p.ContractType = contractType
	return nil
}

// SetEmployment sets the job title, department and payroll group.
func (p *EmployeeProfile) SetEmployment(jobTitle, department, payrollGroup string) error {
	if err := checkLength(jobTitle, maxPersonNameLength, "employee_profile.job_title"); err != nil {
		return err
	}
	if err := checkLength(department, maxPersonNameLength, "employee_profile.department"); err != nil {
		return err
	}
	if err := checkLength(payrollGroup, 64, "employee_profile.payroll_group"); err != nil {
		return err
	}
	p.JobTitle = strings.TrimSpace(jobTitle)
	p.Department = strings.TrimSpace(department)
	p.PayrollGroup = strings.TrimSpace(payrollGroup)
	return nil
}

// SetHiredOn records the start of employment, date-only.
func (p *EmployeeProfile) SetHiredOn(day time.Time) {
	if day.IsZero() {
		p.HiredOn = nil
		return
	}
	date := dateOnly(day)
	p.HiredOn = &date
}

// SetEndedOn records the end of employment, date-only.
//
// A leaver keeps the profile and the role: the person's employment history is a
// fact about the past, and a payslip or a disciplinary record from two years ago
// still needs the department they were in. Employment ending is not the same as
// the role ending, which is what the role's own period records.
func (p *EmployeeProfile) SetEndedOn(day time.Time) error {
	if p.HiredOn != nil {
		ended := dateOnly(day)
		if ended.Before(*p.HiredOn) {
			return apperr.ValidationFailed("employment cannot end before it began").
				WithDetail("field", "employee_profile.ended_on")
		}
	}
	if day.IsZero() {
		p.EndedOn = nil
		return nil
	}
	date := dateOnly(day)
	p.EndedOn = &date
	return nil
}

// IsBillableToThisSchool is not modelled here on purpose: a supplier contact is
// a person with a role, and the supplier itself belongs to the Finance suite.

// GuardianProfile holds the rights a guardian has over students. Whether a
// guardian may be billed is recorded here as a fact about the guardian, but the
// rules that decide who gets an invoice belong to Finance.
type GuardianProfile struct {
	PersonRoleID        ids.UUID
	IsEmergencyContact  bool
	MayCollectStudent   bool
	IsBillingContact    bool
	CanAuthoriseMedical bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// NewGuardianProfile creates a guardian profile with the default rights: a
// guardian is an emergency contact and may collect, but is not by default the
// billing contact, because most families have two and only one is invoiced.
func NewGuardianProfile(personRoleID ids.UUID) *GuardianProfile {
	return &GuardianProfile{
		PersonRoleID:       personRoleID,
		IsEmergencyContact: true,
		MayCollectStudent:  true,
	}
}

// SetRights sets the guardian's rights over students.
func (p *GuardianProfile) SetRights(emergencyContact, mayCollect, isBillingContact, canAuthoriseMedical bool) {
	p.IsEmergencyContact = emergencyContact
	p.MayCollectStudent = mayCollect
	p.IsBillingContact = isBillingContact
	p.CanAuthoriseMedical = canAuthoriseMedical
}
