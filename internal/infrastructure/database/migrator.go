package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/elcomware/edupilot/internal/kernel/clock"
)

// Migration is one released, immutable schema change.
type Migration struct {
	Version  int
	Name     string
	Filename string
	SQL      string
}

// Checksum is the SHA-256 of the migration body, recorded at apply time. A
// released migration whose checksum no longer matches is a hard failure: the
// immutability rule in docs/architecture protects real school data.
func (m Migration) Checksum() string {
	sum := sha256.Sum256([]byte(m.SQL))
	return hex.EncodeToString(sum[:])
}

// LoadMigrations reads and orders every migration in the given filesystem.
func LoadMigrations(files fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, fmt.Errorf("database: reading migrations: %w", err)
	}

	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, name, err := parseMigrationName(entry.Name())
		if err != nil {
			return nil, err
		}

		body, err := fs.ReadFile(files, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("database: reading %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			Filename: entry.Name(),
			SQL:      string(body),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	for i := 1; i < len(migrations); i++ {
		if migrations[i].Version == migrations[i-1].Version {
			return nil, fmt.Errorf("database: duplicate migration version %d", migrations[i].Version)
		}
	}

	return migrations, nil
}

func parseMigrationName(filename string) (int, string, error) {
	base, _, _ := strings.Cut(filename, ".sql")

	prefix, name, found := strings.Cut(base, "_")
	if !found || name == "" {
		return 0, "", fmt.Errorf("database: migration %q must be named <version>_<name>.sql", filename)
	}

	version, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, "", fmt.Errorf("database: migration %q has a non-numeric version", filename)
	}
	if version <= 0 {
		return 0, "", fmt.Errorf("database: migration %q must have a positive version", filename)
	}

	return version, name, nil
}

// MigrationResult reports the outcome of an applied migration.
type MigrationResult struct {
	Version   int
	Name      string
	Applied   bool
	Duration  time.Duration
	Skipped   bool
	Sensitive bool
}

// Migrator applies migrations in version order and records them in
// schema_migrations. There are no down migrations: a released schema only moves
// forward, and a mistake is corrected by a new migration.
type Migrator struct {
	db    DBTX
	clock clock.Clock
	table string
}

// NewMigrator returns a Migrator writing to the given schema table.
func NewMigrator(db DBTX, systemClock clock.Clock) *Migrator {
	return &Migrator{db: db, clock: systemClock, table: "schema_migrations"}
}

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version      INTEGER PRIMARY KEY,
    name         TEXT    NOT NULL,
    checksum     TEXT    NOT NULL,
    applied_at   TEXT    NOT NULL
)`

// Up applies every pending migration and returns what it did.
func (m *Migrator) Up(ctx context.Context, migrations []Migration) ([]MigrationResult, error) {
	// The ambient transaction is the only writer when one is open, and SQLite
	// runs single-writer: going back to the pool here would deadlock.
	db := m.conn(ctx)

	if _, err := db.ExecContext(ctx, createMigrationsTable); err != nil {
		return nil, fmt.Errorf("database: creating schema_migrations: %w", err)
	}

	applied, err := m.appliedVersions(ctx, db)
	if err != nil {
		return nil, err
	}

	if err := m.verifyChecksums(ctx, migrations, applied); err != nil {
		return nil, err
	}

	results := make([]MigrationResult, 0, len(migrations))
	for _, migration := range migrations {
		if _, done := applied[migration.Version]; done {
			results = append(results, MigrationResult{
				Version: migration.Version,
				Name:    migration.Name,
				Skipped: true,
			})
			continue
		}

		started := m.clock.Now()
		if err := m.apply(ctx, migration); err != nil {
			return results, err
		}
		results = append(results, MigrationResult{
			Version:  migration.Version,
			Name:     migration.Name,
			Applied:  true,
			Duration: m.clock.Now().Sub(started),
		})
	}

	return results, nil
}

// conn returns the ambient transaction when one is open, otherwise the pool.
func (m *Migrator) conn(ctx context.Context) DBTX {
	if tx, ok := FromContext(ctx); ok {
		return tx
	}
	return m.db
}

func (m *Migrator) apply(ctx context.Context, migration Migration) error {
	tx, ok := FromContext(ctx)
	if !ok {
		return fmt.Errorf("database: migrations must run inside a transaction")
	}

	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("database: applying migration %d_%s: %w", migration.Version, migration.Name, err)
	}

	_, err := tx.ExecContext(ctx,
		"INSERT INTO "+m.table+" (version, name, checksum, applied_at) VALUES (?, ?, ?, ?)",
		migration.Version, migration.Name, migration.Checksum(), m.clock.Now().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("database: recording migration %d: %w", migration.Version, err)
	}
	return nil
}

func (m *Migrator) appliedVersions(ctx context.Context, db DBTX) (map[int]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT version, checksum FROM "+m.table)
	if err != nil {
		return nil, fmt.Errorf("database: reading applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]string)
	for rows.Next() {
		var version int
		var checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, fmt.Errorf("database: scanning applied migration: %w", err)
		}
		applied[version] = checksum
	}
	return applied, rows.Err()
}

func (m *Migrator) verifyChecksums(ctx context.Context, migrations []Migration, applied map[int]string) error {
	for _, migration := range migrations {
		recorded, done := applied[migration.Version]
		if !done {
			continue
		}
		if recorded != migration.Checksum() {
			return fmt.Errorf(
				"database: migration %s was modified after release (recorded %s, found %s); released migrations are immutable",
				migration.Filename, recorded[:12], migration.Checksum()[:12],
			)
		}
	}
	return nil
}

// Version returns the highest applied migration version, or zero.
func (m *Migrator) Version(ctx context.Context) (int, error) {
	if _, err := m.db.ExecContext(ctx, createMigrationsTable); err != nil {
		return 0, fmt.Errorf("database: creating schema_migrations: %w", err)
	}

	var version sql.NullInt64
	err := m.db.QueryRowContext(ctx, "SELECT MAX(version) FROM "+m.table).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("database: reading schema version: %w", err)
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}
