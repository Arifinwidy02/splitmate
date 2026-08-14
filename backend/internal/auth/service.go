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
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already taken")
)

type userStore interface {
	Create(ctx context.Context, name, email, passwordHash string) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)
}

type Service struct {
	users userStore
}

func NewService(users userStore) *Service {
	return &Service{users: users}
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
