package application_test

import (
	"context"
	"testing"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/clock"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/kernel/money"
	"github.com/elcomware/edupilot/internal/platform/organisation/adapters/sqlite"
	"github.com/elcomware/edupilot/internal/platform/organisation/application"
	"github.com/elcomware/edupilot/internal/platform/organisation/domain"
	"github.com/elcomware/edupilot/internal/testsupport"
)

func newService(t *testing.T) *application.Service {
	t.Helper()

	transactor := database.NewTransactor(testsupport.NewDatabase(t), "")
	return application.NewService(sqlite.NewRepository(transactor), clock.NewFixed(testsupport.FixedTime))
}

func TestCreateOrganisationNormalisesInput(t *testing.T) {
	service := newService(t)

	organisation, err := service.CreateOrganisation(context.Background(), application.CreateOrganisationCommand{
		Name:          "  Ecole Saint-Joseph  ",
		CurrencyCode:  "xof",
		DefaultLocale: "FR",
		Country:       "ci",
	})
	if err != nil {
		t.Fatal(err)
	}

	if organisation.Name != "Ecole Saint-Joseph" {
		t.Errorf("name = %q, want the trimmed value", organisation.Name)
	}
	if organisation.Currency.Code() != "XOF" {
		t.Errorf("currency = %q, want XOF", organisation.Currency.Code())
	}
	if organisation.DefaultLocale != "fr" {
		t.Errorf("locale = %q, want fr", organisation.DefaultLocale)
	}
	if organisation.Country != "CI" {
		t.Errorf("country = %q, want CI", organisation.Country)
	}
	if organisation.LegalName != "Ecole Saint-Joseph" {
		t.Errorf("legal name = %q, want it to default to the display name", organisation.LegalName)
	}
	if !organisation.CreatedAt.Equal(testsupport.FixedTime) {
		t.Errorf("created_at = %v, want the injected clock value", organisation.CreatedAt)
	}
	if organisation.ID.IsNil() {
		t.Error("identifier was not generated")
	}
}

func TestCreateOrganisationRejectsBadInput(t *testing.T) {
	service := newService(t)

	longName := make([]rune, 201)
	for i := range longName {
		longName[i] = 'x'
	}

	cases := map[string]application.CreateOrganisationCommand{
		"empty name":     {Name: "  ", CurrencyCode: "XOF", DefaultLocale: "en", Country: "CI"},
		"bad currency":   {Name: "School", CurrencyCode: "XYZ", DefaultLocale: "en", Country: "CI"},
		"empty locale":   {Name: "School", CurrencyCode: "XOF", DefaultLocale: "", Country: "CI"},
		"bad country":    {Name: "School", CurrencyCode: "XOF", DefaultLocale: "en", Country: "CIV"},
		"overlong name":  {Name: string(longName), CurrencyCode: "XOF", DefaultLocale: "en", Country: "CI"},
		"blank country ": {Name: "School", CurrencyCode: "XOF", DefaultLocale: "en", Country: " "},
	}

	for name, command := range cases {
		if _, err := service.CreateOrganisation(context.Background(), command); err == nil {
			t.Errorf("%s: expected an error", name)
		} else if !apperr.Is(err, apperr.CodeValidationFailed) {
			t.Errorf("%s: code = %s, want %s", name, apperr.CodeOf(err), apperr.CodeValidationFailed)
		}
	}
}

func TestOrganisationRoundTrip(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	created, err := service.CreateOrganisation(ctx, application.CreateOrganisationCommand{
		Name: "Lycee de Cocody", CurrencyCode: "XOF", DefaultLocale: "fr", Country: "CI",
	})
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := service.GetOrganisation(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != created.Name || loaded.Currency.Code() != created.Currency.Code() {
		t.Errorf("loaded %+v, want %+v", loaded, created)
	}
	if loaded.Version != 1 {
		t.Errorf("version = %d, want 1", loaded.Version)
	}
	if !loaded.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("created_at = %v, want %v", loaded.CreatedAt, created.CreatedAt)
	}
}

func TestGetOrganisationRejectsNilIdentifier(t *testing.T) {
	service := newService(t)

	_, err := service.GetOrganisation(context.Background(), ids.UUID{})
	if err == nil || !apperr.Is(err, apperr.CodeValidationFailed) {
		t.Fatalf("err = %v, want a validation failure for a nil identifier", err)
	}
}

func TestGetOrganisationNotFound(t *testing.T) {
	service := newService(t)

	_, err := service.GetOrganisation(context.Background(), ids.New())
	if err == nil || !apperr.Is(err, apperr.CodeNotFound) {
		t.Fatalf("err = %v, want a not-found error for an unknown identifier", err)
	}
}

func TestEnsureOrganisationIsIdempotent(t *testing.T) {
	service := newService(t)
	command := application.CreateOrganisationCommand{
		Name: "Groupe Scolaire Nord", CurrencyCode: "XOF", DefaultLocale: "fr", Country: "SN",
	}

	first, err := service.EnsureOrganisation(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}

	second, err := service.EnsureOrganisation(context.Background(), application.CreateOrganisationCommand{
		Name: "A Completely Different Name", CurrencyCode: "EUR", DefaultLocale: "en", Country: "FR",
	})
	if err != nil {
		t.Fatal(err)
	}

	if first.ID != second.ID {
		t.Fatalf("a second tenant was created: %s and %s", first.ID, second.ID)
	}
	if second.Name != first.Name {
		t.Fatalf("the existing tenant was overwritten: %q became %q", first.Name, second.Name)
	}
}

func TestGetCurrentOrganisationBeforeCreation(t *testing.T) {
	service := newService(t)

	_, err := service.GetCurrentOrganisation(context.Background())
	if err == nil || !apperr.Is(err, apperr.CodeNotFound) {
		t.Fatalf("err = %v, want a not-found error on a fresh installation", err)
	}
}

func TestDomainLifecycle(t *testing.T) {
	organisation, err := domain.NewOrganisationWithID(ids.New(), "School", money.MustCurrency("XOF"), "en", "BJ")
	if err != nil {
		t.Fatal(err)
	}
	if !organisation.IsActive || organisation.Version != 1 {
		t.Errorf("a new organisation must be active at version 1, got %+v", organisation)
	}

	organisation.Deactivate()
	if organisation.IsActive {
		t.Error("Deactivate did not deactivate")
	}
	organisation.Activate()
	if !organisation.IsActive {
		t.Error("Activate did not activate")
	}
}
