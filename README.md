# Stocky Backend

A backend for a stock rewards platform that grants Indian stock shares to users. Built with Go, Gin, GORM, and PostgreSQL.

## Quick Start

```bash
# Stocky Backend

Small Go service to award stock rewards to users. Uses Gin for HTTP and PostgreSQL (DB name: `assignment`).

Quick start

```bash
cp .env.example .env
docker compose up --build
```

Service: http://localhost:8080

Endpoints

- POST /reward — award shares (see docs/API.md for payload)
- GET /today-stocks/{userId} — today's rewards
- GET /historical-inr/{userId} — daily INR history
- GET /stats/{userId} — today's summary + portfolio value
- GET /portfolio/{userId} — current holdings
- GET /health — simple health check

Notes

- Example env: `.env.example` (do not commit real secrets)
- Postman collection: `postman/Stocky.postman_collection.json`
- See `docs/` for short API, schema and design notes.
