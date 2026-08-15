package report

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/balance"
)

var ErrGroupNotFound = errors.New("group not found")

type Service struct {
	repo store
}

func NewService(repo store) *Service {
	return &Service{repo: repo}
}

// BuildReport assembles everything a group report needs: member balances
// (computed by the balance engine — never trusted from the client), settlement
// suggestions, the full expense list with splits, and settlement history.
func (s *Service) BuildReport(ctx context.Context, userID, groupID uuid.UUID) (*Report, error) {
	if err := s.repo.FindMembership(ctx, groupID, userID); err != nil {
		return nil, ErrGroupNotFound
	}

	group, err := s.repo.FindGroup(ctx, groupID)
	if err != nil {
		return nil, ErrGroupNotFound
	}

	expenses, err := s.repo.ExpensesWithSplits(ctx, groupID)
	if err != nil {
		return nil, err
	}

	settlements, err := s.repo.Settlements(ctx, groupID)
	if err != nil {
		return nil, err
	}

	members, err := s.repo.Members(ctx, groupID)
	if err != nil {
		return nil, err
	}

	balances := calculateBalances(expenses, settlements)
	suggestions := buildSuggestions(balances, members)
	sortBalances(balances)

	return &Report{
		GroupName:     group.Name,
		GroupCurrency: group.Currency,
		GeneratedAt:   time.Now().UTC(),
		Balances:      balances,
		Suggestions:   suggestions,
		Expenses:      expenses,
		Settlements:   settlements,
	}, nil
}

// calculateBalances delegates to the balance engine so the exported numbers
// always match the group balances shown in the app.
func calculateBalances(expenses []ExpenseRow, settlements []SettlementRow) []BalanceRow {
	engineExpenses := make([]balance.Expense, 0, len(expenses))
	for _, e := range expenses {
		engineExpenses = append(engineExpenses, balance.Expense{
			PaidBy:    e.PaidBy,
			AmountSen: e.AmountSen,
			Splits:    toEngineSplits(e.Participants),
		})
	}

	engineSettlements := make([]balance.Settlement, 0, len(settlements))
	for _, st := range settlements {
		engineSettlements = append(engineSettlements, balance.Settlement{
			PayerID:    st.PayerID,
			ReceiverID: st.ReceiverID,
			AmountSen:  st.AmountSen,
		})
	}

	raw := balance.CalculateBalances(engineExpenses, engineSettlements)

	balances := make([]BalanceRow, 0, len(raw))
	for userID, sen := range raw {
		balances = append(balances, BalanceRow{UserID: userID, BalanceSen: sen})
	}
	return balances
}

func toEngineSplits(participants []ParticipantRow) []balance.Split {
	splits := make([]balance.Split, 0, len(participants))
	for _, p := range participants {
		splits = append(splits, balance.Split{
			UserID:    p.UserID,
			AmountSen: p.AmountSen,
		})
	}
	return splits
}

func buildSuggestions(balances []BalanceRow, members []MemberRow) []SuggestionRow {
	raw := make(map[uuid.UUID]int64, len(balances))
	for _, b := range balances {
		raw[b.UserID] = b.BalanceSen
	}

	nameByID := make(map[uuid.UUID]string, len(members))
	for _, m := range members {
		nameByID[m.UserID] = m.Name
	}

	transfers := balance.SimplifyDebts(raw)
	suggestions := make([]SuggestionRow, 0, len(transfers))
	for _, t := range transfers {
		suggestions = append(suggestions, SuggestionRow{
			FromName:  nameByID[t.FromUserID],
			ToName:    nameByID[t.ToUserID],
			AmountSen: t.AmountSen,
		})
	}
	return suggestions
}

func sortBalances(balances []BalanceRow) {
	sort.Slice(balances, func(i, j int) bool {
		if balances[i].Name != balances[j].Name {
			return balances[i].Name < balances[j].Name
		}
		return balances[i].UserID.String() < balances[j].UserID.String()
	})
}
