// Package ports defines the repository interfaces the campus domain and
// application layers depend on.
//
// Part of the EduPilot modular monolith. Dependencies point inward only:
// presentation -> application -> domain -> ports (implemented by infrastructure).
package ports

import (
	"context"

	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/campus/domain"
)

// Repository persists campuses. Every read and write is scoped to an
// organisation, so a caller cannot reach another school's campus by passing a
// different identifier.
type Repository interface {
	Create(ctx context.Context, campus *domain.Campus) error
	Update(ctx context.Context, campus *domain.Campus) error
	ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Campus, error)
	List(ctx context.Context, organisationID ids.UUID, includeInactive bool) ([]*domain.Campus, error)
	// ClearPrimary demotes every primary campus of the organisation except the
	// one named, so promoting a campus stays a single-writer operation.
	ClearPrimary(ctx context.Context, organisationID, keep ids.UUID) error
	Primary(ctx context.Context, organisationID ids.UUID) (*domain.Campus, error)
	Count(ctx context.Context, organisationID ids.UUID) (int64, error)
}
