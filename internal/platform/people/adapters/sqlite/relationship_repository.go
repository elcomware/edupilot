package sqlite

import (
	"context"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

const relationshipColumns = `id, organisation_id, academic_year_id, from_person_id, to_person_id,
	relationship_type, relationship_role, is_primary, is_active, version, created_at, updated_at`

// RelationshipRepository stores the directed edges between people in SQLite.
type RelationshipRepository struct {
	transactor *database.Transactor
}

// NewRelationshipRepository builds the SQLite relationship repository.
func NewRelationshipRepository(transactor *database.Transactor) *RelationshipRepository {
	return &RelationshipRepository{transactor: transactor}
}

// Create inserts a new relationship.
func (r *RelationshipRepository) Create(ctx context.Context, relationship *domain.Relationship) error {
	const statement = `INSERT INTO person_relationships (` + relationshipColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		relationship.ID.String(),
		relationship.OrganisationID.String(),
		relationship.AcademicYearID.String(),
		relationship.FromPersonID.String(),
		relationship.ToPersonID.String(),
		string(relationship.Type),
		string(relationship.Role),
		relationship.IsPrimary,
		relationship.IsActive,
		relationship.Version,
		relationship.CreatedAt.Format(time.RFC3339Nano),
		relationship.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return apperr.AlreadyExists("relationship", "edge", string(relationship.Type))
		}
		return apperr.Internal(err, "relationship could not be created")
	}
	return nil
}

// Update writes the relationship.
func (r *RelationshipRepository) Update(ctx context.Context, relationship *domain.Relationship) error {
	const statement = `UPDATE person_relationships SET
			relationship_role = ?, is_primary = ?, is_active = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND organisation_id = ? AND version = ?`

	result, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		string(relationship.Role),
		relationship.IsPrimary,
		relationship.IsActive,
		relationship.UpdatedAt.Format(time.RFC3339Nano),
		relationship.ID.String(),
		relationship.OrganisationID.String(),
		relationship.Version,
	)
	if err != nil {
		return apperr.Internal(err, "relationship could not be updated")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err, "relationship update result could not be read")
	}
	if affected == 0 {
		return apperr.New(apperr.CodeConflict, "the relationship was changed by someone else").
			WithDetail("entity", "relationship").
			WithDetail("id", relationship.ID.String())
	}

	relationship.Version++
	return nil
}

// ByID loads one relationship, or nil when it does not exist in that organisation.
func (r *RelationshipRepository) ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Relationship, error) {
	return r.queryOne(ctx,
		"SELECT "+relationshipColumns+" FROM person_relationships WHERE id = ? AND organisation_id = ?",
		id.String(), organisationID.String())
}

// Outgoing returns the edges leaving a person in a year, optionally filtered by
// type. An empty relationshipType means every type.
func (r *RelationshipRepository) Outgoing(ctx context.Context, organisationID, academicYearID, personID ids.UUID, relationshipType domain.RelationshipType) ([]*domain.Relationship, error) {
	query := "SELECT " + relationshipColumns + ` FROM person_relationships
		WHERE organisation_id = ? AND academic_year_id = ? AND from_person_id = ?`
	arguments := []any{organisationID.String(), academicYearID.String(), personID.String()}

	if relationshipType != "" {
		query += " AND relationship_type = ?"
		arguments = append(arguments, string(relationshipType))
	}
	query += " ORDER BY is_primary DESC, relationship_type, id"

	return r.queryMany(ctx, query, arguments...)
}

// Incoming returns the edges arriving at a person in a year, optionally filtered
// by type. This is how a student's guardians and a staff member's emergency
// contacts are found.
func (r *RelationshipRepository) Incoming(ctx context.Context, organisationID, academicYearID, personID ids.UUID, relationshipType domain.RelationshipType) ([]*domain.Relationship, error) {
	query := "SELECT " + relationshipColumns + ` FROM person_relationships
		WHERE organisation_id = ? AND academic_year_id = ? AND to_person_id = ?`
	arguments := []any{organisationID.String(), academicYearID.String(), personID.String()}

	if relationshipType != "" {
		query += " AND relationship_type = ?"
		arguments = append(arguments, string(relationshipType))
	}
	query += " ORDER BY is_primary DESC, relationship_type, id"

	return r.queryMany(ctx, query, arguments...)
}

// FindSameEdge returns an existing edge between the same two people of the same
// type and label, or nil. Symmetric types are stored in one canonical direction
// by the domain, so the caller can pass them in either order.
func (r *RelationshipRepository) FindSameEdge(ctx context.Context, academicYearID, fromPersonID, toPersonID ids.UUID, relationshipType domain.RelationshipType, role domain.RelationshipRole) (*domain.Relationship, error) {
	from, to := fromPersonID, toPersonID
	if relationshipType.Symmetric() && from.String() > to.String() {
		from, to = to, from
	}

	return r.queryOne(ctx,
		"SELECT "+relationshipColumns+` FROM person_relationships
		 WHERE academic_year_id = ? AND from_person_id = ? AND to_person_id = ?
		   AND relationship_type = ? AND relationship_role = ?`,
		academicYearID.String(), from.String(), to.String(),
		string(relationshipType), string(role))
}

func (r *RelationshipRepository) queryOne(ctx context.Context, query string, arguments ...any) (*domain.Relationship, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, apperr.Internal(err, "relationship lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "relationship lookup failed")
		}
		return nil, nil
	}
	return scanRelationship(rows)
}

func (r *RelationshipRepository) queryMany(ctx context.Context, query string, arguments ...any) ([]*domain.Relationship, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, apperr.Internal(err, "relationship listing failed")
	}
	defer rows.Close()

	relationships := []*domain.Relationship{}
	for rows.Next() {
		relationship, err := scanRelationship(rows)
		if err != nil {
			return nil, err
		}
		relationships = append(relationships, relationship)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err, "relationship listing failed")
	}
	return relationships, nil
}

func scanRelationship(row interface{ Scan(...any) error }) (*domain.Relationship, error) {
	var (
		rawID, rawOrganisationID, rawYearID  string
		rawFromID, rawToID, rawType, rawRole string
		isPrimary, isActive                  bool
		version                              int64
		rawCreatedAt, rawUpdatedAt           string
	)

	err := row.Scan(&rawID, &rawOrganisationID, &rawYearID, &rawFromID, &rawToID,
		&rawType, &rawRole, &isPrimary, &isActive, &version, &rawCreatedAt, &rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "relationship could not be read")
	}

	identifier, err := ids.Parse(rawID)
	if err != nil {
		return nil, apperr.Internal(err, "relationship identifier is invalid")
	}
	owner, err := ids.Parse(rawOrganisationID)
	if err != nil {
		return nil, apperr.Internal(err, "relationship organisation identifier is invalid")
	}
	year, err := ids.Parse(rawYearID)
	if err != nil {
		return nil, apperr.Internal(err, "relationship academic year identifier is invalid")
	}
	from, err := ids.Parse(rawFromID)
	if err != nil {
		return nil, apperr.Internal(err, "relationship source identifier is invalid")
	}
	to, err := ids.Parse(rawToID)
	if err != nil {
		return nil, apperr.Internal(err, "relationship target identifier is invalid")
	}
	created, err := time.Parse(time.RFC3339, rawCreatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "relationship creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "relationship update timestamp is invalid")
	}

	return &domain.Relationship{
		ID:             identifier,
		OrganisationID: owner,
		AcademicYearID: year,
		FromPersonID:   from,
		ToPersonID:     to,
		Type:           domain.RelationshipType(rawType),
		Role:           domain.RelationshipRole(rawRole),
		IsPrimary:      isPrimary,
		IsActive:       isActive,
		Version:        version,
		CreatedAt:      created,
		UpdatedAt:      updated,
	}, nil
}
