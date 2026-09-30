// Package ports defines the repository interfaces the academic domain and
// application layers depend on. As with every EduPilot module, the dependency
// arrow points inward: infrastructure implements these, nothing here imports it.
package ports

import (
	"context"

	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/academic/domain"
)

// YearRepository persists academic years. Every read and write is scoped to an
// organisation, so one school can never read another's calendar.
type YearRepository interface {
	Create(ctx context.Context, year *domain.AcademicYear) error
	Update(ctx context.Context, year *domain.AcademicYear) error
	ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.AcademicYear, error)
	ByName(ctx context.Context, organisationID ids.UUID, name string) (*domain.AcademicYear, error)
	// List returns the organisation's years, most recent first.
	List(ctx context.Context, organisationID ids.UUID) ([]*domain.AcademicYear, error)
	// Current returns the year flagged current, or nil when the organisation has
	// not opened a year yet.
	Current(ctx context.Context, organisationID ids.UUID) (*domain.AcademicYear, error)
	// ClearCurrent demotes every current year of the organisation except one, so
	// opening a new year stays a single-writer operation.
	ClearCurrent(ctx context.Context, organisationID, keep ids.UUID) error
	Count(ctx context.Context, organisationID ids.UUID) (int64, error)
}

// TermRepository persists the terms of an academic year.
type TermRepository interface {
	Create(ctx context.Context, term *domain.Term) error
	Update(ctx context.Context, term *domain.Term) error
	ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Term, error)
	// ListByYear returns the terms of one year in sequence order.
	ListByYear(ctx context.Context, organisationID, academicYearID ids.UUID) ([]*domain.Term, error)
	CountByYear(ctx context.Context, organisationID, academicYearID ids.UUID) (int64, error)
}
