package sqlite

import (
	"context"
	"strings"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

// householdColumnList is the single source of truth for the households column
// order, so the SELECT list and the INSERT placeholders cannot disagree.
var householdColumnList = []string{
	"id", "organisation_id", "academic_year_id", "name",
	"billing_email", "billing_phone",
	"billing_address_line1", "billing_address_line2",
	"billing_city", "billing_postal_code", "billing_country_code",
	"version", "created_at", "updated_at",
}

var householdColumns = strings.Join(householdColumnList, ", ")

// householdUpdatableColumns are the columns an update may write. organisation_id
// and academic_year_id are absent on purpose: a household is never moved between
// tenants or between years, and a family is regrouped by creating a new household.
var householdUpdatableColumns = []string{
	"name", "billing_email", "billing_phone",
	"billing_address_line1", "billing_address_line2",
	"billing_city", "billing_postal_code", "billing_country_code", "updated_at",
}

// HouseholdRepository stores billing parties and their membership in SQLite.
type HouseholdRepository struct {
	transactor *database.Transactor
}

// NewHouseholdRepository builds the SQLite household repository.
func NewHouseholdRepository(transactor *database.Transactor) *HouseholdRepository {
	return &HouseholdRepository{transactor: transactor}
}

// Create inserts a household. Members are written separately by AddMember, because
// household_members references this row.
func (r *HouseholdRepository) Create(ctx context.Context, household *domain.Household) error {
	statement := `INSERT INTO households (` + householdColumns + `) VALUES ` + placeholders(len(householdColumnList))

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement, householdArguments(household)...)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return apperr.AlreadyExists("household", "name in this academic year", household.Name)
		}
		return apperr.Internal(err, "household could not be created")
	}
	return nil
}

// Update writes the household with an optimistic version check.
func (r *HouseholdRepository) Update(ctx context.Context, household *domain.Household) error {
	assignments := "version = version + 1"
	for _, column := range householdUpdatableColumns {
		assignments += ", " + column + " = ?"
	}
	statement := "UPDATE households SET " + assignments +
		" WHERE id = ? AND organisation_id = ? AND version = ?"

	arguments := append(householdUpdateArguments(household),
		household.ID.String(), household.OrganisationID.String(), household.Version)

	result, err := r.transactor.Conn(ctx).ExecContext(ctx, statement, arguments...)
	if err != nil {
		return apperr.Internal(err, "household could not be updated")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err, "household update result could not be read")
	}
	if affected == 0 {
		return apperr.New(apperr.CodeConflict, "the household was changed by someone else").
			WithDetail("entity", "household").
			WithDetail("id", household.ID.String())
	}

	household.Version++
	return nil
}

// ByID loads one household with its members, or nil when it does not exist.
func (r *HouseholdRepository) ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Household, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx,
		"SELECT "+householdColumns+" FROM households WHERE id = ? AND organisation_id = ?",
		id.String(), organisationID.String())
	if err != nil {
		return nil, apperr.Internal(err, "household lookup failed")
	}

	if !rows.Next() {
		err := rows.Err()
		rows.Close()
		if err != nil {
			return nil, apperr.Internal(err, "household lookup failed")
		}
		return nil, nil
	}

	household, err := scanHousehold(rows)
	// The household rows must be released before the members are read. SQLite
	// standalone runs a single connection, so a second open statement would
	// wait for the first to close and the read would deadlock.
	rows.Close()
	if err != nil {
		return nil, err
	}

	if err := r.loadMembers(ctx, household); err != nil {
		return nil, err
	}
	return household, nil
}

// ListByYear returns the organisation's households for a year, each with members.
func (r *HouseholdRepository) ListByYear(ctx context.Context, organisationID, academicYearID ids.UUID) ([]*domain.Household, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx,
		"SELECT "+householdColumns+` FROM households
		 WHERE organisation_id = ? AND academic_year_id = ? ORDER BY name, id`,
		organisationID.String(), academicYearID.String())
	if err != nil {
		return nil, apperr.Internal(err, "household listing failed")
	}

	households := []*domain.Household{}
	for rows.Next() {
		household, err := scanHousehold(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		households = append(households, household)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, apperr.Internal(err, "household listing failed")
	}
	rows.Close()

	// Members are read after the household rows are closed, because SQLite
	// cannot always have two open statements over the same connection.
	for _, household := range households {
		if err := r.loadMembers(ctx, household); err != nil {
			return nil, err
		}
	}
	return households, nil
}

// HouseholdsForPerson returns the households a person belongs to in a year.
func (r *HouseholdRepository) HouseholdsForPerson(ctx context.Context, organisationID, personID, academicYearID ids.UUID) ([]*domain.Household, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx,
		"SELECT "+householdColumns+` FROM households
		 WHERE organisation_id = ? AND academic_year_id = ?
		   AND id IN (SELECT household_id FROM household_members WHERE person_id = ?)
		 ORDER BY name, id`,
		organisationID.String(), academicYearID.String(), personID.String())
	if err != nil {
		return nil, apperr.Internal(err, "household lookup failed")
	}

	households := []*domain.Household{}
	for rows.Next() {
		household, err := scanHousehold(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		households = append(households, household)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, apperr.Internal(err, "household lookup failed")
	}
	rows.Close()

	for _, household := range households {
		if err := r.loadMembers(ctx, household); err != nil {
			return nil, err
		}
	}
	return households, nil
}

// AddMember places one person in a household. A person may be in several
// households, but only once in any one household.
func (r *HouseholdRepository) AddMember(ctx context.Context, member domain.HouseholdMember) error {
	const statement = `INSERT INTO household_members
			(household_id, person_id, member_role, is_billing_party, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (household_id, person_id) DO UPDATE SET
			member_role = excluded.member_role,
			is_billing_party = excluded.is_billing_party`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		member.HouseholdID.String(),
		member.PersonID.String(),
		string(member.Role),
		member.IsBillingPart,
		member.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return apperr.Internal(err, "household member could not be saved")
	}
	return nil
}

func (r *HouseholdRepository) loadMembers(ctx context.Context, household *domain.Household) error {
	const query = `SELECT household_id, person_id, member_role, is_billing_party, created_at
		FROM household_members WHERE household_id = ? ORDER BY member_role, person_id`

	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, household.ID.String())
	if err != nil {
		return apperr.Internal(err, "household members could not be read")
	}
	defer rows.Close()

	household.Members = nil
	for rows.Next() {
		var (
			rawHouseholdID, rawPersonID, rawRole, rawCreatedAt string
			isBillingPart                                      bool
		)
		if err := rows.Scan(&rawHouseholdID, &rawPersonID, &rawRole, &isBillingPart, &rawCreatedAt); err != nil {
			return apperr.Internal(err, "household member could not be read")
		}
		personID, err := ids.Parse(rawPersonID)
		if err != nil {
			return apperr.Internal(err, "household member person identifier is invalid")
		}
		created, err := time.Parse(time.RFC3339, rawCreatedAt)
		if err != nil {
			return apperr.Internal(err, "household member creation timestamp is invalid")
		}

		household.Members = append(household.Members, domain.HouseholdMember{
			HouseholdID:   household.ID,
			PersonID:      personID,
			Role:          domain.MemberRole(rawRole),
			IsBillingPart: isBillingPart,
			CreatedAt:     created,
		})
	}
	if err := rows.Err(); err != nil {
		return apperr.Internal(err, "household members could not be read")
	}
	return nil
}

// householdArguments returns the insert bind values, one per entry of
// householdColumnList.
func householdArguments(household *domain.Household) []any {
	return []any{
		household.ID.String(),
		household.OrganisationID.String(),
		household.AcademicYearID.String(),
		household.Name,
		household.BillingEmail,
		household.BillingPhone,
		household.BillingAddress.Line1,
		household.BillingAddress.Line2,
		household.BillingAddress.City,
		household.BillingAddress.PostalCode,
		household.BillingAddress.Country,
		household.Version,
		household.CreatedAt.Format(time.RFC3339Nano),
		household.UpdatedAt.Format(time.RFC3339Nano),
	}
}

// householdUpdateArguments returns the UPDATE bind values, in the order of
// householdUpdatableColumns.
func householdUpdateArguments(household *domain.Household) []any {
	return []any{
		household.Name,
		household.BillingEmail,
		household.BillingPhone,
		household.BillingAddress.Line1,
		household.BillingAddress.Line2,
		household.BillingAddress.City,
		household.BillingAddress.PostalCode,
		household.BillingAddress.Country,
		household.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func scanHousehold(row interface{ Scan(...any) error }) (*domain.Household, error) {
	var (
		rawID, rawOrganisationID, rawYearID, name string
		email, phone                              string
		line1, line2, city, postalCode, country   string
		version                                   int64
		rawCreatedAt, rawUpdatedAt                string
	)

	err := row.Scan(&rawID, &rawOrganisationID, &rawYearID, &name,
		&email, &phone, &line1, &line2, &city, &postalCode, &country,
		&version, &rawCreatedAt, &rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "household could not be read")
	}

	identifier, err := ids.Parse(rawID)
	if err != nil {
		return nil, apperr.Internal(err, "household identifier is invalid")
	}
	owner, err := ids.Parse(rawOrganisationID)
	if err != nil {
		return nil, apperr.Internal(err, "household organisation identifier is invalid")
	}
	year, err := ids.Parse(rawYearID)
	if err != nil {
		return nil, apperr.Internal(err, "household academic year identifier is invalid")
	}
	created, err := time.Parse(time.RFC3339, rawCreatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "household creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "household update timestamp is invalid")
	}

	return &domain.Household{
		ID:             identifier,
		OrganisationID: owner,
		AcademicYearID: year,
		Name:           name,
		BillingEmail:   email,
		BillingPhone:   phone,
		BillingAddress: domain.Address{
			Line1:      line1,
			Line2:      line2,
			City:       city,
			PostalCode: postalCode,
			Country:    country,
		},
		Version:   version,
		CreatedAt: created,
		UpdatedAt: updated,
	}, nil
}
