// Package application holds the academic year and term use cases.
package application

import (
	"context"
	"time"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/clock"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/academic/domain"
	"github.com/elcomware/edupilot/internal/platform/academic/ports"
)

// Service implements the academic structure use cases.
type Service struct {
	years      ports.YearRepository
	terms      ports.TermRepository
	transactor Transactor
	clock      clock.Clock
}

// Transactor is the unit of work the academic use cases need. The application
// layer depends on this narrow port, not on a concrete SQL transaction.
type Transactor interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// NewService builds the academic application service.
func NewService(years ports.YearRepository, terms ports.TermRepository, transactor Transactor, systemClock clock.Clock) *Service {
	return &Service{years: years, terms: terms, transactor: transactor, clock: systemClock}
}

// CreateYearCommand describes a new academic year.
type CreateYearCommand struct {
	OrganisationID ids.UUID
	Name           string
	StartsOn       time.Time
	EndsOn         time.Time
	// MakeCurrent opens this year as the organisation's current year, demoting
	// whichever year held the flag.
	MakeCurrent bool
}

// CreateYear adds an academic year. Two years may not overlap: an overlapping
// pair would make it ambiguous which year a role, relationship or enrolment
// belongs to, and that ambiguity is very expensive to unpick once fees have been
// billed against it.
func (s *Service) CreateYear(ctx context.Context, command CreateYearCommand) (*domain.AcademicYear, error) {
	if command.OrganisationID.IsNil() {
		return nil, apperr.ValidationFailed("academic year must belong to an organisation").
			WithDetail("field", "academic_year.organisation_id")
	}

	year, err := domain.NewAcademicYear(command.OrganisationID, command.Name, command.StartsOn, command.EndsOn)
	if err != nil {
		return nil, err
	}

	existing, err := s.years.List(ctx, command.OrganisationID)
	if err != nil {
		return nil, err
	}
	for _, other := range existing {
		if year.Overlaps(other) {
			return nil, apperr.New(apperr.CodeConflict, "this academic year overlaps an existing one").
				WithDetail("entity", "academic_year").
				WithDetail("conflictsWith", other.Name)
		}
	}

	now := s.clock.Now().UTC()
	year.CreatedAt = now
	year.UpdatedAt = now

	err = s.transactor.InTx(ctx, func(ctx context.Context) error {
		if command.MakeCurrent {
			if err := s.years.ClearCurrent(ctx, command.OrganisationID, year.ID); err != nil {
				return err
			}
			year.MakeCurrent()
		}
		return s.years.Create(ctx, year)
	})
	if err != nil {
		return nil, err
	}
	return year, nil
}

// GetYear returns one academic year.
func (s *Service) GetYear(ctx context.Context, organisationID, id ids.UUID) (*domain.AcademicYear, error) {
	year, err := s.years.ByID(ctx, organisationID, id)
	if err != nil {
		return nil, err
	}
	if year == nil {
		return nil, apperr.NotFound("academic year", id.String())
	}
	return year, nil
}

// GetCurrentYear returns the organisation's current academic year. Every
// year-scoped use case in Core resolves its year through this, so the platform
// has exactly one notion of "this year".
func (s *Service) GetCurrentYear(ctx context.Context, organisationID ids.UUID) (*domain.AcademicYear, error) {
	year, err := s.years.Current(ctx, organisationID)
	if err != nil {
		return nil, err
	}
	if year == nil {
		return nil, apperr.New(apperr.CodeNotFound, "this organisation has no current academic year").
			WithDetail("entity", "academic_year")
	}
	return year, nil
}

// ListYears returns the organisation's academic years, most recent first.
func (s *Service) ListYears(ctx context.Context, organisationID ids.UUID) ([]*domain.AcademicYear, error) {
	if organisationID.IsNil() {
		return nil, apperr.ValidationFailed("academic year listing requires an organisation").
			WithDetail("field", "academic_year.organisation_id")
	}
	return s.years.List(ctx, organisationID)
}

// SetCurrentYear promotes an academic year to the organisation's current year.
func (s *Service) SetCurrentYear(ctx context.Context, organisationID, id ids.UUID) (*domain.AcademicYear, error) {
	err := s.transactor.InTx(ctx, func(ctx context.Context) error {
		if err := s.years.ClearCurrent(ctx, organisationID, id); err != nil {
			return err
		}
		year, err := s.years.ByID(ctx, organisationID, id)
		if err != nil {
			return err
		}
		if year == nil {
			return apperr.NotFound("academic year", id.String())
		}
		if year.IsCurrent {
			return nil
		}
		year.MakeCurrent()
		year.UpdatedAt = s.clock.Now().UTC()
		return s.years.Update(ctx, year)
	})
	if err != nil {
		return nil, err
	}
	return s.GetYear(ctx, organisationID, id)
}

// CreateTermCommand describes a new term within an academic year.
type CreateTermCommand struct {
	OrganisationID ids.UUID
	AcademicYearID ids.UUID
	Name           string
	Sequence       int
	StartsOn       time.Time
	EndsOn         time.Time
}

// CreateTerm adds a term to an academic year. The term must sit inside its
// year: a term that runs past the end of the year would put fees and reports in
// a period that does not exist on the calendar.
func (s *Service) CreateTerm(ctx context.Context, command CreateTermCommand) (*domain.Term, error) {
	if command.OrganisationID.IsNil() {
		return nil, apperr.ValidationFailed("term must belong to an organisation").
			WithDetail("field", "term.organisation_id")
	}
	if command.AcademicYearID.IsNil() {
		return nil, apperr.ValidationFailed("term must belong to an academic year").
			WithDetail("field", "term.academic_year_id")
	}

	year, err := s.GetYear(ctx, command.OrganisationID, command.AcademicYearID)
	if err != nil {
		return nil, err
	}

	term, err := domain.NewTerm(command.OrganisationID, year.ID, command.Name, command.Sequence, command.StartsOn, command.EndsOn)
	if err != nil {
		return nil, err
	}

	if term.StartsOn.Before(year.StartsOn) || term.EndsOn.After(year.EndsOn) {
		return nil, apperr.ValidationFailed("a term must fall inside its academic year").
			WithDetail("field", "term.period").
			WithDetail("academicYear", year.Name)
	}

	now := s.clock.Now().UTC()
	term.CreatedAt = now
	term.UpdatedAt = now

	if err := s.terms.Create(ctx, term); err != nil {
		return nil, err
	}
	return term, nil
}

// GetTerm returns one term.
func (s *Service) GetTerm(ctx context.Context, organisationID, id ids.UUID) (*domain.Term, error) {
	term, err := s.terms.ByID(ctx, organisationID, id)
	if err != nil {
		return nil, err
	}
	if term == nil {
		return nil, apperr.NotFound("term", id.String())
	}
	return term, nil
}

// ListTerms returns the terms of an academic year in sequence order.
func (s *Service) ListTerms(ctx context.Context, organisationID, academicYearID ids.UUID) ([]*domain.Term, error) {
	if academicYearID.IsNil() {
		return nil, apperr.ValidationFailed("term listing requires an academic year").
			WithDetail("field", "term.academic_year_id")
	}
	return s.terms.ListByYear(ctx, organisationID, academicYearID)
}
