# TaskFlow API
 
## Overview
 
TaskFlow is a task management REST API. Users can register, log in, create projects, add tasks to those projects, and assign tasks to themselves or others.
 
**What it does:**
- User registration and login with JWT authentication
- Create and manage projects
- Add tasks to projects with status, priority, assignee, and due date
- Filter tasks by status or assignee
- Only project owners can edit or delete their projects
- Task delete is restricted to the project owner or the task assignee
---
## Tech Stack
 
- **Language:** Go 1.22
- **Router:** chi
- **Database:** PostgreSQL 16
- **Auth:** JWT (24h expiry, bcrypt passwords)
- **Migrations:** golang-migrate
- **Container:** Docker + Docker Compose
---

## 2. Architecture Decisions 
### Structure 
- `backend/cmd/api`: application entrypoint and HTTP server bootstrap 
- `backend/internal/handlers`: HTTP layer (request parsing, status codes, response shaping) 
- `backend/internal/repository`: DB query layer 
- `backend/internal/models`: API/DB models and validation logic 
- `backend/internal/middleware`: JWT auth middleware 
- `backend/migrations`: schema evolution files 
- `backend/seed.sql`: deterministic local seed data 
### Why this structure 
- Keeps clear boundaries between transport concerns (handlers), persistence (repositories), and domain input validation (models). 
- Makes testing and code review easier by localizing responsibilities. 
- Uses `internal/` to prevent accidental external package usage. 
### Tradeoffs 
- Used a straightforward monolith layout (single API service) for fast delivery and easier reviewer setup. 
- Chose repository methods with explicit SQL over ORM abstractions to keep behavior transparent and predictable. 
- Did not add a separate service layer to avoid over-engineering for current scope. 
### Intentionally left out 
- No frontend app in this repository (scope is backend + API testing). 
- No rate limiting / refresh tokens / RBAC matrix beyond assignment rules. 
- No automated test suite yet (manual Postman-first validation for assignment timeline).
--- 

 ## 3. Running Locally
 
You only need **Docker** installed. Nothing else.
Open your terminal(Linux/Mac) and paste the following commands.
For windows use Git bash.
 
```bash
# 1. Clone the repo
git clone https://github.com/RakeshAero/taskflow-rakeshbalu
cd taskflow-rakeshbalu
 
# 2. Create your .env file
cp .env.example .env
 
# 3. Open .env and replace the JWT_SECRET with a real random value
#    Run this to generate one:
openssl rand -hex 32
#    Paste the output as JWT_SECRET= in your .env file
 
# 4. Start everything
docker compose up --build
```
 
That's it. The API will be running at **http://localhost:8080**
 
Docker starts things in this order automatically:
1. PostgreSQL starts and becomes ready
2. Migrations run
3. Seed data is inserted
4. API server starts
---

## 4. Running Migrations 
Migrations run **automatically** on container start from `backend/entrypoint.sh`: 
1. Wait for PostgreSQL readiness 
2. Run Goose migrations (`up`) 
3. Run seed SQL 
4. Start API server 

## 5. Test Credentials
 
Seed data is loaded automatically on first boot.
 
```
Email:    seed.user@taskflow.local
Password: Taskflow@123
```

## 6. API Reference
 
### Base URL
```
http://localhost:8080
```
 
---
 
### Auth
 
#### Register
```
POST /auth/register
```
Request:
```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "password": "password123"
}
```
Response `201`:
```json
{
  "token": "<jwt>",
  "user": {
    "id": "uuid",
    "name": "Jane Doe",
    "email": "jane@example.com",
    "created_at": "2026-04-16T10:00:00Z"
  }
}
```
 
---
 
#### Login
```
POST /auth/login
```
Request:
```json
{
  "email": "jane@example.com",
  "password": "password123"
}
```
Response `200`:
```json
{
  "token": "<jwt>",
  "user": { ... }
}
```
 
> All endpoints below require this header:
> `Authorization: Bearer <token>`
 
---
 
### Projects
 
#### List projects
```
GET /projects
```
Returns all projects you own or have tasks assigned to you in.
 
Response `200`:
```json
{
  "projects": [
    {
      "id": "uuid",
      "name": "Website Redesign",
      "description": "Q2 project",
      "owner_id": "uuid",
      "created_at": "2026-04-01T10:00:00Z"
    }
  ]
}
```
 
---
 
#### Create project
```
POST /projects
```
Request:
```json
{
  "name": "My Project",
  "description": "Optional description"
}
```
Response `201`: returns the created project object.
 
---
 
#### Get project + its tasks
```
GET /projects/:id
```
Response `200`:
```json
{
  "id": "uuid",
  "name": "Website Redesign",
  "description": "Q2 project",
  "owner_id": "uuid",
  "created_at": "...",
  "tasks": [ ... ]
}
```
 
---
 
#### Update project
```
PATCH /projects/:id
```
Only the project owner can do this.
 
Request (all fields optional):
```json
{
  "name": "New Name",
  "description": "New description"
}
```
Response `200`: returns the updated project object.
 
---
 
#### Delete project
```
DELETE /projects/:id
```
Only the project owner can do this. Deletes all tasks inside it too.
 
Response `204`: no body.
 
---
 
### Tasks
 
#### List tasks
```
GET /projects/:id/tasks
```
Optional filters:
```
?status=todo
?status=in_progress
?status=done
?assignee=<user-uuid>
```
Response `200`:
```json
{
  "tasks": [ ... ]
}
```
 
---
 
#### Create task
```
POST /projects/:id/tasks
```
Request:
```json
{ 
  "title": "Design login screen", 
  "description": "Mobile + desktop", 
  "priority": "high", 
  "assignee_id": "11111111-1111-1111-1111-111111111111" 
} 
```
- `priority` must be: `low`, `medium`, or `high`
- `status` always starts as `todo` — you can change it later via PATCH
Response `201`: returns the created task object.
 
---
 
#### Update task
```
PATCH /tasks/:id
```
All fields are optional. Send only what you want to change.
 
Request:
```json
{ 
"status": "done", 
"priority": "medium" 
}
```
To unassign a task, send `"assignee_id": null`.
 
- `status` must be: `todo`, `in_progress`, or `done`
- `priority` must be: `low`, `medium`, or `high`
Response `200`: returns the updated task object.
 
---
 
#### Delete task
```
DELETE /tasks/:id
```
Only the project owner or the task's assignee can do this.
 
Response `204`: no body.
 
---
 
### Error Responses
 
### Error response format 
Validation error (`400`): 
```json 
{ 
"error": "validation failed",
"fields": { "email": "is required" } 
} 
``` 
Common errors: 
- `401`: unauthenticated / invalid token 
- `403`: authenticated but not allowed 
- `404`: `{ "error": "not found" }` 
--- 
 

## 7. What I'd Do With More Time 
- Add automated tests: - unit tests for validation and middleware.
- Add request rate limiting and audit logging for security-sensitive endpoints. 

