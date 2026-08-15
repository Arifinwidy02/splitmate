# SplitMate — API Specification

Base URL:

```text
/api/v1
```

All protected endpoints require authentication.

---

# 1. Authentication

## POST /auth/register

Request:

```json
{
  "name": "Arifin",
  "email": "arifin@example.com",
  "password": "secure-password"
}
```

Response:

```json
{
  "data": {
    "user": {
      "id": "uuid",
      "name": "Arifin",
      "email": "arifin@example.com"
    }
  }
}
```

---

## POST /auth/login

Request:

```json
{
  "email": "arifin@example.com",
  "password": "secure-password"
}
```

Response sets an HttpOnly session cookie.

---

## POST /auth/logout

Invalidates the current session.

---

## GET /auth/google

Starts the Google Sign In flow.

- Redirects the browser to Google's consent screen (302).
- Sets a short-lived HttpOnly `oauth_state` cookie to protect the callback from
  CSRF.
- Requires `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `OAUTH_REDIRECT_URL` and
  `APP_BASE_URL` to be configured. Returns `503 GOOGLE_NOT_CONFIGURED` otherwise.

**Important:** the endpoint (and the callback below) must be reached through the
frontend origin, so the session cookie is set on the frontend domain. The Next.js
app proxies `/api/v1/*` to the API via rewrites. Register
`{frontend-origin}/api/v1/auth/google/callback` as an authorized redirect URI in
Google Cloud Console.

---

## GET /auth/google/callback

Handles Google's redirect back to the app. Query parameters: `code`, `state`.

- Validates `state` against the `oauth_state` cookie (constant-time compare).
- Exchanges `code` for an access token, fetches the Google profile, then
  finds-or-creates the user:

  - existing `oauth_accounts` row → sign in as that user
  - no oauth account but existing email (password user) → link the account and
    sign in
  - otherwise → create a new user (no password hash) and link the account

- On success: sets the session cookie and redirects to `{APP_BASE_URL}/`.
- On failure: logs the reason server-side and redirects to
  `{APP_BASE_URL}/login?google=error`. Never exposes internal details to the user.

---

## GET /me

Returns the authenticated user.

---

# 2. Groups

## GET /groups

Returns groups for the authenticated user.

Response:

```json
{
  "data": {
    "groups": [
      {
        "id": "uuid",
        "name": "Bali Trip",
        "description": "Trip expenses",
        "currency": "IDR",
        "role": "admin",
        "memberCount": 2,
        "createdAt": "2026-08-14T19:00:00+07:00"
      }
    ]
  }
}
```

---

## POST /groups

Request:

```json
{
  "name": "Bali Trip",
  "description": "Trip expenses",
  "currency": "IDR"
}
```

Validation:

- name: required, 1–100 characters
- description: optional, at most 500 characters
- currency: 3-letter code (e.g. `IDR`, `USD`), uppercased automatically

The creator automatically becomes an admin member. Response includes the group
with `role: "admin"` and `memberCount: 1`.

---

## GET /groups/:groupId

Returns group overview. The user must be a member; non-members receive
`GROUP_NOT_FOUND`.

---

## PATCH /groups/:groupId

Updates group information. Fields are optional (partial update). Admin only;
members receive `FORBIDDEN`.

---

## DELETE /groups/:groupId

Deletes the group if the current user has permission (admin only).

---

# 3. Members

## GET /groups/:groupId/members

Returns group members. The user must be a member.

Response:

```json
{
  "data": {
    "members": [
      {
        "id": "uuid",
        "name": "Arifin",
        "email": "arifin@example.com",
        "role": "admin",
        "joinedAt": "2026-08-14T19:00:00+07:00"
      }
    ]
  }
}
```

---

## POST /groups/:groupId/invitations

Request:

```json
{
  "email": "budi@example.com"
}
```

Admin only. Creates a pending invitation (valid 7 days). The raw token is
returned exactly once in the response; only its hash is stored:

```json
{
  "data": {
    "invitation": {
      "id": "uuid",
      "groupId": "uuid",
      "email": "budi@example.com",
      "status": "pending",
      "expiresAt": "2026-08-21T19:00:00+07:00",
      "token": "SOjjdtnEGn9NJHqVUUgfvIEsFZiRfpnhZtygf4Ttoao"
    }
  }
}
```

Errors:

- `MEMBER_EXISTS` — the email already belongs to a member
- `INVITATION_EXISTS` — a pending invitation already exists for this email

---

## POST /groups/invitations/:token/accept

Accepts a pending invitation using the raw token. The authenticated user's
email must match the invitation email (`INVITATION_FORBIDDEN` otherwise).

Response:

```json
{
  "data": {
    "group": {
      "id": "uuid",
      "name": "Bali Trip",
      "description": "Trip expenses",
      "currency": "IDR",
      "role": "member",
      "memberCount": 2,
      "createdAt": "2026-08-14T19:00:00+07:00"
    }
  }
}
```

Errors:

- `INVITATION_NOT_FOUND` — unknown token
- `INVITATION_EXPIRED` — invitation older than 7 days
- `INVITATION_USED` — invitation already accepted
- `INVITATION_FORBIDDEN` — token used with a different account

---

## DELETE /groups/:groupId/members/:userId

Removes a member when authorized (admin only). An admin cannot remove
themselves.

---

# 4. Expenses

## GET /groups/:groupId/expenses

Returns expenses for a group (paginated). The user must be a member;
non-members receive `GROUP_NOT_FOUND`.

Query parameters:

```text
page       default 1, must be >= 1
limit      default 20, must be between 1 and 100
category   filter by category (exact match, optional)
from       filter by expense date (RFC3339, inclusive, optional)
to         filter by expense date (RFC3339, inclusive, optional)
```

Response:

```json
{
  "data": {
    "expenses": [
      {
        "id": "uuid",
        "groupId": "uuid",
        "description": "Dinner",
        "amount": "600000.00",
        "currency": "IDR",
        "paidBy": "uuid",
        "payerName": "Arifin",
        "createdBy": "uuid",
        "category": "Food & Drinks",
        "expenseDate": "2026-08-14T19:00:00+07:00",
        "participantCount": 3,
        "createdAt": "2026-08-14T19:00:00+07:00"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

---

## POST /groups/:groupId/expenses

Creates an expense. Any member of the group can create.

Categories:

```text
Accommodation
Food & Drinks
Transportation
Shopping
Entertainment
Utilities
Other
```

Amounts are always sent as decimal strings with at most 2 decimal places,
e.g. `"600000.00"`. Equal split:

```json
{
  "description": "Dinner",
  "amount": "600000.00",
  "currency": "IDR",
  "paidBy": "uuid",
  "category": "Food & Drinks",
  "expenseDate": "2026-08-14T19:00:00+07:00",
  "note": "optional, at most 1000 characters",
  "splitType": "equal",
  "participants": ["user-a", "user-b", "user-c"]
}
```

Custom split (`sum(splits.amount) == amount`):

```json
{
  "description": "Dinner",
  "amount": "600000.00",
  "currency": "IDR",
  "paidBy": "uuid",
  "category": "Food & Drinks",
  "expenseDate": "2026-08-14T19:00:00+07:00",
  "splitType": "custom",
  "splits": [
    { "userId": "user-a", "amount": "300000.00" },
    { "userId": "user-b", "amount": "200000.00" },
    { "userId": "user-c", "amount": "100000.00" }
  ]
}
```

Backend validation:

```text
description: required, 1-255 characters
amount:      > 0, at most 99,999,999,999.99, valid decimal string
currency:    must equal the group currency
category:    must be one of the categories above
paidBy:      must be a group member
participants/splits: must be group members and unique
equal split: each share is at least 0.01
custom split: sum(splits.amount) == amount  ->  otherwise INVALID_SPLIT
```

Equal split shares are rounded deterministically: the larger shares (at most
1 unit higher) go to the first participants sorted by user id.

Response:

```json
{
  "data": {
    "expense": {
      "id": "uuid",
      "groupId": "uuid",
      "description": "Dinner",
      "amount": "600000.00",
      "currency": "IDR",
      "paidBy": "uuid",
      "payerName": "",
      "category": "Food & Drinks",
      "expenseDate": "2026-08-14T19:00:00+07:00",
      "note": null,
      "participants": [
        { "userId": "user-a", "amount": "300000.00" },
        { "userId": "user-b", "amount": "200000.00" },
        { "userId": "user-c", "amount": "100000.00" }
      ],
      "createdAt": "2026-08-14T19:00:00+07:00",
      "updatedAt": "2026-08-14T19:00:00+07:00"
    }
  }
}
```

---

## GET /expenses/:expenseId

Returns expense detail including participant names. The user must be a member
of the expense's group; non-members receive `GROUP_NOT_FOUND`, unknown
expenses `EXPENSE_NOT_FOUND`.

---

## PATCH /expenses/:expenseId

Updates an expense. The request body has the same shape as create. Only the
creator can update; other members receive `FORBIDDEN`.

---

## DELETE /expenses/:expenseId

Deletes an expense. Only the creator can delete; other members receive
`FORBIDDEN`.

---

# 5. Balances

## GET /groups/:groupId/balances

Returns the net balance of every group member. The user must be a member;
non-members receive `GROUP_NOT_FOUND`.

Balance formula (computed from expenses, splits, and settlements — never
stored):

```text
balance = payments made - personal shares + settlements paid - settlements received
```

Positive means the member should receive money, negative means they should
pay money.

Response:

```json
{
  "data": {
    "members": [
      {
        "userId": "user-a",
        "name": "Arifin",
        "balance": "700000.00"
      },
      {
        "userId": "user-b",
        "name": "Budi",
        "balance": "-400000.00"
      }
    ]
  }
}
```

---

## GET /groups/:groupId/settlement-suggestions

Returns a deterministic, near-minimal set of transfers that settle all
balances. The largest debtor pays the largest creditor first; ties are broken
by user id. The user must be a member.

Response:

```json
{
  "data": {
    "settlements": [
      {
        "fromUserId": "user-b",
        "toUserId": "user-a",
        "amount": "400000.00"
      }
    ]
  }
}
```

When all balances are zero the array is empty.

---

## GET /me/balance

Returns the authenticated user's balance aggregated across all their groups
(personal balance).

Response:

```json
{
  "data": {
    "owedToUser": "750000.00",
    "userOwes": "320000.00",
    "netBalance": "430000.00"
  }
}
```

---

# 6. Settlements

## GET /groups/:groupId/settlements

Returns settlement history (newest first). The user must be a member;
non-members receive `GROUP_NOT_FOUND`.

Response:

```json
{
  "data": {
    "settlements": [
      {
        "id": "uuid",
        "payerId": "uuid",
        "payerName": "Budi",
        "receiverId": "uuid",
        "receiverName": "Arifin",
        "amount": "400000.00",
        "settledAt": "2026-08-14T19:00:00+07:00",
        "createdAt": "2026-08-14T19:00:00+07:00"
      }
    ]
  }
}
```

---

## POST /groups/:groupId/settlements

Records a repayment. Amounts are decimal strings with at most 2 decimal
places.

Request:

```json
{
  "payerId": "user-b",
  "receiverId": "user-a",
  "amount": "400000.00",
  "settledAt": "2026-08-14T19:00:00+07:00"
}
```

`settledAt` is optional and defaults to now.

Authorization and validation:

- the authenticated user must be the payer (a user cannot settle another
  user's debt) — `FORBIDDEN` otherwise
- the authenticated user must be a group member — `GROUP_NOT_FOUND` otherwise
- payer != receiver
- receiver must be a group member
- amount > 0, at most 99,999,999,999.99

Response (201):

```json
{
  "data": {
    "id": "uuid",
    "payerId": "user-b",
    "payerName": "Budi",
    "receiverId": "user-a",
    "receiverName": "Arifin",
    "amount": "400000.00",
    "settledAt": "2026-08-14T19:00:00+07:00",
    "createdAt": "2026-08-14T19:00:00+07:00"
  }
}
```

Settlements affect balances automatically: `GET /groups/:groupId/balances`
derives balances from expenses, splits, and settlements on every request, so
no stored balance needs to be recalculated.

---

# 7. Dashboard

## GET /dashboard

Returns aggregated dashboard data for the authenticated user. All money
values are decimal strings (at most 2 decimal places).

Response:

```json
{
  "data": {
    "summary": {
      "owedToUser": "750000.00",
      "userOwes": "320000.00",
      "netBalance": "430000.00",
      "totalExpense": "8450000.00",
      "settledAmount": "1000000.00"
    },
    "groups": [
      {
        "id": "uuid",
        "name": "Bali Trip",
        "currency": "IDR",
        "memberCount": 3,
        "balance": "700000.00"
      }
    ],
    "recentExpenses": [
      {
        "id": "uuid",
        "groupId": "uuid",
        "groupName": "Bali Trip",
        "description": "Dinner",
        "payerName": "Arifin",
        "amount": "600000.00",
        "category": "Food & Drinks",
        "expenseDate": "2026-08-14T19:00:00+07:00",
        "participantCount": 3
      }
    ],
    "categories": [
      {
        "category": "Food & Drinks",
        "total": "4500000.00"
      }
    ]
  }
}
```

Notes:

- `summary` is aggregated across all of the user's groups. Positive balances
  count toward `owedToUser`, negative toward `userOwes`; `netBalance` is the
  sum of both. `settledAmount` is the total of settlements where the user is
  payer or receiver.
- `groups` lists the user's groups with the user's balance in each.
- `recentExpenses` contains the 10 most recent expenses across the user's
  groups.
- `categories` contains the expense total per category, ordered by total
  descending.
- The frontend should prefer this endpoint for the initial dashboard instead
  of making many unrelated requests.

---

# 8. Error Response

All errors:

```json
{
  "error": {
    "code": "INVALID_SPLIT",
    "message": "Expense split does not equal the total amount"
  }
}
```

Never expose:

```text
SQL errors
stack traces
password hashes
internal secrets
```

in production API responses.
