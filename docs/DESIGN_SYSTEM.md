# SplitMate — Design System

## 1. Design Direction

SplitMate should feel:

- modern
- trustworthy
- lightweight
- friendly
- financial but not corporate
- data-rich without feeling overwhelming

Reference direction:

**modern fintech dashboard + friendly productivity app**

---

# 2. Visual Principles

## Whitespace

Use generous spacing.

Avoid dense dashboards.

## Hierarchy

Important financial numbers should immediately stand out.

## Cards

Use:

- rounded corners
- subtle border
- very soft shadow
- white surfaces

## Status

Positive and negative balances should be visually obvious.

Positive:

```text
You are owed
+Rp430.000
```

Negative:

```text
You owe
-Rp320.000
```

---

# 3. Colors

Primary:

```text
Green
#16A34A
```

Dark:

```text
#111827
```

Muted text:

```text
#64748B
```

Background:

```text
#F8FAFC
```

Surface:

```text
#FFFFFF
```

Border:

```text
#E2E8F0
```

Positive:

```text
#16A34A
```

Negative:

```text
#EF4444
```

Warning:

```text
#F59E0B
```

Info:

```text
#3B82F6
```

The implementation may adjust exact values slightly if accessibility or contrast requires it.

---

# 4. Typography

Recommended font:

```text
Inter
```

Weights:

```text
400 Regular
500 Medium
600 Semibold
700 Bold
```

Dashboard heading:

```text
32px / 700
```

Section heading:

```text
20px / 600
```

Card amount:

```text
28px / 700
```

Body:

```text
14–16px
```

---

# 5. Layout

Desktop:

```text
Sidebar
240px

Main content
flexible
```

Main content max width:

```text
1440px
```

Page padding:

```text
24–32px
```

---

# 6. Navigation

Sidebar items:

```text
Dashboard
Groups
Expenses
Settlements
Reports
```

Bottom:

```text
Settings
Help & Feedback
```

Active navigation:

- light green background
- green icon
- green text

---

# 7. Dashboard Layout

```text
┌───────────────┬───────────────────────────────┐
│               │ Header                        │
│   Sidebar     ├───────────────────────────────┤
│               │ Welcome                        │
│               │                               │
│               │ Summary Cards                 │
│               │                               │
│               │ Recent Groups | Expenses      │
│               │                               │
│               │ Balance | Categories          │
└───────────────┴───────────────────────────────┘
```

---

# 8. Components

Required reusable components:

```text
Button
IconButton
Input
Select
Dialog
Dropdown
Avatar
Badge
Card
Tabs
Table
EmptyState
Skeleton
Toast
Tooltip
DatePicker
CurrencyInput
ExpenseCard
GroupCard
BalanceCard
```

---

# 9. Expense Form

The Add Expense form should be optimized for speed.

Fields:

```text
Description
Amount
Paid by
Split between
Split method
Category
Date
Note
```

Primary CTA:

```text
Add Expense
```

The user should be able to complete a normal equal split in a few interactions.

---

# 10. Responsive Behavior

Desktop:

```text
sidebar visible
multi-column dashboard
```

Tablet:

```text
collapsible sidebar
two-column cards
```

Mobile:

```text
bottom navigation or compact drawer
single-column layout
floating/add button
```

---

# 11. Accessibility

Requirements:

- keyboard navigable
- visible focus states
- semantic HTML
- proper labels
- accessible dialogs
- sufficient color contrast
- do not communicate status using color alone

---

# 12. Financial Formatting

IDR:

```text
Rp750.000
Rp1.250.000
Rp8.450.000
```

Do not display:

```text
750000
```

Date formatting should follow the user's locale.

---

# 13. Loading States

Never leave blank screens.

Use:

- skeleton cards
- skeleton list items
- button loading state

Example:

```text
[ Saving... ]
```

---

# 14. Empty States

Examples:

```text
No groups yet.

Create your first group to start sharing expenses.
```

and:

```text
No expenses yet.

Add your first expense to start tracking.
```

---

# 15. Error UX

Errors should be understandable.

Bad:

```text
500 Internal Server Error
```

Better:

```text
Something went wrong.
We couldn't save this expense. Please try again.
```

---

# 16. Dashboard Visual Language

Use icons for:

- accommodation
- food
- transportation
- shopping
- entertainment
- utilities
- other

Charts should remain simple.

Avoid excessive gradients, 3D charts, or visual noise.
