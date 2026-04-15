-- +goose Up
CREATE TYPE task_status   AS ENUM ('todo', 'in_progress', 'done');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high');
 
CREATE TABLE IF NOT EXISTS tasks (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT          NOT NULL,
    description TEXT,                                -- nullable
    status      task_status   NOT NULL DEFAULT 'todo',
    priority    task_priority NOT NULL,
    project_id  UUID          NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    assignee_id UUID          REFERENCES users(id) ON DELETE SET NULL, -- nullable
    due_date    DATE,                                -- nullable
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
 
-- Index on project_id — every task list query filters by this.
CREATE INDEX IF NOT EXISTS tasks_project_id_idx   ON tasks(project_id);
 
-- Index on assignee_id — used by the ?assignee= filter and the UNION
-- query that finds projects a user has tasks in.
CREATE INDEX IF NOT EXISTS tasks_assignee_id_idx  ON tasks(assignee_id);
 
-- Index on status — used by the ?status= filter.
CREATE INDEX IF NOT EXISTS tasks_status_idx       ON tasks(status);
 
-- ── Auto-update updated_at ────────────────────────────────────────────────────
-- Postgres doesn't update updated_at automatically like MySQL's ON UPDATE.
-- We need a trigger function + a trigger on the tasks table.
 
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
 
CREATE TRIGGER tasks_set_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER  IF EXISTS tasks_set_updated_at ON tasks;
DROP FUNCTION IF EXISTS set_updated_at();
DROP TABLE    IF EXISTS tasks CASCADE;
DROP TYPE     IF EXISTS task_priority;
DROP TYPE     IF EXISTS task_status;
