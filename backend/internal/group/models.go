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

const (
	maxLogoBytes   = 5 << 20 // 5MB
	logoFieldLimit = 10 << 20
)

var logoContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

type Logo struct {
	Image       []byte
	ContentType string
}

type Group struct {
	ID              uuid.UUID
	Name            string
	Description     *string
	Currency        string
	CreatedBy       uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Role            string
	MemberCount     int
	HasLogo         bool
	LogoImage       []byte
	LogoContentType string
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

type InviteLink struct {
	ID        uuid.UUID
	GroupID   uuid.UUID
	Token     string
	TokenHash string
	CreatedBy uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
	MaxUses   *int
	UsedCount int
	CreatedAt time.Time
}

type GroupPreview struct {
	GroupID     uuid.UUID
	Name        string
	Description *string
	Currency    string
	MemberCount int
	CreatorName string
	MemberNames []string
}
