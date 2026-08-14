package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
	"github.com/Arifinwidy02/splitmate-backend/internal/session"
	"github.com/Arifinwidy02/splitmate-backend/internal/user"
	"github.com/Arifinwidy02/splitmate-backend/pkg/response"
)

func newTestHandler() (*Handler, *fakeStore) {
	store := &fakeStore{}
	tokens := session.NewTokenService([]byte("test-secret"), session.DefaultTokenTTL)
	svc := NewService(store)
	return NewHandler(svc, tokens, false), store
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) response.ErrorBody {
	t.Helper()

	var body struct {
		Error response.ErrorBody `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid error body: %v", err)
	}
	return body.Error
}

func TestRegisterHandlerSuccess(t *testing.T) {
	handler, _ := newTestHandler()

	body := `{"name":"Arifin","email":"arifin@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			User userResponse `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}

	if resp.Data.User.ID == uuid.Nil {
		t.Error("expected non-empty user id")
	}
	if resp.Data.User.Name != "Arifin" || resp.Data.User.Email != "arifin@example.com" {
		t.Errorf("unexpected user: %+v", resp.Data.User)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Error("response must not contain password hash")
	}
}

func TestRegisterHandlerEmailTaken(t *testing.T) {
	handler, _ := newTestHandler()

	first := `{"name":"Arifin","email":"a@b.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(first))
	handler.Register(httptest.NewRecorder(), req)

	second := `{"name":"Arifin","email":"a@b.com","password":"password123"}`
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(second))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
	if got := decodeError(t, rec); got.Code != "EMAIL_TAKEN" {
		t.Errorf("expected code EMAIL_TAKEN, got %q", got.Code)
	}
}

func TestRegisterHandlerInvalidInput(t *testing.T) {
	handler, _ := newTestHandler()

	body := `{"name":"Arifin","email":"a@b.com","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
	}
	if got := decodeError(t, rec); got.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", got.Code)
	}
}

func TestLoginHandlerSetsCookie(t *testing.T) {
	handler, store := newTestHandler()

	if _, err := store.Create(context.Background(), "Arifin", "a@b.com", mustHash("password123")); err != nil {
		t.Fatalf("seed user failed: %v", err)
	}

	body := `{"email":"a@b.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != session.CookieName {
		t.Errorf("expected cookie name %q, got %q", session.CookieName, cookie.Name)
	}
	if cookie.Value == "" {
		t.Error("expected non-empty session token")
	}
	if !cookie.HttpOnly {
		t.Error("session cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite Lax, got %v", cookie.SameSite)
	}
	if cookie.MaxAge <= 0 {
		t.Errorf("expected positive MaxAge, got %d", cookie.MaxAge)
	}
}

func TestLoginHandlerInvalidCredentials(t *testing.T) {
	handler, _ := newTestHandler()

	body := `{"email":"nobody@b.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if got := decodeError(t, rec); got.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected code INVALID_CREDENTIALS, got %q", got.Code)
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, session.CookieName+"=") {
		t.Errorf("expected Set-Cookie for %q, got %q", session.CookieName, setCookie)
	}
	if !strings.Contains(setCookie, "Max-Age=0") {
		t.Errorf("expected Max-Age=0, got %q", setCookie)
	}
}

func TestMeHandler(t *testing.T) {
	handler, store := newTestHandler()

	userID := uuid.New()
	store.users = append(store.users, &user.User{ID: userID, Name: "Arifin", Email: "a@b.com"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(middleware.WithUserID(req.Context(), userID))
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			User userResponse `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if resp.Data.User.ID != userID {
		t.Errorf("expected user id %s, got %s", userID, resp.Data.User.ID)
	}
}

func mustHash(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}
