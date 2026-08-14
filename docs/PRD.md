# SplitMate — Product Requirements Document

## 1. Product Overview

**SplitMate** is a web application for managing shared expenses between friends, roommates, families, trips, and teams.

The core problem is simple:

> People can easily record what they spend, but calculating who owes whom becomes annoying as the number of people and expenses grows.

SplitMate solves this by allowing users to:

- create groups
- invite members
- record expenses
- define who paid
- define who shares an expense
- calculate balances automatically
- simplify debts
- record settlements
- understand spending through dashboards and reports

### Product Positioning

SplitMate is not intended to be a pixel-for-pixel Splitwise clone.

The product should feel like:

> **A modern shared-expense manager with a personal-finance-style dashboard.**

The MVP prioritizes simplicity, correctness, and a polished desktop/mobile-responsive experience.

---

# 2. Goals

## Primary Goals

1. Make adding a shared expense extremely fast.
2. Always calculate balances correctly.
3. Minimize the number of settlement transactions.
4. Make group financial state understandable at a glance.
5. Provide a clean and modern dashboard.
6. Build the application as a production-quality fullstack project.

## Technical Goals

The project should demonstrate:

- Next.js App Router
- React + TypeScript
- Go REST API
- PostgreSQL
- Authentication and authorization
- relational database design
- service/repository architecture
- database transactions
- input validation
- debt calculation algorithms
- API documentation
- automated testing
- Docker-based local development
- production-ready error handling

---

# 3. Non-Goals

The MVP will NOT include:

- bank account synchronization
- credit card synchronization
- real-money payment processing
- cryptocurrency
- AI expense recognition
- OCR receipt scanning
- multi-currency conversion
- native mobile apps
- accounting/tax features

These can be considered future features.

---

# 4. Target Users

## Traveler

A group of friends traveling together and sharing hotels, food, taxis, tickets, etc.

## Roommates

People sharing rent, electricity, internet, groceries, and household expenses.

## Friends

People frequently eating together or sharing activities.

## Small Teams

Teams sharing event, office, or project expenses.

---

# 5. Core User Stories

## Authentication

- As a user, I can register.
- As a user, I can log in.
- As a user, I can log out.
- As a user, I can sign in with Google.
- As a user, I can view and update my profile.

## Groups

- As a user, I can create a group.
- As a user, I can view groups I belong to.
- As a user, I can invite another person.
- As a group member, I can view other members.
- As a group admin, I can manage group members.
- As a group admin, I can edit group information.

## Expenses

- As a group member, I can add an expense.
- I can specify the amount.
- I can specify who paid.
- I can specify who participates.
- I can split equally.
- I can split using custom amounts.
- I can edit an expense.
- I can delete an expense.
- I can view an expense detail.

## Balances

- I can see how much I owe.
- I can see how much others owe me.
- I can see my net balance.
- I can see balances inside a group.
- I can see simplified settlement suggestions.

## Settlements

- I can mark a debt as settled.
- I can view settlement history.
- I can undo/correct a settlement when authorized.

---

# 6. Main User Flow

```text
Register / Login
      ↓
Dashboard
      ↓
Create Group
      ↓
Invite Members
      ↓
Add Expense
      ↓
Select Payer
      ↓
Select Participants
      ↓
Choose Split Method
      ↓
Save Expense
      ↓
Recalculate Balances
      ↓
Show Settlement Suggestions
      ↓
Members Settle
      ↓
Group Balance Becomes Zero
```

---

# 7. Dashboard Requirements

The dashboard is the primary landing page after authentication.

## Header

Display:

- application logo/name
- navigation
- notification indicator
- user avatar
- user name
- Add Expense button

## Summary Cards

Display:

1. You are owed
2. You owe
3. Net balance
4. Total expense

Example:

```text
You are owed
Rp750.000

You owe
Rp320.000

Net balance
+Rp430.000

Total expense
Rp8.450.000
```

## Recent Groups

Display:

- group name
- member count
- user's balance in the group
- balance state:
  - You are owed
  - You owe
  - Settled

## Recent Expenses

Display:

- expense description
- group
- payer
- date
- amount
- participant count

## Balance Overview

Show:

- total owed to user
- total user owes
- settled amount
- net balance

## Expense by Category

Show:

- category
- total
- percentage
- visual indicator

MVP categories:

- Accommodation
- Food & Drinks
- Transportation
- Shopping
- Entertainment
- Utilities
- Other

---

# 8. Group Requirements

A group contains:

- name
- description
- currency
- members
- expenses
- balances
- settlements

Group page tabs:

```text
Overview
Expenses
Balances
Members
Settings
```

## Group Overview

Display:

- total spending
- current user balance
- member balances
- recent expenses
- settlement suggestions

---

# 9. Expense Requirements

Each expense contains:

- description
- total amount
- currency
- category
- payer
- participants
- expense date
- optional note

## Split Methods

### Equal

Example:

```text
Rp600.000 / 3 people

A = Rp200.000
B = Rp200.000
C = Rp200.000
```

### Custom Amount

Example:

```text
A = Rp100.000
B = Rp250.000
C = Rp250.000
```

The backend MUST validate:

```text
sum(split amounts) == expense amount
```

before creating the expense.

---

# 10. Balance Calculation

For every user:

```text
balance = amount_paid - amount_owed
```

Example:

```text
Arifin paid: Rp1.500.000
Arifin owes: Rp800.000

balance = +Rp700.000
```

Interpretation:

```text
positive → user should receive money
negative → user should pay money
zero     → settled
```

---

# 11. Debt Simplification

The backend should transform member balances into a minimal or near-minimal set of transactions.

Example:

```text
A = +700
B = -400
C = -300
```

Result:

```text
B → A = 400
C → A = 300
```

The calculation must happen on the backend.

The frontend should treat the backend result as the source of truth.

---

# 12. Settlement Requirements

A settlement represents an actual repayment.

Fields:

- payer
- receiver
- amount
- group
- timestamp

Example:

```text
Budi paid Arifin Rp400.000
```

After settlement, the group balance should reflect the repayment.

Settlement creation should happen inside a database transaction.

---

# 13. Authorization

Users can only access resources they are authorized to access.

Examples:

- A user cannot view another private group.
- A user cannot add an expense to a group they do not belong to.
- A group member cannot remove an admin unless allowed.
- A user cannot settle another user's debt arbitrarily.

Authorization must be enforced by the Go backend, not only by the Next.js frontend.

---

# 14. API Error Contract

All API errors should use a consistent structure.

Example:

```json
{
  "error": {
    "code": "GROUP_NOT_FOUND",
    "message": "Group not found"
  }
}
```

Never expose internal database errors directly to clients.

---

# 15. Non-Functional Requirements

## Performance

- Dashboard API should normally respond under 500ms in local/typical production conditions.
- Database queries should avoid N+1 patterns.
- Pagination should be used for large expense histories.

## Security

- Passwords must be hashed using Argon2id or bcrypt.
- Authentication tokens must not be stored in localStorage.
- Prefer secure HttpOnly cookies.
- Validate all request payloads.
- Enforce authorization server-side.
- Use parameterized SQL / ORM query APIs.
- Never trust client-calculated balances.

## Reliability

Financial calculations must be deterministic and tested.

---

# 16. MVP Definition of Done

The MVP is complete when a user can:

```text
Register
→ Login
→ Create Group
→ Invite Member
→ Add Expense
→ Split Expense
→ View Balances
→ View Settlement Suggestions
→ Record Settlement
→ See Updated Balance
```

The entire flow must work with real PostgreSQL data.

---

# 17. Future Features

Potential future versions:

- recurring expenses
- receipt uploads
- receipt OCR
- email invitations
- WhatsApp sharing
- payment links
- multi-currency
- currency conversion
- recurring bills
- advanced reports
- CSV export
- PDF reports
- mobile PWA
- native React Native app
- AI expense categorization
