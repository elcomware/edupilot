package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

const roleColumns = `id, organisation_id, person_id, academic_year_id, role,
	starts_on, ends_on, is_current, version, created_at, updated_at`

// RoleRepository stores person roles in SQLite.
type RoleRepository struct {
	transactor *database.Transactor
}

// NewRoleRepository builds the SQLite person role repository.
func NewRoleRepository(transactor *database.Transactor) *RoleRepository {
	return &RoleRepository{transactor: transactor}
}

// Create inserts a new person role.
func (r *RoleRepository) Create(ctx context.Context, role *domain.PersonRole) error {
	const statement = `INSERT INTO person_roles (` + roleColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		role.ID.String(),
		role.OrganisationID.String(),
		role.PersonID.String(),
		role.AcademicYearID.String(),
		string(role.Role),
		optionalDate(role.StartsOn),
		optionalDate(role.EndsOn),
		role.IsCurrent,
		role.Version,
		role.CreatedAt.Format(time.RFC3339Nano),
		role.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return apperr.AlreadyExists("person role", "role in this academic year", string(role.Role))
		}
		return apperr.Internal(err, "person role could not be created")
	}
	return nil
}

// Update writes the person role.
func (r *RoleRepository) Update(ctx context.Context, role *domain.PersonRole) error {
	const statement = `UPDATE person_roles SET
			starts_on = ?, ends_on = ?, is_current = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND organisation_id = ? AND version = ?`

	result, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		optionalDate(role.StartsOn),
		optionalDate(role.EndsOn),
		role.IsCurrent,
		role.UpdatedAt.Format(time.RFC3339Nano),
		role.ID.String(),
		role.OrganisationID.String(),
		role.Version,
	)
	if err != nil {
		return apperr.Internal(err, "person role could not be updated")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err, "person role update result could not be read")
	}
	if affected == 0 {
		return apperr.New(apperr.CodeConflict, "the person role was changed by someone else").
			WithDetail("entity", "person_role").
			WithDetail("id", role.ID.String())
	}

	role.Version++
	return nil
}

// ByID loads one role, or nil when it does not exist in that organisation.
func (r *RoleRepository) ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.PersonRole, error) {
	return r.queryOne(ctx,
		"SELECT "+roleColumns+" FROM person_roles WHERE id = ? AND organisation_id = ?",
		id.String(), organisationID.String())
}

// ByPersonAndRole returns the person's role for one year, or nil.
func (r *RoleRepository) ByPersonAndRole(ctx context.Context, organisationID, personID, academicYearID ids.UUID, role domain.Role) (*domain.PersonRole, error) {
	return r.queryOne(ctx,
		"SELECT "+roleColumns+` FROM person_roles
		 WHERE organisation_id = ? AND person_id = ? AND academic_year_id = ? AND role = ?`,
		organisationID.String(), personID.String(), academicYearID.String(), string(role))
}

// ListByPerson returns every role the person holds in one year.
func (r *RoleRepository) ListByPerson(ctx context.Context, organisationID, personID, academicYearID ids.UUID) ([]*domain.PersonRole, error) {
	return r.queryMany(ctx,
		"SELECT "+roleColumns+` FROM person_roles
		 WHERE organisation_id = ? AND person_id = ? AND academic_year_id = ?
		 ORDER BY role, id`,
		organisationID.String(), personID.String(), academicYearID.String())
}

// ListByPersons returns the roles a set of people hold in one year, in one query.
func (r *RoleRepository) ListByPersons(ctx context.Context, organisationID ids.UUID, personIDs []ids.UUID, academicYearID ids.UUID) ([]*domain.PersonRole, error) {
	if len(personIDs) == 0 {
		return []*domain.PersonRole{}, nil
	}

	arguments := make([]any, 0, len(personIDs)+2)
	arguments = append(arguments, organisationID.String(), academicYearID.String())
	for _, personID := range personIDs {
		arguments = append(arguments, personID.String())
	}

	return r.queryMany(ctx,
		"SELECT "+roleColumns+` FROM person_roles
		 WHERE organisation_id = ? AND academic_year_id = ? AND person_id IN `+placeholders(len(personIDs))+`
		 ORDER BY person_id, role, id`,
		arguments...)
}

// ListByRole returns everyone holding a role in one year, which is how a roster
// or a payroll run is built.
func (r *RoleRepository) ListByRole(ctx context.Context, organisationID, academicYearID ids.UUID, role domain.Role) ([]*domain.PersonRole, error) {
	return r.queryMany(ctx,
		"SELECT "+roleColumns+` FROM person_roles
		 WHERE organisation_id = ? AND academic_year_id = ? AND role = ?
		 ORDER BY person_id`,
		organisationID.String(), academicYearID.String(), string(role))
}

// CountByRole returns how many people hold a role in one year.
func (r *RoleRepository) CountByRole(ctx context.Context, organisationID, academicYearID ids.UUID, role domain.Role) (int64, error) {
	var total int64
	err := r.transactor.Conn(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM person_roles WHERE organisation_id = ? AND academic_year_id = ? AND role = ?",
		organisationID.String(), academicYearID.String(), string(role)).Scan(&total)
	if err != nil {
		return 0, apperr.Internal(err, "person role count failed")
	}
	return total, nil
}

// ClearPrimaryGuardian demotes every primary guardian edge pointing at a student
// in one year except one. It is the single-writer half of the rule that a
// student has at most one billing guardian, mirroring the campus primary flag.
func (r *RoleRepository) ClearPrimaryGuardian(ctx context.Context, organisationID, toPersonID, academicYearID, keep ids.UUID) error {
	const statement = `UPDATE person_relationships
			SET is_primary = 0, version = version + 1
		WHERE organisation_id = ?
		  AND academic_year_id = ?
		  AND to_person_id = ?
		  AND relationship_type = ?
		  AND is_primary = 1
		  AND id <> ?`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		organisationID.String(), academicYearID.String(), toPersonID.String(),
		string(domain.RelationshipGuardianOf), keep.String())
	if err != nil {
		return apperr.Internal(err, "primary guardian could not be cleared")
	}
	return nil
}

func (r *RoleRepository) queryOne(ctx context.Context, query string, arguments ...any) (*domain.PersonRole, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, apperr.Internal(err, "person role lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "person role lookup failed")
		}
		return nil, nil
	}
	return scanRole(rows)
}

func (r *RoleRepository) queryMany(ctx context.Context, query string, arguments ...any) ([]*domain.PersonRole, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, apperr.Internal(err, "person role listing failed")
	}
	defer rows.Close()

	roles := []*domain.PersonRole{}
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err, "person role listing failed")
	}
	return roles, nil
}

func scanRole(row interface{ Scan(...any) error }) (*domain.PersonRole, error) {
	var (
		rawID, rawOrganisationID, rawPersonID, rawYearID, rawRole string
		startsOn, endsOn                                          sql.NullString
		isCurrent                                                 bool
		version                                                   int64
		rawCreatedAt, rawUpdatedAt                                string
	)

	err := row.Scan(&rawID, &rawOrganisationID, &rawPersonID, &rawYearID, &rawRole,
		&startsOn, &endsOn, &isCurrent, &version, &rawCreatedAt, &rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "person role could not be read")
	}

	identifier, err := ids.Parse(rawID)
	if err != nil {
		return nil, apperr.Internal(err, "person role identifier is invalid")
	}
	owner, err := ids.Parse(rawOrganisationID)
	if err != nil {
		return nil, apperr.Internal(err, "person role organisation identifier is invalid")
	}
	person, err := ids.Parse(rawPersonID)
	if err != nil {
		return nil, apperr.Internal(err, "person role person identifier is invalid")
	}
	year, err := ids.Parse(rawYearID)
	if err != nil {
		return nil, apperr.Internal(err, "person role academic year identifier is invalid")
	}
	created, err := time.Parse(time.RFC3339, rawCreatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "person role creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "person role update timestamp is invalid")
	}

	role := &domain.PersonRole{
		ID:             identifier,
		OrganisationID: owner,
		PersonID:       person,
		AcademicYearID: year,
		Role:           domain.Role(rawRole),
		IsCurrent:      isCurrent,
		Version:        version,
		CreatedAt:      created,
		UpdatedAt:      updated,
	}

	if startsOn.Valid && startsOn.String != "" {
		parsed, err := time.Parse(dateLayout, startsOn.String)
		if err != nil {
			return nil, apperr.Internal(err, "person role start date is invalid")
		}
		role.StartsOn = &parsed
	}
	if endsOn.Valid && endsOn.String != "" {
		parsed, err := time.Parse(dateLayout, endsOn.String)
		if err != nil {
			return nil, apperr.Internal(err, "person role end date is invalid")
		}
		role.EndsOn = &parsed
	}
	return role, nil
}

func optionalDate(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format(dateLayout)
}
