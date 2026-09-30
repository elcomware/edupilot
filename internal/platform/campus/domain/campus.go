// Package domain holds the campus aggregate and its invariants. It depends on
// nothing but the kernel.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
)

// Campus is a physical site of a school: a building, or a group of buildings
// with its own head teacher, address and telephone number. A school group
// typically has one campus and occasionally several.
type Campus struct {
	ID             ids.UUID
	OrganisationID ids.UUID
	Name           string
	Code           string
	Address        Address
	Phone          string
	Email          string
	IsPrimary      bool
	IsActive       bool
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Address is a campus postal address.
type Address struct {
	Line1   string
	City    string
	Country string
}

const maxCampusNameLength = 200

// NewCampus creates a campus with a generated identifier.
func NewCampus(organisationID ids.UUID, name, code string, primary bool) (*Campus, error) {
	return NewCampusWithID(ids.New(), organisationID, name, code, primary)
}

// NewCampusWithID creates a campus with a caller-supplied identifier, which is
// what the cloud synchroniser needs.
func NewCampusWithID(id, organisationID ids.UUID, name, code string, primary bool) (*Campus, error) {
	campus := &Campus{
		ID:             id,
		OrganisationID: organisationID,
		IsPrimary:      primary,
		IsActive:       true,
		Version:        1,
	}

	if err := campus.Rename(name); err != nil {
		return nil, err
	}
	if err := campus.SetCode(code); err != nil {
		return nil, err
	}
	return campus, nil
}

// Rename changes the campus name.
func (c *Campus) Rename(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return apperr.ValidationFailed("campus name must not be empty").
			WithDetail("field", "campus.name")
	}
	if utf8.RuneCountInString(trimmed) > maxCampusNameLength {
		return apperr.ValidationFailed("campus name is longer than 200 characters").
			WithDetail("field", "campus.name")
	}
	c.Name = trimmed
	return nil
}

// SetCode sets the short campus code used on timetables and reports, e.g. MAIN.
// The code is unique per organisation, which the repository enforces.
func (c *Campus) SetCode(code string) error {
	trimmed := strings.ToUpper(strings.TrimSpace(code))
	if trimmed == "" {
		return apperr.ValidationFailed("campus code must not be empty").
			WithDetail("field", "campus.code")
	}
	if utf8.RuneCountInString(trimmed) > 32 {
		return apperr.ValidationFailed("campus code is longer than 32 characters").
			WithDetail("field", "campus.code")
	}
	for _, r := range trimmed {
		if r != '-' && r != '_' && !isUpperAlphanumeric(r) {
			return apperr.ValidationFailed("campus code accepts letters, digits, hyphen and underscore only").
				WithDetail("field", "campus.code").
				WithDetail("value", trimmed)
		}
	}
	c.Code = trimmed
	return nil
}

// SetAddress replaces the postal address.
func (c *Campus) SetAddress(address Address) {
	c.Address = address
	if address.Country != "" {
		c.Address.Country = strings.ToUpper(strings.TrimSpace(address.Country))
	}
}

// SetContact sets the phone number and email address.
func (c *Campus) SetContact(phone, email string) {
	c.Phone = strings.TrimSpace(phone)
	c.Email = strings.ToLower(strings.TrimSpace(email))
}

// MakePrimary promotes this campus to the organisation's home campus. Only one
// campus can hold the flag, which the database enforces with a partial unique
// index; the application layer clears the previous holder first.
func (c *Campus) MakePrimary() {
	c.IsPrimary = true
}

// Deactivate retires the campus. Enrolments and timetables are never deleted,
// because an audit of what happened in a given year must stay readable.
func (c *Campus) Deactivate() {
	c.IsActive = false
	c.IsPrimary = false
}

// Activate restores a deactivated campus.
func (c *Campus) Activate() {
	c.IsActive = true
}

func isUpperAlphanumeric(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}
