// Package database provides the EduPilot database port: connection management,
// transaction boundaries and the migration engine. It is infrastructure — the
// domain never imports it, and it never imports the domain.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Pure-Go SQLite driver: no cgo, so EduPilot builds on a school machine
	// with nothing but the Go toolchain (docs/adr/ADR-003).
	_ "modernc.org/sqlite"
)

// Driver names the SQL dialect. Migrations and repositories are written once
// per dialect behind the same ports.
type Driver string

const (
	// DriverSQLite is the standalone database (docs/adr/ADR-003).
	DriverSQLite Driver = "sqlite"
	// DriverPostgres is the site server and cloud database (docs/adr/ADR-010).
	DriverPostgres Driver = "postgres"
)

// DB wraps a *sql.DB with the EduPilot connection policy applied.
type DB struct {
	sqlDB  *sql.DB
	driver Driver
	path   string
}

// Options configures a connection.
type Options struct {
	// Driver selects the SQL dialect.
	Driver Driver
	// Path is the SQLite file path, or a PostgreSQL connection string.
	Path string
	// MaxOpenConns bounds the connection pool. SQLite standalone uses 1 writer.
	MaxOpenConns int
	// BusyTimeout is how long SQLite waits for a lock before failing.
	BusyTimeout time.Duration
}

// Open establishes a connection and applies the connection policy for the
// dialect. Foreign keys are enabled explicitly because SQLite defaults them off,
// and a finance schema without foreign key enforcement is not acceptable.
func Open(ctx context.Context, options Options) (*DB, error) {
	if options.Path == "" {
		return nil, fmt.Errorf("database: a path or connection string is required")
	}

	// SQLite creates no directories on its own, and a first run must not fail
	// merely because the data root is new.
	if options.Driver == DriverSQLite {
		if directory := filepath.Dir(options.Path); directory != "" {
			if err := os.MkdirAll(directory, 0o750); err != nil {
				return nil, fmt.Errorf("database: preparing %s: %w", directory, err)
			}
		}
	}

	dsn, err := dsnFor(options)
	if err != nil {
		return nil, err
	}

	sqlDB, err := sql.Open(sqlDriverName(options.Driver), dsn)
	if err != nil {
		return nil, fmt.Errorf("database: opening %s: %w", options.Driver, err)
	}

	if options.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(options.MaxOpenConns)
		sqlDB.SetMaxIdleConns(options.MaxOpenConns)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database: connecting to %s: %w", options.Driver, err)
	}

	if options.Driver == DriverSQLite {
		if err := applySQLitePragmas(ctx, sqlDB, options); err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
	}

	return &DB{sqlDB: sqlDB, driver: options.Driver, path: options.Path}, nil
}

// SQL exposes the underlying pool for infrastructure concerns such as backup.
func (d *DB) SQL() *sql.DB { return d.sqlDB }

// Driver returns the dialect of this connection.
func (d *DB) Driver() Driver { return d.driver }

// Path returns the configured path or connection string, with any password
// redacted so it is safe to log.
func (d *DB) Path() string { return redact(dsnForRedaction(d.path)) }

// Close releases the connection pool.
func (d *DB) Close() error { return d.sqlDB.Close() }

func sqlDriverName(driver Driver) string {
	if driver == DriverPostgres {
		return "postgres"
	}
	return "sqlite"
}

func dsnFor(options Options) (string, error) {
	if options.Driver == DriverPostgres {
		return options.Path, nil
	}

	busy := options.BusyTimeout
	if busy <= 0 {
		busy = 5 * time.Second
	}

	query := url.Values{}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busy.Milliseconds()))
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(NORMAL)")

	return "file:" + options.Path + "?" + query.Encode(), nil
}

func applySQLitePragmas(ctx context.Context, sqlDB *sql.DB, options Options) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
	}
	if options.BusyTimeout > 0 {
		pragmas = append(pragmas, fmt.Sprintf("PRAGMA busy_timeout = %d", options.BusyTimeout.Milliseconds()))
	}

	for _, pragma := range pragmas {
		if _, err := sqlDB.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("database: %s: %w", pragma, err)
		}
	}
	return nil
}

func redact(connectionString string) string {
	sep := strings.Index(connectionString, "://")
	if sep < 0 {
		return connectionString
	}
	rest := connectionString[sep+3:]
	at := strings.LastIndex(rest, "@")
	if at < 0 {
		return connectionString
	}
	credentials := rest[:at]
	if colon := strings.Index(credentials, ":"); colon >= 0 {
		credentials = credentials[:colon] + ":****"
	}
	return connectionString[:sep+3] + credentials + rest[at:]
}

func dsnForRedaction(path string) string {
	if strings.HasPrefix(path, "file:") {
		return path
	}
	return path
}
