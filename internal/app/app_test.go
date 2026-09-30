package app_test

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/elcomware/edupilot/internal/api"
	"github.com/elcomware/edupilot/internal/app"
	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	campusapp "github.com/elcomware/edupilot/internal/platform/campus/application"
	campusdomain "github.com/elcomware/edupilot/internal/platform/campus/domain"
	"github.com/elcomware/edupilot/internal/testsupport"
)

func newApplication(t *testing.T, settings app.Settings) *app.Application {
	t.Helper()

	edupilot, err := app.New(context.Background(), settings, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("starting the application: %v", err)
	}
	t.Cleanup(func() { _ = edupilot.Close() })
	return edupilot
}

func TestFirstRunBootstrapsTheInstallation(t *testing.T) {
	root := testsupport.TestDataRoot(t)
	edupilot := newApplication(t, app.DefaultSettings(root))

	for _, dir := range []string{"database", "files", "backups", "logs", "runtime"} {
		if info, err := os.Stat(filepath.Join(root, dir)); err != nil || !info.IsDir() {
			t.Fatalf("the data root is missing %q", dir)
		}
	}
	if _, err := os.Stat(testsupport.DatabasePath(root)); err != nil {
		t.Fatalf("the database file was not created: %v", err)
	}

	organisation, err := edupilot.Organisations.GetCurrentOrganisation(context.Background())
	if err != nil {
		t.Fatalf("the tenant was not bootstrapped: %v", err)
	}
	if organisation.Name != "Mon Ecole" {
		t.Errorf("name = %q, want the default school name", organisation.Name)
	}
	if organisation.Currency.Code() != "XOF" {
		t.Errorf("currency = %q, want XOF", organisation.Currency.Code())
	}
}

func TestRestartKeepsTheSameTenant(t *testing.T) {
	root := testsupport.TestDataRoot(t)
	settings := app.DefaultSettings(root)

	first := newApplication(t, settings)
	original, err := first.Organisations.GetCurrentOrganisation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	// A restart is the normal path: it must not create a second tenant.
	second := newApplication(t, settings)
	reloaded, err := second.Organisations.GetCurrentOrganisation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ID != original.ID {
		t.Fatalf("the tenant changed across a restart: %s then %s", original.ID, reloaded.ID)
	}
}

func TestMigrationsAreAppliedOnceAcrossRestarts(t *testing.T) {
	root := testsupport.TestDataRoot(t)
	settings := app.DefaultSettings(root)

	first := newApplication(t, settings)
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	// Booting a second time must succeed: the migration is already applied and
	// its recorded checksum must be left untouched.
	newApplication(t, settings)

	rows, err := openRows(t, root)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no migration was recorded")
	}
	var version int
	var checksum string
	if err := rows.Scan(&version, &checksum); err != nil {
		t.Fatal(err)
	}
	if version != 1 || checksum == "" {
		t.Fatalf("schema_migrations = (%d, %q), want migration 1 with a checksum", version, checksum)
	}
}

func openRows(t *testing.T, root string) (*sql.Rows, error) {
	t.Helper()

	db, err := database.Open(context.Background(), database.SQLiteOptions(root))
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = db.Close() })

	return db.SQL().Query("SELECT version, checksum FROM schema_migrations ORDER BY version")
}

func TestCampusUseCasesAreWiredEndToEnd(t *testing.T) {
	root := testsupport.TestDataRoot(t)
	edupilot := newApplication(t, app.DefaultSettings(root))
	ctx := context.Background()

	organisation, err := edupilot.Organisations.GetCurrentOrganisation(ctx)
	if err != nil {
		t.Fatal(err)
	}

	result, err := edupilot.Campuses.CreateCampus(ctx, campusCommand(organisation.ID, "Campus Central", "CENTRAL", true))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Campus.IsPrimary {
		t.Error("the campus was not created as primary")
	}

	campuses, err := edupilot.Campuses.ListCampuses(ctx, organisation.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(campuses) != 1 {
		t.Fatalf("campus count = %d, want 1", len(campuses))
	}

	view := api.NewCampusView(campuses[0])
	if view.Code != "CENTRAL" || view.OrganisationID != organisation.ID.String() {
		t.Errorf("campus view = %+v, want the persisted campus", view)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	root := testsupport.TestDataRoot(t)
	edupilot := newApplication(t, app.DefaultSettings(root))

	if err := edupilot.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := edupilot.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestNewRejectsAnEmptyDataRoot(t *testing.T) {
	if _, err := app.New(context.Background(), app.Settings{}, slog.New(slog.DiscardHandler)); err == nil {
		t.Fatal("an empty data root was accepted")
	}
}

func campusCommand(organisationID ids.UUID, name, code string, primary bool) campusapp.CreateCampusCommand {
	return campusapp.CreateCampusCommand{
		OrganisationID: organisationID,
		Name:           name,
		Code:           code,
		Address:        campusdomain.Address{Line1: "Rue des Palmiers", City: "Abidjan", Country: "CI"},
		Email:          "campus@ecole.ci",
		IsPrimary:      primary,
	}
}
