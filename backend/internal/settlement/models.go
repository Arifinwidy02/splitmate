package settlement

import (
	"time"

	"github.com/google/uuid"
)

type Settlement struct {
	ID           uuid.UUID
	GroupID      uuid.UUID
	PayerID      uuid.UUID
	PayerName    string
	ReceiverID   uuid.UUID
	ReceiverName string
	AmountSen    int64
	SettledAt    time.Time
	CreatedAt    time.Time
}

type CreateSettlementInput struct {
	PayerID    uuid.UUID
	ReceiverID uuid.UUID
	AmountSen  int64
	SettledAt  time.Time
}
