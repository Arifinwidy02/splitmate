package report

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arifinwidy02/splitmate-backend/pkg/money"
)

var ErrNotFound = errors.New("not found")

type Group struct {
	Name     string
	Currency string
}

type MemberRow struct {
	UserID uuid.UUID
	Name   string
}

type store interface {
	FindGroup(ctx context.Context, groupID uuid.UUID) (*Group, error)
	FindMembership(ctx context.Context, groupID, userID uuid.UUID) error
	ExpensesWithSplits(ctx context.Context, groupID uuid.UUID) ([]ExpenseRow, error)
	Settlements(ctx context.Context, groupID uuid.UUID) ([]SettlementRow, error)
	Members(ctx context.Context, groupID uuid.UUID) ([]MemberRow, error)
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) FindGroup(ctx context.Context, groupID uuid.UUID) (*Group, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT name, currency FROM groups WHERE id = $1`,
		groupID.String())

	var g Group
	if err := row.Scan(&g.Name, &g.Currency); err != nil {
		return nil, fmt.Errorf("find group for report: %w", err)
	}
	return &g, nil
}

func (r *Repository) FindMembership(ctx context.Context, groupID, userID uuid.UUID) error {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2
		 )`,
		groupID.String(), userID.String()).Scan(&exists)
	if err != nil {
		return fmt.Errorf("find membership for report: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ExpensesWithSplits(ctx context.Context, groupID uuid.UUID) ([]ExpenseRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT e.id::text, e.description, e.category, e.paid_by::text, u.name, e.expense_date, e.amount::text, COALESCE(e.note, '')
		 FROM expenses e
		 JOIN users u ON u.id = e.paid_by
		 WHERE e.group_id = $1
		 ORDER BY e.expense_date DESC, e.created_at DESC`,
		groupID.String())
	if err != nil {
		return nil, fmt.Errorf("list expenses for report: %w", err)
	}
	defer rows.Close()

	expenses := []ExpenseRow{}
	expenseIndex := map[string]int{}

	for rows.Next() {
		var (
			row          ExpenseRow
			rawExpenseID string
			rawPaidBy    string
			rawAmount    string
		)

		if err := rows.Scan(&rawExpenseID, &row.Description, &row.Category, &rawPaidBy, &row.PaidByName, &row.ExpenseDate, &rawAmount, &row.Note); err != nil {
			return nil, fmt.Errorf("scan expense for report: %w", err)
		}

		id, err := uuid.Parse(rawExpenseID)
		if err != nil {
			return nil, fmt.Errorf("parse expense id: %w", err)
		}
		paidBy, err := uuid.Parse(rawPaidBy)
		if err != nil {
			return nil, fmt.Errorf("parse expense paid by: %w", err)
		}
		amountSen, err := money.ParseMajor(rawAmount)
		if err != nil {
			return nil, fmt.Errorf("parse expense amount: %w", err)
		}

		row.ID = id
		row.PaidBy = paidBy
		row.AmountSen = amountSen
		expenseIndex[rawExpenseID] = len(expenses)
		expenses = append(expenses, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list expenses for report: %w", err)
	}

	splitRows, err := r.pool.Query(ctx,
		`SELECT es.expense_id::text, es.user_id::text, u.name, es.amount::text
		 FROM expense_splits es
		 JOIN users u ON u.id = es.user_id
		 JOIN expenses e ON e.id = es.expense_id
		 WHERE e.group_id = $1
		 ORDER BY es.expense_id, u.name`,
		groupID.String())
	if err != nil {
		return nil, fmt.Errorf("list splits for report: %w", err)
	}
	defer splitRows.Close()

	for splitRows.Next() {
		var (
			rawExpenseID string
			rawUserID    string
			rawAmount    string
			participant  ParticipantRow
		)

		if err := splitRows.Scan(&rawExpenseID, &rawUserID, &participant.Name, &rawAmount); err != nil {
			return nil, fmt.Errorf("scan split for report: %w", err)
		}

		userID, err := uuid.Parse(rawUserID)
		if err != nil {
			return nil, fmt.Errorf("parse split user id: %w", err)
		}
		amountSen, err := money.ParseMajor(rawAmount)
		if err != nil {
			return nil, fmt.Errorf("parse split amount: %w", err)
		}
		participant.UserID = userID
		participant.AmountSen = amountSen

		idx, ok := expenseIndex[rawExpenseID]
		if !ok {
			continue
		}
		expenses[idx].Participants = append(expenses[idx].Participants, participant)
	}
	if err := splitRows.Err(); err != nil {
		return nil, fmt.Errorf("list splits for report: %w", err)
	}

	return expenses, nil
}

func (r *Repository) Settlements(ctx context.Context, groupID uuid.UUID) ([]SettlementRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.payer_id::text, s.receiver_id::text, pu.name, ru.name, s.amount::text, s.settled_at
		 FROM settlements s
		 JOIN users pu ON pu.id = s.payer_id
		 JOIN users ru ON ru.id = s.receiver_id
		 WHERE s.group_id = $1
		 ORDER BY s.settled_at DESC, s.created_at DESC`,
		groupID.String())
	if err != nil {
		return nil, fmt.Errorf("list settlements for report: %w", err)
	}
	defer rows.Close()

	settlements := []SettlementRow{}
	for rows.Next() {
		var (
			row           SettlementRow
			rawPayerID    string
			rawReceiverID string
			rawAmount     string
		)

		if err := rows.Scan(&rawPayerID, &rawReceiverID, &row.PayerName, &row.ReceiverName, &rawAmount, &row.SettledAt); err != nil {
			return nil, fmt.Errorf("scan settlement for report: %w", err)
		}

		payerID, err := uuid.Parse(rawPayerID)
		if err != nil {
			return nil, fmt.Errorf("parse settlement payer id: %w", err)
		}
		receiverID, err := uuid.Parse(rawReceiverID)
		if err != nil {
			return nil, fmt.Errorf("parse settlement receiver id: %w", err)
		}
		amountSen, err := money.ParseMajor(rawAmount)
		if err != nil {
			return nil, fmt.Errorf("parse settlement amount: %w", err)
		}
		row.PayerID = payerID
		row.ReceiverID = receiverID
		row.AmountSen = amountSen
		settlements = append(settlements, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list settlements for report: %w", err)
	}

	return settlements, nil
}

func (r *Repository) Members(ctx context.Context, groupID uuid.UUID) ([]MemberRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id::text, u.name
		 FROM group_members gm
		 JOIN users u ON u.id = gm.user_id
		 WHERE gm.group_id = $1
		 ORDER BY gm.joined_at`,
		groupID.String())
	if err != nil {
		return nil, fmt.Errorf("list members for report: %w", err)
	}
	defer rows.Close()

	members := []MemberRow{}
	for rows.Next() {
		var (
			row   MemberRow
			rawID string
		)

		if err := rows.Scan(&rawID, &row.Name); err != nil {
			return nil, fmt.Errorf("scan member for report: %w", err)
		}

		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("parse member id: %w", err)
		}
		row.UserID = id
		members = append(members, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list members for report: %w", err)
	}

	return members, nil
}
