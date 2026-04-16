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

 ## 1. Running Locally
 
You only need **Docker** installed. Nothing else.
 
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
