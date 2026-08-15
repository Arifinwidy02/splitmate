package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Arifinwidy02/splitmate-backend/internal/user"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrEmailTaken          = errors.New("email already taken")
	ErrOAuthNotConfigured  = errors.New("google oauth not configured")
	ErrOAuthEmailMissing   = errors.New("google account has no verified email")
	ErrOAuthExchangeFailed = errors.New("failed to complete google sign in")
)

type userStore interface {
	Create(ctx context.Context, name, email, passwordHash string) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	FindByOAuthAccount(ctx context.Context, provider, providerAccountID string) (*user.User, error)
	CreateWithOAuth(ctx context.Context, name, email string, avatarURL *string, provider, providerAccountID string) (*user.User, error)
	LinkOAuthAccount(ctx context.Context, userID uuid.UUID, provider, providerAccountID string) error
}

type Service struct {
	users  userStore
	google googleClient
}

func NewService(users userStore) *Service {
	return &Service{users: users, google: newGoogleHTTPClient()}
}

func (s *Service) Register(ctx context.Context, name, email, password string) (*user.User, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)

	if name == "" || utf8.RuneCountInString(name) > 100 {
		return nil, &apperror.Validation{Message: "Name must be between 1 and 100 characters"}
	}

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 255 {
		return nil, &apperror.Validation{Message: "Enter a valid email address"}
	}

	if len(password) < 8 {
		return nil, &apperror.Validation{Message: "Password must be at least 8 characters"}
	}
	if len(password) > 72 {
		return nil, &apperror.Validation{Message: "Password must be at most 72 characters"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u, err := s.users.Create(ctx, name, email, string(hash))
	if errors.Is(err, user.ErrEmailTaken) {
		return nil, ErrEmailTaken
	}
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*user.User, error) {
	email = normalizeEmail(email)

	u, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, user.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return u, nil
}

func (s *Service) GoogleLogin(ctx context.Context, code, redirectURL, clientID, clientSecret string) (*user.User, error) {
	if s.google == nil {
		return nil, ErrOAuthNotConfigured
	}

	accessToken, err := s.google.ExchangeCode(ctx, code, redirectURL, clientID, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthExchangeFailed, err)
	}

	profile, err := s.google.FetchProfile(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthExchangeFailed, err)
	}

	email := normalizeEmail(profile.Email)
	if email == "" || len(email) > 255 {
		return nil, ErrOAuthEmailMissing
	}

	return s.FindOrCreateByOAuth(ctx, "google", profile.ID, email, strings.TrimSpace(profile.Name), profile.AvatarURL)
}

func (s *Service) FindOrCreateByOAuth(ctx context.Context, provider, providerAccountID, email, name string, avatarURL *string) (*user.User, error) {
	email = normalizeEmail(email)
	if email == "" || len(email) > 255 {
		return nil, ErrOAuthEmailMissing
	}
	u, err := s.users.FindByOAuthAccount(ctx, provider, providerAccountID)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, user.ErrNotFound) {
		return nil, fmt.Errorf("find oauth account: %w", err)
	}

	existing, err := s.users.FindByEmail(ctx, email)
	switch {
	case err == nil:
		if err := s.users.LinkOAuthAccount(ctx, existing.ID, provider, providerAccountID); err != nil {
			return nil, fmt.Errorf("link oauth account: %w", err)
		}
		return existing, nil
	case !errors.Is(err, user.ErrNotFound):
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if name == "" {
		name = email
	}
	u, err = s.users.CreateWithOAuth(ctx, name, email, avatarURL, provider, providerAccountID)
	if errors.Is(err, user.ErrEmailTaken) {
		existing, findErr := s.users.FindByEmail(ctx, email)
		if findErr != nil {
			return nil, fmt.Errorf("find user by email after conflict: %w", findErr)
		}
		if err := s.users.LinkOAuthAccount(ctx, existing.ID, provider, providerAccountID); err != nil {
			return nil, fmt.Errorf("link oauth account after conflict: %w", err)
		}
		return existing, nil
	}
	if err != nil {
		return nil, fmt.Errorf("create oauth user: %w", err)
	}

	return u, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
