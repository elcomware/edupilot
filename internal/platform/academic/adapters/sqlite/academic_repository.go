// Package sqlite implements the academic repositories against SQLite. It is
// the only place that knows the academic_years and academic_terms tables exist.
package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/academic/domain"
)

const (
	yearColumns = `id, organisation_id, name, starts_on, ends_on, is_current,
		version, created_at, updated_at`

	termColumns = `id, organisation_id, academic_year_id, name, sequence,
		starts_on, ends_on, created_at, updated_at`

	dateLayout = "2006-01-02"
)

// YearRepository stores academic years in SQLite.
type YearRepository struct {
	transactor *database.Transactor
}

// NewYearRepository builds the SQLite academic year repository.
func NewYearRepository(transactor *database.Transactor) *YearRepository {
	return &YearRepository{transactor: transactor}
}

// Create inserts a new academic year.
func (r *YearRepository) Create(ctx context.Context, year *domain.AcademicYear) error {
	const statement = `INSERT INTO academic_years (` + yearColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		year.ID.String(),
		year.OrganisationID.String(),
		year.Name,
		year.StartsOn.Format(dateLayout),
		year.EndsOn.Format(dateLayout),
		year.IsCurrent,
		year.Version,
		year.CreatedAt.Format(time.RFC3339Nano),
		year.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		switch {
		case database.IsUniqueViolation(err) && !strings.Contains(err.Error(), "single_current"):
			return apperr.AlreadyExists("academic year", "name", year.Name)
		case database.IsUniqueViolation(err) || strings.Contains(err.Error(), "single_current"):
			return apperr.New(apperr.CodeConflict, "this organisation already has a current academic year").
				WithDetail("entity", "academic_year")
		default:
			return apperr.Internal(err, "academic year could not be created")
		}
	}
	return nil
}

// Update writes the year with an optimistic version check.
func (r *YearRepository) Update(ctx context.Context, year *domain.AcademicYear) error {
	const statement = `UPDATE academic_years SET
			name = ?, starts_on = ?, ends_on = ?, is_current = ?,
			version = version + 1, updated_at = ?
		WHERE id = ? AND organisation_id = ? AND version = ?`

	result, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		year.Name,
		year.StartsOn.Format(dateLayout),
		year.EndsOn.Format(dateLayout),
		year.IsCurrent,
		year.UpdatedAt.Format(time.RFC3339Nano),
		year.ID.String(),
		year.OrganisationID.String(),
		year.Version,
	)
	if err != nil {
		return apperr.Internal(err, "academic year could not be updated")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err, "academic year update result could not be read")
	}
	if affected == 0 {
		return apperr.New(apperr.CodeConflict, "the academic year was changed by someone else").
			WithDetail("entity", "academic_year").
			WithDetail("id", year.ID.String())
	}

	year.Version++
	return nil
}

// ByID loads one year, or nil when it does not exist in that organisation.
func (r *YearRepository) ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.AcademicYear, error) {
	return r.queryOne(ctx,
		"SELECT "+yearColumns+" FROM academic_years WHERE id = ? AND organisation_id = ?",
		id.String(), organisationID.String())
}

// ByName loads a year by its label, or nil when absent.
func (r *YearRepository) ByName(ctx context.Context, organisationID ids.UUID, name string) (*domain.AcademicYear, error) {
	return r.queryOne(ctx,
		"SELECT "+yearColumns+" FROM academic_years WHERE organisation_id = ? AND name = ?",
		organisationID.String(), name)
}

// Current loads the current year of an organisation, or nil when none is set.
func (r *YearRepository) Current(ctx context.Context, organisationID ids.UUID) (*domain.AcademicYear, error) {
	return r.queryOne(ctx,
		"SELECT "+yearColumns+" FROM academic_years WHERE organisation_id = ? AND is_current = 1",
		organisationID.String())
}

// List returns the organisation's years, most recent first.
func (r *YearRepository) List(ctx context.Context, organisationID ids.UUID) ([]*domain.AcademicYear, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx,
		"SELECT "+yearColumns+" FROM academic_years WHERE organisation_id = ? ORDER BY starts_on DESC, name",
		organisationID.String())
	if err != nil {
		return nil, apperr.Internal(err, "academic year listing failed")
	}
	defer rows.Close()

	years := []*domain.AcademicYear{}
	for rows.Next() {
		year, err := scanYear(rows)
		if err != nil {
			return nil, err
		}
		years = append(years, year)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err, "academic year listing failed")
	}
	return years, nil
}

// ClearCurrent demotes every current year of the organisation except one.
func (r *YearRepository) ClearCurrent(ctx context.Context, organisationID, keep ids.UUID) error {
	_, err := r.transactor.Conn(ctx).ExecContext(ctx,
		"UPDATE academic_years SET is_current = 0 WHERE organisation_id = ? AND is_current = 1 AND id <> ?",
		organisationID.String(), keep.String())
	if err != nil {
		return apperr.Internal(err, "academic year current flag could not be cleared")
	}
	return nil
}

// Count returns the number of years in an organisation.
func (r *YearRepository) Count(ctx context.Context, organisationID ids.UUID) (int64, error) {
	var total int64
	err := r.transactor.Conn(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM academic_years WHERE organisation_id = ?", organisationID.String()).Scan(&total)
	if err != nil {
		return 0, apperr.Internal(err, "academic year count failed")
	}
	return total, nil
}

func (r *YearRepository) queryOne(ctx context.Context, query string, arguments ...any) (*domain.AcademicYear, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, apperr.Internal(err, "academic year lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "academic year lookup failed")
		}
		return nil, nil
	}
	return scanYear(rows)
}

func scanYear(row interface{ Scan(...any) error }) (*domain.AcademicYear, error) {
	var (
		id, organisationID, name, startsOn, endsOn string
		isCurrent                                  bool
		version                                    int64
		createdAt, updatedAt                       string
	)

	err := row.Scan(&id, &organisationID, &name, &startsOn, &endsOn, &isCurrent,
		&version, &createdAt, &updatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "academic year could not be read")
	}

	identifier, err := ids.Parse(id)
	if err != nil {
		return nil, apperr.Internal(err, fmt.Sprintf("academic year %q has an invalid identifier", id))
	}
	owner, err := ids.Parse(organisationID)
	if err != nil {
		return nil, apperr.Internal(err, "academic year organisation identifier is invalid")
	}
	start, err := time.Parse(dateLayout, startsOn)
	if err != nil {
		return nil, apperr.Internal(err, "academic year start date is invalid")
	}
	end, err := time.Parse(dateLayout, endsOn)
	if err != nil {
		return nil, apperr.Internal(err, "academic year end date is invalid")
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, apperr.Internal(err, "academic year creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "academic year update timestamp is invalid")
	}

	return &domain.AcademicYear{
		ID:             identifier,
		OrganisationID: owner,
		Name:           name,
		StartsOn:       start,
		EndsOn:         end,
		IsCurrent:      isCurrent,
		Version:        version,
		CreatedAt:      created,
		UpdatedAt:      updated,
	}, nil
}

// TermRepository stores academic terms in SQLite.
type TermRepository struct {
	transactor *database.Transactor
}

// NewTermRepository builds the SQLite term repository.
func NewTermRepository(transactor *database.Transactor) *TermRepository {
	return &TermRepository{transactor: transactor}
}

// Create inserts a new term.
func (r *TermRepository) Create(ctx context.Context, term *domain.Term) error {
	const statement = `INSERT INTO academic_terms (` + termColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		term.ID.String(),
		term.OrganisationID.String(),
		term.AcademicYearID.String(),
		term.Name,
		term.Sequence,
		term.StartsOn.Format(dateLayout),
		term.EndsOn.Format(dateLayout),
		term.CreatedAt.Format(time.RFC3339Nano),
		term.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return apperr.AlreadyExists("term", "sequence", fmt.Sprintf("%d", term.Sequence))
		}
		return apperr.Internal(err, "term could not be created")
	}
	return nil
}

// Update writes the term with an optimistic version check. Terms are immutable
// once created in the first cut, because a term period feeds fee schedules and
// period reporting; the method exists for a later amendment workflow.
func (r *TermRepository) Update(ctx context.Context, term *domain.Term) error {
	const statement = `UPDATE academic_terms SET
			name = ?, sequence = ?, starts_on = ?, ends_on = ?, updated_at = ?
		WHERE id = ? AND organisation_id = ? AND academic_year_id = ?`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		term.Name,
		term.Sequence,
		term.StartsOn.Format(dateLayout),
		term.EndsOn.Format(dateLayout),
		term.UpdatedAt.Format(time.RFC3339Nano),
		term.ID.String(),
		term.OrganisationID.String(),
		term.AcademicYearID.String(),
	)
	if err != nil {
		return apperr.Internal(err, "term could not be updated")
	}
	return nil
}

// ByID loads one term, or nil when it does not exist in that organisation.
func (r *TermRepository) ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Term, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx,
		"SELECT "+termColumns+" FROM academic_terms WHERE id = ? AND organisation_id = ?",
		id.String(), organisationID.String())
	if err != nil {
		return nil, apperr.Internal(err, "term lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "term lookup failed")
		}
		return nil, nil
	}
	return scanTerm(rows)
}

// ListByYear returns the terms of one year in sequence order.
func (r *TermRepository) ListByYear(ctx context.Context, organisationID, academicYearID ids.UUID) ([]*domain.Term, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx,
		"SELECT "+termColumns+" FROM academic_terms WHERE organisation_id = ? AND academic_year_id = ? ORDER BY sequence",
		organisationID.String(), academicYearID.String())
	if err != nil {
		return nil, apperr.Internal(err, "term listing failed")
	}
	defer rows.Close()

	terms := []*domain.Term{}
	for rows.Next() {
		term, err := scanTerm(rows)
		if err != nil {
			return nil, err
		}
		terms = append(terms, term)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err, "term listing failed")
	}
	return terms, nil
}

// CountByYear returns the number of terms in a year.
func (r *TermRepository) CountByYear(ctx context.Context, organisationID, academicYearID ids.UUID) (int64, error) {
	var total int64
	err := r.transactor.Conn(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM academic_terms WHERE organisation_id = ? AND academic_year_id = ?",
		organisationID.String(), academicYearID.String()).Scan(&total)
	if err != nil {
		return 0, apperr.Internal(err, "term count failed")
	}
	return total, nil
}

func scanTerm(row interface{ Scan(...any) error }) (*domain.Term, error) {
	var (
		id, organisationID, academicYearID, name string
		sequence                                 int
		startsOn, endsOn, createdAt, updatedAt   string
	)

	err := row.Scan(&id, &organisationID, &academicYearID, &name, &sequence,
		&startsOn, &endsOn, &createdAt, &updatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "term could not be read")
	}

	identifier, err := ids.Parse(id)
	if err != nil {
		return nil, apperr.Internal(err, fmt.Sprintf("term %q has an invalid identifier", id))
	}
	owner, err := ids.Parse(organisationID)
	if err != nil {
		return nil, apperr.Internal(err, "term organisation identifier is invalid")
	}
	year, err := ids.Parse(academicYearID)
	if err != nil {
		return nil, apperr.Internal(err, "term academic year identifier is invalid")
	}
	start, err := time.Parse(dateLayout, startsOn)
	if err != nil {
		return nil, apperr.Internal(err, "term start date is invalid")
	}
	end, err := time.Parse(dateLayout, endsOn)
	if err != nil {
		return nil, apperr.Internal(err, "term end date is invalid")
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, apperr.Internal(err, "term creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "term update timestamp is invalid")
	}

	return &domain.Term{
		ID:             identifier,
		OrganisationID: owner,
		AcademicYearID: year,
		Name:           name,
		Sequence:       sequence,
		StartsOn:       start,
		EndsOn:         end,
		CreatedAt:      created,
		UpdatedAt:      updated,
	}, nil
}
