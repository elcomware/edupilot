// Package ports defines the repository interfaces the people domain and
// application layers depend on. As with every EduPilot module, the dependency
// arrow points inward: infrastructure implements these, nothing here imports it.
package ports

import (
	"context"

	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

// PersonRepository persists people. Every read and write is scoped to an
// organisation, so one school can never reach another's people by passing a
// different identifier.
type PersonRepository interface {
	Create(ctx context.Context, person *domain.Person) error
	Update(ctx context.Context, person *domain.Person) error
	ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Person, error)
	// List returns people ordered by last name, optionally filtered by a search
	// term over name, preferred name and email.
	List(ctx context.Context, organisationID ids.UUID, search string, includeInactive bool) ([]*domain.Person, error)
	// FindByNationalID returns the person holding a national identifier, or nil.
	// It is how an import recognises somebody who already exists.
	FindByNationalID(ctx context.Context, organisationID ids.UUID, nationalID string) (*domain.Person, error)
	Count(ctx context.Context, organisationID ids.UUID) (int64, error)
}

// RoleRepository persists the roles a person holds in an academic year, together
// with the profile that belongs to each role.
type RoleRepository interface {
	Create(ctx context.Context, role *domain.PersonRole) error
	Update(ctx context.Context, role *domain.PersonRole) error
	ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.PersonRole, error)
	// ByPersonAndRole returns the person's role for one year, or nil.
	ByPersonAndRole(ctx context.Context, organisationID, personID, academicYearID ids.UUID, role domain.Role) (*domain.PersonRole, error)
	// ListByPerson returns every role the person holds in one year.
	ListByPerson(ctx context.Context, organisationID, personID, academicYearID ids.UUID) ([]*domain.PersonRole, error)

	// ListByPersons returns the roles those people hold in one year. A directory
	// page shows a badge beside every row, so loading the roles one person at a
	// time turns one page into a query per row.
	ListByPersons(ctx context.Context, organisationID ids.UUID, personIDs []ids.UUID, academicYearID ids.UUID) ([]*domain.PersonRole, error)
	// ListByRole returns everyone holding a role in one year, which is how a
	// roster or a payroll run is built.
	ListByRole(ctx context.Context, organisationID, academicYearID ids.UUID, role domain.Role) ([]*domain.PersonRole, error)
	// CountByRole returns how many people hold a role in one year.
	CountByRole(ctx context.Context, organisationID, academicYearID ids.UUID, role domain.Role) (int64, error)
	// ClearPrimaryGuardian demotes every primary guardian edge of a student in
	// one year except one, so promoting a new billing guardian is a
	// single-writer operation. It lives here because a guardian edge is
	// identified by the student's role, not by the person's.
	ClearPrimaryGuardian(ctx context.Context, organisationID, toPersonID, academicYearID, keep ids.UUID) error
}

// StudentProfileRepository persists the student-specific facts of a role.
type StudentProfileRepository interface {
	Upsert(ctx context.Context, profile *domain.StudentProfile) error
	ByRoleID(ctx context.Context, roleID ids.UUID) (*domain.StudentProfile, error)
	// FindByNumber returns the student profile with a student number, or nil.
	FindByNumber(ctx context.Context, number string) (*domain.StudentProfile, error)
}

// EmployeeProfileRepository persists the employee-specific facts of a role.
type EmployeeProfileRepository interface {
	Upsert(ctx context.Context, profile *domain.EmployeeProfile) error
	ByRoleID(ctx context.Context, roleID ids.UUID) (*domain.EmployeeProfile, error)
	// FindByNumber returns the employee profile with an employee number, or nil.
	FindByNumber(ctx context.Context, number string) (*domain.EmployeeProfile, error)
}

// GuardianProfileRepository persists the rights a guardian has over students.
type GuardianProfileRepository interface {
	Upsert(ctx context.Context, profile *domain.GuardianProfile) error
	ByRoleID(ctx context.Context, roleID ids.UUID) (*domain.GuardianProfile, error)
}

// RelationshipRepository persists the directed edges between people.
type RelationshipRepository interface {
	Create(ctx context.Context, relationship *domain.Relationship) error
	Update(ctx context.Context, relationship *domain.Relationship) error
	ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Relationship, error)
	// Outgoing returns the edges leaving a person, optionally filtered by type.
	Outgoing(ctx context.Context, organisationID, academicYearID, personID ids.UUID, relationshipType domain.RelationshipType) ([]*domain.Relationship, error)
	// Incoming returns the edges arriving at a person, optionally filtered by
	// type. Incoming GUARDIAN_OF edges are how a student's guardians are found.
	Incoming(ctx context.Context, organisationID, academicYearID, personID ids.UUID, relationshipType domain.RelationshipType) ([]*domain.Relationship, error)
	// FindSameEdge returns an existing edge between the same two people of the
	// same type and label, or nil, so linking twice is a clear error rather
	// than a duplicate.
	FindSameEdge(ctx context.Context, academicYearID, fromPersonID, toPersonID ids.UUID, relationshipType domain.RelationshipType, role domain.RelationshipRole) (*domain.Relationship, error)
}

// HouseholdRepository persists billing parties and their membership.
type HouseholdRepository interface {
	Create(ctx context.Context, household *domain.Household) error
	Update(ctx context.Context, household *domain.Household) error
	ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Household, error)
	ListByYear(ctx context.Context, organisationID, academicYearID ids.UUID) ([]*domain.Household, error)
	// AddMember adds one person to a household.
	AddMember(ctx context.Context, member domain.HouseholdMember) error
	// HouseholdsForPerson returns the households a person belongs to in a year.
	HouseholdsForPerson(ctx context.Context, organisationID, personID, academicYearID ids.UUID) ([]*domain.Household, error)
}
