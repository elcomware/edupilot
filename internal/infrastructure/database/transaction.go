package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNoActiveTransaction is returned when a repository needs a transaction and
// none is present on the context.
var ErrNoActiveTransaction = errors.New("database: no active transaction")

// DBTX is the subset of database/sql shared by *sql.DB and *sql.Tx. Every
// EduPilot repository accepts a DBTX, so the same repository code runs inside
// or outside a transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

type transactionKey struct{}

// Transactor owns EduPilot transaction boundaries. A business operation is one
// transaction, and an outbox row written inside it commits atomically with the
// business change (docs/adr/ADR-007).
type Transactor struct {
	db  *DB
	iso string
}

// NewTransactor returns a Transactor for the given connection. The isolation
// level string is dialect specific; an empty string uses the driver default.
func NewTransactor(db *DB, isolation string) *Transactor {
	return &Transactor{db: db, iso: isolation}
}

// DB returns the connection the transactor runs against.
func (t *Transactor) DB() *DB { return t.db }

// InTx runs fn inside a transaction, committing on success and rolling back on
// error or panic. The transaction is placed on the context so repositories
// reached from fn join it rather than opening a second connection.
func (t *Transactor) InTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := FromContext(ctx); ok {
		return fn(ctx)
	}

	options := &sql.TxOptions{}
	if t.iso != "" {
		options.Isolation = sql.IsolationLevel(sql.LevelDefault)
	}

	tx, err := t.db.sqlDB.BeginTx(ctx, options)
	if err != nil {
		return fmt.Errorf("database: begin: %w", err)
	}

	committed := false
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := fn(WithTx(ctx, tx)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("database: commit: %w", err)
	}
	committed = true
	return nil
}

// WithTx returns a context carrying the given transaction.
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, transactionKey{}, tx)
}

// FromContext returns the transaction carried by the context, if any.
func FromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(transactionKey{}).(*sql.Tx)
	return tx, ok
}

// Conn returns the DBTX a repository should use for the given context: the
// ambient transaction when one exists, otherwise the connection pool.
func (t *Transactor) Conn(ctx context.Context) DBTX {
	if tx, ok := FromContext(ctx); ok {
		return tx
	}
	return t.db.sqlDB
}

// Pool returns the connection pool, for read paths that must not join an
// ambient transaction.
func (t *Transactor) Pool() DBTX {
	return t.db.sqlDB
}
