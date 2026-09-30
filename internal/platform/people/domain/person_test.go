package domain_test

import (
	"testing"
	"time"

	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

// mustDate is a calendar date in UTC, the only kind of time these records hold.
func mustDate(t *testing.T, year int, month time.Month, day int) time.Time {
	t.Helper()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// A name is rendered on every list in the interface, so a gap left by a missing
// middle name would be seen a thousand times a day.
func TestFullNameSkipsThePartsThatAreMissing(t *testing.T) {
	cases := []struct {
		name        string
		first       string
		middle      string
		last        string
		preferred   string
		wantFull    string
		wantDisplay string
	}{
		{
			name:        "both given names",
			first:       "Grace",
			middle:      "Marie",
			last:        "Tanyi",
			wantFull:    "Grace Marie Tanyi",
			wantDisplay: "Grace Marie Tanyi",
		},
		{
			name:        "no middle name",
			first:       "Grace",
			last:        "Tanyi",
			wantFull:    "Grace Tanyi",
			wantDisplay: "Grace Tanyi",
		},
		{
			name:        "only a first name",
			first:       "Grace",
			wantFull:    "Grace",
			wantDisplay: "Grace",
		},
		{
			name:        "a preferred name wins for display",
			first:       "Grace",
			middle:      "Marie",
			last:        "Tanyi",
			preferred:   "Gracie",
			wantFull:    "Grace Marie Tanyi",
			wantDisplay: "Gracie",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			person := &domain.Person{
				FirstName:     testCase.first,
				MiddleName:    testCase.middle,
				LastName:      testCase.last,
				PreferredName: testCase.preferred,
			}

			if got := person.FullName(); got != testCase.wantFull {
				t.Errorf("FullName() = %q, want %q", got, testCase.wantFull)
			}
			if got := person.DisplayName(); got != testCase.wantDisplay {
				t.Errorf("DisplayName() = %q, want %q", got, testCase.wantDisplay)
			}
		})
	}
}

// Employment ending before it began is not a date the interface should ever
// accept, because every later question about a leaver's record depends on it.
func TestEmploymentCannotEndBeforeItBegan(t *testing.T) {
	profile := domain.NewEmployeeProfile(ids.New(), "EMP-1")
	profile.SetHiredOn(mustDate(t, 2025, 9, 1))

	if err := profile.SetEndedOn(mustDate(t, 2024, 1, 1)); err == nil {
		t.Fatal("ending employment before it began should fail")
	}
	if profile.EndedOn != nil {
		t.Errorf("EndedOn = %v, want the rejected date to be discarded", profile.EndedOn)
	}

	if err := profile.SetEndedOn(mustDate(t, 2026, 6, 30)); err != nil {
		t.Fatalf("ending employment after it began: %v", err)
	}
	if profile.EndedOn == nil {
		t.Fatal("the employment end date was not recorded")
	}
}

// A leaver keeps the profile: the department they worked in is a fact about the
// past that a payslip from two years ago still needs.
func TestALeaversProfileIsKept(t *testing.T) {
	profile := domain.NewEmployeeProfile(ids.New(), "EMP-2")
	if err := profile.SetEmployment("Teacher", "Mathematics", ""); err != nil {
		t.Fatal(err)
	}
	profile.SetHiredOn(mustDate(t, 2025, 9, 1))
	if err := profile.SetEndedOn(mustDate(t, 2026, 6, 30)); err != nil {
		t.Fatal(err)
	}

	if profile.JobTitle != "Teacher" || profile.Department != "Mathematics" {
		t.Errorf("profile = %q in %q, want the employment details to survive the leaver", profile.JobTitle, profile.Department)
	}
	if profile.EmployeeNumber != "EMP-2" {
		t.Errorf("employee number = %q, want the number to survive the leaver", profile.EmployeeNumber)
	}
}
