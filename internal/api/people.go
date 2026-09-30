package api

import (
	"time"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	academicdomain "github.com/elcomware/edupilot/internal/platform/academic/domain"
	peopleapp "github.com/elcomware/edupilot/internal/platform/people/application"
	peopledomain "github.com/elcomware/edupilot/internal/platform/people/domain"
)

// The people and academic views are the contract the frontend compiles against.
// They are deliberately flatter than the domain: the interface shows "2025-2026"
// and "ENROLLED", never a time zone or a version number, and it never receives a
// domain identifier as anything but a string.

// AcademicYearView is one academic year.
type AcademicYearView struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisationId"`
	Name           string `json:"name"`
	StartsOn       string `json:"startsOn"`
	EndsOn         string `json:"endsOn"`
	IsCurrent      bool   `json:"isCurrent"`
}

// NewAcademicYearView maps an academic year to the wire shape. Dates go over the
// wire as plain calendar days, because a school year boundary is a date and not
// an instant: sending a timestamp would let a browser in another time zone
// display 31 August as 1 September.
func NewAcademicYearView(year *academicdomain.AcademicYear) AcademicYearView {
	return AcademicYearView{
		ID:             year.ID.String(),
		OrganisationID: year.OrganisationID.String(),
		Name:           year.Name,
		StartsOn:       CalendarDate(year.StartsOn),
		EndsOn:         CalendarDate(year.EndsOn),
		IsCurrent:      year.IsCurrent,
	}
}

// NewAcademicYearViews maps a slice of years to the wire shape.
func NewAcademicYearViews(years []*academicdomain.AcademicYear) []AcademicYearView {
	views := make([]AcademicYearView, 0, len(years))
	for _, year := range years {
		views = append(views, NewAcademicYearView(year))
	}
	return views
}

// TermView is one term of an academic year.
type TermView struct {
	ID             string `json:"id"`
	AcademicYearID string `json:"academicYearId"`
	Name           string `json:"name"`
	Sequence       int    `json:"sequence"`
	StartsOn       string `json:"startsOn"`
	EndsOn         string `json:"endsOn"`
}

// NewTermView maps a term to the wire shape.
func NewTermView(term *academicdomain.Term) TermView {
	return TermView{
		ID:             term.ID.String(),
		AcademicYearID: term.AcademicYearID.String(),
		Name:           term.Name,
		Sequence:       term.Sequence,
		StartsOn:       CalendarDate(term.StartsOn),
		EndsOn:         CalendarDate(term.EndsOn),
	}
}

// NewTermViews maps a slice of terms to the wire shape.
func NewTermViews(terms []*academicdomain.Term) []TermView {
	views := make([]TermView, 0, len(terms))
	for _, term := range terms {
		views = append(views, NewTermView(term))
	}
	return views
}

// PersonView is one person, with the roles they hold in one academic year.
//
// Roles and Relationships are the reason this is not a flat record: the
// interface has to show that one person is a student in one year and an employee
// in another, and that a guardian is linked to several students, without the
// frontend inventing that structure itself.
type PersonView struct {
	ID             string           `json:"id"`
	OrganisationID string           `json:"organisationId"`
	FirstName      string           `json:"firstName"`
	MiddleName     string           `json:"middleName"`
	LastName       string           `json:"lastName"`
	PreferredName  string           `json:"preferredName"`
	DisplayName    string           `json:"displayName"`
	Email          string           `json:"email"`
	Phone          string           `json:"phone"`
	SecondaryPhone string           `json:"secondaryPhone"`
	DateOfBirth    string           `json:"dateOfBirth"`
	Gender         string           `json:"gender"`
	Nationality    string           `json:"nationality"`
	NationalID     string           `json:"nationalId"`
	Address        AddressView      `json:"address"`
	PhotoDocument  string           `json:"photoDocumentId"`
	IsActive       bool             `json:"isActive"`
	Roles          []PersonRoleView `json:"roles"`
	GuardiansOf    []string         `json:"guardiansOf"`
	GuardianOf     []string         `json:"guardianOf"`
}

// AddressView is a postal address. It is a value object on the wire so the
// frontend does not have to know which tables hold which part of it.
type AddressView struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

// NewAddressView maps a domain address to the wire shape.
func NewAddressView(address peopledomain.Address) AddressView {
	return AddressView{
		Line1:      address.Line1,
		Line2:      address.Line2,
		City:       address.City,
		PostalCode: address.PostalCode,
		Country:    address.Country,
	}
}

// PersonRoleView is one role a person holds in an academic year, with the
// profile that belongs to that role. The role-specific fields are grouped rather
// than flattened, because a student has no employee number and an employee has no
// student number, and a flat shape would carry empty fields in every row.
type PersonRoleView struct {
	ID             string `json:"id"`
	PersonID       string `json:"personId"`
	AcademicYearID string `json:"academicYearId"`
	Role           string `json:"role"`
	StartsOn       string `json:"startsOn"`
	EndsOn         string `json:"endsOn"`
	IsCurrent      bool   `json:"isCurrent"`

	Student *StudentRoleDetailsView `json:"student,omitempty"`
	// Employee and Guardian are separate optional blocks, not one shared struct,
	// so the interface can decide the labels from the role rather than guess.
	Employee *EmployeeRoleDetailsView `json:"employee,omitempty"`
	Guardian *GuardianRoleDetailsView `json:"guardian,omitempty"`
}

// StudentRoleDetailsView is the student-specific part of a role.
type StudentRoleDetailsView struct {
	StudentNumber  string `json:"studentNumber"`
	Status         string `json:"status"`
	AdmissionDate  string `json:"admissionDate"`
	PreviousSchool string `json:"previousSchool"`
	IsBoarding     bool   `json:"isBoarding"`
}

// EmployeeRoleDetailsView is the employee-specific part of a role.
type EmployeeRoleDetailsView struct {
	EmployeeNumber string `json:"employeeNumber"`
	JobTitle       string `json:"jobTitle"`
	Department     string `json:"department"`
	HiredOn        string `json:"hiredOn"`
	EndedOn        string `json:"endedOn"`
	ContractType   string `json:"contractType"`
	PayrollGroup   string `json:"payrollGroup"`
}

// GuardianRoleDetailsView is the rights a guardian has, as opposed to the family
// structure that the relationships describe.
type GuardianRoleDetailsView struct {
	IsEmergencyContact  bool `json:"isEmergencyContact"`
	MayCollectStudent   bool `json:"mayCollectStudent"`
	IsBillingContact    bool `json:"isBillingContact"`
	CanAuthoriseMedical bool `json:"canAuthoriseMedical"`
}

// NewPersonRoleView maps a role to the wire shape. The profile argument is the
// one that matches the role, and is nil when it has not been written yet, so the
// interface can tell "no employee number" from "not an employee".
func NewPersonRoleView(role *peopledomain.PersonRole, student *peopledomain.StudentProfile, employee *peopledomain.EmployeeProfile, guardian *peopledomain.GuardianProfile) PersonRoleView {
	view := PersonRoleView{
		ID:             role.ID.String(),
		PersonID:       role.PersonID.String(),
		AcademicYearID: role.AcademicYearID.String(),
		Role:           string(role.Role),
		StartsOn:       OptionalCalendarDate(role.StartsOn),
		EndsOn:         OptionalCalendarDate(role.EndsOn),
		IsCurrent:      role.IsCurrent,
	}

	if student != nil {
		view.Student = &StudentRoleDetailsView{
			StudentNumber:  student.StudentNumber,
			Status:         string(student.Status),
			AdmissionDate:  OptionalCalendarDate(student.AdmissionDate),
			PreviousSchool: student.PreviousSchool,
			IsBoarding:     student.IsBoarding,
		}
	}
	if employee != nil {
		view.Employee = &EmployeeRoleDetailsView{
			EmployeeNumber: employee.EmployeeNumber,
			JobTitle:       employee.JobTitle,
			Department:     employee.Department,
			HiredOn:        OptionalCalendarDate(employee.HiredOn),
			EndedOn:        OptionalCalendarDate(employee.EndedOn),
			ContractType:   string(employee.ContractType),
			PayrollGroup:   employee.PayrollGroup,
		}
	}
	if guardian != nil {
		view.Guardian = &GuardianRoleDetailsView{
			IsEmergencyContact:  guardian.IsEmergencyContact,
			MayCollectStudent:   guardian.MayCollectStudent,
			IsBillingContact:    guardian.IsBillingContact,
			CanAuthoriseMedical: guardian.CanAuthoriseMedical,
		}
	}
	return view
}

// NewPersonRoleViews maps a slice of roles to the wire shape.
func NewPersonRoleViews(roles []*peopledomain.PersonRole) []PersonRoleView {
	views := make([]PersonRoleView, 0, len(roles))
	for _, role := range roles {
		views = append(views, NewPersonRoleView(role, nil, nil, nil))
	}
	return views
}

// NewPersonView maps a person to the wire shape. Roles may be empty when the
// caller is listing people across years and has not resolved any.
func NewPersonView(person *peopledomain.Person, roles []*peopledomain.PersonRole) PersonView {
	view := PersonView{
		ID:             person.ID.String(),
		OrganisationID: person.OrganisationID.String(),
		FirstName:      person.FirstName,
		MiddleName:     person.MiddleName,
		LastName:       person.LastName,
		PreferredName:  person.PreferredName,
		DisplayName:    person.DisplayName(),
		Email:          person.Email,
		Phone:          person.Phone,
		SecondaryPhone: person.SecondaryPhone,
		DateOfBirth:    OptionalCalendarDate(person.DateOfBirth),
		Gender:         person.Gender,
		Nationality:    person.Nationality,
		NationalID:     person.NationalID,
		Address:        NewAddressView(person.Address),
		IsActive:       person.IsActive,
		Roles:          make([]PersonRoleView, 0, len(roles)),
		GuardiansOf:    []string{},
		GuardianOf:     []string{},
	}
	if person.PhotoDocumentID != nil {
		view.PhotoDocument = person.PhotoDocumentID.String()
	}
	for _, role := range roles {
		view.Roles = append(view.Roles, NewPersonRoleView(role, nil, nil, nil))
	}
	return view
}

// NewPersonViewWithRoles maps a person whose roles have already been read
// together with their profiles.
//
// This is what a person screen needs: a list of roles without their profiles
// would show "STUDENT" with no student number, which is the one field anybody
// opening that screen is looking for.
func NewPersonViewWithRoles(person *peopledomain.Person, roles []*peopleapp.RoleDetails) PersonView {
	view := NewPersonView(person, nil)
	view.Roles = make([]PersonRoleView, 0, len(roles))
	for _, details := range roles {
		view.Roles = append(view.Roles, NewPersonRoleView(
			details.Role, details.Student, details.Employee, details.Guardian,
		))
	}
	return view
}

// NewPersonViewWithRole maps a person showing one role in particular, which is
// what a roster wants: the screen is asking "who are this year's students", so
// the role it filtered on is the role it shows.
func NewPersonViewWithRole(person *peopledomain.Person, details *peopleapp.RoleDetails) PersonView {
	view := NewPersonView(person, nil)
	view.Roles = []PersonRoleView{
		NewPersonRoleView(details.Role, details.Student, details.Employee, details.Guardian),
	}
	return view
}

// RelationshipView is one directed edge between two people in an academic year.
type RelationshipView struct {
	ID             string `json:"id"`
	AcademicYearID string `json:"academicYearId"`
	FromPersonID   string `json:"fromPersonId"`
	ToPersonID     string `json:"toPersonId"`
	Type           string `json:"type"`
	Role           string `json:"role"`
	IsPrimary      bool   `json:"isPrimary"`
	IsActive       bool   `json:"isActive"`
}

// NewRelationshipView maps a relationship to the wire shape.
func NewRelationshipView(relationship *peopledomain.Relationship) RelationshipView {
	return RelationshipView{
		ID:             relationship.ID.String(),
		AcademicYearID: relationship.AcademicYearID.String(),
		FromPersonID:   relationship.FromPersonID.String(),
		ToPersonID:     relationship.ToPersonID.String(),
		Type:           string(relationship.Type),
		Role:           string(relationship.Role),
		IsPrimary:      relationship.IsPrimary,
		IsActive:       relationship.IsActive,
	}
}

// NewRelationshipViews maps a slice of relationships to the wire shape.
func NewRelationshipViews(relationships []*peopledomain.Relationship) []RelationshipView {
	views := make([]RelationshipView, 0, len(relationships))
	for _, relationship := range relationships {
		views = append(views, NewRelationshipView(relationship))
	}
	return views
}

// HouseholdView is a billing party for one academic year, with the people who
// belong to it. The member list is billing membership, not a family tree: the
// family relationships are read from the relationship edges.
type HouseholdView struct {
	ID             string                `json:"id"`
	AcademicYearID string                `json:"academicYearId"`
	Name           string                `json:"name"`
	BillingEmail   string                `json:"billingEmail"`
	BillingPhone   string                `json:"billingPhone"`
	BillingAddress AddressView           `json:"billingAddress"`
	Members        []HouseholdMemberView `json:"members"`
}

// HouseholdMemberView is one person's place in a household.
type HouseholdMemberView struct {
	PersonID      string `json:"personId"`
	Role          string `json:"role"`
	IsBillingPart bool   `json:"isBillingPart"`
}

// NewHouseholdView maps a household to the wire shape.
func NewHouseholdView(household *peopledomain.Household) HouseholdView {
	view := HouseholdView{
		ID:             household.ID.String(),
		AcademicYearID: household.AcademicYearID.String(),
		Name:           household.Name,
		BillingEmail:   household.BillingEmail,
		BillingPhone:   household.BillingPhone,
		BillingAddress: NewAddressView(household.BillingAddress),
		Members:        make([]HouseholdMemberView, 0, len(household.Members)),
	}
	for _, member := range household.Members {
		view.Members = append(view.Members, HouseholdMemberView{
			PersonID:      member.PersonID.String(),
			Role:          string(member.Role),
			IsBillingPart: member.IsBillingPart,
		})
	}
	return view
}

// NewHouseholdViews maps a slice of households to the wire shape.
func NewHouseholdViews(households []*peopledomain.Household) []HouseholdView {
	views := make([]HouseholdView, 0, len(households))
	for _, household := range households {
		views = append(views, NewHouseholdView(household))
	}
	return views
}

// CalendarDate formats an instant as the calendar day it falls on, in UTC. School
// years and terms are delimited by dates rather than instants, and a term that
// begins "31 August" must not render as "1 September" for a reader east of
// Greenwich.
func CalendarDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

// OptionalCalendarDate formats a nullable date, returning an empty string rather
// than a zero date, so the frontend can render "no date" without parsing
// "0001-01-01".
func OptionalCalendarDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return CalendarDate(*value)
}

// ParseCalendarDate reads a calendar day received from the interface. An empty
// string is accepted and yields a nil time, because "no date" and "the first of
// January 0001" are not the same request.
func ParseCalendarDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, apperr.ValidationFailed("a date must be written as YYYY-MM-DD").
			WithDetail("field", "date").
			WithDetail("value", value)
	}
	return &parsed, nil
}

// ParseRequiredCalendarDate reads a calendar day that must be present, such as
// the start and end of an academic year.
func ParseRequiredCalendarDate(field, value string) (time.Time, error) {
	parsed, err := ParseCalendarDate(value)
	if err != nil {
		return time.Time{}, err
	}
	if parsed == nil {
		return time.Time{}, apperr.ValidationFailed("a date is required").
			WithDetail("field", field)
	}
	return *parsed, nil
}

// The request types below are the transport-facing shapes for the people and
// academic use cases. A transport receives these, never a domain aggregate, and
// the organisation is resolved from the installation rather than trusted from the
// request: a single-tenant desktop build has exactly one tenant, and accepting an
// organisation from the client would let a caller name somebody else's school.

// ListPeopleRequest asks for a page of people in the current installation.
// Every bound transport method takes either nothing or exactly one request
// object, and returns a Result. Wails binds arguments positionally, so a method
// with a bare string parameter would have to be called as GetPerson("id") while
// its neighbours are called with an object. Mixing the two shapes is how a
// gateway silently stops working, so the shape is uniform: one object in, one
// envelope out, and a field added to a request later never changes the binding.

// GetPersonRequest identifies one person.
type GetPersonRequest struct {
	ID string `json:"id"`
}

// GetRelationshipRequest identifies one relationship.
type GetRelationshipRequest struct {
	ID string `json:"id"`
}

// UnlinkRelationshipRequest identifies the relationship to end.
type UnlinkRelationshipRequest struct {
	ID string `json:"id"`
}

// ListTermsRequest asks for the terms of one academic year.
type ListTermsRequest struct {
	AcademicYearID string `json:"academicYearId"`
}

// ListHouseholdsRequest asks for the households of a year. An empty year means
// the school's current one, which is what a bursar almost always wants.
type ListHouseholdsRequest struct {
	AcademicYearID string `json:"academicYearId"`
}

// ListPeopleWithRoleRequest asks for everyone holding one role in a year, which
// is how a roster is built. An empty year means the current one.
type ListPeopleWithRoleRequest struct {
	AcademicYearID string `json:"academicYearId"`
	Role           string `json:"role"`
}

type ListPeopleRequest struct {
	PageRequest
	// IncludeInactive returns deactivated people as well. A school needs them to
	// resolve an old statement, but not on the main roster.
	IncludeInactive bool `json:"includeInactive"`
	// AcademicYearID scopes the roles shown on each row. Empty means the current
	// year, because a role held last year says nothing about this one and a
	// directory that mixed them would be a lie.
	AcademicYearID string `json:"academicYearId"`
}

// RegisterPersonRequest adds a person.
type RegisterPersonRequest struct {
	FirstName      string      `json:"firstName"`
	MiddleName     string      `json:"middleName"`
	LastName       string      `json:"lastName"`
	PreferredName  string      `json:"preferredName"`
	Email          string      `json:"email"`
	Phone          string      `json:"phone"`
	SecondaryPhone string      `json:"secondaryPhone"`
	DateOfBirth    string      `json:"dateOfBirth"`
	Gender         string      `json:"gender"`
	Nationality    string      `json:"nationality"`
	NationalID     string      `json:"nationalId"`
	Address        AddressView `json:"address"`
}

// AssignRoleRequest gives a person a role in an academic year, together with the
// profile that belongs to that role.
//
// The role-specific fields are all present on one request because a single form
// collects them; only the ones matching Role are read. Keeping them apart would
// mean four near-identical request types for one use case.
type AssignRoleRequest struct {
	PersonID       string `json:"personId"`
	AcademicYearID string `json:"academicYearId"`
	Role           string `json:"role"`
	StartsOn       string `json:"startsOn"`
	EndsOn         string `json:"endsOn"`

	StudentNumber  string `json:"studentNumber"`
	AdmissionDate  string `json:"admissionDate"`
	StudentStatus  string `json:"studentStatus"`
	PreviousSchool string `json:"previousSchool"`
	IsBoarding     bool   `json:"isBoarding"`

	EmployeeNumber string `json:"employeeNumber"`
	JobTitle       string `json:"jobTitle"`
	Department     string `json:"department"`
	HiredOn        string `json:"hiredOn"`
	EndedOn        string `json:"endedOn"`
	ContractType   string `json:"contractType"`
	PayrollGroup   string `json:"payrollGroup"`

	IsEmergencyContact  bool `json:"isEmergencyContact"`
	MayCollectStudent   bool `json:"mayCollectStudent"`
	IsBillingContact    bool `json:"isBillingContact"`
	CanAuthoriseMedical bool `json:"canAuthoriseMedical"`
}

// LinkRelationshipRequest records a directed relationship between two people.
type LinkRelationshipRequest struct {
	FromPersonID   string `json:"fromPersonId"`
	ToPersonID     string `json:"toPersonId"`
	AcademicYearID string `json:"academicYearId"`
	Type           string `json:"type"`
	Role           string `json:"role"`
	MakePrimary    bool   `json:"makePrimary"`
}

// ListRelationshipsRequest asks for the edges around one person in a year. An
// empty Type means every kind of edge, which is what a family view wants.
type ListRelationshipsRequest struct {
	PersonID       string `json:"personId"`
	AcademicYearID string `json:"academicYearId"`
	Type           string `json:"type"`
	// Incoming reads the edges pointing at the person, so their guardians. The
	// default, false, reads the edges leaving them, so their dependants.
	Incoming bool `json:"incoming"`
}

// CreateYearRequest opens an academic year.
type CreateYearRequest struct {
	Name     string `json:"name"`
	StartsOn string `json:"startsOn"`
	EndsOn   string `json:"endsOn"`
	// MakeCurrent opens this year as the current one, demoting the previous.
	MakeCurrent bool `json:"makeCurrent"`
}

// CreateHouseholdRequest creates a billing party for a year.
type CreateHouseholdRequest struct {
	AcademicYearID string                 `json:"academicYearId"`
	Name           string                 `json:"name"`
	BillingEmail   string                 `json:"billingEmail"`
	BillingPhone   string                 `json:"billingPhone"`
	BillingAddress AddressView            `json:"billingAddress"`
	Members        []HouseholdMemberInput `json:"members"`
}

// HouseholdMemberInput places one person in a household.
type HouseholdMemberInput struct {
	PersonID      string `json:"personId"`
	Role          string `json:"role"`
	IsBillingPart bool   `json:"isBillingPart"`
}
