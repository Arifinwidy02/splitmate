package group

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/user"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
)

var (
	ErrGroupNotFound       = errors.New("group not found")
	ErrMemberNotFound      = errors.New("member not found")
	ErrForbidden           = errors.New("forbidden")
	ErrInvitationNotFound  = errors.New("invitation not found")
	ErrInvitationExists    = errors.New("invitation already exists")
	ErrInvitationExpired   = errors.New("invitation expired")
	ErrInvitationUsed      = errors.New("invitation already used")
	ErrInvitationForbidden = errors.New("invitation is for a different email")
	ErrInviteLinkRevoked   = errors.New("invite link revoked")
	ErrInviteLinkLimit     = errors.New("invite link usage limit reached")
	ErrNoLogo              = errors.New("group has no logo")
)

const (
	invitationTTL     = 7 * 24 * time.Hour
	maxDescriptionLen = 500
	maxEmailLen       = 255
	maxBulkInvites    = 50
	previewMemberCap  = 8
)

const (
	ReasonMemberExists     = "MEMBER_EXISTS"
	ReasonInvitationExists = "INVITATION_EXISTS"
	ReasonDuplicate        = "DUPLICATE"
)

type BulkInvitationResult struct {
	Invitation *Invitation
	Token      string
}

type InvitationFailure struct {
	Email  string
	Reason string
}

var currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)

type userFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)
}

type Service struct {
	store store
	users userFinder
}

func NewService(store store, users userFinder) *Service {
	return &Service{store: store, users: users}
}

func (s *Service) CreateGroup(ctx context.Context, userID uuid.UUID, name, description, currency string, logo *Logo) (*Group, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	currency = strings.ToUpper(strings.TrimSpace(currency))

	if err := validateGroupName(name); err != nil {
		return nil, err
	}
	if !currencyRe.MatchString(currency) {
		return nil, &apperror.Validation{Message: "Currency must be a 3-letter code (e.g. IDR, USD)"}
	}
	if err := validateLogo(logo); err != nil {
		return nil, err
	}

	var desc *string
	if description != "" {
		if utf8.RuneCountInString(description) > maxDescriptionLen {
			return nil, &apperror.Validation{Message: "Description must be at most 500 characters"}
		}
		desc = &description
	}

	g, err := s.store.CreateGroupWithAdmin(ctx, &Group{
		Name:            name,
		Description:     desc,
		Currency:        currency,
		LogoImage:       logoImage(logo),
		LogoContentType: logoContentType(logo),
	}, userID)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	g.Role = RoleAdmin
	g.MemberCount = 1

	return g, nil
}

func (s *Service) ListGroups(ctx context.Context, userID uuid.UUID) ([]*Group, error) {
	groups, err := s.store.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	return groups, nil
}

func (s *Service) GetGroup(ctx context.Context, userID, groupID uuid.UUID) (*Group, error) {
	m, err := s.requireMembership(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}

	g, err := s.store.FindByID(ctx, groupID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find group: %w", err)
	}
	g.Role = m.Role

	return g, nil
}

func (s *Service) UpdateGroup(ctx context.Context, userID, groupID uuid.UUID, name, description, currency *string) (*Group, error) {
	m, err := s.requireAdmin(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}

	g, err := s.store.FindByID(ctx, groupID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find group: %w", err)
	}
	g.Role = m.Role

	if name != nil {
		n := strings.TrimSpace(*name)
		if err := validateGroupName(n); err != nil {
			return nil, err
		}
		g.Name = n
	}
	if description != nil {
		d := strings.TrimSpace(*description)
		if utf8.RuneCountInString(d) > maxDescriptionLen {
			return nil, &apperror.Validation{Message: "Description must be at most 500 characters"}
		}
		if d == "" {
			g.Description = nil
		} else {
			g.Description = &d
		}
	}
	if currency != nil {
		c := strings.ToUpper(strings.TrimSpace(*currency))
		if !currencyRe.MatchString(c) {
			return nil, &apperror.Validation{Message: "Currency must be a 3-letter code (e.g. IDR, USD)"}
		}
		g.Currency = c
	}

	if err := s.store.Update(ctx, g); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("update group: %w", err)
	}

	return g, nil
}

func (s *Service) DeleteGroup(ctx context.Context, userID, groupID uuid.UUID) error {
	if _, err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return err
	}

	if err := s.store.Delete(ctx, groupID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrGroupNotFound
		}
		return fmt.Errorf("delete group: %w", err)
	}

	return nil
}

func (s *Service) GetLogo(ctx context.Context, userID, groupID uuid.UUID) (*Logo, error) {
	if _, err := s.requireMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}

	logo, err := s.store.FindLogo(ctx, groupID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if errors.Is(err, ErrNoLogo) {
		return nil, ErrNoLogo
	}
	if err != nil {
		return nil, fmt.Errorf("find group logo: %w", err)
	}

	return logo, nil
}

func (s *Service) ListMembers(ctx context.Context, userID, groupID uuid.UUID) ([]*Member, error) {
	if _, err := s.requireMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}

	members, err := s.store.ListMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	return members, nil
}

func (s *Service) RemoveMember(ctx context.Context, userID, groupID, memberID uuid.UUID) error {
	if _, err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return err
	}
	if memberID == userID {
		return &apperror.Validation{Message: "You cannot remove yourself from the group"}
	}

	if _, err := s.store.FindMembership(ctx, groupID, memberID); errors.Is(err, ErrNotFound) {
		return ErrMemberNotFound
	} else if err != nil {
		return fmt.Errorf("find member: %w", err)
	}

	if err := s.store.RemoveMember(ctx, groupID, memberID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("remove member: %w", err)
	}

	return nil
}

func (s *Service) CreateInvitation(ctx context.Context, userID, groupID uuid.UUID, email string) (*Invitation, string, error) {
	if _, err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return nil, "", err
	}

	email = normalizeEmail(email)
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > maxEmailLen {
		return nil, "", &apperror.Validation{Message: "Enter a valid email address"}
	}

	if _, err := s.store.FindMembershipByEmail(ctx, groupID, email); err == nil {
		return nil, "", ErrMemberExists
	} else if !errors.Is(err, ErrNotFound) {
		return nil, "", fmt.Errorf("find member by email: %w", err)
	}

	if _, err := s.store.FindPendingInvitation(ctx, groupID, email); err == nil {
		return nil, "", ErrInvitationExists
	} else if !errors.Is(err, ErrNotFound) {
		return nil, "", fmt.Errorf("find pending invitation: %w", err)
	}

	token, err := newInvitationToken()
	if err != nil {
		return nil, "", fmt.Errorf("generate invitation token: %w", err)
	}

	now := time.Now()
	inv := &Invitation{
		GroupID:   groupID,
		Email:     email,
		InvitedBy: userID,
		TokenHash: hashToken(token),
		Status:    statusPending,
		ExpiresAt: now.Add(invitationTTL),
	}

	if err := s.store.CreateInvitation(ctx, inv); err != nil {
		return nil, "", fmt.Errorf("create invitation: %w", err)
	}

	return inv, token, nil
}

func (s *Service) CreateBulkInvitations(ctx context.Context, userID, groupID uuid.UUID, emails []string) ([]BulkInvitationResult, []InvitationFailure, error) {
	if _, err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return nil, nil, err
	}
	if len(emails) == 0 {
		return nil, nil, &apperror.Validation{Message: "Enter at least one email address"}
	}
	if len(emails) > maxBulkInvites {
		return nil, nil, &apperror.Validation{Message: "At most 50 email addresses per request"}
	}

	normalized := make([]string, 0, len(emails))
	for _, email := range emails {
		email = normalizeEmail(email)
		addr, err := mail.ParseAddress(email)
		if err != nil || addr.Address != email || len(email) > maxEmailLen {
			return nil, nil, &apperror.Validation{Message: "Enter valid email addresses"}
		}
		normalized = append(normalized, email)
	}

	unique := make([]string, 0, len(normalized))
	seen := make(map[string]bool, len(normalized))
	var failures []InvitationFailure
	for _, email := range normalized {
		if seen[email] {
			failures = append(failures, InvitationFailure{Email: email, Reason: ReasonDuplicate})
			continue
		}
		seen[email] = true
		unique = append(unique, email)
	}

	members, err := s.store.MembersByEmails(ctx, groupID, unique)
	if err != nil {
		return nil, nil, fmt.Errorf("find members by emails: %w", err)
	}
	pending, err := s.store.PendingInvitationsByEmails(ctx, groupID, unique)
	if err != nil {
		return nil, nil, fmt.Errorf("find pending invitations by emails: %w", err)
	}

	tokens := make(map[string]string, len(unique))
	invites := make([]*Invitation, 0, len(unique))
	for _, email := range unique {
		if members[email] {
			failures = append(failures, InvitationFailure{Email: email, Reason: ReasonMemberExists})
			continue
		}
		if pending[email] {
			failures = append(failures, InvitationFailure{Email: email, Reason: ReasonInvitationExists})
			continue
		}

		token, err := newInvitationToken()
		if err != nil {
			return nil, nil, fmt.Errorf("generate invitation token: %w", err)
		}
		tokens[email] = token
		invites = append(invites, &Invitation{
			GroupID:   groupID,
			Email:     email,
			InvitedBy: userID,
			TokenHash: hashToken(token),
			Status:    statusPending,
			ExpiresAt: time.Now().Add(invitationTTL),
		})
	}

	created, err := s.store.CreateInvitations(ctx, invites)
	if err != nil {
		return nil, nil, fmt.Errorf("create invitations: %w", err)
	}

	results := make([]BulkInvitationResult, 0, len(created))
	for _, inv := range created {
		results = append(results, BulkInvitationResult{Invitation: inv, Token: tokens[inv.Email]})
	}

	return results, failures, nil
}

func (s *Service) AcceptInvitation(ctx context.Context, userID uuid.UUID, token string) (*Group, error) {
	if token == "" {
		return nil, ErrInvitationNotFound
	}

	inv, err := s.store.FindInvitationByTokenHash(ctx, hashToken(token))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvitationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invitation: %w", err)
	}

	if inv.Status != statusPending {
		return nil, ErrInvitationUsed
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if u.Email != inv.Email {
		return nil, ErrInvitationForbidden
	}

	if err := s.store.AcceptInvitation(ctx, inv, userID); err != nil {
		if errors.Is(err, ErrMemberExists) {
			return nil, ErrMemberExists
		}
		return nil, fmt.Errorf("accept invitation: %w", err)
	}

	g, err := s.store.FindByID(ctx, inv.GroupID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find group: %w", err)
	}
	g.Role = RoleMember

	return g, nil
}

func (s *Service) GetOrCreateInviteLink(ctx context.Context, userID, groupID uuid.UUID) (*InviteLink, error) {
	if _, err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return nil, err
	}

	link, err := s.store.FindActiveInviteLink(ctx, groupID)
	if err == nil {
		return link, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("find active invite link: %w", err)
	}

	token, err := newInvitationToken()
	if err != nil {
		return nil, fmt.Errorf("generate invite token: %w", err)
	}

	link = &InviteLink{
		GroupID:   groupID,
		Token:     token,
		TokenHash: hashToken(token),
		CreatedBy: userID,
		ExpiresAt: time.Now().Add(invitationTTL),
	}

	if err := s.store.CreateInviteLink(ctx, link); err != nil {
		return nil, fmt.Errorf("create invite link: %w", err)
	}

	return link, nil
}

func (s *Service) RevokeInviteLink(ctx context.Context, userID, groupID uuid.UUID) error {
	if _, err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return err
	}

	if err := s.store.RevokeInviteLinks(ctx, groupID); err != nil {
		return fmt.Errorf("revoke invite links: %w", err)
	}

	return nil
}

func (s *Service) PreviewInviteLink(ctx context.Context, token string, viewerID *uuid.UUID) (*GroupPreview, bool, error) {
	link, err := s.findValidInviteLink(ctx, token)
	if err != nil {
		return nil, false, err
	}

	preview, err := s.store.FindGroupPreview(ctx, link.GroupID)
	if errors.Is(err, ErrNotFound) {
		return nil, false, ErrGroupNotFound
	}
	if err != nil {
		return nil, false, fmt.Errorf("find group preview: %w", err)
	}
	if len(preview.MemberNames) > previewMemberCap {
		preview.MemberNames = preview.MemberNames[:previewMemberCap]
	}

	isMember := false
	if viewerID != nil {
		if _, err := s.store.FindMembership(ctx, link.GroupID, *viewerID); err == nil {
			isMember = true
		} else if !errors.Is(err, ErrNotFound) {
			return nil, false, fmt.Errorf("find membership: %w", err)
		}
	}

	return preview, isMember, nil
}

func (s *Service) JoinGroupViaLink(ctx context.Context, userID uuid.UUID, token string) (*Group, bool, error) {
	link, err := s.findValidInviteLink(ctx, token)
	if err != nil {
		return nil, false, err
	}

	g, err := s.store.FindByID(ctx, link.GroupID)
	if errors.Is(err, ErrNotFound) {
		return nil, false, ErrGroupNotFound
	}
	if err != nil {
		return nil, false, fmt.Errorf("find group: %w", err)
	}
	g.Role = RoleMember

	if _, err := s.store.FindMembership(ctx, link.GroupID, userID); err == nil {
		return g, true, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, false, fmt.Errorf("find membership: %w", err)
	}

	if link.MaxUses != nil && link.UsedCount >= *link.MaxUses {
		return nil, false, ErrInviteLinkLimit
	}

	if err := s.store.JoinViaInviteLink(ctx, link, userID); err != nil {
		if errors.Is(err, ErrMemberExists) {
			return g, true, nil
		}
		if errors.Is(err, ErrInviteLinkLimit) {
			return nil, false, ErrInviteLinkLimit
		}
		return nil, false, fmt.Errorf("join via invite link: %w", err)
	}

	return g, false, nil
}

func (s *Service) findValidInviteLink(ctx context.Context, token string) (*InviteLink, error) {
	if token == "" {
		return nil, ErrInvitationNotFound
	}

	link, err := s.store.FindInviteLinkByTokenHash(ctx, hashToken(token))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvitationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invite link: %w", err)
	}
	if link.RevokedAt != nil {
		return nil, ErrInviteLinkRevoked
	}
	if time.Now().After(link.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	return link, nil
}

func (s *Service) requireMembership(ctx context.Context, groupID, userID uuid.UUID) (*Membership, error) {
	m, err := s.store.FindMembership(ctx, groupID, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find membership: %w", err)
	}
	return m, nil
}

func (s *Service) requireAdmin(ctx context.Context, groupID, userID uuid.UUID) (*Membership, error) {
	m, err := s.requireMembership(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if m.Role != RoleAdmin {
		return nil, ErrForbidden
	}
	return m, nil
}

func validateGroupName(name string) error {
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return &apperror.Validation{Message: "Name must be between 1 and 100 characters"}
	}
	return nil
}

func validateLogo(logo *Logo) error {
	if logo == nil {
		return nil
	}
	if len(logo.Image) == 0 {
		return &apperror.Validation{Message: "Logo image is empty"}
	}
	if len(logo.Image) > maxLogoBytes {
		return &apperror.Validation{Message: "Logo image must be at most 5MB"}
	}
	if !logoContentTypes[logo.ContentType] {
		return &apperror.Validation{Message: "Logo must be a JPEG, PNG, WebP or GIF image"}
	}
	return nil
}

func logoImage(logo *Logo) []byte {
	if logo == nil {
		return nil
	}
	return logo.Image
}

func logoContentType(logo *Logo) string {
	if logo == nil {
		return ""
	}
	return logo.ContentType
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func newInvitationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
