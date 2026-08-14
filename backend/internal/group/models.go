package group

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleAdmin  = "admin"
	RoleMember = "member"

	statusPending   = "pending"
	statusAccepted  = "accepted"
	statusExpired   = "expired"
	statusCancelled = "cancelled"
)

type Group struct {
	ID          uuid.UUID
	Name        string
	Description *string
	Currency    string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Role        string
	MemberCount int
}

type Member struct {
	UserID   uuid.UUID
	Name     string
	Email    string
	Role     string
	JoinedAt time.Time
}

type Membership struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
	Role    string
}

type Invitation struct {
	ID        uuid.UUID
	GroupID   uuid.UUID
	Email     string
	InvitedBy uuid.UUID
	TokenHash string
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
}
