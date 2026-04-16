# TaskFlow API
 
## A task management REST API built with Go, PostgreSQL, and Docker.
 
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
- No automated test suite yet (manual Postman-first validation for assignment timeline). --- 

 ## 3. Running Locally
 
You only need **Docker** installed. Nothing else.
Open your terminal(Linux/Mac) and paste the following commands.
For windows use Git bash.
 
```bash
# 1. Clone the repo
git clone https://github.com/RakeshAero/taskflow-rakeshbalu
cd taskflow-yourname
 
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

## 7. What I'd Do With More Time 
- Add automated tests: - unit tests for validation and middleware.
- Add request rate limiting and audit logging for security-sensitive endpoints. 