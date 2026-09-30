// Package domain holds the academic aggregates EduPilot Core needs before any
// business domain runs. Finance requires only the structural concepts: a year
// and its terms. Teaching, assessment and curriculum belong to the Academics
// domain and are not modelled here (docs/adr/ADR-011).
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
)

// AcademicYear is one school year for an organisation, for example 2026-2027.
// It is the platform's time axis: roles, relationships, households and
// enrolments all belong to a year. Finance keeps its own fiscal calendar
// separately, because a September-June year is usually booked on a July-June
// fiscal year.
type AcademicYear struct {
	ID             ids.UUID
	OrganisationID ids.UUID
	Name           string
	StartsOn       time.Time
	EndsOn         time.Time
	IsCurrent      bool
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Term is a division of an academic year, for example Term 1 or the Spring
// term. A school that runs three terms defines three; one that runs none
// leaves the year without terms.
type Term struct {
	ID             ids.UUID
	OrganisationID ids.UUID
	AcademicYearID ids.UUID
	Name           string
	Sequence       int
	StartsOn       time.Time
	EndsOn         time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const maxAcademicYearNameLength = 64

// NewAcademicYear creates an academic year with a generated identifier.
func NewAcademicYear(organisationID ids.UUID, name string, startsOn, endsOn time.Time) (*AcademicYear, error) {
	return NewAcademicYearWithID(ids.New(), organisationID, name, startsOn, endsOn)
}

// NewAcademicYearWithID creates an academic year with a caller-supplied
// identifier, which is what the cloud synchroniser needs.
func NewAcademicYearWithID(id, organisationID ids.UUID, name string, startsOn, endsOn time.Time) (*AcademicYear, error) {
	year := &AcademicYear{
		ID:             id,
		OrganisationID: organisationID,
		Version:        1,
	}

	if err := year.Rename(name); err != nil {
		return nil, err
	}
	if err := year.SetPeriod(startsOn, endsOn); err != nil {
		return nil, err
	}
	return year, nil
}

// Rename sets the year label, which is free text so a school can write
// "2026-2027", "Year 2026/27" or whatever its reports already use.
func (y *AcademicYear) Rename(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return apperr.ValidationFailed("academic year name must not be empty").
			WithDetail("field", "academic_year.name")
	}
	if utf8.RuneCountInString(trimmed) > maxAcademicYearNameLength {
		return apperr.ValidationFailed("academic year name is longer than 64 characters").
			WithDetail("field", "academic_year.name")
	}
	y.Name = trimmed
	return nil
}

// SetPeriod sets the first and last day of the year. The end must be after the
// start, and neither is moved to the first of the month: a school's year is a
// real date range that fee schedules and reporting are cut against.
func (y *AcademicYear) SetPeriod(startsOn, endsOn time.Time) error {
	start := dateOnly(startsOn)
	end := dateOnly(endsOn)

	if start.IsZero() || end.IsZero() {
		return apperr.ValidationFailed("academic year start and end dates are required").
			WithDetail("field", "academic_year.period")
	}
	if !end.After(start) {
		return apperr.ValidationFailed("academic year must end after it starts").
			WithDetail("field", "academic_year.period").
			WithDetail("startsOn", start.Format(dateLayout)).
			WithDetail("endsOn", end.Format(dateLayout))
	}

	y.StartsOn = start
	y.EndsOn = end
	return nil
}

// MakeCurrent marks this year as the organisation's current year. The database
// holds at most one current year per organisation with a partial unique index,
// so the application layer demotes the previous holder in the same transaction.
func (y *AcademicYear) MakeCurrent() {
	y.IsCurrent = true
}

// Contains reports whether a date falls inside the year. Reports and fee
// schedules use it instead of comparing strings.
func (y *AcademicYear) Contains(day time.Time) bool {
	date := dateOnly(day)
	return !date.Before(y.StartsOn) && !date.After(y.EndsOn)
}

// Overlaps reports whether this year shares any day with another. Two years
// that overlap would make a role, relationship or enrolment ambiguous, so the
// application layer refuses to create one.
func (y *AcademicYear) Overlaps(other *AcademicYear) bool {
	if other == nil {
		return false
	}
	return !y.StartsOn.After(other.EndsOn) && !other.StartsOn.After(y.EndsOn)
}

// NewTerm creates a term with a generated identifier.
func NewTerm(organisationID, academicYearID ids.UUID, name string, sequence int, startsOn, endsOn time.Time) (*Term, error) {
	return NewTermWithID(ids.New(), organisationID, academicYearID, name, sequence, startsOn, endsOn)
}

// NewTermWithID creates a term with a caller-supplied identifier.
func NewTermWithID(id, organisationID, academicYearID ids.UUID, name string, sequence int, startsOn, endsOn time.Time) (*Term, error) {
	term := &Term{
		ID:             id,
		OrganisationID: organisationID,
		AcademicYearID: academicYearID,
	}

	if err := term.Rename(name); err != nil {
		return nil, err
	}
	if err := term.SetSequence(sequence); err != nil {
		return nil, err
	}
	if err := term.SetPeriod(startsOn, endsOn); err != nil {
		return nil, err
	}
	return term, nil
}

// Rename sets the term name.
func (t *Term) Rename(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return apperr.ValidationFailed("term name must not be empty").
			WithDetail("field", "term.name")
	}
	if utf8.RuneCountInString(trimmed) > maxAcademicYearNameLength {
		return apperr.ValidationFailed("term name is longer than 64 characters").
			WithDetail("field", "term.name")
	}
	t.Name = trimmed
	return nil
}

// SetSequence sets the term order within its year. Terms are ordered by
// sequence rather than by date so a school can define a sequence that matches
// the order it already reports in.
func (t *Term) SetSequence(sequence int) error {
	if sequence <= 0 {
		return apperr.ValidationFailed("term sequence must be a positive number").
			WithDetail("field", "term.sequence").
			WithDetail("value", sequence)
	}
	t.Sequence = sequence
	return nil
}

// SetPeriod sets the first and last day of the term.
func (t *Term) SetPeriod(startsOn, endsOn time.Time) error {
	start := dateOnly(startsOn)
	end := dateOnly(endsOn)

	if start.IsZero() || end.IsZero() {
		return apperr.ValidationFailed("term start and end dates are required").
			WithDetail("field", "term.period")
	}
	if !end.After(start) {
		return apperr.ValidationFailed("term must end after it starts").
			WithDetail("field", "term.period").
			WithDetail("startsOn", start.Format(dateLayout)).
			WithDetail("endsOn", end.Format(dateLayout))
	}

	t.StartsOn = start
	t.EndsOn = end
	return nil
}

// dateLayout is the storage format for a date-only value.
const dateLayout = "2006-01-02"

// dateOnly strips the time component so a year or term boundary never depends
// on the timezone it was entered in.
func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
