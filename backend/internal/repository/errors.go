package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// isUniqueViolation returns true when err is a Postgres unique_violation (23505).
// We use this in CreateUser (duplicate email) and anywhere else a UNIQUE
// constraint could be hit.
//
// pgconn.PgError is the concrete error type pgx returns for DB-level errors.
// It carries the SQLState code we can check precisely.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return false
}