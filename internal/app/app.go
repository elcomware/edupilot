// Package app is the EduPilot composition root. It is the only place that
// knows which concrete adapters satisfy which ports, so cmd/desktop (Mode A),
// cmd/server (Mode B) and the future cloud host (Mode C) wire the same object
// graph with the same behaviour.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/clock"
	academicadapter "github.com/elcomware/edupilot/internal/platform/academic/adapters/sqlite"
	academicapp "github.com/elcomware/edupilot/internal/platform/academic/application"
	campusadapter "github.com/elcomware/edupilot/internal/platform/campus/adapters/sqlite"
	campusapp "github.com/elcomware/edupilot/internal/platform/campus/application"
	organisationadapter "github.com/elcomware/edupilot/internal/platform/organisation/adapters/sqlite"
	organisationapp "github.com/elcomware/edupilot/internal/platform/organisation/application"
	peopleadapter "github.com/elcomware/edupilot/internal/platform/people/adapters/sqlite"
	peopleapp "github.com/elcomware/edupilot/internal/platform/people/application"
)

// The composition root is the only place that knows both modules, so it is also
// where the proof that the Academic service satisfies the narrow port People
// declares lives. Asserting it here keeps Academic free of any import of People.
var _ peopleapp.AcademicYearScope = (*academicapp.Service)(nil)

// Settings describes how an EduPilot installation is configured on disk.
type Settings struct {
	// DataRoot holds the database, managed files, backups and logs. School data
	// never lives next to the application binaries (docs/adr/ADR-003).
	DataRoot string
	// CurrencyCode is the ISO 4217 code of the tenant's base currency.
	CurrencyCode string
	// Locale is the default interface language.
	Locale string
	// Country is the ISO 3166-1 alpha-2 code of the tenant's country.
	Country string
	// SchoolName is used to bootstrap the tenant on first run.
	SchoolName string
}

// DefaultSettings returns the settings used by a fresh installation.
func DefaultSettings(dataRoot string) Settings {
	return Settings{
		DataRoot:     dataRoot,
		CurrencyCode: "XOF",
		Locale:       "fr",
		Country:      "CI",
		SchoolName:   "Mon Ecole",
	}
}

// Application is the wired object graph. Transport adapters hold one of these
// and call the application services directly; they never reach past it into
// infrastructure.
type Application struct {
	Database *database.DB
	Settings Settings

	Organisations *organisationapp.Service
	Campuses      *campusapp.Service
	Academic      *academicapp.Service
	People        *peopleapp.Service

	closer func() error
}

// New opens the database, applies migrations, ensures the tenant exists and
// returns the wired application.
func New(ctx context.Context, settings Settings, logger *slog.Logger) (*Application, error) {
	if settings.DataRoot == "" {
		return nil, fmt.Errorf("app: a data root is required")
	}
	if err := database.EnsureDirectories(settings.DataRoot); err != nil {
		return nil, err
	}

	db, err := database.Open(ctx, database.SQLiteOptions(settings.DataRoot))
	if err != nil {
		return nil, err
	}

	transactor := database.NewTransactor(db, "")
	systemClock := clock.System{}

	results, err := database.MigrateSQLite(ctx, transactor, systemClock)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, result := range results {
		if result.Applied {
			logger.Info("migration applied", "version", result.Version, "name", result.Name, "took", result.Duration)
		}
	}

	organisations := organisationapp.NewService(organisationadapter.NewRepository(transactor), systemClock)
	campuses := campusapp.NewService(campusadapter.NewRepository(transactor), transactor, systemClock)
	academic := academicapp.NewService(
		academicadapter.NewYearRepository(transactor),
		academicadapter.NewTermRepository(transactor),
		transactor,
		systemClock,
	)
	people := peopleapp.NewService(peopleapp.Repositories{
		People:        peopleadapter.NewPersonRepository(transactor),
		Roles:         peopleadapter.NewRoleRepository(transactor),
		Students:      peopleadapter.NewStudentProfileRepository(transactor),
		Employees:     peopleadapter.NewEmployeeProfileRepository(transactor),
		Guardians:     peopleadapter.NewGuardianProfileRepository(transactor),
		Relationships: peopleadapter.NewRelationshipRepository(transactor),
		Households:    peopleadapter.NewHouseholdRepository(transactor),
	}, academic, transactor, systemClock)

	if _, err := organisations.EnsureOrganisation(ctx, organisationapp.CreateOrganisationCommand{
		Name:          settings.SchoolName,
		CurrencyCode:  settings.CurrencyCode,
		DefaultLocale: settings.Locale,
		Country:       settings.Country,
	}); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("app: bootstrapping the tenant failed: %w", err)
	}

	return &Application{
		Database:      db,
		Settings:      settings,
		Organisations: organisations,
		Campuses:      campuses,
		Academic:      academic,
		People:        people,
		closer:        db.Close,
	}, nil
}

// Close releases the database handle.
func (a *Application) Close() error {
	if a == nil || a.closer == nil {
		return nil
	}
	return a.closer()
}

// StartupTimeout bounds first-run work, which includes creating the data root
// and applying migrations on a school machine that may be slow.
const StartupTimeout = 30 * time.Second

// StartupContext returns the context used while wiring the application.
func StartupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), StartupTimeout)
}
