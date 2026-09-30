// Package sqlite implements the campus repository against SQLite. It is the
// only place that knows the campuses table exists.
package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/campus/domain"
)

const campusColumns = `id, organisation_id, name, code, address_line1, city, country_code,
	phone, email, is_primary, is_active, version, created_at, updated_at`

// Repository stores campuses in SQLite.
type Repository struct {
	transactor *database.Transactor
}

// NewRepository builds the SQLite campus repository.
func NewRepository(transactor *database.Transactor) *Repository {
	return &Repository{transactor: transactor}
}

// Create inserts a new campus.
func (r *Repository) Create(ctx context.Context, campus *domain.Campus) error {
	const statement = `INSERT INTO campuses (` + campusColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement, campusArguments(campus)...)
	if err != nil {
		switch {
		case database.IsUniqueViolation(err) && !strings.Contains(err.Error(), "single_primary"):
			return apperr.AlreadyExists("campus", "code", campus.Code)
		case database.IsUniqueViolation(err) || strings.Contains(strings.ToLower(err.Error()), "single_primary"):
			return apperr.New(apperr.CodeConflict, "this organisation already has a primary campus").
				WithDetail("entity", "campus")
		default:
			return apperr.Internal(err, "campus could not be created")
		}
	}
	return nil
}

// Update writes the campus with an optimistic version check.
func (r *Repository) Update(ctx context.Context, campus *domain.Campus) error {
	const statement = `UPDATE campuses SET
			name = ?, code = ?, address_line1 = ?, city = ?, country_code = ?,
			phone = ?, email = ?, is_primary = ?, is_active = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND organisation_id = ? AND version = ?`

	result, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		campus.Name,
		campus.Code,
		campus.Address.Line1,
		campus.Address.City,
		campus.Address.Country,
		campus.Phone,
		campus.Email,
		campus.IsPrimary,
		campus.IsActive,
		campus.UpdatedAt.Format(time.RFC3339Nano),
		campus.ID.String(),
		campus.OrganisationID.String(),
		campus.Version,
	)
	if err != nil {
		return apperr.Internal(err, "campus could not be updated")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err, "campus update result could not be read")
	}
	if affected == 0 {
		return apperr.New(apperr.CodeConflict, "the campus was changed by someone else").
			WithDetail("entity", "campus").
			WithDetail("id", campus.ID.String())
	}

	campus.Version++
	return nil
}

// ByID loads one campus, or nil when it does not exist in that organisation.
func (r *Repository) ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Campus, error) {
	return r.queryOne(ctx,
		"SELECT "+campusColumns+" FROM campuses WHERE id = ? AND organisation_id = ?",
		id.String(), organisationID.String())
}

// Primary loads the home campus of an organisation, or nil when none is set.
func (r *Repository) Primary(ctx context.Context, organisationID ids.UUID) (*domain.Campus, error) {
	return r.queryOne(ctx,
		"SELECT "+campusColumns+" FROM campuses WHERE organisation_id = ? AND is_primary = 1",
		organisationID.String())
}

// List returns the campuses of an organisation ordered by name.
func (r *Repository) List(ctx context.Context, organisationID ids.UUID, includeInactive bool) ([]*domain.Campus, error) {
	query := "SELECT " + campusColumns + " FROM campuses WHERE organisation_id = ?"
	if !includeInactive {
		query += " AND is_active = 1"
	}
	query += " ORDER BY is_primary DESC, name, id"

	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, organisationID.String())
	if err != nil {
		return nil, apperr.Internal(err, "campus listing failed")
	}
	defer rows.Close()

	campuses := []*domain.Campus{}
	for rows.Next() {
		campus, err := scanCampus(rows)
		if err != nil {
			return nil, err
		}
		campuses = append(campuses, campus)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err, "campus listing failed")
	}
	return campuses, nil
}

// ClearPrimary demotes every primary campus of the organisation except one.
func (r *Repository) ClearPrimary(ctx context.Context, organisationID, keep ids.UUID) error {
	_, err := r.transactor.Conn(ctx).ExecContext(ctx,
		"UPDATE campuses SET is_primary = 0 WHERE organisation_id = ? AND is_primary = 1 AND id <> ?",
		organisationID.String(), keep.String())
	if err != nil {
		return apperr.Internal(err, "campus primary flag could not be cleared")
	}
	return nil
}

// Count returns the number of campuses in an organisation.
func (r *Repository) Count(ctx context.Context, organisationID ids.UUID) (int64, error) {
	var total int64
	err := r.transactor.Conn(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM campuses WHERE organisation_id = ?", organisationID.String()).Scan(&total)
	if err != nil {
		return 0, apperr.Internal(err, "campus count failed")
	}
	return total, nil
}

func (r *Repository) queryOne(ctx context.Context, query string, arguments ...any) (*domain.Campus, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, apperr.Internal(err, "campus lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "campus lookup failed")
		}
		return nil, nil
	}

	return scanCampus(rows)
}

func campusArguments(campus *domain.Campus) []any {
	return []any{
		campus.ID.String(),
		campus.OrganisationID.String(),
		campus.Name,
		campus.Code,
		campus.Address.Line1,
		campus.Address.City,
		campus.Address.Country,
		campus.Phone,
		campus.Email,
		campus.IsPrimary,
		campus.IsActive,
		campus.Version,
		campus.CreatedAt.Format(time.RFC3339Nano),
		campus.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func scanCampus(row interface{ Scan(...any) error }) (*domain.Campus, error) {
	var (
		id, organisationID, name, code     string
		line1, city, country, phone, email string
		isPrimary, isActive                bool
		version                            int64
		createdAt, updatedAt               string
	)

	err := row.Scan(&id, &organisationID, &name, &code, &line1, &city, &country,
		&phone, &email, &isPrimary, &isActive, &version, &createdAt, &updatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "campus could not be read")
	}

	identifier, err := ids.Parse(id)
	if err != nil {
		return nil, apperr.Internal(err, fmt.Sprintf("campus %q has an invalid identifier", id))
	}
	owner, err := ids.Parse(organisationID)
	if err != nil {
		return nil, apperr.Internal(err, "campus organisation identifier is invalid")
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, apperr.Internal(err, "campus creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "campus update timestamp is invalid")
	}

	return &domain.Campus{
		ID:             identifier,
		OrganisationID: owner,
		Name:           name,
		Code:           code,
		Address: domain.Address{
			Line1:   line1,
			City:    city,
			Country: country,
		},
		Phone:     phone,
		Email:     email,
		IsPrimary: isPrimary,
		IsActive:  isActive,
		Version:   version,
		CreatedAt: created,
		UpdatedAt: updated,
	}, nil
}
