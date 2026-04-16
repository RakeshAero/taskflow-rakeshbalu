#!/bin/sh
set -eu

echo "Waiting for PostgreSQL..."
until pg_isready -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" >/dev/null 2>&1; do
  sleep 1
done

echo "Running migrations..."
goose -dir /app/migrations postgres "${DATABASE_URL}" up

echo "Seeding database..."
psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f /app/seed.sql

echo "Starting API..."
exec /app/taskflow