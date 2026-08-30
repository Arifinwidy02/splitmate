package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/session"
)

func newTestSessionService() *session.Service {
	tokens := session.NewTokenServiceWithDefaults([]byte("test-secret"))
	// ParseAccessToken does not touch the repository, so a nil repository is
	// safe for these tests.
	return session.NewService(tokens, nil)
}

func TestRequireAuthNoCookie(t *testing.T) {
	sessions := newTestSessionService()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	RequireAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAuthValidToken(t *testing.T) {
	sessions := newTestSessionService()
	userID := uuid.New()

	tokens := session.NewTokenServiceWithDefaults([]byte("test-secret"))
	tokenString, _, err := tokens.IssueAccessToken(userID)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.AccessTokenCookie, Value: tokenString})

	var got uuid.UUID
	RequireAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	sessions := newTestSessionService()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.AccessTokenCookie, Value: "garbage-token"})

	RequireAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestOptionalAuthAttachesUserID(t *testing.T) {
	sessions := newTestSessionService()
	tokens := session.NewTokenServiceWithDefaults([]byte("test-secret"))
	userID := uuid.New()

	token, _, err := tokens.IssueAccessToken(userID)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.AccessTokenCookie, Value: token})

	var got uuid.UUID
	var ok bool
	OptionalAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !ok || got != userID {
		t.Errorf("expected user id %s, got %s (ok=%v)", userID, got, ok)
	}
}

func TestOptionalAuthPassesThroughWithoutCookie(t *testing.T) {
	sessions := newTestSessionService()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	called := false
	OptionalAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := UserIDFromContext(r.Context()); ok {
			t.Error("user id must not be set without a cookie")
		}
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !called {
		t.Error("handler should still be called")
	}
}

func TestOptionalAuthPassesThroughInvalidToken(t *testing.T) {
	sessions := newTestSessionService()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.AccessTokenCookie, Value: "garbage-token"})

	called := false
	OptionalAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := UserIDFromContext(r.Context()); ok {
			t.Error("user id must not be set for an invalid token")
		}
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !called {
		t.Error("handler should still be called")
	}
}
