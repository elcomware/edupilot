package sqlite

import (
	"testing"
	"time"

	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

// A person insert whose placeholder count disagrees with its column list fails at
// runtime with "N values for M columns", not at compile time. This repository
// shipped exactly that bug once, so the counts are asserted rather than trusted.
func TestPersonStatementArgumentsMatchTheirColumns(t *testing.T) {
	birth := time.Date(2015, 5, 20, 0, 0, 0, 0, time.UTC)
	photo := ids.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	populated := &domain.Person{
		ID:              ids.New(),
		OrganisationID:  ids.New(),
		FirstName:       "Awa",
		LastName:        "Traore",
		DateOfBirth:     &birth,
		PhotoDocumentID: &photo,
		Address:         domain.Address{Line1: "Rue", City: "Abidjan", Country: "CI"},
		Version:         1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	// The empty case is the one that changes the argument count, because the
	// optional date and photo columns collapse to a single untyped nil.
	bare := &domain.Person{
		ID:             ids.New(),
		OrganisationID: ids.New(),
		FirstName:      "Ibrahim",
		LastName:       "Kone",
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	cases := map[string]*domain.Person{"populated": populated, "bare": bare}
	for name, person := range cases {
		t.Run(name, func(t *testing.T) {
			if got := len(personArguments(person)); got != len(personColumnList) {
				t.Errorf("insert binds %d values for %d columns", got, len(personColumnList))
			}
			if got := len(personUpdateArguments(person)); got != len(personUpdatableColumns) {
				t.Errorf("update binds %d values for %d columns", got, len(personUpdatableColumns))
			}
		})
	}
}

// An update must never be able to move a person to another organisation, so the
// organisation is not a writable column.
func TestPersonUpdateCannotRewriteIdentityOrTenant(t *testing.T) {
	for _, forbidden := range []string{"id", "organisation_id", "version", "created_at"} {
		for _, column := range personUpdatableColumns {
			if column == forbidden {
				t.Errorf("%q must not be an updatable column", column)
			}
		}
	}
}
