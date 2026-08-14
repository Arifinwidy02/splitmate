package balance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arifinwidy02/splitmate-backend/pkg/money"
)

type store interface {
	ExpensesWithSplits(ctx context.Context, groupID uuid.UUID) ([]Expense, error)
	Settlements(ctx context.Context, groupID uuid.UUID) ([]Settlement, error)
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ExpensesWithSplits(ctx context.Context, groupID uuid.UUID) ([]Expense, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT e.id::text, e.paid_by::text, e.amount::text
		 FROM expenses e
		 WHERE e.group_id = $1
		 ORDER BY e.created_at`,
		groupID.String())
	if err != nil {
		return nil, fmt.Errorf("list expenses for balance: %w", err)
	}
	defer rows.Close()

	expenses := []Expense{}
	expenseIDs := []string{}
	expenseIndex := map[string]int{}

	for rows.Next() {
		var (
			rawExpenseID string
			rawPaidBy    string
			rawAmount    string
		)

		if err := rows.Scan(&rawExpenseID, &rawPaidBy, &rawAmount); err != nil {
			return nil, fmt.Errorf("scan expense for balance: %w", err)
		}

		paidBy, err := uuid.Parse(rawPaidBy)
		if err != nil {
			return nil, fmt.Errorf("parse paid by: %w", err)
		}
		amountSen, err := money.ParseMajor(rawAmount)
		if err != nil {
			return nil, fmt.Errorf("parse expense amount: %w", err)
		}

		expenseIndex[rawExpenseID] = len(expenseIDs)
		expenseIDs = append(expenseIDs, rawExpenseID)
		expenses = append(expenses, Expense{PaidBy: paidBy, AmountSen: amountSen})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list expenses for balance: %w", err)
	}

	splitsByExpense := map[string][]Split{}

	splitRows, err := r.pool.Query(ctx,
		`SELECT es.expense_id::text, es.user_id::text, es.amount::text
		 FROM expense_splits es
		 JOIN expenses e ON e.id = es.expense_id
		 WHERE e.group_id = $1`,
		groupID.String())
	if err != nil {
		return nil, fmt.Errorf("list splits for balance: %w", err)
	}
	defer splitRows.Close()

	for splitRows.Next() {
		var (
			rawExpenseID string
			rawUserID    string
			rawAmount    string
		)

		if err := splitRows.Scan(&rawExpenseID, &rawUserID, &rawAmount); err != nil {
			return nil, fmt.Errorf("scan split for balance: %w", err)
		}

		userID, err := uuid.Parse(rawUserID)
		if err != nil {
			return nil, fmt.Errorf("parse split user id: %w", err)
		}
		amountSen, err := money.ParseMajor(rawAmount)
		if err != nil {
			return nil, fmt.Errorf("parse split amount: %w", err)
		}

		splitsByExpense[rawExpenseID] = append(splitsByExpense[rawExpenseID], Split{UserID: userID, AmountSen: amountSen})
	}
	if err := splitRows.Err(); err != nil {
		return nil, fmt.Errorf("list splits for balance: %w", err)
	}

	for i, id := range expenseIDs {
		expenses[i].Splits = splitsByExpense[id]
	}

	return expenses, nil
}

func (r *Repository) Settlements(ctx context.Context, groupID uuid.UUID) ([]Settlement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.payer_id::text, s.receiver_id::text, s.amount::text
		 FROM settlements s
		 WHERE s.group_id = $1
		 ORDER BY s.created_at`,
		groupID.String())
	if err != nil {
		return nil, fmt.Errorf("list settlements for balance: %w", err)
	}
	defer rows.Close()

	settlements := []Settlement{}
	for rows.Next() {
		var (
			rawPayerID    string
			rawReceiverID string
			rawAmount     string
		)

		if err := rows.Scan(&rawPayerID, &rawReceiverID, &rawAmount); err != nil {
			return nil, fmt.Errorf("scan settlement for balance: %w", err)
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

		settlements = append(settlements, Settlement{PayerID: payerID, ReceiverID: receiverID, AmountSen: amountSen})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list settlements for balance: %w", err)
	}

	return settlements, nil
}
