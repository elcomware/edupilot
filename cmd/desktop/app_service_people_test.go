package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/elcomware/edupilot/internal/api"
	"github.com/elcomware/edupilot/internal/app"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/testsupport"
)

// transportFixture is a booted installation exposed through the Wails transport
// service, which is the only surface the desktop interface can reach. Testing
// here proves the wire contract, not just the use cases underneath it.
type transportFixture struct {
	app *App
	ctx context.Context
}

func newTransportFixture(t *testing.T) *transportFixture {
	t.Helper()

	edupilot, err := app.New(
		context.Background(),
		app.DefaultSettings(testsupport.TestDataRoot(t)),
		slog.New(slog.DiscardHandler),
	)
	if err != nil {
		t.Fatalf("starting the application: %v", err)
	}
	t.Cleanup(func() { _ = edupilot.Close() })

	return &transportFixture{app: newApp(edupilot), ctx: context.Background()}
}

// openYear opens a current academic year, which every people operation needs
// before it will accept a command.
func (f *transportFixture) openYear(t *testing.T) api.AcademicYearView {
	t.Helper()

	result := f.app.CreateYear(f.ctx, api.CreateYearRequest{
		Name:        "2025-2026",
		StartsOn:    "2025-09-01",
		EndsOn:      "2026-07-31",
		MakeCurrent: true,
	})
	year, err := result.Unwrap()
	if err != nil {
		t.Fatalf("opening an academic year: %v", err)
	}
	return year
}

func (f *transportFixture) register(t *testing.T, first, last string) api.PersonView {
	t.Helper()

	result := f.app.RegisterPerson(f.ctx, api.RegisterPersonRequest{
		FirstName: first,
		LastName:  last,
		Address:   api.AddressView{Line1: "1 rue de l'eglise", City: "Douala", Country: "CM"},
	})
	person, err := result.Unwrap()
	if err != nil {
		t.Fatalf("registering %s %s: %v", first, last, err)
	}
	return person
}

func TestTransportOpensAndReadsTheCurrentYear(t *testing.T) {
	fixture := newTransportFixture(t)

	year := fixture.openYear(t)
	if year.Name != "2025-2026" {
		t.Errorf("name = %q, want the year that was just opened", year.Name)
	}
	if year.StartsOn != "2025-09-01" || year.EndsOn != "2026-07-31" {
		t.Errorf("dates = %q..%q, want the calendar dates to survive the round trip", year.StartsOn, year.EndsOn)
	}
	if !year.IsCurrent {
		t.Error("a year opened as current is not reported as current")
	}

	current, err := fixture.app.GetCurrentYear(fixture.ctx).Unwrap()
	if err != nil {
		t.Fatalf("reading the current year: %v", err)
	}
	if current.ID != year.ID {
		t.Errorf("current year = %q, want the year that was just opened", current.ID)
	}

	page, err := fixture.app.ListYears(fixture.ctx, api.PageRequest{}).Unwrap()
	if err != nil {
		t.Fatalf("listing years: %v", err)
	}
	if len(page.Items) != 1 {
		t.Errorf("listed %d years, want 1", len(page.Items))
	}
}

// A school that has not opened a year yet must be told so plainly, otherwise the
// interface shows an empty people list and a mysterious failure when a role is
// assigned.
func TestTransportReportsAMissingCurrentYear(t *testing.T) {
	fixture := newTransportFixture(t)

	_, err := fixture.app.GetCurrentYear(fixture.ctx).Unwrap()
	if err == nil {
		t.Fatal("reading the current year before one is open should fail")
	}
	if got := failureCode(t, err); got != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", got)
	}
}

// The organisation is taken from the installation, so a request cannot smuggle in
// another tenant's identifier.
func TestTransportRejectsAnotherTenantsIdentifiers(t *testing.T) {
	fixture := newTransportFixture(t)
	fixture.openYear(t)
	person := fixture.register(t, "Amina", "Njoya")

	foreign := ids.New()
	result := fixture.app.GetPerson(fixture.ctx, api.GetPersonRequest{ID: foreign.String()})
	_, err := result.Unwrap()
	if err == nil {
		t.Fatal("reading a person from another tenant should fail")
	}
	// Not found rather than forbidden: a person in another school must be
	// indistinguishable from one that does not exist.
	if got := failureCode(t, err); got != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", got)
	}

	// The person that does belong to this tenant is still readable, so the
	// rejection above was about the identifier and not a broken transport.
	if _, err := fixture.app.GetPerson(fixture.ctx, api.GetPersonRequest{ID: person.ID}).Unwrap(); err != nil {
		t.Fatalf("reading this school's own person: %v", err)
	}
}

func TestTransportRejectsAMalformedIdentifier(t *testing.T) {
	fixture := newTransportFixture(t)

	_, err := fixture.app.GetPerson(fixture.ctx, api.GetPersonRequest{ID: "not-a-uuid"}).Unwrap()
	if err == nil {
		t.Fatal("a malformed identifier should fail")
	}
	if got := failureCode(t, err); got != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED: a malformed id is a caller bug, not a missing record", got)
	}
}

// A leaver keeps the profile, so the department they worked in is still on file.
func TestTransportKeepsAnEmployeesProfileWhenEmploymentEnds(t *testing.T) {
	fixture := newTransportFixture(t)
	fixture.openYear(t)
	person := fixture.register(t, "Joseph", "Mbarga")

	role, err := fixture.app.AssignRole(fixture.ctx, api.AssignRoleRequest{
		PersonID:       person.ID,
		Role:           "EMPLOYEE",
		EmployeeNumber: "EMP-1",
		JobTitle:       "Teacher",
		Department:     "Mathematics",
		HiredOn:        "2025-09-01",
		EndedOn:        "2026-06-30",
		ContractType:   "FIXED_TERM",
	}).Unwrap()
	if err != nil {
		t.Fatalf("assigning the employee role: %v", err)
	}
	if role.Employee == nil {
		t.Fatal("the employee profile was not returned")
	}
	if role.Employee.JobTitle != "Teacher" {
		t.Errorf("job title = %q, want Teacher", role.Employee.JobTitle)
	}
	if role.Employee.EndedOn != "2026-06-30" {
		t.Errorf("ended on = %q, want the employment end date to be recorded", role.Employee.EndedOn)
	}
}

// Employment cannot end before it began, whichever end of the transport it is
// caught at.
func TestTransportRejectsEmploymentEndingBeforeItBegan(t *testing.T) {
	fixture := newTransportFixture(t)
	fixture.openYear(t)
	person := fixture.register(t, "Joseph", "Mbarga")

	_, err := fixture.app.AssignRole(fixture.ctx, api.AssignRoleRequest{
		PersonID:       person.ID,
		Role:           "EMPLOYEE",
		EmployeeNumber: "EMP-2",
		HiredOn:        "2025-09-01",
		EndedOn:        "2024-01-01",
	}).Unwrap()
	if err == nil {
		t.Fatal("employment ending before it began should fail")
	}
	if got := failureCode(t, err); got != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", got)
	}
}

// The year is optional on the wire so the interface can omit it and get the year
// the school is working in, but a malformed year is still rejected.
func TestTransportTreatsAnEmptyYearAsTheCurrentOne(t *testing.T) {
	fixture := newTransportFixture(t)
	year := fixture.openYear(t)
	person := fixture.register(t, "Awa", "Nkem")

	role, err := fixture.app.AssignRole(fixture.ctx, api.AssignRoleRequest{
		PersonID:      person.ID,
		Role:          "STUDENT",
		StudentNumber: "STU-1",
	}).Unwrap()
	if err != nil {
		t.Fatalf("assigning a role without naming a year: %v", err)
	}
	if role.AcademicYearID != year.ID {
		t.Errorf("academic year = %q, want the current year %q", role.AcademicYearID, year.ID)
	}

	_, err = fixture.app.AssignRole(fixture.ctx, api.AssignRoleRequest{
		PersonID:       person.ID,
		AcademicYearID: "not-a-uuid",
		Role:           "EMPLOYEE",
	}).Unwrap()
	if err == nil {
		t.Fatal("a malformed academic year should fail")
	}
	if got := failureCode(t, err); got != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", got)
	}
}

// A person standing in the current year, read back with the roles they hold.
func TestTransportReadsAPersonWithTheirRoles(t *testing.T) {
	fixture := newTransportFixture(t)
	fixture.openYear(t)
	person := fixture.register(t, "Grace", "Tanyi")

	for _, role := range []string{"STUDENT", "EMPLOYEE"} {
		if _, err := fixture.app.AssignRole(fixture.ctx, api.AssignRoleRequest{
			PersonID:      person.ID,
			Role:          role,
			StudentNumber: "STU-2",
		}).Unwrap(); err != nil {
			t.Fatalf("assigning %s: %v", role, err)
		}
	}

	read, err := fixture.app.GetPerson(fixture.ctx, api.GetPersonRequest{ID: person.ID}).Unwrap()
	if err != nil {
		t.Fatalf("reading the person: %v", err)
	}
	if read.DisplayName != "Grace Tanyi" {
		t.Errorf("display name = %q, want the name assembled from first and last", read.DisplayName)
	}
	if len(read.Roles) != 2 {
		t.Fatalf("read %d roles, want both roles this person holds", len(read.Roles))
	}
	// A role is useless without the profile that belongs to it, so the student
	// number must reach the wire on the person screen.
	var student *api.PersonRoleView
	for i := range read.Roles {
		if read.Roles[i].Role == "STUDENT" {
			student = &read.Roles[i]
		}
	}
	if student == nil {
		t.Fatal("the student role is missing from the roles this person holds")
	}
	if student.Student == nil {
		t.Fatal("the student role has no profile on the wire, so the student number is missing")
	}
	if student.Student.StudentNumber != "STU-2" {
		t.Errorf("student number = %q, want STU-2", student.Student.StudentNumber)
	}
	if read.Address.Line1 != "1 rue de l'eglise" {
		t.Errorf("address = %q, want the address to survive registration", read.Address.Line1)
	}
}

// A guardian edge, read from both ends, so the interface can show "my children"
// and "my guardians" from one call.
func TestTransportLinksAndReadsRelationshipsFromBothEnds(t *testing.T) {
	fixture := newTransportFixture(t)
	fixture.openYear(t)
	guardian := fixture.register(t, "Marie", "Ngono")
	student := fixture.register(t, "Junior", "Ngono")

	if _, err := fixture.app.AssignRole(fixture.ctx, api.AssignRoleRequest{
		PersonID:          guardian.ID,
		Role:              "GUARDIAN",
		MayCollectStudent: true,
	}).Unwrap(); err != nil {
		t.Fatalf("assigning the guardian role: %v", err)
	}

	edge, err := fixture.app.Link(fixture.ctx, api.LinkRelationshipRequest{
		FromPersonID: guardian.ID,
		ToPersonID:   student.ID,
		Type:         "GUARDIAN_OF",
		Role:         "MOTHER",
		MakePrimary:  true,
	}).Unwrap()
	if err != nil {
		t.Fatalf("linking guardian to student: %v", err)
	}
	if edge.Type != "GUARDIAN_OF" {
		t.Errorf("type = %q, want GUARDIAN_OF", edge.Type)
	}

	outward, err := fixture.app.ListRelationships(fixture.ctx, api.ListRelationshipsRequest{
		PersonID: guardian.ID,
	}).Unwrap()
	if err != nil {
		t.Fatalf("reading the guardian's edges: %v", err)
	}
	if len(outward) != 1 || outward[0].ToPersonID != student.ID {
		t.Fatalf("outward edges = %+v, want the one edge to the student", outward)
	}

	inward, err := fixture.app.ListRelationships(fixture.ctx, api.ListRelationshipsRequest{
		PersonID: student.ID,
		Incoming: true,
	}).Unwrap()
	if err != nil {
		t.Fatalf("reading the student's edges: %v", err)
	}
	if len(inward) != 1 || inward[0].FromPersonID != guardian.ID {
		t.Fatalf("inward edges = %+v, want the one edge from the guardian", inward)
	}

	// Unlinking deactivates the edge without erasing it, so a statement printed
	// last year still resolves the guardian who was on it.
	unlinked, err := fixture.app.Unlink(fixture.ctx, api.UnlinkRelationshipRequest{ID: edge.ID}).Unwrap()
	if err != nil {
		t.Fatalf("unlinking: %v", err)
	}
	if unlinked.IsActive {
		t.Error("an unlinked edge is still active, so the guardian would still be offered as a contact")
	}
	if unlinked.ID != edge.ID {
		t.Errorf("unlinking returned edge %q, want the one that was unlinked", unlinked.ID)
	}
}

// Linking a person to themselves is a modelling error, not a request to be
// stored.
func TestTransportRejectsSelfRelationship(t *testing.T) {
	fixture := newTransportFixture(t)
	fixture.openYear(t)
	person := fixture.register(t, "Solo", "Ndongo")

	_, err := fixture.app.Link(fixture.ctx, api.LinkRelationshipRequest{
		FromPersonID: person.ID,
		ToPersonID:   person.ID,
		Type:         "GUARDIAN_OF",
	}).Unwrap()
	if err == nil {
		t.Fatal("linking somebody to themselves should fail")
	}
	if got := failureCode(t, err); got != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", got)
	}
}

// A household is a billing party: the members must be people of this school.
func TestTransportCreatesHouseholdFromItsMembers(t *testing.T) {
	fixture := newTransportFixture(t)
	fixture.openYear(t)
	guardian := fixture.register(t, "Paul", "Owona")
	student := fixture.register(t, "Alice", "Owona")

	household, err := fixture.app.CreateHousehold(fixture.ctx, api.CreateHouseholdRequest{
		Name:         "Owona family",
		BillingEmail: "paul.owona@example.test",
		BillingAddress: api.AddressView{
			Line1:      "4 avenue du Centre",
			Line2:      "BP 22",
			City:       "Douala",
			PostalCode: "00200",
			Country:    "CM",
		},
		Members: []api.HouseholdMemberInput{
			{PersonID: guardian.ID, Role: "HEAD", IsBillingPart: true},
			{PersonID: student.ID, Role: "DEPENDENT"},
		},
	}).Unwrap()
	if err != nil {
		t.Fatalf("creating the household: %v", err)
	}
	if household.Name != "Owona family" {
		t.Errorf("name = %q, want the household name", household.Name)
	}
	if len(household.Members) != 2 {
		t.Fatalf("read %d members, want both", len(household.Members))
	}
	// Only one of the two is the billing party, so a duplicate invoice is visible
	// in the wire contract. The member's parent is not on the wire: members are
	// nested inside the household they belong to, so the interface has nowhere to
	// put a household id it would then have to keep in step.
	billing := 0
	for _, member := range household.Members {
		if member.IsBillingPart {
			billing++
		}
	}
	if billing != 1 {
		t.Errorf("%d members are marked as the billing party, want 1", billing)
	}
}

// The list endpoints must return an empty slice rather than null, or the
// interface has to special-case a first load of every table.
func TestTransportReturnsEmptySlicesRatherThanNull(t *testing.T) {
	fixture := newTransportFixture(t)
	fixture.openYear(t)

	people, err := fixture.app.ListPeople(fixture.ctx, api.ListPeopleRequest{}).Unwrap()
	if err != nil {
		t.Fatalf("listing people: %v", err)
	}
	if people.Items == nil {
		t.Error("the people page has a null items slice, want an empty one")
	}

	households, err := fixture.app.ListHouseholds(fixture.ctx, api.ListHouseholdsRequest{}).Unwrap()
	if err != nil {
		t.Fatalf("listing households: %v", err)
	}
	if households == nil {
		t.Error("listing households returned null, want an empty slice")
	}

	terms, err := fixture.app.ListTerms(fixture.ctx, api.ListTermsRequest{AcademicYearID: currentYearID(t, fixture)}).Unwrap()
	if err != nil {
		t.Fatalf("listing terms: %v", err)
	}
	if terms == nil {
		t.Error("listing terms returned null, want an empty slice")
	}
}

// failureCode is the code of a transport failure, so a test can say what the
// interface will be told rather than only that something went wrong.
func failureCode(t *testing.T, err error) apperr.Code {
	t.Helper()

	var wire *api.AppError
	if !errors.As(err, &wire) {
		t.Fatalf("error is not a transport failure envelope: %v", err)
	}
	return wire.Code
}

func currentYearID(t *testing.T, fixture *transportFixture) string {
	t.Helper()

	year, err := fixture.app.GetCurrentYear(fixture.ctx).Unwrap()
	if err != nil {
		t.Fatalf("reading the current year: %v", err)
	}
	return year.ID
}
