package api_test

import (
	"testing"
	"time"

	"github.com/elcomware/edupilot/internal/api"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	academicdomain "github.com/elcomware/edupilot/internal/platform/academic/domain"
	peopledomain "github.com/elcomware/edupilot/internal/platform/people/domain"
)

// A term that begins on 31 August must read as 31 August everywhere. Rendering
// it as 1 September for a reader east of Greenwich would put the first day of
// term in the wrong week.
func TestCalendarDateDoesNotShiftAcrossTimeZones(t *testing.T) {
	boundary := time.Date(2025, 8, 31, 23, 30, 0, 0, time.UTC)
	if got := api.CalendarDate(boundary); got != "2025-08-31" {
		t.Errorf("CalendarDate = %q, want 2025-08-31", got)
	}
	if got := api.CalendarDate(time.Time{}); got != "" {
		t.Errorf("CalendarDate of a zero time = %q, want an empty string", got)
	}
	if got := api.OptionalCalendarDate(nil); got != "" {
		t.Errorf("OptionalCalendarDate(nil) = %q, want an empty string", got)
	}

	day := time.Date(2015, 5, 20, 0, 0, 0, 0, time.UTC)
	if got := api.OptionalCalendarDate(&day); got != "2015-05-20" {
		t.Errorf("OptionalCalendarDate = %q, want 2015-05-20", got)
	}
}

func TestNewAcademicYearView(t *testing.T) {
	year, err := academicdomain.NewAcademicYear(ids.New(), "2025-2026",
		time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	view := api.NewAcademicYearView(year)
	if view.Name != "2025-2026" || view.StartsOn != "2025-09-01" || view.EndsOn != "2026-07-31" {
		t.Errorf("year view = %+v, want the persisted year as calendar days", view)
	}
	if view.OrganisationID != year.OrganisationID.String() {
		t.Errorf("organisationId = %q, want %q", view.OrganisationID, year.OrganisationID)
	}
}

// A person is shown by the name they want to be called, but the formal name is
// kept for documents, so both must survive the mapping.
func TestNewPersonViewPrefersTheDisplayName(t *testing.T) {
	person, err := peopledomain.NewPerson(ids.New(), "Awa", "Traore")
	if err != nil {
		t.Fatal(err)
	}
	if err := person.SetPreferredName("Awaa"); err != nil {
		t.Fatal(err)
	}
	if err := person.SetContact("awa@ecole.ci", "+2250700000000", ""); err != nil {
		t.Fatal(err)
	}

	view := api.NewPersonView(person, nil)
	if view.DisplayName != "Awaa" {
		t.Errorf("displayName = %q, want the preferred name", view.DisplayName)
	}
	if view.LastName != "Traore" {
		t.Errorf("lastName = %q, want the formal name kept for documents", view.LastName)
	}
	if view.Email != "awa@ecole.ci" {
		t.Errorf("email = %q, want the persisted address", view.Email)
	}
	// The optional collections must be empty, never nil, so the frontend can
	// iterate them without a null check.
	if view.Roles == nil || view.GuardiansOf == nil || view.GuardianOf == nil {
		t.Error("an optional list was serialised as null instead of an empty array")
	}
}

// The role-specific details belong in the block that matches the role, so the
// interface never has to ask whether a student number is meaningful for an
// employee.
func TestNewPersonRoleViewGroupsProfileDetailsByRole(t *testing.T) {
	studentRole, err := peopledomain.NewPersonRole(ids.New(), ids.New(), ids.New(), peopledomain.RoleStudent)
	if err != nil {
		t.Fatal(err)
	}
	studentProfile := peopledomain.NewStudentProfile(studentRole.ID, "STU-0001")
	if err := studentProfile.SetStatus(peopledomain.StudentEnrolled); err != nil {
		t.Fatal(err)
	}

	view := api.NewPersonRoleView(studentRole, studentProfile, nil, nil)
	if view.Student == nil {
		t.Fatal("the student details were dropped")
	}
	if view.Student.StudentNumber != "STU-0001" || view.Student.Status != "ENROLLED" {
		t.Errorf("student details = %+v, want the persisted profile", view.Student)
	}
	if view.Employee != nil || view.Guardian != nil {
		t.Error("a student role carries employee or guardian details")
	}

	// A role with no profile yet must still project cleanly, so the interface can
	// tell "not written" from "not applicable".
	bare := api.NewPersonRoleView(studentRole, nil, nil, nil)
	if bare.Student != nil || bare.Employee != nil || bare.Guardian != nil {
		t.Error("a role with no profile produced a details block")
	}
	if bare.Role != "STUDENT" {
		t.Errorf("role = %q, want STUDENT", bare.Role)
	}
}

func TestNewHouseholdViewReportsMembers(t *testing.T) {
	organisation, year := ids.New(), ids.New()
	household, err := peopledomain.NewHousehold(organisation, year, "Famille Traore")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := household.AddMember(peopledomain.HouseholdMember{
		PersonID:      ids.New(),
		Role:          peopledomain.MemberHead,
		IsBillingPart: true,
	}); err != nil {
		t.Fatal(err)
	}

	view := api.NewHouseholdView(household)
	if view.Name != "Famille Traore" {
		t.Errorf("name = %q, want the billing name", view.Name)
	}
	if len(view.Members) != 1 || !view.Members[0].IsBillingPart {
		t.Errorf("members = %+v, want one billing member", view.Members)
	}
	if view.AcademicYearID != year.String() {
		t.Errorf("academicYearId = %q, want %q", view.AcademicYearID, year)
	}
}

// A zero slice must serialise as [], not null, or a list view has to guard
// against null on every render.
func TestEmptySlicesSerialiseAsArrays(t *testing.T) {
	personWithoutRoles, err := peopledomain.NewPerson(ids.New(), "Awa", "Traore")
	if err != nil {
		t.Fatal(err)
	}
	if got := api.NewPersonViewWithRoles(personWithoutRoles, nil); got.Roles == nil {
		t.Error("a person with no roles produced a nil Roles, which serialises as null rather than []")
	}
	if got := api.NewAcademicYearViews(nil); got == nil {
		t.Error("NewAcademicYearViews(nil) returned nil")
	}
	if got := api.NewRelationshipViews(nil); got == nil {
		t.Error("NewRelationshipViews(nil) returned nil")
	}
	if got := api.NewHouseholdViews(nil); got == nil {
		t.Error("NewHouseholdViews(nil) returned nil")
	}
	if got := api.NewTermViews(nil); got == nil {
		t.Error("NewTermViews(nil) returned nil")
	}
}
