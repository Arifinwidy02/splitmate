package expense

import (
	"time"

	"github.com/google/uuid"
)

const (
	SplitEqual  = "equal"
	SplitCustom = "custom"

	maxNoteLen        = 1000
	maxReceiptBytes   = 5 << 20 // 5MB
	receiptFieldLimit = 10 << 20
)

var (
	Categories = []string{
		"Accommodation",
		"Food & Drinks",
		"Transportation",
		"Shopping",
		"Entertainment",
		"Utilities",
		"Other",
	}

	receiptContentTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
)

type Expense struct {
	ID                 uuid.UUID
	GroupID            uuid.UUID
	Description        string
	AmountSen          int64
	Currency           string
	PaidBy             uuid.UUID
	PayerName          string
	Category           string
	ExpenseDate        time.Time
	Note               *string
	ReceiptImage       []byte
	ReceiptContentType string
	CreatedBy          uuid.UUID
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
	HasReceipt       bool
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
	Receipt     *Receipt
}

type Receipt struct {
	Image       []byte
	ContentType string
}
