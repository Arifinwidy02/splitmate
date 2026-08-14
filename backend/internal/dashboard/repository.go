package dashboard

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arifinwidy02/splitmate-backend/pkg/money"
)

type Store interface {
	TotalExpense(ctx context.Context, userID uuid.UUID) (int64, error)
	SettledAmount(ctx context.Context, userID uuid.UUID) (int64, error)
	RecentExpenses(ctx context.Context, userID uuid.UUID, limit int) ([]RecentExpense, error)
	CategoryTotals(ctx context.Context, userID uuid.UUID) ([]CategoryTotal, error)
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) TotalExpense(ctx context.Context, userID uuid.UUID) (int64, error) {
	var rawTotal string

	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(e.amount), 0)::text
		 FROM expenses e
		 JOIN group_members gm ON gm.group_id = e.group_id
		 WHERE gm.user_id = $1`,
		userID.String()).Scan(&rawTotal)
	if err != nil {
		return 0, fmt.Errorf("sum total expense: %w", err)
	}

	totalSen, err := money.ParseMajor(rawTotal)
	if err != nil {
		return 0, fmt.Errorf("parse total expense: %w", err)
	}

	return totalSen, nil
}

func (r *Repository) SettledAmount(ctx context.Context, userID uuid.UUID) (int64, error) {
	var rawTotal string

	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(s.amount), 0)::text
		 FROM settlements s
		 WHERE s.payer_id = $1 OR s.receiver_id = $1`,
		userID.String()).Scan(&rawTotal)
	if err != nil {
		return 0, fmt.Errorf("sum settled amount: %w", err)
	}

	totalSen, err := money.ParseMajor(rawTotal)
	if err != nil {
		return 0, fmt.Errorf("parse settled amount: %w", err)
	}

	return totalSen, nil
}

func (r *Repository) RecentExpenses(ctx context.Context, userID uuid.UUID, limit int) ([]RecentExpense, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT e.id::text, e.group_id::text, g.name, e.description, u.name, e.amount::text, e.category, e.expense_date,
		        (SELECT count(*) FROM expense_splits es WHERE es.expense_id = e.id)
		 FROM expenses e
		 JOIN groups g ON g.id = e.group_id
		 JOIN users u ON u.id = e.paid_by
		 JOIN group_members gm ON gm.group_id = e.group_id
		 WHERE gm.user_id = $1
		 ORDER BY e.expense_date DESC, e.created_at DESC
		 LIMIT $2`,
		userID.String(), limit)
	if err != nil {
		return nil, fmt.Errorf("list recent expenses: %w", err)
	}
	defer rows.Close()

	expenses := []RecentExpense{}
	for rows.Next() {
		var (
			rawExpenseID string
			rawGroupID   string
			rawAmount    string
			expense      RecentExpense
		)

		if err := rows.Scan(
			&rawExpenseID, &rawGroupID, &expense.GroupName, &expense.Description, &expense.PayerName,
			&rawAmount, &expense.Category, &expense.ExpenseDate, &expense.ParticipantCount,
		); err != nil {
			return nil, fmt.Errorf("scan recent expense: %w", err)
		}

		expense.ID, err = uuid.Parse(rawExpenseID)
		if err != nil {
			return nil, fmt.Errorf("parse recent expense id: %w", err)
		}
		expense.GroupID, err = uuid.Parse(rawGroupID)
		if err != nil {
			return nil, fmt.Errorf("parse recent expense group id: %w", err)
		}
		amountSen, err := money.ParseMajor(rawAmount)
		if err != nil {
			return nil, fmt.Errorf("parse recent expense amount: %w", err)
		}
		expense.Amount = money.FormatMajor(amountSen)

		expenses = append(expenses, expense)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list recent expenses: %w", err)
	}

	return expenses, nil
}

func (r *Repository) CategoryTotals(ctx context.Context, userID uuid.UUID) ([]CategoryTotal, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT e.category, SUM(e.amount)::text
		 FROM expenses e
		 JOIN group_members gm ON gm.group_id = e.group_id
		 WHERE gm.user_id = $1
		 GROUP BY e.category
		 ORDER BY SUM(e.amount) DESC`,
		userID.String())
	if err != nil {
		return nil, fmt.Errorf("list category totals: %w", err)
	}
	defer rows.Close()

	categories := []CategoryTotal{}
	for rows.Next() {
		var (
			rawTotal string
			category CategoryTotal
		)

		if err := rows.Scan(&category.Category, &rawTotal); err != nil {
			return nil, fmt.Errorf("scan category total: %w", err)
		}

		totalSen, err := money.ParseMajor(rawTotal)
		if err != nil {
			return nil, fmt.Errorf("parse category total: %w", err)
		}
		category.Total = money.FormatMajor(totalSen)

		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list category totals: %w", err)
	}

	return categories, nil
}
