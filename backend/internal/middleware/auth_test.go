package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/session"
)

func newTestTokens() *session.TokenService {
	return session.NewTokenService([]byte("test-secret"), session.DefaultTokenTTL)
}

func TestRequireAuthNoCookie(t *testing.T) {
	tokens := newTestTokens()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	RequireAuth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAuthValidToken(t *testing.T) {
	tokens := newTestTokens()
	userID := uuid.New()

	token, _, err := tokens.Issue(userID)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: token})

	var got uuid.UUID
	RequireAuth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		got, ok = UserIDFromContext(r.Context())
		if !ok {
			t.Error("user id missing from context")
		}
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got != userID {
		t.Errorf("expected user id %s, got %s", userID, got)
	}
}

func TestRequireAuthInvalidToken(t *testing.T) {
	tokens := newTestTokens()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "garbage-token"})

	RequireAuth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
