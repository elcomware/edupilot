package application

import (
	"context"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

// LinkRelationshipCommand records a directed relationship between two people.
type LinkRelationshipCommand struct {
	OrganisationID ids.UUID
	AcademicYearID ids.UUID
	FromPersonID   ids.UUID
	ToPersonID     ids.UUID
	Type           domain.RelationshipType
	Role           domain.RelationshipRole
	MakePrimary    bool
}

// Link records that FromPersonID stands in the given relationship to
// ToPersonID for one academic year.
//
// There is no special case for "an employee who is also a guardian" or "a
// guardian who is an emergency contact for a colleague". Both are just edges,
// which is what lets HR, Academics, Transport and Finance each read the same
// family view instead of each building their own.
func (s *Service) Link(ctx context.Context, command LinkRelationshipCommand) (*domain.Relationship, error) {
	if command.OrganisationID.IsNil() {
		return nil, apperr.ValidationFailed("a relationship must belong to an organisation").
			WithDetail("field", "relationship.organisation_id")
	}

	academicYearID, err := s.resolveYear(ctx, command.OrganisationID, command.AcademicYearID)
	if err != nil {
		return nil, err
	}

	// Both people must exist in this organisation. Checking both is what stops
	// a school from linking its student to somebody in another school.
	if _, err := s.GetPerson(ctx, command.OrganisationID, command.FromPersonID); err != nil {
		return nil, err
	}
	if _, err := s.GetPerson(ctx, command.OrganisationID, command.ToPersonID); err != nil {
		return nil, err
	}

	relationship, err := domain.NewRelationship(
		command.OrganisationID, academicYearID,
		command.FromPersonID, command.ToPersonID, command.Type,
	)
	if err != nil {
		return nil, err
	}
	if err := relationship.SetRole(command.Role); err != nil {
		return nil, err
	}
	if command.MakePrimary {
		if err := relationship.MakePrimary(); err != nil {
			return nil, err
		}
	}

	duplicate, err := s.relationships.FindSameEdge(
		ctx, academicYearID, relationship.FromPersonID, relationship.ToPersonID,
		relationship.Type, relationship.Role,
	)
	if err != nil {
		return nil, err
	}
	if duplicate != nil {
		return nil, apperr.AlreadyExists("relationship", "edge", string(relationship.Type))
	}

	now := s.clock.Now().UTC()
	relationship.CreatedAt = now
	relationship.UpdatedAt = now

	// Promoting a guardian to primary demotes the previous holder inside the
	// same transaction, so a student never has two billing guardians.
	err = s.transactor.InTx(ctx, func(ctx context.Context) error {
		if relationship.IsPrimary {
			if err := s.roles.ClearPrimaryGuardian(ctx, command.OrganisationID, relationship.ToPersonID, academicYearID, relationship.ID); err != nil {
				return err
			}
		}
		return s.relationships.Create(ctx, relationship)
	})
	if err != nil {
		return nil, err
	}
	return relationship, nil
}

// GetRelationship returns one relationship of this organisation.
func (s *Service) GetRelationship(ctx context.Context, organisationID, id ids.UUID) (*domain.Relationship, error) {
	relationship, err := s.relationships.ByID(ctx, organisationID, id)
	if err != nil {
		return nil, err
	}
	if relationship == nil {
		return nil, apperr.NotFound("relationship", id.String())
	}
	return relationship, nil
}

// Unlink ends a relationship without deleting it, so a statement printed last
// year still resolves the guardian who was on it.
func (s *Service) Unlink(ctx context.Context, organisationID, id ids.UUID) error {
	relationship, err := s.relationships.ByID(ctx, organisationID, id)
	if err != nil {
		return err
	}
	if relationship == nil {
		return apperr.NotFound("relationship", id.String())
	}

	relationship.Deactivate()
	relationship.UpdatedAt = s.clock.Now().UTC()
	return s.relationships.Update(ctx, relationship)
}

// RelationshipsOf returns the edges leaving a person in a year, optionally
// filtered by type. This is the "who are my dependants" view.
func (s *Service) RelationshipsOf(ctx context.Context, organisationID, personID, academicYearID ids.UUID, relationshipType domain.RelationshipType) ([]*domain.Relationship, error) {
	year, err := s.resolveYear(ctx, organisationID, academicYearID)
	if err != nil {
		return nil, err
	}
	return s.relationships.Outgoing(ctx, organisationID, year, personID, relationshipType)
}

// RelationshipsTo returns the edges arriving at a person in a year, optionally
// filtered by type. Incoming GUARDIAN_OF edges are how a student's guardians are
// found, and incoming EMERGENCY_CONTACT_OF edges are how a staff member's
// emergency contacts are found.
func (s *Service) RelationshipsTo(ctx context.Context, organisationID, personID, academicYearID ids.UUID, relationshipType domain.RelationshipType) ([]*domain.Relationship, error) {
	year, err := s.resolveYear(ctx, organisationID, academicYearID)
	if err != nil {
		return nil, err
	}
	return s.relationships.Incoming(ctx, organisationID, year, personID, relationshipType)
}

// GuardiansOf returns the people who are guardians of a person in a year, newest
// relationship first, with the primary billing guardian when one is set.
func (s *Service) GuardiansOf(ctx context.Context, organisationID, personID, academicYearID ids.UUID) ([]*domain.Relationship, error) {
	return s.RelationshipsTo(ctx, organisationID, personID, academicYearID, domain.RelationshipGuardianOf)
}

// DependantsOf returns everyone a person is a guardian of in a year. One
// guardian with children in several classes returns several students here, and
// each student's class membership is an enrolment owned by Academics.
func (s *Service) DependantsOf(ctx context.Context, organisationID, personID, academicYearID ids.UUID) ([]*domain.Relationship, error) {
	return s.RelationshipsOf(ctx, organisationID, personID, academicYearID, domain.RelationshipGuardianOf)
}

// CreateHouseholdCommand describes a billing party.
type CreateHouseholdCommand struct {
	OrganisationID ids.UUID
	AcademicYearID ids.UUID
	Name           string
	BillingEmail   string
	BillingPhone   string
	BillingAddress domain.Address
	// Members are the people who belong to the household. Their positions in the
	// school are not implied here: a member may be a guardian, a student or both.
	Members []HouseholdMemberCommand
}

// HouseholdMemberCommand is one person's place in a household.
type HouseholdMemberCommand struct {
	PersonID      ids.UUID
	Role          domain.MemberRole
	IsBillingPart bool
}

// CreateHousehold creates a billing party for one academic year. A household is
// not a family tree: it exists so Finance has a party to invoice, and a family
// can be re-grouped between years without touching a person's record.
func (s *Service) CreateHousehold(ctx context.Context, command CreateHouseholdCommand) (*domain.Household, error) {
	if command.OrganisationID.IsNil() {
		return nil, apperr.ValidationFailed("a household must belong to an organisation").
			WithDetail("field", "household.scope")
	}

	academicYearID, err := s.resolveYear(ctx, command.OrganisationID, command.AcademicYearID)
	if err != nil {
		return nil, err
	}

	household, err := domain.NewHousehold(command.OrganisationID, academicYearID, command.Name)
	if err != nil {
		return nil, err
	}
	if err := household.SetBillingContact(command.BillingEmail, command.BillingPhone, command.BillingAddress); err != nil {
		return nil, err
	}

	now := s.clock.Now().UTC()
	household.CreatedAt = now
	household.UpdatedAt = now

	err = s.transactor.InTx(ctx, func(ctx context.Context) error {
		// The household is inserted first because household_members references it;
		// writing the members first would violate the foreign key.
		if err := s.households.Create(ctx, household); err != nil {
			return err
		}
		for _, requested := range command.Members {
			if _, err := s.GetPerson(ctx, command.OrganisationID, requested.PersonID); err != nil {
				return err
			}
			// The aggregate decides the stored form of the member, stamping the
			// household identifier and normalising an empty role. What is
			// returned is what gets written.
			stored, err := household.AddMember(domain.HouseholdMember{
				PersonID:      requested.PersonID,
				Role:          requested.Role,
				IsBillingPart: requested.IsBillingPart,
				CreatedAt:     now,
			})
			if err != nil {
				return err
			}
			if err := s.households.AddMember(ctx, stored); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return household, nil
}

// GetHousehold returns one household with its members.
func (s *Service) GetHousehold(ctx context.Context, organisationID, id ids.UUID) (*domain.Household, error) {
	household, err := s.households.ByID(ctx, organisationID, id)
	if err != nil {
		return nil, err
	}
	if household == nil {
		return nil, apperr.NotFound("household", id.String())
	}
	return household, nil
}

// ListHouseholds returns the organisation's households for a year.
func (s *Service) ListHouseholds(ctx context.Context, organisationID, academicYearID ids.UUID) ([]*domain.Household, error) {
	year, err := s.resolveYear(ctx, organisationID, academicYearID)
	if err != nil {
		return nil, err
	}
	return s.households.ListByYear(ctx, organisationID, year)
}
