// Package domain holds the person aggregate, the roles a person holds and the
// relationships between people. It depends on nothing but the kernel.
//
// The model is set out in docs/adr/ADR-011. Three decisions carry most of the
// weight:
//
//   - A person is an identity, not a role. There is no "type" column. A teacher
//     who is also a parent is one Person holding two PersonRole rows in the same
//     academic year, and "which roles does this person hold" is always a query.
//   - A role belongs to an academic year, because a child is a student in some
//     years and an alumnus later.
//   - Every relationship between two people is one directed, typed edge. No
//     domain owns a family tree.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
)

// Person is a human being's identity at one organisation: who they are,
// independent of what they do at the school. It is deliberately not year-scoped
// — a person exists before they enrol and after they graduate.
//
// Class membership is not here. A student's place in a class is an enrolment
// owned by the academic module, so People never carries a school structure.
type Person struct {
	ID              ids.UUID
	OrganisationID  ids.UUID
	FirstName       string
	MiddleName      string
	LastName        string
	PreferredName   string
	Email           string
	Phone           string
	SecondaryPhone  string
	DateOfBirth     *time.Time
	Gender          string
	Nationality     string
	NationalID      string
	Address         Address
	PhotoDocumentID *ids.UUID
	IsActive        bool
	Version         int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Address is a person's residential address. It is the default address; a
// guardian may be billed or contacted somewhere else through a household.
type Address struct {
	Line1      string
	Line2      string
	City       string
	PostalCode string
	Country    string
}

const (
	maxPersonNameLength  = 120
	maxPersonEmailLength = 254
	maxPersonPhoneLength = 32
	maxNationalIDLength  = 64
	maxAddressLineLength = 200
)

// NewPerson creates a person with a generated identifier.
func NewPerson(organisationID ids.UUID, firstName, lastName string) (*Person, error) {
	return NewPersonWithID(ids.New(), organisationID, firstName, lastName)
}

// NewPersonWithID creates a person with a caller-supplied identifier, which is
// what the cloud synchroniser needs.
func NewPersonWithID(id, organisationID ids.UUID, firstName, lastName string) (*Person, error) {
	person := &Person{
		ID:             id,
		OrganisationID: organisationID,
		IsActive:       true,
		Version:        1,
	}

	if err := person.SetName(firstName, "", lastName); err != nil {
		return nil, err
	}
	return person, nil
}

// SetName sets the person's names. A school needs a first and a last name to
// file anything against, so both are required; the middle and preferred names
// are optional, because plenty of schools never record them.
func (p *Person) SetName(firstName, middleName, lastName string) error {
	first := strings.TrimSpace(firstName)
	last := strings.TrimSpace(lastName)

	if first == "" {
		return apperr.ValidationFailed("first name must not be empty").
			WithDetail("field", "person.first_name")
	}
	if last == "" {
		return apperr.ValidationFailed("last name must not be empty").
			WithDetail("field", "person.last_name")
	}
	if err := checkNameLength(first, "person.first_name"); err != nil {
		return err
	}
	if err := checkNameLength(last, "person.last_name"); err != nil {
		return err
	}
	if err := checkNameLength(strings.TrimSpace(middleName), "person.middle_name"); err != nil {
		return err
	}

	p.FirstName = first
	p.MiddleName = strings.TrimSpace(middleName)
	p.LastName = last
	return nil
}

// SetPreferredName sets the name the person answers to, used on cards, rosters
// and statements when it differs from the legal name.
func (p *Person) SetPreferredName(preferred string) error {
	trimmed := strings.TrimSpace(preferred)
	if err := checkNameLength(trimmed, "person.preferred_name"); err != nil {
		return err
	}
	p.PreferredName = trimmed
	return nil
}

// DisplayName is the name to show a user: the preferred name when there is one,
// otherwise the full name.
func (p *Person) DisplayName() string {
	if p.PreferredName != "" {
		return p.PreferredName
	}
	return p.FullName()
}

// FullName is the person's name as filed, ignoring the preferred name.
//
// The parts are joined rather than concatenated with spaces, because a school
// records a middle name for some people and not others, and a name that renders
// as "Grace  Tanyi" is wrong on every list in the interface.
func (p *Person) FullName() string {
	parts := make([]string, 0, 3)
	for _, part := range []string{p.FirstName, p.MiddleName, p.LastName} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " ")
}

// SetContact sets the primary email and phone. An empty email is allowed,
// because not every guardian has one, but if one is given it must look like an
// address: the system sends statements and reminders to it.
func (p *Person) SetContact(email, phone, secondaryPhone string) error {
	normalisedEmail, err := normaliseEmail(email)
	if err != nil {
		return err
	}
	if err := checkPhone(phone, "person.phone"); err != nil {
		return err
	}
	if err := checkPhone(secondaryPhone, "person.secondary_phone"); err != nil {
		return err
	}

	p.Email = normalisedEmail
	p.Phone = strings.TrimSpace(phone)
	p.SecondaryPhone = strings.TrimSpace(secondaryPhone)
	return nil
}

// SetDateOfBirth records the date of birth, or clears it. The date is stored
// date-only: a birth time is never needed and a timezone would make two
// birthdays differ by a day between sites.
func (p *Person) SetDateOfBirth(day time.Time) error {
	if day.IsZero() {
		p.DateOfBirth = nil
		return nil
	}
	date := dateOnly(day)
	p.DateOfBirth = &date
	return nil
}

// ClearDateOfBirth removes a recorded date of birth.
func (p *Person) ClearDateOfBirth() {
	p.DateOfBirth = nil
}

// SetIdentity records the demographic facts a school keeps about a person.
func (p *Person) SetIdentity(gender, nationality, nationalID string) error {
	if err := checkLength(nationality, maxPersonNameLength, "person.nationality"); err != nil {
		return err
	}
	if err := checkLength(nationalID, maxNationalIDLength, "person.national_id"); err != nil {
		return err
	}

	p.Gender = strings.TrimSpace(gender)
	p.Nationality = strings.TrimSpace(nationality)
	p.NationalID = strings.TrimSpace(nationalID)
	return nil
}

// SetAddress replaces the residential address.
func (p *Person) SetAddress(address Address) error {
	if err := checkLength(address.Line1, maxAddressLineLength, "person.address_line1"); err != nil {
		return err
	}
	if err := checkLength(address.Line2, maxAddressLineLength, "person.address_line2"); err != nil {
		return err
	}

	address.Line1 = strings.TrimSpace(address.Line1)
	address.Line2 = strings.TrimSpace(address.Line2)
	address.City = strings.TrimSpace(address.City)
	address.PostalCode = strings.TrimSpace(address.PostalCode)
	address.Country = strings.ToUpper(strings.TrimSpace(address.Country))
	p.Address = address
	return nil
}

// SetPhoto links the person's photograph through the document storage
// abstraction. Passing a nil identifier removes the photograph.
func (p *Person) SetPhoto(documentID *ids.UUID) {
	p.PhotoDocumentID = documentID
}

// AgeOn returns the person's age in whole years at a given date.
func (p *Person) AgeOn(day time.Time) (int, bool) {
	if p.DateOfBirth == nil {
		return 0, false
	}
	date := dateOnly(day)
	if date.Before(*p.DateOfBirth) {
		return 0, false
	}

	years := date.Year() - p.DateOfBirth.Year()
	if date.Month() < p.DateOfBirth.Month() ||
		(date.Month() == p.DateOfBirth.Month() && date.Day() < p.DateOfBirth.Day()) {
		years--
	}
	return years, true
}

// Deactivate retires the person. Historical records keep referring to them, and
// a person is never deleted: a receipt must still name the guardian who paid it.
func (p *Person) Deactivate() {
	p.IsActive = false
}

// Activate restores a deactivated person.
func (p *Person) Activate() {
	p.IsActive = true
}

// SortName is the "Last, First" form used for rosters and alphabetical lists.
func (p *Person) SortName() string {
	return p.LastName + ", " + p.FirstName
}

func checkNameLength(value, field string) error {
	return checkLength(value, maxPersonNameLength, field)
}

func checkLength(value string, maximum int, field string) error {
	if utf8.RuneCountInString(value) > maximum {
		return apperr.ValidationFailed("value is longer than the allowed length").
			WithDetail("field", field).
			WithDetail("maxLength", maximum)
	}
	return nil
}

func checkPhone(value, field string) error {
	return checkLength(value, maxPersonPhoneLength, field)
}

// normaliseEmail lower-cases an address and rejects one that is obviously not
// an address, because statements and reminders are sent to it and a typo here
// silently loses money from a family.
func normaliseEmail(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "", nil
	}
	if utf8.RuneCountInString(trimmed) > maxPersonEmailLength {
		return "", apperr.ValidationFailed("email address is longer than the allowed length").
			WithDetail("field", "person.email").
			WithDetail("maxLength", maxPersonEmailLength)
	}

	at := strings.Index(trimmed, "@")
	if at <= 0 || at == len(trimmed)-1 || strings.Count(trimmed, "@") != 1 {
		return "", apperr.ValidationFailed("email address is not valid").
			WithDetail("field", "person.email")
	}
	if strings.ContainsAny(trimmed, " \t\r\n") {
		return "", apperr.ValidationFailed("email address is not valid").
			WithDetail("field", "person.email")
	}
	return trimmed, nil
}
