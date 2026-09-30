package application_test

import (
	"context"
	"testing"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/clock"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	campussqlite "github.com/elcomware/edupilot/internal/platform/campus/adapters/sqlite"
	campusapp "github.com/elcomware/edupilot/internal/platform/campus/application"
	campusdomain "github.com/elcomware/edupilot/internal/platform/campus/domain"
	"github.com/elcomware/edupilot/internal/platform/organisation/adapters/sqlite"
	organisationapp "github.com/elcomware/edupilot/internal/platform/organisation/application"
	"github.com/elcomware/edupilot/internal/testsupport"
)

type fixture struct {
	campuses      *campusapp.Service
	organisations *organisationapp.Service
	transactor    *database.Transactor
	owner         ids.UUID
	tenantB       ids.UUID
}

func newFixture(t *testing.T) fixture {
	t.Helper()

	transactor := database.NewTransactor(testsupport.NewDatabase(t), "")
	systemClock := clock.NewFixed(testsupport.FixedTime)

	organisations := organisationapp.NewService(sqlite.NewRepository(transactor), systemClock)
	campuses := campusapp.NewService(campussqlite.NewRepository(transactor), transactor, systemClock)
	ctx := context.Background()

	owner, err := organisations.CreateOrganisation(ctx, organisationapp.CreateOrganisationCommand{
		Name: "Ecole Principale", CurrencyCode: "XOF", DefaultLocale: "fr", Country: "CI",
	})
	if err != nil {
		t.Fatal(err)
	}

	// A second tenant, so tenancy isolation can be tested against real data
	// rather than a mock that agrees with whatever the code does.
	tenantB, err := organisations.CreateOrganisation(ctx, organisationapp.CreateOrganisationCommand{
		Name: "Ecole Privée du Plateau", CurrencyCode: "XOF", DefaultLocale: "fr", Country: "CI",
	})
	if err != nil {
		t.Fatal(err)
	}

	return fixture{campuses: campuses, organisations: organisations, transactor: transactor, owner: owner.ID, tenantB: tenantB.ID}
}

func createCampus(t *testing.T, f fixture, organisationID ids.UUID, name, code string, primary bool) *campusdomain.Campus {
	t.Helper()

	result, err := f.campuses.CreateCampus(context.Background(), campusapp.CreateCampusCommand{
		OrganisationID: organisationID,
		Name:           name,
		Code:           code,
		Address:        campusdomain.Address{Line1: "Rue des Palmiers", City: "Abidjan", Country: "ci"},
		Phone:          "+225 07 00 00 00 00",
		Email:          "Contact@Ecole.CI",
		IsPrimary:      primary,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Campus
}

func TestCreateCampusNormalisesInput(t *testing.T) {
	f := newFixture(t)

	campus := createCampus(t, f, f.owner, "  Campus Principal ", " main ", true)

	if campus.Name != "Campus Principal" {
		t.Errorf("name = %q, want the trimmed value", campus.Name)
	}
	if campus.Code != "MAIN" {
		t.Errorf("code = %q, want MAIN", campus.Code)
	}
	if campus.Address.Country != "CI" {
		t.Errorf("country = %q, want CI", campus.Address.Country)
	}
	if campus.Email != "contact@ecole.ci" {
		t.Errorf("email = %q, want the lower-cased address", campus.Email)
	}
	if !campus.IsPrimary {
		t.Error("the campus was requested as primary")
	}
	if campus.Version != 1 || !campus.IsActive {
		t.Errorf("a new campus must be active at version 1, got %+v", campus)
	}
}

func TestCreateCampusRejectsInvalidCode(t *testing.T) {
	f := newFixture(t)

	_, err := f.campuses.CreateCampus(context.Background(), campusapp.CreateCampusCommand{
		OrganisationID: f.owner,
		Name:           "Campus",
		Code:           "Main Campus!",
	})
	if err == nil || !apperr.Is(err, apperr.CodeValidationFailed) {
		t.Fatalf("err = %v, want a validation failure for an unusable code", err)
	}
}

func TestCreateCampusRejectsDuplicateCode(t *testing.T) {
	f := newFixture(t)
	createCampus(t, f, f.owner, "Campus One", "MAIN", true)

	_, err := f.campuses.CreateCampus(context.Background(), campusapp.CreateCampusCommand{
		OrganisationID: f.owner,
		Name:           "Campus Two",
		Code:           "MAIN",
	})
	if err == nil || !apperr.Is(err, apperr.CodeAlreadyExists) {
		t.Fatalf("err = %v, want ALREADY_EXISTS for a duplicate campus code", err)
	}
}

func TestCreateCampusRequiresOrganisation(t *testing.T) {
	f := newFixture(t)

	_, err := f.campuses.CreateCampus(context.Background(), campusapp.CreateCampusCommand{
		Name: "Orphan Campus",
		Code: "ORPH",
	})
	if err == nil || !apperr.Is(err, apperr.CodeValidationFailed) {
		t.Fatalf("err = %v, want a validation failure for a missing organisation", err)
	}
}

func TestPrimaryCampusIsDemotedWhenAnotherIsPromoted(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	first := createCampus(t, f, f.owner, "Campus One", "ONE", true)
	second := createCampus(t, f, f.owner, "Campus Two", "TWO", false)

	promoted, err := f.campuses.SetPrimaryCampus(ctx, f.owner, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !promoted.IsPrimary {
		t.Fatal("the promoted campus is not primary")
	}

	demoted, err := f.campuses.GetCampus(ctx, f.owner, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if demoted.IsPrimary {
		t.Fatal("the previous primary campus was not demoted")
	}

	primaries, err := f.transactor.Conn(ctx).QueryContext(ctx,
		"SELECT COUNT(*) FROM campuses WHERE organisation_id = ? AND is_primary = 1", f.owner.String())
	if err != nil {
		t.Fatal(err)
	}
	defer primaries.Close()
	if !primaries.Next() {
		t.Fatal("count query returned no row")
	}
	var count int
	if err := primaries.Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("primary campuses = %d, want exactly 1", count)
	}
}

func TestCreateCampusAsPrimaryDemotesThePreviousOne(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	createCampus(t, f, f.owner, "Campus One", "ONE", true)
	result, err := f.campuses.CreateCampus(ctx, campusapp.CreateCampusCommand{
		OrganisationID: f.owner,
		Name:           "Campus Two",
		Code:           "TWO",
		IsPrimary:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.DemotedPrimary == nil {
		t.Fatal("the demoted campus was not reported")
	}
	if result.DemotedPrimary.Name != "Campus One" {
		t.Errorf("demoted %q, want Campus One", result.DemotedPrimary.Name)
	}
	if !result.Campus.IsPrimary {
		t.Error("the new campus is not primary")
	}
}

func TestPrimaryCampusesAreIndependentPerOrganisation(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	createCampus(t, f, f.owner, "Campus One", "ONE", true)
	createCampus(t, f, f.tenantB, "Other Campus", "ONE", true)

	// The same code is free in another tenant, and neither demotes the other.
	listed, err := f.campuses.ListCampuses(ctx, f.tenantB, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || !listed[0].IsPrimary {
		t.Fatalf("tenant B campuses = %+v, want one primary campus", listed)
	}
}

func TestCampusListIsScopedToOrganisation(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	createCampus(t, f, f.owner, "Campus One", "ONE", true)
	createCampus(t, f, f.owner, "Campus Two", "TWO", false)
	foreign := createCampus(t, f, f.tenantB, "Other Campus", "OTH", true)

	listed, err := f.campuses.ListCampuses(ctx, f.owner, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("campus count = %d, want 2", len(listed))
	}
	for _, campus := range listed {
		if campus.OrganisationID != f.owner {
			t.Errorf("campus %q from another organisation leaked into the list", campus.Name)
		}
		if campus.ID == foreign.ID {
			t.Error("another tenant's campus is visible")
		}
	}

	// Reading another tenant's campus by id must fail, not return the record.
	if _, err := f.campuses.GetCampus(ctx, f.owner, foreign.ID); err == nil {
		t.Fatal("cross-tenant read was allowed")
	} else if !apperr.Is(err, apperr.CodeNotFound) {
		t.Fatalf("code = %s, want NOT_FOUND", apperr.CodeOf(err))
	}
}

func TestListCampusesRequiresOrganisation(t *testing.T) {
	f := newFixture(t)

	_, err := f.campuses.ListCampuses(context.Background(), ids.UUID{}, false)
	if err == nil || !apperr.Is(err, apperr.CodeValidationFailed) {
		t.Fatalf("err = %v, want a validation failure", err)
	}
}

func TestInactiveCampusIsHiddenUnlessRequested(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	active := createCampus(t, f, f.owner, "Campus One", "ONE", true)
	retired := createCampus(t, f, f.owner, "Old Campus", "OLD", false)

	retired.Deactivate()
	if _, err := f.transactor.Conn(ctx).ExecContext(ctx,
		"UPDATE campuses SET is_active = 0, is_primary = 0 WHERE id = ?", retired.ID.String()); err != nil {
		t.Fatal(err)
	}

	visible, err := f.campuses.ListCampuses(ctx, f.owner, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != active.ID {
		t.Fatalf("visible campuses = %+v, want only the active one", visible)
	}

	all, err := f.campuses.ListCampuses(ctx, f.owner, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("campus count = %d, want 2 when inactive campuses are included", len(all))
	}
}
