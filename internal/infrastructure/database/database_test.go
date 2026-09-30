package database_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sqlitemigrations "github.com/elcomware/edupilot/database/migrations/sqlite"
	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/clock"
	"github.com/elcomware/edupilot/internal/testsupport"
)

const insertOrganisationSQL = `INSERT INTO organisations (id, name, currency, created_at, updated_at)
	VALUES (?, ?, 'XOF', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`

func TestOpenAppliesConnectionPolicy(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLiteOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var foreignKeys int
	if err := db.SQL().QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatal("foreign keys are not enforced; a finance schema cannot rely on SQLite defaults")
	}

	var journalMode string
	if err := db.SQL().QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}
}

func TestMigrationsApplyAndAreIdempotent(t *testing.T) {
	db := testsupport.NewDatabase(t)
	tx := database.NewTransactor(db, "")

	version, err := database.NewMigrator(tx.DB().SQL(), clock.System{}).Version(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if version < 1 {
		t.Fatalf("schema version = %d, want at least 1", version)
	}

	results, err := database.MigrateSQLite(context.Background(), tx, clock.System{})
	if err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}
	for _, result := range results {
		if result.Applied {
			t.Fatalf("migration %d was applied twice", result.Version)
		}
	}
}

func TestCoreSchemaHasExpectedTables(t *testing.T) {
	db := testsupport.NewDatabase(t)

	rows, err := db.SQL().Query("SELECT name FROM sqlite_master WHERE type = 'table'")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	present := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	required := []string{
		"schema_migrations", "organisations", "sites", "campuses", "academic_years", "academic_terms",
		"people", "user_accounts", "roles", "permissions", "role_permissions", "user_roles",
		"audit_entries", "outbox_events", "inbox_messages", "settings", "documents",
	}
	for _, table := range required {
		if !present[table] {
			t.Errorf("table %q is missing from the core schema", table)
		}
	}
}

func TestMigrationsAreOrderedAndChecksummed(t *testing.T) {
	migrations, err := database.LoadMigrations(sqlitemigrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 {
		t.Fatal("no migrations were loaded")
	}

	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].Version >= migrations[i].Version {
			t.Fatalf("migrations are not ordered: %d before %d", migrations[i-1].Version, migrations[i].Version)
		}
	}

	if migrations[0].Checksum() == "" {
		t.Fatal("checksum is empty")
	}
}

func TestReleasedMigrationEditIsDetected(t *testing.T) {
	db := testsupport.NewDatabase(t)
	tx := database.NewTransactor(db, "")

	migrations, err := database.LoadMigrations(sqlitemigrations.FS)
	if err != nil {
		t.Fatal(err)
	}

	tampered := make([]database.Migration, len(migrations))
	copy(tampered, migrations)
	tampered[0].SQL += "\n-- edited after release\n"

	migrator := database.NewMigrator(tx.DB().SQL(), clock.System{})
	err = tx.InTx(context.Background(), func(ctx context.Context) error {
		_, upErr := migrator.Up(ctx, tampered)
		return upErr
	})
	if err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("err = %v, want an immutability failure", err)
	}
}

func TestEnsureDirectoriesCreatesTheDataLayout(t *testing.T) {
	root := t.TempDir()
	if err := database.EnsureDirectories(root); err != nil {
		t.Fatal(err)
	}

	for _, dir := range []string{"database", "files", "backups", "logs", "runtime"} {
		info, err := os.Stat(filepath.Join(root, dir))
		if err != nil || !info.IsDir() {
			t.Fatalf("directory %q was not created", dir)
		}
	}
}

func TestTransactionRollsBackOnError(t *testing.T) {
	db := testsupport.NewDatabase(t)
	tx := database.NewTransactor(db, "")
	forced := errors.New("forced rollback")

	err := tx.InTx(context.Background(), func(ctx context.Context) error {
		if _, err := tx.Conn(ctx).ExecContext(ctx, insertOrganisationSQL, "id-rollback", "Rollback School"); err != nil {
			return err
		}
		return forced
	})
	if !errors.Is(err, forced) {
		t.Fatalf("err = %v, want the callback error", err)
	}

	var count int
	if err := db.SQL().QueryRow("SELECT COUNT(*) FROM organisations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("organisations count = %d, want 0 after rollback", count)
	}
}

func TestTransactionCommitsAndNests(t *testing.T) {
	db := testsupport.NewDatabase(t)
	tx := database.NewTransactor(db, "")

	err := tx.InTx(context.Background(), func(ctx context.Context) error {
		if _, err := tx.Conn(ctx).ExecContext(ctx, insertOrganisationSQL, "id-commit", "Committed School"); err != nil {
			return err
		}
		return tx.InTx(ctx, func(inner context.Context) error {
			_, execErr := tx.Conn(inner).ExecContext(inner, insertOrganisationSQL, "id-nested", "Nested School")
			return execErr
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.SQL().QueryRow("SELECT COUNT(*) FROM organisations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("organisations count = %d, want 2", count)
	}
}
