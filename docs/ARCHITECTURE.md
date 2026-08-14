# SplitMate — System Architecture

## 1. High-Level Architecture

```text
┌───────────────────────────┐
│        Browser            │
│   Next.js + React + TS    │
└─────────────┬─────────────┘
              │ HTTPS
              ▼
┌───────────────────────────┐
│        Go API              │
│                            │
│ HTTP Handler               │
│      ↓                     │
│ Middleware                 │
│      ↓                     │
│ Service                    │
│      ↓                     │
│ Repository                 │
└─────────────┬─────────────┘
              │
              ▼
┌───────────────────────────┐
│       PostgreSQL           │
└───────────────────────────┘
```

Optional infrastructure:

```text
Browser
  │
  ▼
Next.js
  │
  ▼
Go API
  │
  ├── PostgreSQL
  └── Email provider
```

---

# 2. Frontend

Technology:

- Next.js
- App Router
- TypeScript
- Tailwind CSS
- React
- TanStack Query where client-side server state is useful
- Zod for client-side validation
- Recharts for dashboard visualization

Frontend responsibilities:

- rendering
- navigation
- form interactions
- loading states
- optimistic UI where safe
- displaying backend data
- client-side validation for UX

Frontend must NOT own financial truth.

---

# 3. Backend

Technology:

- Go
- REST API
- PostgreSQL
- pgx
- chi or Gin

Recommended:

```text
Go
chi
pgx
go-playground/validator
```

Backend responsibilities:

- authentication
- authorization
- validation
- business rules
- expense calculations
- balance calculations
- settlement calculations
- database transactions

---

# 4. Go Project Structure

```text
backend/
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── auth/
│   ├── user/
│   ├── group/
│   ├── expense/
│   ├── settlement/
│   ├── balance/
│   │
│   ├── middleware/
│   ├── database/
│   └── server/
│
├── migrations/
│
├── pkg/
│   └── response/
│
├── tests/
│
├── Dockerfile
├── go.mod
└── go.sum
```

---

# 5. Layer Responsibilities

## Handler

HTTP concerns only.

Responsibilities:

- parse request
- validate basic request format
- call service
- return HTTP response

Handler must NOT contain complex business logic.

---

## Service

Business logic.

Examples:

```text
CreateGroup()
CreateExpense()
CalculateBalance()
SimplifyDebts()
CreateSettlement()
```

---

## Repository

Database access only.

Examples:

```text
Create()
FindByID()
FindByGroupID()
Update()
Delete()
```

Repository should not decide business rules.

---

# 6. Authentication

Recommended MVP approach:

```text
Browser
  ↓
Login
  ↓
Go API
  ↓
Verify credentials
  ↓
Create session
  ↓
Set HttpOnly Secure cookie
```

Avoid storing authentication tokens in:

```text
localStorage
sessionStorage
```

The browser should automatically send the secure cookie to the API.

For local development, HTTPS may be disabled, but production cookies must be Secure.

---

# 7. Authorization

Every protected endpoint should identify:

```text
current_user_id
```

Then verify resource access.

Example:

```text
GET /api/groups/:id

→ authenticate user
→ check group membership
→ fetch group
→ return response
```

Never rely on:

```text
if frontend hides button
```

Authorization must exist on the backend.

---

# 8. API Communication

Recommended API prefix:

```text
/api/v1
```

Example:

```text
GET    /api/v1/me
GET    /api/v1/groups
POST   /api/v1/groups
GET    /api/v1/groups/:groupId
POST   /api/v1/groups/:groupId/expenses
GET    /api/v1/groups/:groupId/balances
GET    /api/v1/groups/:groupId/settlements
POST   /api/v1/groups/:groupId/settlements
```

---

# 9. Money Handling

Never use floating point.

Bad:

```go
float64
```

Preferred:

```text
PostgreSQL NUMERIC(14,2)
```

In Go, consider representing money as integer minor units for calculations:

```text
Rp10.500 → 1050000
```

or use a decimal library consistently.

The implementation must choose one strategy and use it everywhere.

---

# 10. Debt Calculation

The balance engine is a core domain module.

Input:

```text
map[userID]balance
```

Example:

```text
A: +700
B: -400
C: -300
```

Output:

```text
B → A 400
C → A 300
```

Algorithm should be:

- deterministic
- pure where possible
- independently unit tested
- independent from HTTP/database code

---

# 11. Error Handling

Use typed domain errors.

Example:

```text
ErrUnauthorized
ErrForbidden
ErrGroupNotFound
ErrExpenseNotFound
ErrInvalidSplit
ErrInvalidSettlement
```

Map domain errors to HTTP:

```text
401 Unauthorized
403 Forbidden
404 Not Found
409 Conflict
422 Unprocessable Entity
500 Internal Server Error
```

---

# 12. Observability

MVP:

- structured logs
- request ID
- error logging
- basic health endpoint

Endpoints:

```text
GET /health
GET /ready
```

Future:

- OpenTelemetry
- metrics
- tracing
- Sentry

---

# 13. Deployment

Recommended:

```text
Frontend
Next.js → Vercel

Backend
Go → containerized deployment

Database
PostgreSQL → managed PostgreSQL
```

The architecture should remain portable so the Go service can also run on Railway, Render, Fly.io, Zeabur, or another container platform.

---

# 14. Environment Variables

Frontend:

```text
NEXT_PUBLIC_API_URL=
```

Backend:

```text
PORT=
DATABASE_URL=
JWT_SECRET=
COOKIE_DOMAIN=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
APP_ENV=
```

Never commit `.env`.

Commit:

```text
.env.example
```
