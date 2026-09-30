// Package sqlite implements the organisation repository against SQLite. It is
// the only place that knows the organisations table exists.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/kernel/money"
	"github.com/elcomware/edupilot/internal/platform/organisation/domain"
)

const organisationColumns = `id, name, legal_name, currency, country, default_locale, tax_number,
	address_line1, address_line2, city, postal_code, country_code, is_active, version, created_at, updated_at`

// Repository stores organisations in SQLite.
type Repository struct {
	transactor *database.Transactor
}

// NewRepository builds the SQLite organisation repository.
func NewRepository(transactor *database.Transactor) *Repository {
	return &Repository{transactor: transactor}
}

// Create inserts a new organisation.
func (r *Repository) Create(ctx context.Context, organisation *domain.Organisation) error {
	const statement = `INSERT INTO organisations (` + organisationColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement, organisationArguments(organisation)...)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return apperr.AlreadyExists("organisation", "id", organisation.ID.String())
		}
		return apperr.Internal(err, "organisation could not be created")
	}
	return nil
}

// Update writes the organisation with an optimistic version check. A stale
// writer is rejected rather than silently overwriting a colleague's change.
func (r *Repository) Update(ctx context.Context, organisation *domain.Organisation) error {
	const statement = `UPDATE organisations SET
			name = ?, legal_name = ?, currency = ?, country = ?, default_locale = ?, tax_number = ?,
			address_line1 = ?, address_line2 = ?, city = ?, postal_code = ?, country_code = ?,
			is_active = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND version = ?`

	result, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		organisation.Name,
		organisation.LegalName,
		organisation.Currency.Code(),
		organisation.Country,
		organisation.DefaultLocale,
		organisation.TaxNumber,
		organisation.Address.Line1,
		organisation.Address.Line2,
		organisation.Address.City,
		organisation.Address.PostalCode,
		organisation.Address.Country,
		organisation.IsActive,
		organisation.UpdatedAt.Format(time.RFC3339Nano),
		organisation.ID.String(),
		organisation.Version,
	)
	if err != nil {
		return apperr.Internal(err, "organisation could not be updated")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err, "organisation update result could not be read")
	}
	if affected == 0 {
		return apperr.New(apperr.CodeConflict, "the organisation was changed by someone else").
			WithDetail("entity", "organisation").
			WithDetail("id", organisation.ID.String())
	}

	organisation.Version++
	return nil
}

// ByID loads an organisation, or nil when it does not exist.
func (r *Repository) ByID(ctx context.Context, id ids.UUID) (*domain.Organisation, error) {
	return r.queryOne(ctx,
		"SELECT "+organisationColumns+" FROM organisations WHERE id = ?", id.String())
}

// First loads the oldest organisation, which is the tenant of a
// single-organisation installation.
func (r *Repository) First(ctx context.Context) (*domain.Organisation, error) {
	return r.queryOne(ctx,
		"SELECT "+organisationColumns+" FROM organisations ORDER BY created_at, id LIMIT 1")
}

// Exists reports whether an organisation id is already taken.
func (r *Repository) Exists(ctx context.Context, id ids.UUID) (bool, error) {
	var found int
	err := r.transactor.Conn(ctx).QueryRowContext(ctx,
		"SELECT 1 FROM organisations WHERE id = ?", id.String()).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, apperr.Internal(err, "organisation lookup failed")
	}
	return true, nil
}

// Count returns the number of organisations.
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var total int64
	if err := r.transactor.Conn(ctx).QueryRowContext(ctx, "SELECT COUNT(*) FROM organisations").Scan(&total); err != nil {
		return 0, apperr.Internal(err, "organisation count failed")
	}
	return total, nil
}

func (r *Repository) queryOne(ctx context.Context, query string, arguments ...any) (*domain.Organisation, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, apperr.Internal(err, "organisation lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "organisation lookup failed")
		}
		return nil, nil
	}

	organisation, err := scanOrganisation(rows)
	if err != nil {
		return nil, err
	}
	return organisation, nil
}

func organisationArguments(organisation *domain.Organisation) []any {
	return []any{
		organisation.ID.String(),
		organisation.Name,
		organisation.LegalName,
		organisation.Currency.Code(),
		organisation.Country,
		organisation.DefaultLocale,
		organisation.TaxNumber,
		organisation.Address.Line1,
		organisation.Address.Line2,
		organisation.Address.City,
		organisation.Address.PostalCode,
		organisation.Address.Country,
		organisation.IsActive,
		organisation.Version,
		organisation.CreatedAt.Format(time.RFC3339Nano),
		organisation.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func scanOrganisation(row interface{ Scan(...any) error }) (*domain.Organisation, error) {
	var (
		id, name, legalName, currencyCode, country, locale, taxNumber string
		line1, line2, city, postalCode, addressCountry                string
		isActive                                                      bool
		version                                                       int64
		createdAt, updatedAt                                          string
	)

	err := row.Scan(&id, &name, &legalName, &currencyCode, &country, &locale, &taxNumber,
		&line1, &line2, &city, &postalCode, &addressCountry,
		&isActive, &version, &createdAt, &updatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "organisation could not be read")
	}

	identifier, err := ids.Parse(id)
	if err != nil {
		return nil, apperr.Internal(err, fmt.Sprintf("organisation %q has an invalid identifier", id))
	}

	currency, err := money.LookupCurrency(currencyCode)
	if err != nil {
		return nil, apperr.Internal(err, fmt.Sprintf("organisation %q has an unsupported currency", id))
	}

	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, apperr.Internal(err, "organisation creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "organisation update timestamp is invalid")
	}

	return &domain.Organisation{
		ID:            identifier,
		Name:          name,
		LegalName:     legalName,
		Currency:      currency,
		Country:       country,
		DefaultLocale: locale,
		TaxNumber:     taxNumber,
		Address: domain.Address{
			Line1:      line1,
			Line2:      line2,
			City:       city,
			PostalCode: postalCode,
			Country:    addressCountry,
		},
		IsActive:  isActive,
		Version:   version,
		CreatedAt: created,
		UpdatedAt: updated,
	}, nil
}
