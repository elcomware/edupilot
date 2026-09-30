package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/elcomware/edupilot/internal/app"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	academicapp "github.com/elcomware/edupilot/internal/platform/academic/application"
	organisationapp "github.com/elcomware/edupilot/internal/platform/organisation/application"
	peopleapp "github.com/elcomware/edupilot/internal/platform/people/application"
	peopledomain "github.com/elcomware/edupilot/internal/platform/people/domain"
	"github.com/elcomware/edupilot/internal/testsupport"
)

// peopleFixture is a booted installation with one open academic year, which is
// the state every people use case needs before it will accept a command.
type peopleFixture struct {
	people         *peopleapp.Service
	academic       *academicapp.Service
	organisations  *organisationapp.Service
	organisationID ids.UUID
	yearID         ids.UUID
}

func newPeopleFixture(t *testing.T) *peopleFixture {
	t.Helper()

	edupilot := newApplication(t, app.DefaultSettings(testsupport.TestDataRoot(t)))
	ctx := context.Background()

	organisation, err := edupilot.Organisations.GetCurrentOrganisation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	year, err := edupilot.Academic.CreateYear(ctx, academicapp.CreateYearCommand{
		OrganisationID: organisation.ID,
		Name:           "2025-2026",
		StartsOn:       time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:         time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		MakeCurrent:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	return &peopleFixture{
		people:         edupilot.People,
		academic:       edupilot.Academic,
		organisations:  edupilot.Organisations,
		organisationID: organisation.ID,
		yearID:         year.ID,
	}
}

// addYear opens another academic year and returns its identifier.
func (f *peopleFixture) addYear(t *testing.T, name string, startsOn, endsOn time.Time) ids.UUID {
	t.Helper()

	year, err := f.academic.CreateYear(context.Background(), academicapp.CreateYearCommand{
		OrganisationID: f.organisationID,
		Name:           name,
		StartsOn:       startsOn,
		EndsOn:         endsOn,
		MakeCurrent:    true,
	})
	if err != nil {
		t.Fatalf("opening %s: %v", name, err)
	}
	return year.ID
}

func (f *peopleFixture) register(t *testing.T, first, last, email string) ids.UUID {
	t.Helper()

	person, err := f.people.RegisterPerson(context.Background(), peopleapp.RegisterPersonCommand{
		OrganisationID: f.organisationID,
		FirstName:      first,
		LastName:       last,
		Email:          email,
		Phone:          "+2250700000000",
		Address: peopledomain.Address{
			Line1:   "Rue des Palmiers",
			City:    "Abidjan",
			Country: "CI",
		},
	})
	if err != nil {
		t.Fatalf("registering %s %s: %v", first, last, err)
	}
	return person.ID
}

func (f *peopleFixture) assign(t *testing.T, personID ids.UUID, command peopleapp.AssignRoleCommand) *peopledomain.PersonRole {
	t.Helper()

	command.OrganisationID = f.organisationID
	command.PersonID = personID
	role, err := f.people.AssignRole(context.Background(), command)
	if err != nil {
		t.Fatalf("assigning role %s: %v", command.Role, err)
	}
	return role
}

func (f *peopleFixture) link(t *testing.T, from, to ids.UUID, relationshipType peopledomain.RelationshipType, role peopledomain.RelationshipRole, primary bool) *peopledomain.Relationship {
	t.Helper()

	relationship, err := f.people.Link(context.Background(), peopleapp.LinkRelationshipCommand{
		OrganisationID: f.organisationID,
		FromPersonID:   from,
		ToPersonID:     to,
		Type:           relationshipType,
		Role:           role,
		MakePrimary:    primary,
	})
	if err != nil {
		t.Fatalf("linking %s %s->%s: %v", relationshipType, from, to, err)
	}
	return relationship
}

// TestPeopleUseCasesAreWiredEndToEnd drives the whole people slice against a real
// migrated SQLite database. Every repository statement, column name and foreign
// key is exercised here for the first time; the domain tests cannot catch a
// mistyped column, and a broken statement would otherwise only surface at runtime.
func TestPeopleUseCasesAreWiredEndToEnd(t *testing.T) {
	fixture := newPeopleFixture(t)
	ctx := context.Background()

	// One identity holding two different roles in the same year is the central
	// claim of ADR-011: a person is not a type.
	teacher := fixture.register(t, "Awa", "Traore", "awa.traore@ecole.ci")
	fixture.assign(t, teacher, peopleapp.AssignRoleCommand{
		Role:           peopledomain.RoleEmployee,
		EmployeeNumber: "EMP-0001",
		JobTitle:       "Professeur",
	})
	fixture.assign(t, teacher, peopleapp.AssignRoleCommand{
		Role: peopledomain.RoleGuardian,
	})

	roles, err := fixture.people.ListRoles(ctx, fixture.organisationID, teacher, ids.UUID{})
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 2 {
		t.Fatalf("role count = %d, want 2 roles on one identity", len(roles))
	}

	pupil := fixture.register(t, "Mamadou", "Kone", "mamadou@ecole.ci")
	fixture.assign(t, pupil, peopleapp.AssignRoleCommand{
		Role:          peopledomain.RoleStudent,
		StudentNumber: "STU-0001",
		StudentStatus: peopledomain.StudentEnrolled,
	})

	// An employee who is also a guardian is just an edge: no new table, no new type.
	first := fixture.link(t, teacher, pupil,
		peopledomain.RelationshipGuardianOf, peopledomain.RelationshipMother, true)
	if !first.IsPrimary {
		t.Error("the billing guardian was not marked primary")
	}

	guardians, err := fixture.people.GuardiansOf(ctx, fixture.organisationID, pupil, ids.UUID{})
	if err != nil {
		t.Fatal(err)
	}
	if len(guardians) != 1 || guardians[0].FromPersonID != first.FromPersonID {
		t.Fatalf("the pupil has %d guardians, want the one that was linked", len(guardians))
	}

	dependants, err := fixture.people.DependantsOf(ctx, fixture.organisationID, teacher, ids.UUID{})
	if err != nil {
		t.Fatal(err)
	}
	if len(dependants) != 1 || dependants[0].ToPersonID != first.ToPersonID {
		t.Fatalf("the guardian has %d dependants, want one", len(dependants))
	}

	// Promoting a second guardian must demote the first, or Finance would have
	// two billing guardians for one pupil.
	second := fixture.register(t, "Ibrahim", "Traore", "ibrahim@ecole.ci")
	fixture.assign(t, second, peopleapp.AssignRoleCommand{Role: peopledomain.RoleGuardian})
	fixture.link(t, second, pupil,
		peopledomain.RelationshipGuardianOf, peopledomain.RelationshipFather, true)

	guardians, err = fixture.people.GuardiansOf(ctx, fixture.organisationID, pupil, ids.UUID{})
	if err != nil {
		t.Fatal(err)
	}
	if len(guardians) != 2 {
		t.Fatalf("guardian count = %d, want 2: demotion must not delete the edge", len(guardians))
	}
	primaries := 0
	for _, guardian := range guardians {
		if !guardian.IsPrimary {
			continue
		}
		primaries++
		if guardian.FromPersonID != second {
			t.Error("the wrong guardian is primary")
		}
	}
	if primaries != 1 {
		t.Errorf("primary guardian count = %d, want exactly 1", primaries)
	}

	// A household is an invoicing party, not a family tree. This path inserts the
	// parent before its members, so a foreign key failure here means the write
	// order regressed.
	household, err := fixture.people.CreateHousehold(ctx, peopleapp.CreateHouseholdCommand{
		OrganisationID: fixture.organisationID,
		Name:           "Famille Traore",
		BillingEmail:   "facturation@traore.ci",
		BillingPhone:   "+2250711223333",
		BillingAddress: peopledomain.Address{Line1: "Cocody Riviera", City: "Abidjan", Country: "CI"},
		Members: []peopleapp.HouseholdMemberCommand{
			{PersonID: teacher, Role: peopledomain.MemberHead, IsBillingPart: true},
			{PersonID: second, Role: peopledomain.MemberPartner},
			{PersonID: pupil, Role: peopledomain.MemberDependent},
		},
	})
	if err != nil {
		t.Fatalf("creating the household: %v", err)
	}
	if len(household.Members) != 3 {
		t.Errorf("member count = %d, want 3", len(household.Members))
	}

	reloaded, err := fixture.people.GetHousehold(ctx, fixture.organisationID, household.ID)
	if err != nil {
		t.Fatalf("reloading the household: %v", err)
	}
	if reloaded.Name != "Famille Traore" || reloaded.BillingEmail != "facturation@traore.ci" {
		t.Errorf("household = %+v, want the persisted billing party", reloaded)
	}
	if len(reloaded.Members) != 3 {
		t.Errorf("reloaded member count = %d, want 3", len(reloaded.Members))
	}

	households, err := fixture.people.ListHouseholds(ctx, fixture.organisationID, ids.UUID{})
	if err != nil {
		t.Fatal(err)
	}
	if len(households) != 1 {
		t.Fatalf("household listing = %d, want 1", len(households))
	}

	// Unlinking ends the edge without deleting it, so a statement printed last
	// year still resolves the guardian who was on it then.
	if err := fixture.people.Unlink(ctx, fixture.organisationID, first.ID); err != nil {
		t.Fatalf("unlinking: %v", err)
	}
	remaining, err := fixture.people.GuardiansOf(ctx, fixture.organisationID, pupil, ids.UUID{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, guardian := range remaining {
		if guardian.ID != first.ID {
			continue
		}
		found = true
		if guardian.IsActive {
			t.Error("the unlinked edge is still active")
		}
	}
	if !found {
		t.Error("unlinking deleted the edge instead of ending it")
	}
}

// TestPeopleAreScopedToTheTenant checks that neither a search nor a bare
// identifier can reach another organisation's people, and that a LIKE wildcard
// typed into the search box is treated as a literal character.
func TestPeopleAreScopedToTheTenant(t *testing.T) {
	fixture := newPeopleFixture(t)
	ctx := context.Background()

	other, err := fixture.organisations.CreateOrganisation(ctx, organisationapp.CreateOrganisationCommand{
		Name:          "Ecole Deux",
		CurrencyCode:  "XOF",
		DefaultLocale: "fr",
		Country:       "CI",
	})
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := fixture.people.RegisterPerson(ctx, peopleapp.RegisterPersonCommand{
		OrganisationID: other.ID,
		FirstName:      "Zeta",
		LastName:       "Zebra",
		Email:          "zeta@ecole2.ci",
	})
	if err != nil {
		t.Fatal(err)
	}

	// The stranger exists but must be invisible from the first tenant.
	people, err := fixture.people.ListPeople(ctx, fixture.organisationID, "Zebra", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 0 {
		t.Errorf("the first tenant sees %d people from another tenant", len(people))
	}
	if _, err := fixture.people.GetPerson(ctx, fixture.organisationID, stranger.ID); err == nil {
		t.Error("a person from another tenant was readable by identifier alone")
	}

	// A wildcard typed into the search box must be treated as a literal
	// character, not as a pattern that matches the whole list.
	fixture.register(t, "Ada", "Percent", "ada@ecole.ci")
	people, err = fixture.people.ListPeople(ctx, fixture.organisationID, "%", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 0 {
		t.Errorf("a literal %% matched %d people; the wildcard was not escaped", len(people))
	}

	// The same holds for the single-character wildcard.
	people, err = fixture.people.ListPeople(ctx, fixture.organisationID, "_", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 0 {
		t.Errorf("a literal _ matched %d people; the wildcard was not escaped", len(people))
	}

	// An ordinary term still finds the person, so escaping did not break search.
	people, err = fixture.people.ListPeople(ctx, fixture.organisationID, "Percent", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 1 || people[0].FirstName != "Ada" {
		t.Errorf("searching a plain term returned %d people, want 1", len(people))
	}
}

// TestRolesAreScopedToTheAcademicYear checks the year scoping that lets a school
// carry a pupil forward without carrying their guardian list forward too.
func TestRolesAreScopedToTheAcademicYear(t *testing.T) {
	fixture := newPeopleFixture(t)
	ctx := context.Background()

	guardian := fixture.register(t, "Awa", "Traore", "awa@ecole.ci")
	fixture.assign(t, guardian, peopleapp.AssignRoleCommand{Role: peopledomain.RoleGuardian})

	next := fixture.addYear(t, "2026-2027",
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 31, 0, 0, 0, 0, time.UTC))
	if next == fixture.yearID {
		t.Fatal("the second year reused the first identifier")
	}

	roles, err := fixture.people.ListRoles(ctx, fixture.organisationID, guardian, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 0 {
		t.Errorf("the guardian carries %d roles into a new year, want 0", len(roles))
	}

	// An academic year belonging to another tenant is indistinguishable from one
	// that does not exist, so a role cannot be filed against a foreign calendar.
	_, err = fixture.people.ListRoles(ctx, ids.New(), guardian, next)
	if err == nil {
		t.Error("a foreign organisation read the year-scoped role listing")
	}
}
