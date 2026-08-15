package report

import (
	"time"

	"github.com/google/uuid"
)

// ExpenseRow is one expense with its splits, ready for export.
type ExpenseRow struct {
	ID           uuid.UUID
	Description  string
	Category     string
	PaidBy       uuid.UUID
	PaidByName   string
	ExpenseDate  time.Time
	AmountSen    int64
	Note         string
	Participants []ParticipantRow
}

// ParticipantRow is a single member's share of an expense.
type ParticipantRow struct {
	UserID    uuid.UUID
	Name      string
	AmountSen int64
}

// BalanceRow is a member's net balance in the group.
type BalanceRow struct {
	UserID     uuid.UUID
	Name       string
	BalanceSen int64
}

// SettlementRow is one recorded repayment.
type SettlementRow struct {
	PayerID      uuid.UUID
	ReceiverID   uuid.UUID
	PayerName    string
	ReceiverName string
	AmountSen    int64
	SettledAt    time.Time
}

// SuggestionRow is one suggested transfer.
type SuggestionRow struct {
	FromName  string
	ToName    string
	AmountSen int64
}

// Report is the complete data needed to render the Excel workbook.
type Report struct {
	GroupName     string
	GroupCurrency string
	GeneratedAt   time.Time
	Balances      []BalanceRow
	Suggestions   []SuggestionRow
	Expenses      []ExpenseRow
	Settlements   []SettlementRow
}
