// Package application holds the organisation use cases. It orchestrates the
// domain, the repository ports and the transaction boundary; it never talks to
// SQLite directly.
package application

import (
	"context"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/clock"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/organisation/domain"
	"github.com/elcomware/edupilot/internal/platform/organisation/ports"
)

// CreateOrganisationCommand describes a new organisation. Exactly one
// organisation exists per installation: it is the tenant, not a member of a
// collection.
type CreateOrganisationCommand struct {
	Name          string
	CurrencyCode  string
	DefaultLocale string
	Country       string
}

// CreateOrganisation creates the tenant and stamps its audit timestamps.
func (s *Service) CreateOrganisation(ctx context.Context, command CreateOrganisationCommand) (*domain.Organisation, error) {
	organisation, err := domain.NewOrganisation(command.Name, command.CurrencyCode, command.DefaultLocale, command.Country)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	organisation.CreatedAt = now
	organisation.UpdatedAt = now

	if err := s.repository.Create(ctx, organisation); err != nil {
		return nil, err
	}
	return organisation, nil
}

// GetOrganisation returns an organisation by id.
func (s *Service) GetOrganisation(ctx context.Context, id ids.UUID) (*domain.Organisation, error) {
	if id.IsNil() {
		return nil, apperr.ValidationFailed("organisation id must not be empty").
			WithDetail("field", "organisation.id")
	}

	organisation, err := s.repository.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if organisation == nil {
		return nil, apperr.NotFound("organisation", id.String())
	}
	return organisation, nil
}

// GetCurrentOrganisation returns the single tenant of this installation. It is
// how a fresh install bootstraps and how the shell resolves "my school".
func (s *Service) GetCurrentOrganisation(ctx context.Context) (*domain.Organisation, error) {
	organisation, err := s.repository.First(ctx)
	if err != nil {
		return nil, err
	}
	if organisation == nil {
		return nil, apperr.NotFound("organisation", "current")
	}
	return organisation, nil
}

// EnsureOrganisation creates the tenant when the installation is empty and
// returns the existing one otherwise. It makes first run idempotent, which is
// what the desktop shell needs on every launch.
func (s *Service) EnsureOrganisation(ctx context.Context, command CreateOrganisationCommand) (*domain.Organisation, error) {
	existing, err := s.repository.First(ctx)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return s.CreateOrganisation(ctx, command)
}

// Service implements the organisation use cases.
type Service struct {
	repository ports.Repository
	clock      clock.Clock
}

// NewService builds the organisation application service.
func NewService(repository ports.Repository, systemClock clock.Clock) *Service {
	return &Service{repository: repository, clock: systemClock}
}
