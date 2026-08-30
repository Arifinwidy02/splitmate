package session

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestTokenRoundTrip(t *testing.T) {
	svc := NewTokenServiceWithDefaults([]byte("test-secret"))
	userID := uuid.New()

	token, expiresAt, err := svc.Issue(userID)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	if expiresAt.Before(time.Now().Add(AccessTokenTTL - time.Minute)) {
		t.Errorf("unexpected expiry %v", expiresAt)
	}

	parsed, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed != userID {
		t.Errorf("expected %s, got %s", userID, parsed)
	}
}

func TestTokenWrongSecret(t *testing.T) {
	svc := NewTokenServiceWithDefaults([]byte("test-secret"))
	other := NewTokenServiceWithDefaults([]byte("other-secret"))

	token, _, err := other.Issue(uuid.New())
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	if _, err := svc.Parse(token); err == nil {
		t.Error("expected parse error for token signed with different secret")
	}
}

func TestTokenGarbage(t *testing.T) {
	svc := NewTokenServiceWithDefaults([]byte("test-secret"))

	if _, err := svc.Parse("not-a-jwt"); err == nil {
		t.Error("expected parse error for garbage token")
	}
}

func TestTokenExpired(t *testing.T) {
	secret := []byte("test-secret")
	svc := NewTokenServiceWithDefaults(secret)

	claims := jwt.RegisteredClaims{
		Subject:   uuid.New().String(),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("sign expired token failed: %v", err)
	}

	if _, err := svc.Parse(expired); err == nil {
		t.Error("expected parse error for expired token")
	}
}

func TestTokenWrongAlg(t *testing.T) {
	secret := []byte("test-secret")
	svc := NewTokenServiceWithDefaults(secret)

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject: uuid.New().String(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign token failed: %v", err)
	}

	if _, err := svc.Parse(none); err == nil {
		t.Error("expected parse error for token with alg=none")
	}
}
