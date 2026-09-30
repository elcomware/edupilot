// Package testsupport provides shared fixtures for EduPilot tests. It is
// imported only by test binaries.
package testsupport

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/clock"
)

// FixedTime is the instant every deterministic test starts from.
var FixedTime = time.Date(2026, time.September, 29, 9, 0, 0, 0, time.UTC)

// NewDatabase opens a migrated SQLite database in a temporary directory. Each
// test gets its own file, so tests never share state.
func NewDatabase(t *testing.T) *database.DB {
	t.Helper()

	options := database.SQLiteOptions(t.TempDir())
	db, err := database.Open(context.Background(), options)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := database.MigrateSQLite(context.Background(), database.NewTransactor(db, ""), clock.System{}); err != nil {
		t.Fatalf("migrating test database: %v", err)
	}

	return db
}

// NewTransactor returns a Transactor for the test database.
func NewTransactor(t *testing.T) *database.Transactor {
	t.Helper()
	return database.NewTransactor(NewDatabase(t), "")
}

// TestDataRoot returns a temporary EduPilot data root with the production
// directory layout in place.
func TestDataRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := database.EnsureDirectories(root); err != nil {
		t.Fatalf("preparing data root: %v", err)
	}
	return root
}

// DatabasePath returns the SQLite file inside a data root.
func DatabasePath(dataRoot string) string {
	return filepath.Join(dataRoot, "database", "edupilot.db")
}
