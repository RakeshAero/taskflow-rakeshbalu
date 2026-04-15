package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver with database/sql
)

// Connect opens a connection pool to PostgreSQL and verifies it with a ping.
// Returns the pool on success, or an error if the DB is unreachable.

func Connect(dsn string) (*sqlx.DB, error) {
	// sqlx.Open does NOT actually dial the DB — it just validates the DSN format.
	// The real connection happens on the first query, or when we call db.PingContext().
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// ── Connection Pool Tuning ─────────────────────────────────────────────────
	// These are conservative defaults suitable for a small/medium app.
	// You can raise MaxOpenConns if you see "too many clients" errors under load.

	// Maximum number of open (in-use + idle) connections to the DB.
	// PostgreSQL's default max_connections is 100 — stay well below it.
	db.SetMaxOpenConns(25)

	// Maximum number of idle connections kept in the pool between requests.
	// Idle connections are reused instantly — no TCP handshake overhead.
	db.SetMaxIdleConns(10)

	// How long a connection can stay in the pool before being closed and replaced.
	// Prevents stale connections that the DB or a firewall may have silently dropped.
	db.SetConnMaxLifetime(30 * time.Minute)

	// How long an idle connection can sit in the pool unused before being closed.
	db.SetConnMaxIdleTime(5 * time.Minute)

	// ── Verify connectivity ────────────────────────────────────────────────────
	// PingContext actually dials the DB. We give it 5 seconds — if Docker
	// starts the API before Postgres is ready, main.go will exit and Docker
	// will restart the container (depends_on + healthcheck in docker-compose.yml).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Database connection pool ready",
		"max_open_conns", 25,
		"max_idle_conns", 10,
	)

	return db, nil
}

// HealthCheck pings the DB and returns an error if it is unreachable.
// Called by GET /health so we can check DB status, not just HTTP status.
func HealthCheck(db *sqlx.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}