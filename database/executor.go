package database

import (
	"context"
	"database/sql"
)

// Query executes a query that returns rows, typically a SELECT.
func Query(db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
func QueryRow(db *sql.DB, query string, args ...interface{}) *sql.Row {
	return db.QueryRow(query, args...)
}

// Exec executes a query without returning any rows.
func Exec(db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	return db.Exec(query, args...)
}

// QueryContext executes a query that returns rows with context.
func QueryContext(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row with context.
func QueryRowContext(ctx context.Context, db *sql.DB, query string, args ...interface{}) *sql.Row {
	return db.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a query without returning any rows with context.
func ExecContext(ctx context.Context, db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	return db.ExecContext(ctx, query, args...)
}

// --- Transaction Support ---

// TxQuery executes a query within a transaction.
func TxQuery(tx *sql.Tx, query string, args ...interface{}) (*sql.Rows, error) {
	return tx.Query(query, args...)
}

// TxQueryRow executes a query within a transaction that returns one row.
func TxQueryRow(tx *sql.Tx, query string, args ...interface{}) *sql.Row {
	return tx.QueryRow(query, args...)
}

// TxExec executes a command within a transaction.
func TxExec(tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	return tx.Exec(query, args...)
}

// TxQueryContext executes a query within a transaction with context.
func TxQueryContext(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (*sql.Rows, error) {
	return tx.QueryContext(ctx, query, args...)
}

// TxExecContext executes a command within a transaction with context.
func TxExecContext(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	return tx.ExecContext(ctx, query, args...)
}
