-- +goose Up
-- Creates the users table.
-- Run automatically on startup via golang-migrate.
 
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- gives us gen_random_uuid()
 
CREATE TABLE IF NOT EXISTS users (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL,
    password   TEXT        NOT NULL,             -- bcrypt hash, never plaintext
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
 
-- Unique index on email — enforces no duplicate accounts.
-- Also speeds up the login query (SELECT * FROM users WHERE email = $1).
CREATE UNIQUE INDEX IF NOT EXISTS users_email_idx ON users(email);


-- +goose Down
DROP TABLE IF EXISTS users CASCADE;
