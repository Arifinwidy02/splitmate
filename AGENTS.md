# SplitMate — OpenCode Agent Instructions

## 1. Project Mission

You are working on **SplitMate**, a production-oriented shared expense web application.

The project consists of:

```text
Frontend
Next.js + TypeScript

Backend
Go REST API

Database
PostgreSQL
```

The product must prioritize:

1. correctness
2. security
3. maintainability
4. good UX
5. simple architecture

Do not optimize for speed of coding at the expense of correctness.

---

# 2. Source of Truth

Before implementing a feature, read:

```text
PRD.md
ERD.md
ARCHITECTURE.md
API.md
DESIGN_SYSTEM.md
MILESTONE.md
```

If implementation conflicts with these documents:

1. identify the conflict
2. do not silently invent a new behavior
3. prefer updating the documentation before implementing a major architectural change

---

# 3. Architecture Rules

## Frontend

Next.js should be responsible for:

- UI
- routing
- presentation
- client interactions
- forms
- displaying server state

Do not put core financial business logic in React components.

---

## Backend

Go owns:

- business rules
- authorization
- validation
- financial calculations
- database writes
- balance calculations
- settlement calculations

The backend is the source of truth.

---

# 4. Go Architecture

Follow:

```text
Handler
  ↓
Service
  ↓
Repository
  ↓
Database
```

## Handler

Only HTTP concerns.

Do not put:

- SQL queries
- complex calculations
- business rules

inside handlers.

---

## Service

Services contain business logic.

Examples:

```text
CreateExpense
CalculateGroupBalance
GenerateSettlementSuggestions
CreateSettlement
```

---

## Repository

Repositories contain persistence logic.

Repositories should not know HTTP details.

---

# 5. Financial Rules

These rules are non-negotiable.

## Never use float for money

Avoid:

```go
float64
```

Use integer minor units or a proper decimal representation consistently.

---

## Never trust frontend calculations

The frontend may calculate values for preview.

The backend must recalculate and validate.

---

## Expense invariant

```text
sum(expense splits) == expense amount
```

If this is false:

```text
reject request
```

---

## Settlement invariant

```text
payer != receiver
amount > 0
```

---

# 6. Database Rules

Use PostgreSQL.

All schema changes must be done through migrations.

Never manually modify production tables.

Prefer:

```text
foreign keys
unique constraints
check constraints
indexes
```

when the database can enforce the invariant.

---

# 7. Authentication Rules

Never store authentication tokens in:

```text
localStorage
```

Prefer secure:

```text
HttpOnly
Secure
SameSite
```

cookies in production.

Passwords must never be stored in plaintext.

---

# 8. Authorization Rules

Every protected resource must verify authorization on the backend.

Example:

```text
User requests group
        ↓
Authenticate user
        ↓
Check group membership
        ↓
Allow / reject
```

Never rely on frontend route protection alone.

---

# 9. API Rules

All APIs should use:

```text
/api/v1
```

Use consistent JSON responses.

Success:

```json
{
  "data": {}
}
```

Error:

```json
{
  "error": {
    "code": "SOME_ERROR",
    "message": "Human-readable message"
  }
}
```

Never expose stack traces or SQL errors to users.

---

# 10. Frontend Rules

Use TypeScript strictly.

Avoid:

```ts
any
```

unless there is a documented reason.

Prefer small reusable components.

Do not create giant components such as:

```text
Dashboard.tsx
```

with hundreds of lines.

Split by responsibility.

---

# 11. Server State

Do not duplicate server state unnecessarily.

For data-heavy interactive pages, use an appropriate server-state approach.

Possible options:

```text
TanStack Query
```

or server components + server actions/fetching where appropriate.

Choose one approach per use case instead of introducing multiple competing patterns.

---

# 12. Forms

Forms must have:

- validation
- loading state
- error state
- success feedback
- disabled submit state when appropriate

Validation should exist:

```text
Frontend → UX validation
Backend  → authoritative validation
```

---

# 13. Error Handling

Never silently swallow errors.

Bad:

```ts
try {
  await save()
} catch {}
```

Better:

```text
catch
→ log appropriate diagnostic information
→ show useful user feedback
```

Do not expose internal errors.

---

# 14. Testing

Every important business rule must have tests.

Highest priority:

```text
balance calculation
debt simplification
expense split validation
settlement calculation
authorization
```

When modifying financial logic:

```text
write/update tests first when practical
```

---

# 15. Debt Simplification

The debt algorithm must be isolated from:

- HTTP
- database
- framework
- UI

It should be possible to test it with simple inputs and outputs.

Example:

```text
Input

A +700
B -400
C -300

Output

B → A 400
C → A 300
```

Do not bury this algorithm inside a handler.

---

# 16. UI Design Rules

Follow `DESIGN_SYSTEM.md`.

The approved visual direction is:

```text
modern
minimal
fintech-inspired
friendly
clean
```

Avoid:

- excessive gradients
- excessive animations
- overly dense tables
- unnecessary modals
- visual noise

---

# 17. Dashboard Rules

The dashboard must contain:

```text
Welcome header
Summary cards
Recent groups
Recent expenses
Balance overview
Expense categories
```

Primary CTA:

```text
+ Add Expense
```

---

# 18. Responsive Design

The application must work on:

```text
desktop
tablet
mobile
```

Do not design desktop first and ignore mobile.

---

# 19. Accessibility

All interactive elements must be keyboard accessible.

Use semantic HTML.

Inputs must have labels.

Dialogs must trap focus appropriately.

Do not use color as the only indication of financial state.

---

# 20. API Security

Validate:

- authentication
- authorization
- input shape
- ownership/membership
- amounts
- UUIDs
- pagination parameters

Never trust:

```text
userId
groupId
amount
payerId
```

sent from the frontend.

---

# 21. Git Rules

Use conventional commits where practical:

```text
feat:
fix:
refactor:
test:
docs:
chore:
```

Examples:

```text
feat(expense): add equal expense splitting
fix(balance): correct settlement calculation
test(balance): add multiple creditor cases
```

Keep commits focused.

---

# 22. Coding Workflow

For every task:

## Step 1

Read the relevant documentation.

## Step 2

Identify affected layers.

Example:

```text
Add Expense

Frontend
→ form

API
→ endpoint

Service
→ business logic

Repository
→ database

Tests
→ validation
```

## Step 3

Implement the smallest correct change.

## Step 4

Run tests.

## Step 5

Run lint/typecheck/build.

## Step 6

Review for security and authorization.

## Step 7

Update documentation if behavior changed.

---

# 23. Do Not Overengineer

Avoid introducing:

- microservices
- event buses
- Kafka
- Redis
- Kubernetes
- CQRS
- event sourcing

unless there is a real requirement.

The initial architecture should remain:

```text
Next.js
+
Go API
+
PostgreSQL
```

A modular monolith is preferred.

---

# 24. When Requirements Are Ambiguous

Do not invent complex product behavior.

Prefer:

1. existing PRD
2. existing ERD
3. existing architecture
4. simplest implementation

If ambiguity materially affects data or architecture, stop and explain the ambiguity before making a large change.

---

# 25. Definition of Done

A task is not complete merely because the code compiles.

Before considering a task complete:

- [ ] implementation works
- [ ] validation exists
- [ ] authorization is correct
- [ ] errors are handled
- [ ] tests exist for important logic
- [ ] lint passes
- [ ] typecheck passes
- [ ] build passes
- [ ] documentation is updated if necessary

---

# 26. Agent Behavior

When using OpenCode:

- inspect existing code before creating new files
- reuse existing utilities
- do not duplicate logic
- do not rewrite unrelated code
- do not change architecture without reason
- keep changes scoped to the current milestone
- explain assumptions in code comments only when necessary
- prefer readable code over clever code
- never fake API/database behavior in production code

The agent should behave like a senior engineer working on a real product, not like a code generator.
