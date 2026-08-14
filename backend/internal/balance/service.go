package balance

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/group"
)

var (
	ErrGroupNotFound = errors.New("group not found")
)

type groupStore interface {
	FindMembership(ctx context.Context, groupID, userID uuid.UUID) (*group.Membership, error)
	ListMembers(ctx context.Context, groupID uuid.UUID) ([]*group.Member, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*group.Group, error)
}

type Service struct {
	repo       store
	groupStore groupStore
}

func NewService(repo store, groupStore groupStore) *Service {
	return &Service{repo: repo, groupStore: groupStore}
}

type MemberBalance struct {
	UserID     uuid.UUID
	Name       string
	BalanceSen int64
}

type PersonalSummary struct {
	OwedToUserSen int64
	UserOwesSen   int64
	NetBalanceSen int64
}

func (s *Service) GroupBalances(ctx context.Context, userID, groupID uuid.UUID) ([]MemberBalance, error) {
	if _, err := s.groupStore.FindMembership(ctx, groupID, userID); err != nil {
		return nil, ErrGroupNotFound
	}

	balances, err := s.calculateGroupBalances(ctx, groupID)
	if err != nil {
		return nil, err
	}

	members, err := s.groupStore.ListMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	result := make([]MemberBalance, 0, len(members))
	for _, m := range members {
		result = append(result, MemberBalance{
			UserID:     m.UserID,
			Name:       m.Name,
			BalanceSen: balances[m.UserID],
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].UserID.String() < result[j].UserID.String()
	})

	return result, nil
}

func (s *Service) SettlementSuggestions(ctx context.Context, userID, groupID uuid.UUID) ([]Transfer, error) {
	if _, err := s.groupStore.FindMembership(ctx, groupID, userID); err != nil {
		return nil, ErrGroupNotFound
	}

	balances, err := s.calculateGroupBalances(ctx, groupID)
	if err != nil {
		return nil, err
	}

	return SimplifyDebts(balances), nil
}

func (s *Service) PersonalBalance(ctx context.Context, userID uuid.UUID) (PersonalSummary, error) {
	groups, err := s.groupStore.ListByUserID(ctx, userID)
	if err != nil {
		return PersonalSummary{}, fmt.Errorf("list groups: %w", err)
	}

	var summary PersonalSummary
	for _, g := range groups {
		balances, err := s.calculateGroupBalances(ctx, g.ID)
		if err != nil {
			return PersonalSummary{}, err
		}

		switch b := balances[userID]; {
		case b > 0:
			summary.OwedToUserSen += b
			summary.NetBalanceSen += b
		case b < 0:
			summary.UserOwesSen += -b
			summary.NetBalanceSen += b
		}
	}

	return summary, nil
}

func (s *Service) calculateGroupBalances(ctx context.Context, groupID uuid.UUID) (map[uuid.UUID]int64, error) {
	expenses, err := s.repo.ExpensesWithSplits(ctx, groupID)
	if err != nil {
		return nil, err
	}

	settlements, err := s.repo.Settlements(ctx, groupID)
	if err != nil {
		return nil, err
	}

	return CalculateBalances(expenses, settlements), nil
}
