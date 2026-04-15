-- Idempotent seed data for local/dev environments.
-- Known credentials:
--   email: seed.user@taskflow.local
--   password: Taskflow@123

INSERT INTO users (id, name, email, password)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  'Seed User',
  'seed.user@taskflow.local',
  crypt('Taskflow@123', gen_salt('bf', 12))
)
ON CONFLICT (email) DO UPDATE
SET
  name = EXCLUDED.name,
  password = EXCLUDED.password;

INSERT INTO projects (id, name, description, owner_id)
VALUES (
  '22222222-2222-2222-2222-222222222222',
  'Seed Project',
  'Project created by seed script',
  '11111111-1111-1111-1111-111111111111'
)
ON CONFLICT (id) DO UPDATE
SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  owner_id = EXCLUDED.owner_id;

INSERT INTO tasks (id, title, description, status, priority, project_id, created_by, assignee_id)
VALUES
(
  '33333333-3333-3333-3333-333333333331',
  'Seed task: todo',
  'Seeded task with todo status',
  'todo',
  'low',
  '22222222-2222-2222-2222-222222222222',
  '11111111-1111-1111-1111-111111111111',
  '11111111-1111-1111-1111-111111111111'
),
(
  '33333333-3333-3333-3333-333333333332',
  'Seed task: in progress',
  'Seeded task with in_progress status',
  'in_progress',
  'medium',
  '22222222-2222-2222-2222-222222222222',
  '11111111-1111-1111-1111-111111111111',
  '11111111-1111-1111-1111-111111111111'
),
(
  '33333333-3333-3333-3333-333333333333',
  'Seed task: done',
  'Seeded task with done status',
  'done',
  'high',
  '22222222-2222-2222-2222-222222222222',
  '11111111-1111-1111-1111-111111111111',
  '11111111-1111-1111-1111-111111111111'
)
ON CONFLICT (id) DO UPDATE
SET
  title = EXCLUDED.title,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  priority = EXCLUDED.priority,
  project_id = EXCLUDED.project_id,
  created_by = EXCLUDED.created_by,
  assignee_id = EXCLUDED.assignee_id;
