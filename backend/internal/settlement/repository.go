package settlement

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arifinwidy02/splitmate-backend/pkg/money"
)

var ErrNotFound = errors.New("not found")

type store interface {
	Create(ctx context.Context, s *Settlement) error
	ListByGroup(ctx context.Context, groupID uuid.UUID) ([]*Settlement, error)
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, s *Settlement) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO settlements (group_id, payer_id, receiver_id, amount, settled_at)
		 VALUES ($1, $2, $3, $4::numeric, $5)
		 RETURNING id::text, created_at`,
		s.GroupID.String(), s.PayerID.String(), s.ReceiverID.String(), money.FormatMajor(s.AmountSen), s.SettledAt)

	var rawID string
	if err := row.Scan(&rawID, &s.CreatedAt); err != nil {
		return fmt.Errorf("insert settlement: %w", err)
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse settlement id: %w", err)
	}
	s.ID = id

	err = r.pool.QueryRow(ctx,
		`SELECT pu.name, ru.name
		 FROM users pu, users ru
		 WHERE pu.id = $1 AND ru.id = $2`,
		s.PayerID.String(), s.ReceiverID.String()).Scan(&s.PayerName, &s.ReceiverName)
	if err != nil {
		return fmt.Errorf("fetch settlement names: %w", err)
	}

	return nil
}

func (r *Repository) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]*Settlement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.id::text, s.payer_id::text, pu.name, s.receiver_id::text, ru.name, s.amount::text, s.settled_at, s.created_at
		 FROM settlements s
		 JOIN users pu ON pu.id = s.payer_id
		 JOIN users ru ON ru.id = s.receiver_id
		 WHERE s.group_id = $1
		 ORDER BY s.settled_at DESC, s.created_at DESC`,
		groupID.String())
	if err != nil {
		return nil, fmt.Errorf("list settlements: %w", err)
	}
	defer rows.Close()

	settlements := []*Settlement{}
	for rows.Next() {
		var (
			s          Settlement
			rawID      string
			rawPayerID string
			rawRecvID  string
			rawAmount  string
		)

		if err := rows.Scan(&rawID, &rawPayerID, &s.PayerName, &rawRecvID, &s.ReceiverName, &rawAmount, &s.SettledAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan settlement: %w", err)
		}

		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("parse settlement id: %w", err)
		}
		payerID, err := uuid.Parse(rawPayerID)
		if err != nil {
			return nil, fmt.Errorf("parse payer id: %w", err)
		}
		receiverID, err := uuid.Parse(rawRecvID)
		if err != nil {
			return nil, fmt.Errorf("parse receiver id: %w", err)
		}
		amountSen, err := money.ParseMajor(rawAmount)
		if err != nil {
			return nil, fmt.Errorf("parse settlement amount: %w", err)
		}

		s.ID = id
		s.GroupID = groupID
		s.PayerID = payerID
		s.ReceiverID = receiverID
		s.AmountSen = amountSen
		settlements = append(settlements, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list settlements: %w", err)
	}

	return settlements, nil
}
