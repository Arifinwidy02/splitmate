package expense

import (
	"time"

	"github.com/google/uuid"
)

const (
	SplitEqual  = "equal"
	SplitCustom = "custom"

	maxNoteLen = 1000
)

var Categories = []string{
	"Accommodation",
	"Food & Drinks",
	"Transportation",
	"Shopping",
	"Entertainment",
	"Utilities",
	"Other",
}

type Expense struct {
	ID          uuid.UUID
	GroupID     uuid.UUID
	Description string
	AmountSen   int64
	Currency    string
	PaidBy      uuid.UUID
	PayerName   string
	Category    string
	ExpenseDate time.Time
	Note        *string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Participant struct {
	UserID    uuid.UUID
	Name      string
	AmountSen int64
}

type ExpenseWithSplits struct {
	Expense
	Participants []Participant
}

type ExpenseSummary struct {
	Expense
	PayerName        string
	ParticipantCount int
}

type SplitAmount struct {
	UserID    uuid.UUID
	AmountSen int64
}

type CreateExpenseInput struct {
	Description string
	AmountSen   int64
	Currency    string
	PaidBy      uuid.UUID
	Category    string
	ExpenseDate time.Time
	Note        *string
	SplitType   string
	EqualIDs    []uuid.UUID
	Splits      []SplitAmount
}
