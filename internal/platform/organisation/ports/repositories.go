// Package ports defines the repository interfaces the organisation domain and
// application layers depend on.
//
// Part of the EduPilot modular monolith. Dependencies point inward only:
// presentation -> application -> domain -> ports (implemented by infrastructure).
package ports

import (
	"context"

	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/organisation/domain"
)

// Repository persists organisations.
type Repository interface {
	Create(ctx context.Context, organisation *domain.Organisation) error
	Update(ctx context.Context, organisation *domain.Organisation) error
	ByID(ctx context.Context, id ids.UUID) (*domain.Organisation, error)
	// First returns the oldest organisation, which is the tenant of a
	// single-organisation installation. It returns nil when none exists.
	First(ctx context.Context) (*domain.Organisation, error)
	Exists(ctx context.Context, id ids.UUID) (bool, error)
	Count(ctx context.Context) (int64, error)
}
