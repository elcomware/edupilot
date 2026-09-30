// Package domain holds the organisation aggregate and its invariants. It
// depends on nothing but the kernel.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/kernel/money"
)

// Organisation is the legal and fiscal boundary of the whole system. Every
// other record in EduPilot belongs to exactly one organisation, and nothing
// crosses that boundary.
type Organisation struct {
	ID            ids.UUID
	Name          string
	LegalName     string
	Currency      money.Currency
	Country       string
	DefaultLocale string
	TaxNumber     string
	Address       Address
	IsActive      bool
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Address is a postal address. Optional parts stay empty rather than null, so a
// country with no postal code needs no special case.
type Address struct {
	Line1      string
	Line2      string
	City       string
	PostalCode string
	Country    string
}

const (
	maxNameLength   = 200
	maxLocaleLength = 35
)

// NewOrganisation creates an organisation with a generated identifier. The
// currency is resolved from an ISO 4217 code, because a school that invoices in
// the wrong currency corrupts every downstream ledger.
func NewOrganisation(name, currencyCode, locale, country string) (*Organisation, error) {
	currency, err := money.LookupCurrency(currencyCode)
	if err != nil {
		return nil, apperr.ValidationFailed("organisation.currency is not a supported ISO 4217 code").
			WithDetail("field", "organisation.currency").
			WithDetail("value", currencyCode)
	}
	return NewOrganisationWithID(ids.New(), name, currency, locale, country)
}

// NewOrganisationWithID creates an organisation with a caller-supplied
// identifier, which is what the cloud synchroniser needs.
func NewOrganisationWithID(id ids.UUID, name string, currency money.Currency, locale, country string) (*Organisation, error) {
	organisation := &Organisation{
		ID:       id,
		Currency: currency,
		IsActive: true,
		Version:  1,
	}

	if err := organisation.Rename(name); err != nil {
		return nil, err
	}
	if err := organisation.SetLocale(locale); err != nil {
		return nil, err
	}
	if err := organisation.SetCountry(country); err != nil {
		return nil, err
	}
	return organisation, nil
}

// Rename changes the display name.
func (o *Organisation) Rename(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return apperr.ValidationFailed("organisation name must not be empty").
			WithDetail("field", "organisation.name")
	}
	if utf8.RuneCountInString(trimmed) > maxNameLength {
		return apperr.ValidationFailed("organisation name is longer than 200 characters").
			WithDetail("field", "organisation.name")
	}

	o.Name = trimmed
	if o.LegalName == "" {
		o.LegalName = trimmed
	}
	return nil
}

// SetLocale sets the default interface language, e.g. en or fr.
func (o *Organisation) SetLocale(locale string) error {
	trimmed := strings.ToLower(strings.TrimSpace(locale))
	if trimmed == "" {
		return apperr.ValidationFailed("default locale must not be empty").
			WithDetail("field", "organisation.default_locale")
	}
	if len(trimmed) > maxLocaleLength {
		return apperr.ValidationFailed("default locale is too long").
			WithDetail("field", "organisation.default_locale")
	}
	o.DefaultLocale = trimmed
	return nil
}

// SetCountry sets the ISO 3166-1 alpha-2 country code.
func (o *Organisation) SetCountry(country string) error {
	trimmed := strings.ToUpper(strings.TrimSpace(country))
	if len(trimmed) != 2 {
		return apperr.ValidationFailed("country must be a two-letter code").
			WithDetail("field", "organisation.country").
			WithDetail("value", trimmed)
	}
	o.Country = trimmed
	o.Address.Country = trimmed
	return nil
}

// SetAddress replaces the postal address.
func (o *Organisation) SetAddress(address Address) {
	o.Address = address
	if address.Country != "" {
		o.Address.Country = strings.ToUpper(strings.TrimSpace(address.Country))
	}
}

// Deactivate retires the organisation. Historical records are never deleted;
// EduPilot keeps them for statutory reporting.
func (o *Organisation) Deactivate() {
	o.IsActive = false
}

// Activate restores a deactivated organisation.
func (o *Organisation) Activate() {
	o.IsActive = true
}
