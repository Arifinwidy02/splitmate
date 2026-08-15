package dashboard

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/balance"
	"github.com/Arifinwidy02/splitmate-backend/internal/group"
	"github.com/Arifinwidy02/splitmate-backend/pkg/money"
)

const recentExpenseLimit = 10

type groupStore interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*group.Group, error)
}

type balanceStore interface {
	ExpensesWithSplits(ctx context.Context, groupID uuid.UUID) ([]balance.Expense, error)
	Settlements(ctx context.Context, groupID uuid.UUID) ([]balance.Settlement, error)
}

type Service struct {
	repo        Store
	groupStore  groupStore
	balanceRepo balanceStore
}

func NewService(repo Store, groupStore groupStore, balanceRepo balanceStore) *Service {
	return &Service{repo: repo, groupStore: groupStore, balanceRepo: balanceRepo}
}

func (s *Service) GetDashboard(ctx context.Context, userID uuid.UUID) (*Dashboard, error) {
	groups, err := s.groupStore.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	var summary Summary
	var owedToUserSen, userOwesSen, netBalanceSen int64

	groupOverviews := make([]GroupOverview, 0, len(groups))
	for _, g := range groups {
		balances, err := s.calculateGroupBalances(ctx, g.ID)
		if err != nil {
			return nil, err
		}

		b := balances[userID]
		switch {
		case b > 0:
			owedToUserSen += b
			netBalanceSen += b
		case b < 0:
			userOwesSen += -b
			netBalanceSen += b
		}

		groupOverviews = append(groupOverviews, GroupOverview{
			ID:          g.ID,
			Name:        g.Name,
			Currency:    g.Currency,
			MemberCount: g.MemberCount,
			HasLogo:     g.HasLogo,
			Balance:     money.FormatMajor(b),
		})
	}

	totalExpenseSen, err := s.repo.TotalExpense(ctx, userID)
	if err != nil {
		return nil, err
	}

	settledSen, err := s.repo.SettledAmount(ctx, userID)
	if err != nil {
		return nil, err
	}

	recentExpenses, err := s.repo.RecentExpenses(ctx, userID, recentExpenseLimit)
	if err != nil {
		return nil, err
	}

	categories, err := s.repo.CategoryTotals(ctx, userID)
	if err != nil {
		return nil, err
	}

	summary.OwedToUser = money.FormatMajor(owedToUserSen)
	summary.UserOwes = money.FormatMajor(userOwesSen)
	summary.NetBalance = money.FormatMajor(netBalanceSen)
	summary.TotalExpense = money.FormatMajor(totalExpenseSen)
	summary.SettledAmount = money.FormatMajor(settledSen)

	return &Dashboard{
		Summary:        summary,
		Groups:         groupOverviews,
		RecentExpenses: recentExpenses,
		Categories:     categories,
	}, nil
}

func (s *Service) calculateGroupBalances(ctx context.Context, groupID uuid.UUID) (map[uuid.UUID]int64, error) {
	expenses, err := s.balanceRepo.ExpensesWithSplits(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list expenses for dashboard: %w", err)
	}

	settlements, err := s.balanceRepo.Settlements(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list settlements for dashboard: %w", err)
	}

	return balance.CalculateBalances(expenses, settlements), nil
}
