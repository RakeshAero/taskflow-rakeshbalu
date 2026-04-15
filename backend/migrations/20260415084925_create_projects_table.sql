-- +goose Up
CREATE TABLE IF NOT EXISTS projects (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    description TEXT,                            -- nullable, no DEFAULT needed
    owner_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
 
-- Index on owner_id — speeds up "list projects owned by user" query.
CREATE INDEX IF NOT EXISTS projects_owner_id_idx ON projects(owner_id);

-- +goose Down
DROP TABLE IF EXISTS projects CASCADE;
