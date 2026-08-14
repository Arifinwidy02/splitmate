package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Arifinwidy02/splitmate-backend/internal/user"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
)

type fakeStore struct {
	users []*user.User
}

func (f *fakeStore) Create(ctx context.Context, name, email, passwordHash string) (*user.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return nil, user.ErrEmailTaken
		}
	}

	u := &user.User{
		ID:           uuid.New(),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
	}
	f.users = append(f.users, u)
	return u, nil
}

func (f *fakeStore) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}

func (f *fakeStore) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}

func newTestService(store userStore) *Service {
	return NewService(store)
}

func TestRegisterInvalidInput(t *testing.T) {
	svc := newTestService(&fakeStore{})

	tests := []struct {
		name     string
		input    [3]string
		expected string
	}{
		{"empty name", [3]string{"", "a@b.com", "password123"}, "Name"},
		{"name too long", [3]string{strings.Repeat("a", 101), "a@b.com", "password123"}, "Name"},
		{"invalid email", [3]string{"Arifin", "not-an-email", "password123"}, "valid email"},
		{"missing domain", [3]string{"Arifin", "a@", "password123"}, "valid email"},
		{"short password", [3]string{"Arifin", "a@b.com", "short"}, "8 characters"},
		{"long password", [3]string{"Arifin", "a@b.com", strings.Repeat("x", 73)}, "72 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), tt.input[0], tt.input[1], tt.input[2])

			var valErr *apperror.Validation
			if !asValidationError(err, &valErr) {
				t.Fatalf("expected ValidationError, got %v", err)
			}
			if !strings.Contains(valErr.Message, tt.expected) {
				t.Errorf("expected message containing %q, got %q", tt.expected, valErr.Message)
			}
		})
	}
}

func TestRegisterSuccess(t *testing.T) {
	store := &fakeStore{}
	svc := newTestService(store)

	u, err := svc.Register(context.Background(), "  Arifin  ", "Arifin@Example.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.Name != "Arifin" {
		t.Errorf("expected trimmed name %q, got %q", "Arifin", u.Name)
	}
	if u.Email != "arifin@example.com" {
		t.Errorf("expected lowercased email %q, got %q", "arifin@example.com", u.Email)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("password123")); err != nil {
		t.Errorf("password hash does not match: %v", err)
	}
	if u.PasswordHash == "password123" {
		t.Error("password stored in plaintext")
	}
}

func TestRegisterEmailTaken(t *testing.T) {
	store := &fakeStore{}
	svc := newTestService(store)

	if _, err := svc.Register(context.Background(), "Arifin", "a@b.com", "password123"); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	_, err := svc.Register(context.Background(), "Arifin", "A@B.COM", "password123")
	if err != ErrEmailTaken {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	store := &fakeStore{}
	svc := newTestService(store)

	if _, err := svc.Register(context.Background(), "Arifin", "a@b.com", "password123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"unknown email", "nobody@b.com", "password123"},
		{"wrong password", "a@b.com", "wrong-password"},
		{"wrong case email", "A@B.COM", "wrong-password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Login(context.Background(), tt.email, tt.password)
			if err != ErrInvalidCredentials {
				t.Errorf("expected ErrInvalidCredentials, got %v", err)
			}
		})
	}
}

func TestLoginSuccess(t *testing.T) {
	store := &fakeStore{}
	svc := newTestService(store)

	if _, err := svc.Register(context.Background(), "Arifin", "a@b.com", "password123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	u, err := svc.Login(context.Background(), "A@B.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "a@b.com" {
		t.Errorf("expected email %q, got %q", "a@b.com", u.Email)
	}
}

func TestGetUserNotFound(t *testing.T) {
	svc := newTestService(&fakeStore{})

	_, err := svc.GetUser(context.Background(), uuid.New())
	if err != user.ErrNotFound {
		t.Errorf("expected user.ErrNotFound, got %v", err)
	}
}

func asValidationError(err error, target **apperror.Validation) bool {
	if err == nil {
		return false
	}
	valErr, ok := err.(*apperror.Validation)
	if !ok {
		return false
	}
	*target = valErr
	return true
}
