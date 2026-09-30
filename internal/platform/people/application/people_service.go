// Package application holds the people use cases: registering a person, giving
// them a role in an academic year, and recording who is related to whom.
package application

import (
	"context"
	"time"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/clock"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
	"github.com/elcomware/edupilot/internal/platform/people/ports"
)

// AcademicYearScope is the narrow port through which People checks that an
// academic year belongs to the organisation it is being used with.
//
// It is deliberately not a dependency on the academic module's repositories.
// People needs to know that a year is real and belongs to this tenant; it must
// not be able to edit a term or reorder a school calendar as a side effect of
// registering a student. The academic module supplies the implementation.
type AcademicYearScope interface {
	BelongsToOrganisation(ctx context.Context, organisationID, academicYearID ids.UUID) (bool, error)
	CurrentYearID(ctx context.Context, organisationID ids.UUID) (ids.UUID, error)
}

// Service implements the people use cases.
type Service struct {
	people        ports.PersonRepository
	roles         ports.RoleRepository
	students      ports.StudentProfileRepository
	employees     ports.EmployeeProfileRepository
	guardians     ports.GuardianProfileRepository
	relationships ports.RelationshipRepository
	households    ports.HouseholdRepository
	years         AcademicYearScope
	transactor    Transactor
	clock         clock.Clock
}

// Transactor is the unit of work the people use cases need. The application
// layer depends on this narrow port, not on a concrete SQL transaction.
type Transactor interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Repositories bundles the persistence ports so a caller wires them once.
type Repositories struct {
	People        ports.PersonRepository
	Roles         ports.RoleRepository
	Students      ports.StudentProfileRepository
	Employees     ports.EmployeeProfileRepository
	Guardians     ports.GuardianProfileRepository
	Relationships ports.RelationshipRepository
	Households    ports.HouseholdRepository
}

// NewService builds the people application service.
func NewService(
	repositories Repositories,
	years AcademicYearScope,
	transactor Transactor,
	systemClock clock.Clock,
) *Service {
	return &Service{
		people:        repositories.People,
		roles:         repositories.Roles,
		students:      repositories.Students,
		employees:     repositories.Employees,
		guardians:     repositories.Guardians,
		relationships: repositories.Relationships,
		households:    repositories.Households,
		years:         years,
		transactor:    transactor,
		clock:         systemClock,
	}
}

// RegisterPersonCommand describes a new person.
type RegisterPersonCommand struct {
	OrganisationID ids.UUID
	FirstName      string
	MiddleName     string
	LastName       string
	PreferredName  string
	Email          string
	Phone          string
	SecondaryPhone string
	DateOfBirth    *time.Time
	Gender         string
	Nationality    string
	NationalID     string
	Address        domain.Address
}

// RegisterPerson adds a person to an organisation. It records who somebody is
// and nothing about what they do at the school: roles are assigned separately,
// because a person can hold several and can gain or lose them between years.
func (s *Service) RegisterPerson(ctx context.Context, command RegisterPersonCommand) (*domain.Person, error) {
	if command.OrganisationID.IsNil() {
		return nil, apperr.ValidationFailed("a person must belong to an organisation").
			WithDetail("field", "person.organisation_id")
	}

	person, err := domain.NewPerson(command.OrganisationID, command.FirstName, command.LastName)
	if err != nil {
		return nil, err
	}
	if err := person.SetName(command.FirstName, command.MiddleName, command.LastName); err != nil {
		return nil, err
	}
	if err := person.SetPreferredName(command.PreferredName); err != nil {
		return nil, err
	}
	if err := person.SetContact(command.Email, command.Phone, command.SecondaryPhone); err != nil {
		return nil, err
	}
	if err := person.SetIdentity(command.Gender, command.Nationality, command.NationalID); err != nil {
		return nil, err
	}
	if err := person.SetAddress(command.Address); err != nil {
		return nil, err
	}
	if command.DateOfBirth != nil {
		if err := person.SetDateOfBirth(*command.DateOfBirth); err != nil {
			return nil, err
		}
	}

	now := s.clock.Now().UTC()
	person.CreatedAt = now
	person.UpdatedAt = now

	if err := s.people.Create(ctx, person); err != nil {
		return nil, err
	}
	return person, nil
}

// GetPerson returns one person.
func (s *Service) GetPerson(ctx context.Context, organisationID, id ids.UUID) (*domain.Person, error) {
	person, err := s.people.ByID(ctx, organisationID, id)
	if err != nil {
		return nil, err
	}
	if person == nil {
		return nil, apperr.NotFound("person", id.String())
	}
	return person, nil
}

// ListPeople returns the organisation's people, optionally filtered by a search
// term and including or excluding deactivated records.
func (s *Service) ListPeople(ctx context.Context, organisationID ids.UUID, search string, includeInactive bool) ([]*domain.Person, error) {
	if organisationID.IsNil() {
		return nil, apperr.ValidationFailed("people listing requires an organisation").
			WithDetail("field", "person.organisation_id")
	}
	return s.people.List(ctx, organisationID, search, includeInactive)
}

// UpdatePersonDetailsCommand carries the editable facts about a person. A nil
// field means "leave it as it is", so a form that only edits the phone number
// does not blank the address.
type UpdatePersonDetailsCommand struct {
	OrganisationID ids.UUID
	PersonID       ids.UUID
	FirstName      *string
	MiddleName     *string
	LastName       *string
	PreferredName  *string
	Email          *string
	Phone          *string
	SecondaryPhone *string
	DateOfBirth    *time.Time
	ClearBirthDate bool
	Gender         *string
	Nationality    *string
	NationalID     *string
	Address        *domain.Address
	PhotoDocument  *ids.UUID
	ClearPhoto     bool
}

// UpdatePersonDetails edits a person. The person itself is never replaced and
// never deleted, because historical financial records must keep naming the
// person who was involved.
func (s *Service) UpdatePersonDetails(ctx context.Context, command UpdatePersonDetailsCommand) (*domain.Person, error) {
	person, err := s.GetPerson(ctx, command.OrganisationID, command.PersonID)
	if err != nil {
		return nil, err
	}

	if command.FirstName != nil || command.MiddleName != nil || command.LastName != nil {
		first, middle, last := person.FirstName, person.MiddleName, person.LastName
		if command.FirstName != nil {
			first = *command.FirstName
		}
		if command.MiddleName != nil {
			middle = *command.MiddleName
		}
		if command.LastName != nil {
			last = *command.LastName
		}
		if err := person.SetName(first, middle, last); err != nil {
			return nil, err
		}
	}

	if command.PreferredName != nil {
		if err := person.SetPreferredName(*command.PreferredName); err != nil {
			return nil, err
		}
	}
	if command.Email != nil || command.Phone != nil || command.SecondaryPhone != nil {
		email, phone, secondary := person.Email, person.Phone, person.SecondaryPhone
		if command.Email != nil {
			email = *command.Email
		}
		if command.Phone != nil {
			phone = *command.Phone
		}
		if command.SecondaryPhone != nil {
			secondary = *command.SecondaryPhone
		}
		if err := person.SetContact(email, phone, secondary); err != nil {
			return nil, err
		}
	}
	if command.Gender != nil || command.Nationality != nil || command.NationalID != nil {
		gender, nationality, nationalID := person.Gender, person.Nationality, person.NationalID
		if command.Gender != nil {
			gender = *command.Gender
		}
		if command.Nationality != nil {
			nationality = *command.Nationality
		}
		if command.NationalID != nil {
			nationalID = *command.NationalID
		}
		if err := person.SetIdentity(gender, nationality, nationalID); err != nil {
			return nil, err
		}
	}
	if command.Address != nil {
		if err := person.SetAddress(*command.Address); err != nil {
			return nil, err
		}
	}

	switch {
	case command.ClearBirthDate:
		person.ClearDateOfBirth()
	case command.DateOfBirth != nil:
		if err := person.SetDateOfBirth(*command.DateOfBirth); err != nil {
			return nil, err
		}
	}

	switch {
	case command.ClearPhoto:
		person.SetPhoto(nil)
	case command.PhotoDocument != nil:
		person.SetPhoto(command.PhotoDocument)
	}

	person.UpdatedAt = s.clock.Now().UTC()
	if err := s.people.Update(ctx, person); err != nil {
		return nil, err
	}
	return person, nil
}

// AssignRoleCommand gives a person a role in an academic year.
type AssignRoleCommand struct {
	OrganisationID ids.UUID
	PersonID       ids.UUID
	// AcademicYearID may be left nil to use the organisation's current year,
	// which is what a bursar almost always wants.
	AcademicYearID ids.UUID
	Role           domain.Role
	StartsOn       *time.Time
	EndsOn         *time.Time

	// Role-specific details, applied only for the matching role.
	StudentNumber  string
	AdmissionDate  *time.Time
	StudentStatus  domain.StudentStatus
	PreviousSchool string
	IsBoarding     bool
	EmployeeNumber string
	JobTitle       string
	Department     string
	HiredOn        *time.Time
	// EndedOn is when employment finished, which is not the same as the role's
	// own end date: a person can stop being an employee and keep some other role
	// in the same year.
	EndedOn        *time.Time
	PayrollGroup   string
	ContractType   domain.ContractType
	GuardianRights *GuardianRights
}

// GuardianRights are the permissions a guardian has over students.
type GuardianRights struct {
	IsEmergencyContact  bool
	MayCollectStudent   bool
	IsBillingContact    bool
	CanAuthoriseMedical bool
}

// AssignRole gives a person a role in an academic year and creates the profile
// that belongs to that role.
//
// A person may hold the same role in different years, and several different
// roles in the same year. That is the whole point of the model: a teacher who is
// also a parent is one person with a STUDENT-free EMPLOYEE role and a GUARDIAN
// role, not two people and not one person with an ambiguous "type".
func (s *Service) AssignRole(ctx context.Context, command AssignRoleCommand) (*domain.PersonRole, error) {
	if command.OrganisationID.IsNil() {
		return nil, apperr.ValidationFailed("a role must belong to an organisation").
			WithDetail("field", "person_role.organisation_id")
	}
	if command.PersonID.IsNil() {
		return nil, apperr.ValidationFailed("a role must belong to a person").
			WithDetail("field", "person_role.person_id")
	}

	// Confirm the person exists in this organisation before giving them a role,
	// so a typo cannot create a role for a person in another school.
	if _, err := s.GetPerson(ctx, command.OrganisationID, command.PersonID); err != nil {
		return nil, err
	}

	academicYearID, err := s.resolveYear(ctx, command.OrganisationID, command.AcademicYearID)
	if err != nil {
		return nil, err
	}

	existing, err := s.roles.ByPersonAndRole(ctx, command.OrganisationID, command.PersonID, academicYearID, command.Role)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperr.AlreadyExists("person role", "role in this academic year",
			string(command.Role)+" ("+academicYearID.String()+")")
	}

	role, err := domain.NewPersonRole(command.OrganisationID, command.PersonID, academicYearID, command.Role)
	if err != nil {
		return nil, err
	}
	if err := role.SetPeriod(command.StartsOn, command.EndsOn); err != nil {
		return nil, err
	}

	now := s.clock.Now().UTC()
	role.CreatedAt = now
	role.UpdatedAt = now

	err = s.transactor.InTx(ctx, func(ctx context.Context) error {
		if err := s.roles.Create(ctx, role); err != nil {
			return err
		}
		return s.writeProfile(ctx, command, role, now)
	})
	if err != nil {
		return nil, err
	}
	return role, nil
}

// resolveYear returns the requested academic year, or the organisation's current
// one when the caller did not name a year.
func (s *Service) resolveYear(ctx context.Context, organisationID, academicYearID ids.UUID) (ids.UUID, error) {
	if academicYearID.IsNil() {
		current, err := s.years.CurrentYearID(ctx, organisationID)
		if err != nil {
			return ids.UUID{}, err
		}
		if current.IsNil() {
			return ids.UUID{}, apperr.New(apperr.CodeNotFound, "this organisation has no current academic year; open one before assigning roles").
				WithDetail("entity", "academic_year")
		}
		return current, nil
	}

	belongs, err := s.years.BelongsToOrganisation(ctx, organisationID, academicYearID)
	if err != nil {
		return ids.UUID{}, err
	}
	if !belongs {
		// Reported as not-found rather than forbidden: a year from another
		// tenant must be indistinguishable from one that does not exist.
		return ids.UUID{}, apperr.NotFound("academic year", academicYearID.String())
	}
	return academicYearID, nil
}

// writeProfile stores the profile that belongs to the assigned role.
func (s *Service) writeProfile(ctx context.Context, command AssignRoleCommand, role *domain.PersonRole, now time.Time) error {
	switch role.Role {
	case domain.RoleStudent:
		profile := domain.NewStudentProfile(role.ID, command.StudentNumber)
		if command.StudentStatus != "" {
			if err := profile.SetStatus(command.StudentStatus); err != nil {
				return err
			}
		}
		profile.PreviousSchool = command.PreviousSchool
		profile.IsBoarding = command.IsBoarding
		if command.AdmissionDate != nil {
			profile.SetAdmissionDate(*command.AdmissionDate)
		}
		profile.CreatedAt = now
		profile.UpdatedAt = now
		return s.students.Upsert(ctx, profile)

	case domain.RoleEmployee:
		profile := domain.NewEmployeeProfile(role.ID, command.EmployeeNumber)
		if command.ContractType != "" {
			if err := profile.SetContractType(command.ContractType); err != nil {
				return err
			}
		}
		if err := profile.SetEmployment(command.JobTitle, command.Department, command.PayrollGroup); err != nil {
			return err
		}
		if command.HiredOn != nil {
			profile.SetHiredOn(*command.HiredOn)
		}
		if command.EndedOn != nil {
			if err := profile.SetEndedOn(*command.EndedOn); err != nil {
				return err
			}
		}
		profile.CreatedAt = now
		profile.UpdatedAt = now
		return s.employees.Upsert(ctx, profile)

	case domain.RoleGuardian:
		profile := domain.NewGuardianProfile(role.ID)
		if command.GuardianRights != nil {
			profile.SetRights(
				command.GuardianRights.IsEmergencyContact,
				command.GuardianRights.MayCollectStudent,
				command.GuardianRights.IsBillingContact,
				command.GuardianRights.CanAuthoriseMedical,
			)
		}
		profile.CreatedAt = now
		profile.UpdatedAt = now
		return s.guardians.Upsert(ctx, profile)

	default:
		// A supplier contact has no profile in Core: the supplier itself is
		// owned by the Finance suite, which reads the person it points at.
		return nil
	}
}

// EndRole closes a role at a date, keeping the row so the year's history stays
// readable.
func (s *Service) EndRole(ctx context.Context, organisationID, roleID ids.UUID, on time.Time) error {
	role, err := s.roles.ByID(ctx, organisationID, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return apperr.NotFound("person role", roleID.String())
	}
	if err := role.End(on); err != nil {
		return err
	}
	role.UpdatedAt = s.clock.Now().UTC()
	return s.roles.Update(ctx, role)
}

// ListRoles returns every role a person holds in an academic year.
func (s *Service) ListRoles(ctx context.Context, organisationID, personID, academicYearID ids.UUID) ([]*domain.PersonRole, error) {
	year, err := s.resolveYear(ctx, organisationID, academicYearID)
	if err != nil {
		return nil, err
	}
	return s.roles.ListByPerson(ctx, organisationID, personID, year)
}

// RoleDetails is a role together with the profile that belongs to its role type.
//
// The profile lives in a different table from the role, and the one that applies
// depends on the role. Reading a role without its profile would leave a bursar
// looking at a "STUDENT" label with no student number beside it, so the two are
// read together here rather than left for each caller to reassemble.
type RoleDetails struct {
	Role     *domain.PersonRole
	Student  *domain.StudentProfile
	Employee *domain.EmployeeProfile
	Guardian *domain.GuardianProfile
}

// withProfile loads the profile that matches a role, if one exists. A missing
// profile is not an error: a role can be recorded a moment before its profile is
// written, and a supplier contact has no profile in Core at all.
func (s *Service) withProfile(ctx context.Context, role *domain.PersonRole) (*RoleDetails, error) {
	details := &RoleDetails{Role: role}

	var err error
	switch role.Role {
	case domain.RoleStudent:
		details.Student, err = s.students.ByRoleID(ctx, role.ID)
	case domain.RoleEmployee:
		details.Employee, err = s.employees.ByRoleID(ctx, role.ID)
	case domain.RoleGuardian:
		details.Guardian, err = s.guardians.ByRoleID(ctx, role.ID)
	}
	if err != nil {
		return nil, err
	}
	return details, nil
}

// GetRole returns one role of this organisation with its profile.
func (s *Service) GetRole(ctx context.Context, organisationID, roleID ids.UUID) (*RoleDetails, error) {
	role, err := s.roles.ByID(ctx, organisationID, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, apperr.NotFound("person role", roleID.String())
	}
	return s.withProfile(ctx, role)
}

// ListRoleDetails returns the roles a person holds in an academic year, each with
// the profile that belongs to it.
func (s *Service) ListRoleDetails(ctx context.Context, organisationID, personID, academicYearID ids.UUID) ([]*RoleDetails, error) {
	roles, err := s.ListRoles(ctx, organisationID, personID, academicYearID)
	if err != nil {
		return nil, err
	}

	details := make([]*RoleDetails, 0, len(roles))
	for _, role := range roles {
		withProfile, err := s.withProfile(ctx, role)
		if err != nil {
			return nil, err
		}
		details = append(details, withProfile)
	}
	return details, nil
}

// ListRolesForPeople returns the roles a set of people hold in one year, keyed by
// person. A directory page shows a role badge beside every row and the whole
// year at once, so the roles are read in one query and grouped here rather than
// by asking per person.
func (s *Service) ListRolesForPeople(ctx context.Context, organisationID ids.UUID, personIDs []ids.UUID, academicYearID ids.UUID) (map[ids.UUID][]*RoleDetails, error) {
	byPerson := make(map[ids.UUID][]*RoleDetails, len(personIDs))
	if len(personIDs) == 0 {
		return byPerson, nil
	}

	year, err := s.resolveYear(ctx, organisationID, academicYearID)
	if err != nil {
		return nil, err
	}

	roles, err := s.roles.ListByPersons(ctx, organisationID, personIDs, year)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		details, err := s.withProfile(ctx, role)
		if err != nil {
			return nil, err
		}
		byPerson[role.PersonID] = append(byPerson[role.PersonID], details)
	}
	return byPerson, nil
}

// ListPeopleWithRole returns everyone holding a role in an academic year, which
// is how a roster, a payroll run or a fee run is built.
func (s *Service) ListPeopleWithRole(ctx context.Context, organisationID, academicYearID ids.UUID, role domain.Role) ([]*domain.PersonRole, error) {
	year, err := s.resolveYear(ctx, organisationID, academicYearID)
	if err != nil {
		return nil, err
	}
	return s.roles.ListByRole(ctx, organisationID, year, role)
}
