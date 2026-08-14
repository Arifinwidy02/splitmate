package settlement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/group"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
)

var (
	ErrGroupNotFound = errors.New("group not found")
	ErrForbidden     = errors.New("forbidden")
)

const maxAmountSen = int64(99_999_999_999_999)

type groupStore interface {
	FindMembership(ctx context.Context, groupID, userID uuid.UUID) (*group.Membership, error)
}

type Service struct {
	repo       store
	groupStore groupStore
}

func NewService(repo store, groupStore groupStore) *Service {
	return &Service{repo: repo, groupStore: groupStore}
}

// CreateSettlement records a repayment. Only the payer can record the
// settlement (a user cannot settle another user's debt).
func (s *Service) CreateSettlement(ctx context.Context, userID, groupID uuid.UUID, input CreateSettlementInput) (*Settlement, error) {
	if _, err := s.groupStore.FindMembership(ctx, groupID, userID); err != nil {
		return nil, ErrGroupNotFound
	}

	if input.PayerID != userID {
		return nil, ErrForbidden
	}

	if input.PayerID == input.ReceiverID {
		return nil, apperror.NewValidation("Payer and receiver must be different")
	}
	if input.AmountSen <= 0 {
		return nil, apperror.NewValidation("Settlement amount must be greater than zero")
	}
	if input.AmountSen > maxAmountSen {
		return nil, apperror.NewValidation("Settlement amount is too large")
	}

	if _, err := s.groupStore.FindMembership(ctx, groupID, input.ReceiverID); err != nil {
		return nil, apperror.NewValidation("Receiver must be a group member")
	}

	settledAt := input.SettledAt
	if settledAt.IsZero() {
		settledAt = time.Now()
	}

	st := &Settlement{
		GroupID:    groupID,
		PayerID:    input.PayerID,
		ReceiverID: input.ReceiverID,
		AmountSen:  input.AmountSen,
		SettledAt:  settledAt,
	}

	if err := s.repo.Create(ctx, st); err != nil {
		return nil, fmt.Errorf("create settlement: %w", err)
	}

	return st, nil
}

func (s *Service) ListSettlements(ctx context.Context, userID, groupID uuid.UUID) ([]*Settlement, error) {
	if _, err := s.groupStore.FindMembership(ctx, groupID, userID); err != nil {
		return nil, ErrGroupNotFound
	}

	settlements, err := s.repo.ListByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	return settlements, nil
}
