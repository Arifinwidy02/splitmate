package dashboard

import (
	"time"

	"github.com/google/uuid"
)

type Summary struct {
	OwedToUser    string `json:"owedToUser"`
	UserOwes      string `json:"userOwes"`
	NetBalance    string `json:"netBalance"`
	TotalExpense  string `json:"totalExpense"`
	SettledAmount string `json:"settledAmount"`
}

type GroupOverview struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Currency    string    `json:"currency"`
	MemberCount int       `json:"memberCount"`
	Balance     string    `json:"balance"`
}

type RecentExpense struct {
	ID               uuid.UUID `json:"id"`
	GroupID          uuid.UUID `json:"groupId"`
	GroupName        string    `json:"groupName"`
	Description      string    `json:"description"`
	PayerName        string    `json:"payerName"`
	Amount           string    `json:"amount"`
	Category         string    `json:"category"`
	ExpenseDate      time.Time `json:"expenseDate"`
	ParticipantCount int       `json:"participantCount"`
}

type CategoryTotal struct {
	Category string `json:"category"`
	Total    string `json:"total"`
}

type Dashboard struct {
	Summary        Summary         `json:"summary"`
	Groups         []GroupOverview `json:"groups"`
	RecentExpenses []RecentExpense `json:"recentExpenses"`
	Categories     []CategoryTotal `json:"categories"`
}
