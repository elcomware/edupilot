// Package sqlite implements the people repositories against SQLite. It is the
// only place that knows the people tables exist.
package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

const (
	dateLayout = "2006-01-02"
)

// personColumnList is the single source of truth for the people column order.
// The SELECT list and the INSERT placeholders are both derived from it, so a
// column added here cannot drift out of step with the number of bind values.
var personColumnList = []string{
	"id", "organisation_id", "first_name", "middle_name", "last_name", "preferred_name",
	"email", "phone", "secondary_phone", "date_of_birth", "gender", "nationality", "national_id",
	"address_line1", "address_line2", "city", "postal_code", "country_code", "photo_document_id",
	"is_active", "version", "created_at", "updated_at",
}

var (
	personColumns           = strings.Join(personColumnList, ", ")
	personInsertPlaceholdes = placeholders(len(personColumnList))
)

// placeholders builds "(?, ?, ?)" for the given number of columns.
func placeholders(count int) string {
	if count <= 0 {
		return "()"
	}
	return "(" + strings.TrimSuffix(strings.Repeat("?, ", count), ", ") + ")"
}

// PersonRepository stores people in SQLite.
type PersonRepository struct {
	transactor *database.Transactor
}

// NewPersonRepository builds the SQLite person repository.
func NewPersonRepository(transactor *database.Transactor) *PersonRepository {
	return &PersonRepository{transactor: transactor}
}

// Create inserts a new person.
func (r *PersonRepository) Create(ctx context.Context, person *domain.Person) error {
	statement := `INSERT INTO people (` + personColumns + `) VALUES ` + personInsertPlaceholdes

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement, personArguments(person)...)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return apperr.AlreadyExists("person", "identifier", person.ID.String())
		}
		return apperr.Internal(err, "person could not be created")
	}
	return nil
}

// personUpdatableColumns are the columns an update may write, in the order the
// UPDATE statement binds them. organisation_id is absent on purpose: a person is
// never moved between schools by editing them, and id and version are handled by
// the WHERE clause.
var personUpdatableColumns = []string{
	"first_name", "middle_name", "last_name", "preferred_name",
	"email", "phone", "secondary_phone", "date_of_birth",
	"gender", "nationality", "national_id",
	"address_line1", "address_line2", "city", "postal_code", "country_code",
	"photo_document_id", "is_active", "updated_at",
}

// Update writes the person with an optimistic version check.
func (r *PersonRepository) Update(ctx context.Context, person *domain.Person) error {
	assignments := "version = version + 1"
	for _, column := range personUpdatableColumns {
		assignments += ", " + column + " = ?"
	}
	statement := "UPDATE people SET " + assignments +
		" WHERE id = ? AND organisation_id = ? AND version = ?"

	arguments := append(personUpdateArguments(person),
		person.ID.String(), person.OrganisationID.String(), person.Version)

	result, err := r.transactor.Conn(ctx).ExecContext(ctx, statement, arguments...)
	if err != nil {
		return apperr.Internal(err, "person could not be updated")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err, "person update result could not be read")
	}
	if affected == 0 {
		return apperr.New(apperr.CodeConflict, "the person was changed by someone else").
			WithDetail("entity", "person").
			WithDetail("id", person.ID.String())
	}

	person.Version++
	return nil
}

// ByID loads one person, or nil when it does not exist in that organisation.
func (r *PersonRepository) ByID(ctx context.Context, organisationID, id ids.UUID) (*domain.Person, error) {
	return r.queryOne(ctx,
		"SELECT "+personColumns+" FROM people WHERE id = ? AND organisation_id = ?",
		id.String(), organisationID.String())
}

// List returns people ordered by last name, optionally filtered by a search term
// over the names and email address. The term is always a bound parameter and is
// escaped for LIKE, never interpolated into the statement.
func (r *PersonRepository) List(ctx context.Context, organisationID ids.UUID, search string, includeInactive bool) ([]*domain.Person, error) {
	query := "SELECT " + personColumns + " FROM people WHERE organisation_id = ?"
	arguments := []any{organisationID.String()}

	if trimmed := strings.TrimSpace(search); trimmed != "" {
		pattern := "%" + escapeLike(trimmed) + "%"
		query += ` AND (last_name LIKE ? ESCAPE '\' OR first_name LIKE ? ESCAPE '\' ` +
			`OR preferred_name LIKE ? ESCAPE '\' OR email LIKE ? ESCAPE '\')`
		arguments = append(arguments, pattern, pattern, pattern, pattern)
	}
	if !includeInactive {
		query += " AND is_active = 1"
	}
	query += " ORDER BY last_name, first_name, id"

	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, apperr.Internal(err, "people listing failed")
	}
	defer rows.Close()

	people := []*domain.Person{}
	for rows.Next() {
		person, err := scanPerson(rows)
		if err != nil {
			return nil, err
		}
		people = append(people, person)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err, "people listing failed")
	}
	return people, nil
}

// FindByNationalID returns the person holding a national identifier, or nil. It
// is how an import recognises somebody who already exists.
func (r *PersonRepository) FindByNationalID(ctx context.Context, organisationID ids.UUID, nationalID string) (*domain.Person, error) {
	if strings.TrimSpace(nationalID) == "" {
		return nil, nil
	}
	return r.queryOne(ctx,
		"SELECT "+personColumns+" FROM people WHERE organisation_id = ? AND national_id = ?",
		organisationID.String(), nationalID)
}

// Count returns the number of people in an organisation.
func (r *PersonRepository) Count(ctx context.Context, organisationID ids.UUID) (int64, error) {
	var total int64
	err := r.transactor.Conn(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM people WHERE organisation_id = ?", organisationID.String()).Scan(&total)
	if err != nil {
		return 0, apperr.Internal(err, "people count failed")
	}
	return total, nil
}

func (r *PersonRepository) queryOne(ctx context.Context, query string, arguments ...any) (*domain.Person, error) {
	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, apperr.Internal(err, "person lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "person lookup failed")
		}
		return nil, nil
	}
	return scanPerson(rows)
}

// personArguments returns the insert column order, one value per entry of
// personColumnList. person_update_arguments_test.go asserts the two stay the
// same length, which is the failure this repository previously shipped.
func personArguments(person *domain.Person) []any {
	var birth any
	if person.DateOfBirth != nil {
		birth = person.DateOfBirth.Format(dateLayout)
	}
	var photo any
	if person.PhotoDocumentID != nil {
		photo = person.PhotoDocumentID.String()
	}

	return []any{
		person.ID.String(),
		person.OrganisationID.String(),
		person.FirstName,
		person.MiddleName,
		person.LastName,
		person.PreferredName,
		person.Email,
		person.Phone,
		person.SecondaryPhone,
		birth,
		person.Gender,
		person.Nationality,
		person.NationalID,
		person.Address.Line1,
		person.Address.Line2,
		person.Address.City,
		person.Address.PostalCode,
		person.Address.Country,
		photo,
		person.IsActive,
		person.Version,
		person.CreatedAt.Format(time.RFC3339Nano),
		person.UpdatedAt.Format(time.RFC3339Nano),
	}
}

// personUpdateArguments returns the bind values for the UPDATE statement, in the
// order of personUpdatableColumns. The insert and update column orders are
// different, so the two argument lists are built separately rather than sliced
// out of one another.
func personUpdateArguments(person *domain.Person) []any {
	var birth any
	if person.DateOfBirth != nil {
		birth = person.DateOfBirth.Format(dateLayout)
	}
	var photo any
	if person.PhotoDocumentID != nil {
		photo = person.PhotoDocumentID.String()
	}

	return []any{
		person.FirstName,
		person.MiddleName,
		person.LastName,
		person.PreferredName,
		person.Email,
		person.Phone,
		person.SecondaryPhone,
		birth,
		person.Gender,
		person.Nationality,
		person.NationalID,
		person.Address.Line1,
		person.Address.Line2,
		person.Address.City,
		person.Address.PostalCode,
		person.Address.Country,
		photo,
		person.IsActive,
		person.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func scanPerson(row interface{ Scan(...any) error }) (*domain.Person, error) {
	var (
		rawID, rawOrganisationID                       string
		firstName, middleName, lastName, preferredName string
		email, phone, secondaryPhone                   string
		birthDate, gender, nationality, nationalID     sql.NullString
		line1, line2, city, postalCode, country        string
		photoDocumentID                                sql.NullString
		isActive                                       bool
		version                                        int64
		rawCreatedAt, rawUpdatedAt                     string
	)

	err := row.Scan(&rawID, &rawOrganisationID, &firstName, &middleName, &lastName, &preferredName,
		&email, &phone, &secondaryPhone, &birthDate, &gender, &nationality, &nationalID,
		&line1, &line2, &city, &postalCode, &country, &photoDocumentID,
		&isActive, &version, &rawCreatedAt, &rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "person could not be read")
	}

	identifier, err := ids.Parse(rawID)
	if err != nil {
		return nil, apperr.Internal(err, "person identifier is invalid")
	}
	owner, err := ids.Parse(rawOrganisationID)
	if err != nil {
		return nil, apperr.Internal(err, "person organisation identifier is invalid")
	}
	created, err := time.Parse(time.RFC3339, rawCreatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "person creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "person update timestamp is invalid")
	}

	person := &domain.Person{
		ID:             identifier,
		OrganisationID: owner,
		FirstName:      firstName,
		MiddleName:     middleName,
		LastName:       lastName,
		PreferredName:  preferredName,
		Email:          email,
		Phone:          phone,
		SecondaryPhone: secondaryPhone,
		Gender:         gender.String,
		Nationality:    nationality.String,
		NationalID:     nationalID.String,
		Address: domain.Address{
			Line1:      line1,
			Line2:      line2,
			City:       city,
			PostalCode: postalCode,
			Country:    country,
		},
		IsActive:  isActive,
		Version:   version,
		CreatedAt: created,
		UpdatedAt: updated,
	}

	if birthDate.Valid && birthDate.String != "" {
		parsed, err := time.Parse(dateLayout, birthDate.String)
		if err != nil {
			return nil, apperr.Internal(err, "person date of birth is invalid")
		}
		person.DateOfBirth = &parsed
	}
	if photoDocumentID.Valid && photoDocumentID.String != "" {
		parsed, err := ids.Parse(photoDocumentID.String)
		if err != nil {
			return nil, apperr.Internal(err, "person photo identifier is invalid")
		}
		person.PhotoDocumentID = &parsed
	}
	return person, nil
}

// escapeLike neutralises the LIKE wildcards in a user's search term so that
// searching for "100%" does not match every row.
func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
