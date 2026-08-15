# SplitMate — Development Milestones

## Milestone 0 — Project Foundation

### Goal

Create the monorepo and development environment.

### Tasks

- [x] Create repository
- [x] Create Next.js app
- [x] Create Go API
- [x] Create PostgreSQL local environment
- [x] Add Docker Compose
- [x] Add `.env.example`
- [x] Add Git ignore rules
- [x] Add basic README
- [x] Add health endpoint
- [x] Configure linting
- [x] Configure formatting

### Definition of Done

```text
Next.js runs
Go API runs
PostgreSQL runs
Next.js can reach Go API
```

---

# Milestone 1 — Database

### Goal

Implement PostgreSQL schema and migrations.

### Tasks

- [x] users
- [x] oauth_accounts
- [x] groups
- [x] group_members
- [x] group_invitations
- [x] expenses
- [x] expense_splits
- [x] settlements
- [x] indexes
- [x] foreign keys
- [x] constraints

### Definition of Done

Database can be recreated from zero using migrations.

---

# Milestone 2 — Authentication

### Goal

Users can securely authenticate.

### Tasks

- [x] Register
- [x] Login
- [x] Logout
- [x] Session
- [x] Password hashing
- [x] Auth middleware
- [x] Current user endpoint
- [x] Next.js auth state
- [x] Protected routes

### Optional

- [ ] Google OAuth

### Definition of Done

A user can register, log in, refresh the browser, and remain authenticated.

---

# Milestone 3 — Groups

### Goal

Users can create and manage groups.

### Tasks

- [x] Create group
- [x] List groups
- [x] Get group
- [x] Update group
- [x] Delete group
- [x] Add members
- [x] Invitations
- [x] Membership authorization

### Definition of Done

Two users can belong to the same group.

---

# Milestone 4 — Expenses

### Goal

Users can record real expenses.

### Tasks

- [x] Create expense
- [x] Equal split
- [x] Custom split
- [x] Edit expense
- [x] Delete expense
- [x] Expense detail
- [x] Expense list
- [x] Categories
- [x] Validation

### Important Tests

- [x] split total equals expense total
- [x] negative amount rejected
- [x] non-member rejected
- [x] invalid payer rejected

---

# Milestone 5 — Balance Engine

### Goal

Correctly calculate financial balances.

### Tasks

- [x] Calculate member balances
- [x] Calculate personal balance
- [x] Generate settlement suggestions
- [x] Debt simplification
- [x] Unit tests
- [x] Edge cases

### Edge Cases

```text
2 people
3 people
10+ people
zero balance
multiple creditors
multiple debtors
very small amounts
exact settlements
```

This milestone is a core engineering milestone.

---

# Milestone 6 — Settlements

### Goal

Users can record repayments.

### Tasks

- [x] Settlement history
- [x] Create settlement
- [x] Authorization
- [x] Transaction handling
- [x] Recalculate balances
- [x] Settlement UI (delivered in M7 — quick settle + record payment panel in the group page)

### Notes

Balances are never stored; the balance engine derives them from expenses,
splits, and settlements on every read, so no recalculations are needed at
write time.

---

# Milestone 7 — Dashboard

### Goal

Implement the approved dashboard design.

### Tasks

- [x] Sidebar (desktop sidebar + mobile bottom navigation)
- [x] Header
- [x] Summary cards
- [x] Recent groups
- [x] Recent expenses
- [x] Balance overview
- [x] Expense categories
- [x] Charts (CSS bars, no chart library)
- [x] Responsive layout

### Notes

- Backend `GET /api/v1/dashboard` aggregates summary, groups, recent
  expenses, and category totals in one request.
- Supporting pages delivered with the dashboard: groups list with create and
  join-by-token forms, and group detail with balances, add-expense form,
  members with invite (token shown once with copy), settlement panel, and
  settlement history. This also completes the M6 Settlement UI task.
- No frontend tests yet; covered by live E2E during the milestone and
  deferred to M9.

---

# Milestone 8 — UX Polish

### Tasks

- [x] Loading states
- [x] Skeletons
- [x] Empty states
- [x] Error states
- [x] Toasts
- [x] Confirmation dialogs
- [x] Responsive design
- [x] Accessibility
- [x] Keyboard navigation

### Notes

- Skeletons via `loading.tsx` for `/`, `/groups`, `/groups/[id]`.
- Page-level errors via `app/(app)/error.tsx` (retry) and branded `app/not-found.tsx`.
- Toasts: success feedback via `?success=` query param after server-action redirects,
  rendered by `components/toast.tsx` (auto-dismiss, dismissible, `aria-live`).
- Confirmation dialogs: native `<dialog>` (`showModal`) with backdrop + ESC handling
  (`components/confirm-dialog.tsx`), used for delete expense and delete group.
- Delete expense UI (creator-only button, backend `DELETE /api/v1/expenses/{id}`,
  already existed) and delete group UI (admin-only) added in this milestone.
- Accessibility: skip-to-content link, `:focus-visible` outline, `prefers-reduced-motion`
  support, semantic headings, `aria-label` on icon buttons.
- API addition: expense list/detail responses now include `createdBy`
  (backend `expenseResponse`/`summaryResponse`) so the UI can show the delete
  button only to the creator.

---

# Milestone 9 — Testing

### Backend

- [x] Unit tests
- [x] Service tests
- [x] Balance algorithm tests
- [x] Repository tests
- [x] API integration tests

### Frontend

- [x] Component tests
- [x] Form tests
- [x] Critical user flow tests

### E2E

Test:

```text
Register
→ Login
→ Create Group
→ Add Member
→ Add Expense
→ Calculate Balance
→ Settle
```

### Notes

- Backend unit/service tests cover auth, sessions, money parsing, expense split
  validation, balance calculation, debt simplification and settlement service
  rules (`pkg/money`, `internal/{auth,session,expense,balance,settlement,group,dashboard}`).
- Repository + API integration coverage via `internal/integration/integration_test.go`
  against a real PostgreSQL instance (creates/drops `splitmate_integration` DB;
  skips when the DB is unavailable). Covers the full user journey
  (register → login → group → invite → accept → expense → balance → settle →
  dashboard → delete) plus authorization (non-member 404s, non-creator delete 403,
  payer-only settlement 403, member delete group 403) and financial validation
  (zero amount, split-sum mismatch, payer == receiver).
- Frontend: 41 Vitest tests (format helpers, `server-api`, toast, confirm dialog,
  delete buttons, add-expense form, settle panel, group forms) with jsdom +
  Testing Library.
- E2E via Playwright (`frontend/tests/e2e/main-flow.spec.ts`): the official flow
  above (registration intentionally redirects to the sign-in page — register API
  does not set a session cookie), plus a non-member 404 check. The journey test
  deletes its group at the end so test data does not accumulate.
- Bug found by E2E: `redirect()` inside `try/catch` in server actions swallowed
  `NEXT_REDIRECT` and surfaced an error alert; redirects moved outside the catch
  in `app/actions/{groups,expenses,settlements}.ts`.
- Bug found by E2E: `<input type="datetime-local">` values are not valid RFC3339;
  `toRFC3339` in `lib/format.ts` converts client-side (hidden `expenseDateRfc`
  field) with a server-side fallback.

---

# Milestone 10 — Production

### Tasks

- [x] Production environment
- [x] PostgreSQL production
- [x] Go container
- [x] Next.js deployment
- [x] Environment secrets
- [ ] CORS configuration (not needed — frontend calls the API server-side only; revisit if OAuth redirect flows are added)
- [x] Secure cookies
- [x] Logging
- [x] Health checks
- [ ] Database backups
- [ ] Error monitoring

### Notes

- Next.js deployed on Vercel (`https://splitmate-phi.vercel.app`). The frontend lives in
  `frontend/`, so the Vercel project's Root Directory must be set to `frontend`
  (the `rootDirectory` property in `vercel.json` is rejected by Vercel — managed in
  the dashboard). Framework Preset must be Next.js, otherwise the deployment has no
  routing manifest and every path returns a platform 404.
- Go API deployed on Zeabur (`https://splitmate.zeabur.app`) via `backend/Dockerfile`
  (multi-stage, runs `migrate` then `api`; `.dockerignore` excludes `.env`). Service
  Root Directory on Zeabur is set to `backend`.
- PostgreSQL production on Zeabur; connection string injected via `DATABASE_URL`.
- Environment secrets: `JWT_SECRET` (strong random), `DATABASE_URL`, `APP_ENV=production`
  (enables `Secure` cookies), `NEXT_PUBLIC_API_URL` on Vercel (inlined at build time —
  redeploy required after changing it).
- Verified with a Playwright smoke test against the production URL: register → login →
  create group → add expense → dashboard totals → delete group (cleanup), all green.
- Remaining: database backups and error monitoring are intentionally deferred until
  real usage exists.

---

# Recommended Coding Order

```text
M0
 ↓
M1
 ↓
M2
 ↓
M3
 ↓
M4
 ↓
M5
 ↓
M6
 ↓
M7
 ↓
M8
 ↓
M9
 ↓
M10
```

Do not jump directly to dashboard polish before the financial domain is correct.
