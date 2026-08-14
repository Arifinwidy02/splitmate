# SplitMate

A modern shared-expense management web application.

## Stack

### Frontend

- Next.js
- TypeScript
- Tailwind CSS

### Backend

- Go
- REST API

### Database

- PostgreSQL

## Repository Structure

```text
splitmate/
├── frontend/
├── backend/
├── docs/
├── docker-compose.yml
└── AGENTS.md
```

## Documentation

```text
PRD.md
ERD.md
ARCHITECTURE.md
API.md
DESIGN_SYSTEM.md
MILESTONE.md
AGENTS.md
```

## Local Development

The expected local stack:

```text
Next.js
Go API
PostgreSQL
```

Docker Compose should provide PostgreSQL and any supporting infrastructure.

### Backend

```bash
docker compose up -d postgres   # start PostgreSQL
cd backend
cp .env.example .env            # required: JWT_SECRET must be set
go run ./cmd/migrate            # apply database migrations
go run ./cmd/api                # start the API (default :8080)
```

Configuration is read from environment variables (see `backend/.env.example`).

### Frontend

```bash
cd frontend
cp .env.example .env.local      # optional: NEXT_PUBLIC_API_URL
bun dev                         # start Next.js (default :3000)
```

Open http://localhost:3000, register an account, and sign in. Sessions use
HttpOnly cookies issued by the Go API.

## Product Flow

```text
Login
 ↓
Dashboard
 ↓
Create Group
 ↓
Invite Members
 ↓
Add Expense
 ↓
Calculate Balance
 ↓
Simplify Debt
 ↓
Settle
```

## MVP

The MVP focuses on:

- authentication
- groups
- members
- expenses
- equal/custom splitting
- balance calculation
- debt simplification
- settlements
- dashboard
- responsive UI

## Important

Financial calculations must be performed and validated by the Go backend.

The frontend is never the source of truth for balances.
