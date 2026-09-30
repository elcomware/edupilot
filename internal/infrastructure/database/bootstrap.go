package database

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	sqlitemigrations "github.com/elcomware/edupilot/database/migrations/sqlite"
	"github.com/elcomware/edupilot/internal/kernel/clock"
)

// MigrateSQLite applies the embedded SQLite migrations to the open connection.
func MigrateSQLite(ctx context.Context, tx *Transactor, systemClock clock.Clock) ([]MigrationResult, error) {
	migrations, err := LoadMigrations(sqlitemigrations.FS)
	if err != nil {
		return nil, err
	}

	migrator := NewMigrator(tx.DB().SQL(), systemClock)

	var results []MigrationResult
	err = tx.InTx(ctx, func(ctx context.Context) error {
		applied, err := migrator.Up(ctx, migrations)
		results = applied
		return err
	})
	if err != nil {
		return results, err
	}
	return results, nil
}

// SQLiteOptions returns the recommended connection options for a standalone
// or site-server SQLite database.
func SQLiteOptions(dataRoot string) Options {
	return Options{
		Driver:       DriverSQLite,
		Path:         filepath.Join(dataRoot, "database", "edupilot.db"),
		MaxOpenConns: 1,
		BusyTimeout:  5 * time.Second,
	}
}

// EnsureDirectories creates the EduPilot data root layout. Application files
// and school data are always separate (docs/adr/ADR-003).
func EnsureDirectories(dataRoot string) error {
	for _, dir := range []string{"database", "files", "backups", "logs", "runtime"} {
		if err := ensureDirectory(filepath.Join(dataRoot, dir)); err != nil {
			return fmt.Errorf("database: preparing %s: %w", dir, err)
		}
	}
	return nil
}
