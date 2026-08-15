package expense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arifinwidy02/splitmate-backend/pkg/money"
)

var (
	ErrNotFound = errors.New("not found")
)

type store interface {
	CreateExpenseWithSplits(ctx context.Context, e *Expense, splits []SplitAmount) (*Expense, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Expense, []Participant, error)
	UpdateExpenseWithSplits(ctx context.Context, e *Expense, splits []SplitAmount) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByGroup(ctx context.Context, groupID uuid.UUID, page, limit int, category string, from, to *time.Time) ([]*ExpenseSummary, int, error)
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const expenseColumns = "e.id::text, e.group_id::text, e.description, e.amount::text, e.currency, e.paid_by::text, e.category, e.expense_date, e.note, e.created_by::text, e.created_at, e.updated_at"

const expenseDetailColumns = expenseColumns + ", e.receipt_image, e.receipt_content_type"

func scanExpenseRow(row pgx.Row) (*Expense, error) {
	var (
		e          Expense
		rawID      string
		rawGroup   string
		rawAmount  string
		rawPaidBy  string
		rawCreated string
	)

	if err := row.Scan(&rawID, &rawGroup, &e.Description, &rawAmount, &e.Currency, &rawPaidBy, &e.Category, &e.ExpenseDate, &e.Note, &rawCreated, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}

	amountSen, err := money.ParseMajor(rawAmount)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("parse expense id: %w", err)
	}
	groupID, err := uuid.Parse(rawGroup)
	if err != nil {
		return nil, fmt.Errorf("parse group id: %w", err)
	}
	paidBy, err := uuid.Parse(rawPaidBy)
	if err != nil {
		return nil, fmt.Errorf("parse paid by: %w", err)
	}
	createdBy, err := uuid.Parse(rawCreated)
	if err != nil {
		return nil, fmt.Errorf("parse created by: %w", err)
	}

	e.ID = id
	e.GroupID = groupID
	e.AmountSen = amountSen
	e.PaidBy = paidBy
	e.CreatedBy = createdBy

	return &e, nil
}

func (r *Repository) CreateExpenseWithSplits(ctx context.Context, e *Expense, splits []SplitAmount) (*Expense, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx,
		`INSERT INTO expenses AS e (group_id, description, amount, currency, paid_by, category, expense_date, note, receipt_image, receipt_content_type, created_by)
		 VALUES ($1, $2, $3::numeric, $4, $5, $6, $7, $8, NULLIF($9, ''::bytea), NULLIF($10, ''), $11)
		 RETURNING `+expenseColumns,
		e.GroupID.String(), e.Description, money.FormatMajor(e.AmountSen), e.Currency, e.PaidBy.String(), e.Category, e.ExpenseDate, e.Note, e.ReceiptImage, e.ReceiptContentType, e.CreatedBy.String())

	created, err := scanExpenseRow(row)
	if err != nil {
		return nil, fmt.Errorf("insert expense: %w", err)
	}

	if err := insertSplits(ctx, tx, created.ID, splits); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Expense, []Participant, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+expenseDetailColumns+`, u.name
		 FROM expenses e
		 JOIN users u ON u.id = e.paid_by
		 WHERE e.id = $1`,
		id.String())

	var (
		payerName          string
		rawID              string
		rawGroup           string
		rawAmount          string
		rawPaidBy          string
		rawCreate          string
		receiptImage       []byte
		receiptContentType *string
		e                  Expense
	)

	if err := row.Scan(&rawID, &rawGroup, &e.Description, &rawAmount, &e.Currency, &rawPaidBy, &e.Category, &e.ExpenseDate, &e.Note, &rawCreate, &e.CreatedAt, &e.UpdatedAt, &receiptImage, &receiptContentType, &payerName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("find expense: %w", err)
	}
	if receiptContentType != nil {
		e.ReceiptContentType = *receiptContentType
	}

	amountSen, err := money.ParseMajor(rawAmount)
	if err != nil {
		return nil, nil, fmt.Errorf("parse amount: %w", err)
	}
	expenseID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse expense id: %w", err)
	}
	groupID, err := uuid.Parse(rawGroup)
	if err != nil {
		return nil, nil, fmt.Errorf("parse group id: %w", err)
	}
	paidBy, err := uuid.Parse(rawPaidBy)
	if err != nil {
		return nil, nil, fmt.Errorf("parse paid by: %w", err)
	}
	createdBy, err := uuid.Parse(rawCreate)
	if err != nil {
		return nil, nil, fmt.Errorf("parse created by: %w", err)
	}

	e.ID = expenseID
	e.GroupID = groupID
	e.AmountSen = amountSen
	e.PaidBy = paidBy
	e.CreatedBy = createdBy
	e.ReceiptImage = receiptImage

	rows, err := r.pool.Query(ctx,
		`SELECT es.user_id::text, u.name, es.amount::text
		 FROM expense_splits es
		 JOIN users u ON u.id = es.user_id
		 WHERE es.expense_id = $1
		 ORDER BY u.name`,
		id.String())
	if err != nil {
		return nil, nil, fmt.Errorf("find splits: %w", err)
	}
	defer rows.Close()

	participants := []Participant{}
	for rows.Next() {
		var (
			p         Participant
			rawUserID string
			rawAmount string
		)

		if err := rows.Scan(&rawUserID, &p.Name, &rawAmount); err != nil {
			return nil, nil, fmt.Errorf("scan split: %w", err)
		}

		userID, err := uuid.Parse(rawUserID)
		if err != nil {
			return nil, nil, fmt.Errorf("parse split user id: %w", err)
		}
		amountSen, err := money.ParseMajor(rawAmount)
		if err != nil {
			return nil, nil, fmt.Errorf("parse split amount: %w", err)
		}
		p.UserID = userID
		p.AmountSen = amountSen
		participants = append(participants, p)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("find splits: %w", err)
	}

	e.PayerName = payerName

	return &e, participants, nil
}

func (r *Repository) UpdateExpenseWithSplits(ctx context.Context, e *Expense, splits []SplitAmount) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var updatedAt time.Time
	if e.ReceiptImage != nil {
		err = tx.QueryRow(ctx,
			`UPDATE expenses AS e
			 SET description = $1, amount = $2::numeric, currency = $3, paid_by = $4, category = $5, expense_date = $6, note = $7, receipt_image = $8, receipt_content_type = $9, updated_at = now()
			 WHERE id = $10
			 RETURNING e.updated_at`,
			e.Description, money.FormatMajor(e.AmountSen), e.Currency, e.PaidBy.String(), e.Category, e.ExpenseDate, e.Note, e.ReceiptImage, e.ReceiptContentType, e.ID.String()).Scan(&updatedAt)
	} else {
		err = tx.QueryRow(ctx,
			`UPDATE expenses AS e
			 SET description = $1, amount = $2::numeric, currency = $3, paid_by = $4, category = $5, expense_date = $6, note = $7, updated_at = now()
			 WHERE id = $8
			 RETURNING e.updated_at`,
			e.Description, money.FormatMajor(e.AmountSen), e.Currency, e.PaidBy.String(), e.Category, e.ExpenseDate, e.Note, e.ID.String()).Scan(&updatedAt)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("update expense: %w", err)
	}
	e.UpdatedAt = updatedAt

	if _, err := tx.Exec(ctx, `DELETE FROM expense_splits WHERE expense_id = $1`, e.ID.String()); err != nil {
		return fmt.Errorf("delete splits: %w", err)
	}

	if err := insertSplits(ctx, tx, e.ID, splits); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func insertSplits(ctx context.Context, tx pgx.Tx, expenseID uuid.UUID, splits []SplitAmount) error {
	for _, s := range splits {
		if _, err := tx.Exec(ctx,
			`INSERT INTO expense_splits (expense_id, user_id, amount)
			 VALUES ($1, $2, $3::numeric)`,
			expenseID.String(), s.UserID.String(), money.FormatMajor(s.AmountSen)); err != nil {
			return fmt.Errorf("insert split: %w", err)
		}
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM expenses WHERE id = $1`, id.String())
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListByGroup(ctx context.Context, groupID uuid.UUID, page, limit int, category string, from, to *time.Time) ([]*ExpenseSummary, int, error) {
	where := "e.group_id = $1"
	args := []any{groupID.String()}
	param := 2

	if category != "" {
		where += fmt.Sprintf(" AND e.category = $%d", param)
		args = append(args, category)
		param++
	}
	if from != nil {
		where += fmt.Sprintf(" AND e.expense_date >= $%d", param)
		args = append(args, *from)
		param++
	}
	if to != nil {
		where += fmt.Sprintf(" AND e.expense_date <= $%d", param)
		args = append(args, *to)
		param++
	}

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM expenses e WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count expenses: %w", err)
	}

	limitArg := param
	offsetArg := param + 1
	queryArgs := append(append([]any{}, args...), limit, (page-1)*limit)

	rows, err := r.pool.Query(ctx,
		`SELECT `+expenseColumns+`, u.name,
		        (SELECT count(*) FROM expense_splits es WHERE es.expense_id = e.id),
		        e.receipt_image IS NOT NULL
		 FROM expenses e
		 JOIN users u ON u.id = e.paid_by
		 WHERE `+where+`
		 ORDER BY e.expense_date DESC, e.created_at DESC
		 LIMIT $`+itoa(limitArg)+` OFFSET $`+itoa(offsetArg),
		queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	summaries := []*ExpenseSummary{}
	for rows.Next() {
		var (
			s                ExpenseSummary
			rawID            string
			rawGroup         string
			rawAmount        string
			rawPaidBy        string
			rawCreated       string
			participantCount int
		)

		if err := rows.Scan(&rawID, &rawGroup, &s.Description, &rawAmount, &s.Currency, &rawPaidBy, &s.Category, &s.ExpenseDate, &s.Note, &rawCreated, &s.CreatedAt, &s.UpdatedAt, &s.PayerName, &participantCount, &s.HasReceipt); err != nil {
			return nil, 0, fmt.Errorf("scan expense: %w", err)
		}

		amountSen, err := money.ParseMajor(rawAmount)
		if err != nil {
			return nil, 0, fmt.Errorf("parse amount: %w", err)
		}
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, 0, fmt.Errorf("parse expense id: %w", err)
		}
		groupIDParsed, err := uuid.Parse(rawGroup)
		if err != nil {
			return nil, 0, fmt.Errorf("parse group id: %w", err)
		}
		paidBy, err := uuid.Parse(rawPaidBy)
		if err != nil {
			return nil, 0, fmt.Errorf("parse paid by: %w", err)
		}
		createdBy, err := uuid.Parse(rawCreated)
		if err != nil {
			return nil, 0, fmt.Errorf("parse created by: %w", err)
		}

		s.ID = id
		s.GroupID = groupIDParsed
		s.AmountSen = amountSen
		s.PaidBy = paidBy
		s.CreatedBy = createdBy
		s.ParticipantCount = participantCount
		summaries = append(summaries, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", err)
	}

	return summaries, total, nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
