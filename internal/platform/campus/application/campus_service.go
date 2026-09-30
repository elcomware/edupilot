// Package application holds the campus use cases.
package application

import (
	"context"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/clock"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/campus/domain"
	"github.com/elcomware/edupilot/internal/platform/campus/ports"
)

// CreateCampusCommand describes a new campus.
type CreateCampusCommand struct {
	OrganisationID ids.UUID
	Name           string
	Code           string
	Address        domain.Address
	Phone          string
	Email          string
	IsPrimary      bool
}

// CreateCampusCommandResult carries the new campus and the campus it demoted
// when the command asked for a new primary campus.
type CreateCampusCommandResult struct {
	Campus         *domain.Campus
	DemotedPrimary *domain.Campus
}

// CreateCampus adds a campus to an organisation. Promoting a campus to primary
// demotes the previous one inside the same transaction, so the organisation
// never has two home campuses.
func (s *Service) CreateCampus(ctx context.Context, command CreateCampusCommand) (CreateCampusCommandResult, error) {
	var result CreateCampusCommandResult

	if command.OrganisationID.IsNil() {
		return result, apperr.ValidationFailed("campus must belong to an organisation").
			WithDetail("field", "campus.organisation_id")
	}

	campus, err := domain.NewCampus(command.OrganisationID, command.Name, command.Code, false)
	if err != nil {
		return result, err
	}
	campus.SetAddress(command.Address)
	campus.SetContact(command.Phone, command.Email)

	now := s.clock.Now().UTC()
	campus.CreatedAt = now
	campus.UpdatedAt = now

	err = s.transactor.InTx(ctx, func(ctx context.Context) error {
		previous, demoteErr := s.demoteCurrentPrimary(ctx, command.OrganisationID, campus.ID)
		if demoteErr != nil {
			return demoteErr
		}
		result.DemotedPrimary = previous

		if command.IsPrimary {
			campus.MakePrimary()
		}
		return s.repository.Create(ctx, campus)
	})
	if err != nil {
		return CreateCampusCommandResult{}, err
	}

	result.Campus = campus
	return result, nil
}

// GetCampus returns one campus of an organisation.
func (s *Service) GetCampus(ctx context.Context, organisationID, id ids.UUID) (*domain.Campus, error) {
	campus, err := s.repository.ByID(ctx, organisationID, id)
	if err != nil {
		return nil, err
	}
	if campus == nil {
		return nil, apperr.NotFound("campus", id.String())
	}
	return campus, nil
}

// ListCampuses returns the campuses of an organisation, newest name order.
func (s *Service) ListCampuses(ctx context.Context, organisationID ids.UUID, includeInactive bool) ([]*domain.Campus, error) {
	if organisationID.IsNil() {
		return nil, apperr.ValidationFailed("campus listing requires an organisation").
			WithDetail("field", "campus.organisation_id")
	}
	return s.repository.List(ctx, organisationID, includeInactive)
}

// SetPrimaryCampus promotes a campus to the organisation's home campus.
func (s *Service) SetPrimaryCampus(ctx context.Context, organisationID, id ids.UUID) (*domain.Campus, error) {
	err := s.transactor.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.demoteCurrentPrimary(ctx, organisationID, id); err != nil {
			return err
		}

		campus, err := s.repository.ByID(ctx, organisationID, id)
		if err != nil {
			return err
		}
		if campus == nil {
			return apperr.NotFound("campus", id.String())
		}

		if campus.IsPrimary {
			return nil
		}

		campus.MakePrimary()
		campus.UpdatedAt = s.clock.Now().UTC()
		return s.repository.Update(ctx, campus)
	})
	if err != nil {
		return nil, err
	}

	return s.repository.ByID(ctx, organisationID, id)
}

func (s *Service) demoteCurrentPrimary(ctx context.Context, organisationID, keep ids.UUID) (*domain.Campus, error) {
	previous, err := s.repository.Primary(ctx, organisationID)
	if err != nil {
		return nil, err
	}
	if previous == nil || previous.ID == keep {
		return nil, nil
	}

	previous.IsPrimary = false
	previous.UpdatedAt = s.clock.Now().UTC()
	if err := s.repository.Update(ctx, previous); err != nil {
		return nil, err
	}

	if err := s.repository.ClearPrimary(ctx, organisationID, keep); err != nil {
		return nil, err
	}
	return previous, nil
}

// Service implements the campus use cases.
type Service struct {
	repository ports.Repository
	transactor Transactor
	clock      clock.Clock
}

// Transactor is the unit of work the campus use cases need. The application
// layer depends on this narrow port, not on a concrete SQL transaction.
type Transactor interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// NewService builds the campus application service.
func NewService(repository ports.Repository, transactor Transactor, systemClock clock.Clock) *Service {
	return &Service{repository: repository, transactor: transactor, clock: systemClock}
}
