package expense

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/group"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
)

var (
	ErrExpenseNotFound = errors.New("expense not found")
	ErrGroupNotFound   = errors.New("group not found")
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidSplit    = errors.New("invalid split")
	ErrNoReceipt       = errors.New("expense has no receipt")
)

const maxAmountSen = 99_999_999_999_999

type groupStore interface {
	FindMembership(ctx context.Context, groupID, userID uuid.UUID) (*group.Membership, error)
	ListMembers(ctx context.Context, groupID uuid.UUID) ([]*group.Member, error)
	FindByID(ctx context.Context, id uuid.UUID) (*group.Group, error)
}

type Service struct {
	store store
	group groupStore
}

func NewService(store store, groupStore groupStore) *Service {
	return &Service{store: store, group: groupStore}
}

func (s *Service) CreateExpense(ctx context.Context, userID, groupID uuid.UUID, input CreateExpenseInput) (*ExpenseWithSplits, error) {
	if _, err := s.requireMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}

	if err := s.validateInput(ctx, groupID, input); err != nil {
		return nil, err
	}

	splits, err := buildSplits(input)
	if err != nil {
		return nil, err
	}

	e := &Expense{
		GroupID:            groupID,
		Description:        input.Description,
		AmountSen:          input.AmountSen,
		Currency:           input.Currency,
		PaidBy:             input.PaidBy,
		Category:           input.Category,
		ExpenseDate:        input.ExpenseDate,
		Note:               input.Note,
		ReceiptImage:       receiptImage(input),
		ReceiptContentType: receiptContentType(input),
		CreatedBy:          userID,
	}

	created, err := s.store.CreateExpenseWithSplits(ctx, e, splits)
	if err != nil {
		return nil, fmt.Errorf("create expense: %w", err)
	}

	participants, err := s.toParticipants(ctx, splits)
	if err != nil {
		return nil, err
	}

	return &ExpenseWithSplits{Expense: *created, Participants: participants}, nil
}

func (s *Service) ListExpenses(ctx context.Context, userID, groupID uuid.UUID, page, limit int, category string, from, to *time.Time) ([]*ExpenseSummary, int, error) {
	if _, err := s.requireMembership(ctx, groupID, userID); err != nil {
		return nil, 0, err
	}

	summaries, total, err := s.store.ListByGroup(ctx, groupID, page, limit, category, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", err)
	}

	return summaries, total, nil
}

func (s *Service) GetExpense(ctx context.Context, userID, expenseID uuid.UUID) (*ExpenseWithSplits, error) {
	e, participants, err := s.store.FindByID(ctx, expenseID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrExpenseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find expense: %w", err)
	}

	if _, err := s.requireMembership(ctx, e.GroupID, userID); err != nil {
		return nil, err
	}

	return &ExpenseWithSplits{Expense: *e, Participants: participants}, nil
}

func (s *Service) UpdateExpense(ctx context.Context, userID, expenseID uuid.UUID, input CreateExpenseInput) (*ExpenseWithSplits, error) {
	e, participants, err := s.store.FindByID(ctx, expenseID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrExpenseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find expense: %w", err)
	}

	if _, err := s.requireMembership(ctx, e.GroupID, userID); err != nil {
		return nil, err
	}
	if e.CreatedBy != userID {
		return nil, ErrForbidden
	}

	if err := s.validateInput(ctx, e.GroupID, input); err != nil {
		return nil, err
	}

	splits, err := buildSplits(input)
	if err != nil {
		return nil, err
	}

	updated := &Expense{
		ID:                 expenseID,
		GroupID:            e.GroupID,
		Description:        input.Description,
		AmountSen:          input.AmountSen,
		Currency:           input.Currency,
		PaidBy:             input.PaidBy,
		Category:           input.Category,
		ExpenseDate:        input.ExpenseDate,
		Note:               input.Note,
		ReceiptImage:       receiptImage(input),
		ReceiptContentType: receiptContentType(input),
		CreatedBy:          e.CreatedBy,
	}

	if err := s.store.UpdateExpenseWithSplits(ctx, updated, splits); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrExpenseNotFound
		}
		return nil, fmt.Errorf("update expense: %w", err)
	}

	_ = participants

	newParticipants, err := s.toParticipants(ctx, splits)
	if err != nil {
		return nil, err
	}

	updated.CreatedAt = e.CreatedAt
	updated.PayerName = e.PayerName

	return &ExpenseWithSplits{Expense: *updated, Participants: newParticipants}, nil
}

func (s *Service) DeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error {
	e, _, err := s.store.FindByID(ctx, expenseID)
	if errors.Is(err, ErrNotFound) {
		return ErrExpenseNotFound
	}
	if err != nil {
		return fmt.Errorf("find expense: %w", err)
	}

	if _, err := s.requireMembership(ctx, e.GroupID, userID); err != nil {
		return err
	}
	if e.CreatedBy != userID {
		return ErrForbidden
	}

	if err := s.store.Delete(ctx, expenseID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrExpenseNotFound
		}
		return fmt.Errorf("delete expense: %w", err)
	}

	return nil
}

func (s *Service) GetReceipt(ctx context.Context, userID, expenseID uuid.UUID) ([]byte, string, error) {
	e, _, err := s.store.FindByID(ctx, expenseID)
	if errors.Is(err, ErrNotFound) {
		return nil, "", ErrExpenseNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("find expense: %w", err)
	}

	if _, err := s.requireMembership(ctx, e.GroupID, userID); err != nil {
		return nil, "", err
	}

	if len(e.ReceiptImage) == 0 {
		return nil, "", ErrNoReceipt
	}

	return e.ReceiptImage, e.ReceiptContentType, nil
}

func receiptImage(input CreateExpenseInput) []byte {
	if input.Receipt == nil {
		return nil
	}
	return input.Receipt.Image
}

func receiptContentType(input CreateExpenseInput) string {
	if input.Receipt == nil {
		return ""
	}
	return input.Receipt.ContentType
}

func (s *Service) requireMembership(ctx context.Context, groupID, userID uuid.UUID) (*group.Membership, error) {
	m, err := s.group.FindMembership(ctx, groupID, userID)
	if errors.Is(err, group.ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find membership: %w", err)
	}
	return m, nil
}

func (s *Service) validateInput(ctx context.Context, groupID uuid.UUID, input CreateExpenseInput) error {
	input.Description = strings.TrimSpace(input.Description)
	if input.Description == "" || utf8.RuneCountInString(input.Description) > 255 {
		return &apperror.Validation{Message: "Description must be between 1 and 255 characters"}
	}

	if input.AmountSen <= 0 {
		return &apperror.Validation{Message: "Amount must be greater than zero"}
	}
	if input.AmountSen > maxAmountSen {
		return &apperror.Validation{Message: "Amount is too large"}
	}

	g, err := s.group.FindByID(ctx, groupID)
	if errors.Is(err, group.ErrNotFound) {
		return ErrGroupNotFound
	}
	if err != nil {
		return fmt.Errorf("find group: %w", err)
	}

	if input.Currency != g.Currency {
		return &apperror.Validation{Message: "Expense currency must match the group currency"}
	}

	if !isValidCategory(input.Category) {
		return &apperror.Validation{Message: "Unknown expense category"}
	}

	if input.ExpenseDate.IsZero() {
		return &apperror.Validation{Message: "Expense date is required"}
	}

	if input.Note != nil && utf8.RuneCountInString(*input.Note) > maxNoteLen {
		return &apperror.Validation{Message: "Note must be at most 1000 characters"}
	}

	if err := validateReceipt(input.Receipt); err != nil {
		return err
	}

	members, err := s.group.ListMembers(ctx, groupID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}

	memberIDs := map[uuid.UUID]bool{}
	for _, m := range members {
		memberIDs[m.UserID] = true
	}

	if !memberIDs[input.PaidBy] {
		return &apperror.Validation{Message: "Payer must be a group member"}
	}

	if input.SplitType == SplitEqual {
		if len(input.EqualIDs) == 0 {
			return &apperror.Validation{Message: "At least one participant is required"}
		}
		seen := map[uuid.UUID]bool{}
		for _, id := range input.EqualIDs {
			if !memberIDs[id] {
				return &apperror.Validation{Message: "Participants must be group members"}
			}
			if seen[id] {
				return &apperror.Validation{Message: "Participants must be unique"}
			}
			seen[id] = true
		}
		if len(input.EqualIDs) > int(input.AmountSen) {
			return &apperror.Validation{Message: "Each participant must receive at least 0.01"}
		}
		return nil
	}

	if input.SplitType == SplitCustom {
		if len(input.Splits) == 0 {
			return &apperror.Validation{Message: "At least one split is required"}
		}
		seen := map[uuid.UUID]bool{}
		var totalSen int64
		for _, sp := range input.Splits {
			if !memberIDs[sp.UserID] {
				return &apperror.Validation{Message: "Participants must be group members"}
			}
			if seen[sp.UserID] {
				return &apperror.Validation{Message: "Participants must be unique"}
			}
			seen[sp.UserID] = true
			if sp.AmountSen <= 0 {
				return &apperror.Validation{Message: "Each split amount must be greater than zero"}
			}
			totalSen += sp.AmountSen
		}
		if totalSen != input.AmountSen {
			return ErrInvalidSplit
		}
		return nil
	}

	return &apperror.Validation{Message: "splitType must be \"equal\" or \"custom\""}
}

func buildSplits(input CreateExpenseInput) ([]SplitAmount, error) {
	if input.SplitType == SplitEqual {
		return equalSplit(input.AmountSen, input.EqualIDs)
	}

	splits := make([]SplitAmount, 0, len(input.Splits))
	for _, sp := range input.Splits {
		splits = append(splits, SplitAmount{UserID: sp.UserID, AmountSen: sp.AmountSen})
	}
	sort.Slice(splits, func(i, j int) bool {
		return splits[i].UserID.String() < splits[j].UserID.String()
	})
	return splits, nil
}

func equalSplit(amountSen int64, userIDs []uuid.UUID) ([]SplitAmount, error) {
	n := int64(len(userIDs))
	if n == 0 || amountSen < n {
		return nil, ErrInvalidSplit
	}

	base := amountSen / n
	remainder := amountSen % n

	ids := make([]uuid.UUID, len(userIDs))
	copy(ids, userIDs)
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].String() < ids[j].String()
	})

	splits := make([]SplitAmount, 0, len(ids))
	for i, id := range ids {
		amount := base
		if int64(i) < remainder {
			amount++
		}
		splits = append(splits, SplitAmount{UserID: id, AmountSen: amount})
	}

	return splits, nil
}

func (s *Service) toParticipants(ctx context.Context, splits []SplitAmount) ([]Participant, error) {
	participants := make([]Participant, 0, len(splits))
	for _, sp := range splits {
		participants = append(participants, Participant{UserID: sp.UserID, AmountSen: sp.AmountSen})
	}
	return participants, nil
}

func isValidCategory(category string) bool {
	for _, c := range Categories {
		if c == category {
			return true
		}
	}
	return false
}

func validateReceipt(r *Receipt) error {
	if r == nil {
		return nil
	}
	if len(r.Image) == 0 {
		return &apperror.Validation{Message: "Receipt image is empty"}
	}
	if len(r.Image) > maxReceiptBytes {
		return &apperror.Validation{Message: "Receipt image must be at most 5MB"}
	}
	if !receiptContentTypes[r.ContentType] {
		return &apperror.Validation{Message: "Receipt must be a JPEG, PNG, WebP or GIF image"}
	}
	return nil
}
