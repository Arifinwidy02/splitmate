package auth

import (
	"context"
	"encoding/json"
	"errors"
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
	store := &fakeStore{oauthLinks: map[string]uuid.UUID{}}
	tokens := session.NewTokenService([]byte("test-secret"), session.DefaultTokenTTL)
	svc := NewService(store)
	return NewHandler(svc, tokens, false, nil), store
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

func newTestOAuthHandler() (*Handler, *fakeStore) {
	store := &fakeStore{oauthLinks: map[string]uuid.UUID{}}
	tokens := session.NewTokenService([]byte("test-secret"), session.DefaultTokenTTL)
	svc := NewService(store)
	svc.google = &fakeGoogleClient{
		profile: &GoogleProfile{ID: "google-id-1", Email: "a@b.com", Name: "Google User"},
	}
	oauth := &OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:3000/api/v1/auth/google/callback",
		AppBaseURL:   "http://localhost:3000",
	}
	return NewHandler(svc, tokens, false, oauth), store
}

func TestGoogleLoginNotConfigured(t *testing.T) {
	handler, _ := newTestHandler()

	rec := httptest.NewRecorder()
	handler.GoogleLogin(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	if err := decodeError(t, rec); err.Code != "GOOGLE_NOT_CONFIGURED" {
		t.Errorf("expected GOOGLE_NOT_CONFIGURED, got %q", err.Code)
	}
}

func TestGoogleLoginRedirectsWithStateCookie(t *testing.T) {
	handler, _ := newTestOAuthHandler()

	rec := httptest.NewRecorder()
	handler.GoogleLogin(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "https://accounts.google.com/o/oauth2/v2/auth?") {
		t.Errorf("expected redirect to google, got %q", location)
	}
	if !strings.Contains(location, "client_id=client-id") {
		t.Errorf("expected client_id in redirect, got %q", location)
	}
	if !strings.Contains(location, "redirect_uri=") || !strings.Contains(location, "state=") {
		t.Errorf("expected redirect_uri and state in redirect, got %q", location)
	}

	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, oauthStateCookie+"=") {
		t.Errorf("expected oauth state cookie, got %q", setCookie)
	}
	if !strings.Contains(setCookie, "HttpOnly") {
		t.Error("oauth state cookie must be HttpOnly")
	}
}

func TestGoogleCallbackMissingState(t *testing.T) {
	handler, _ := newTestOAuthHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=abc", nil)
	handler.GoogleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got status %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "http://localhost:3000/login?google=error" {
		t.Errorf("expected error redirect, got %q", got)
	}
}

func TestGoogleCallbackStateMismatch(t *testing.T) {
	handler, _ := newTestOAuthHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=abc&state=wrong", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "correct"})
	handler.GoogleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got status %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "http://localhost:3000/login?google=error" {
		t.Errorf("expected error redirect, got %q", got)
	}
}

func TestGoogleCallbackSuccess(t *testing.T) {
	handler, store := newTestOAuthHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=abc&state=abc123", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "abc123"})
	handler.GoogleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got status %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "http://localhost:3000/" {
		t.Errorf("expected redirect to app root, got %q", got)
	}

	cookies := rec.Result().Cookies()
	var hasSession bool
	for _, c := range cookies {
		if c.Name == session.CookieName && c.Value != "" {
			hasSession = true
		}
	}
	if !hasSession {
		t.Errorf("expected session cookie, got %v", cookies)
	}

	if len(store.users) != 1 || store.users[0].Email != "a@b.com" {
		t.Errorf("expected the google user to be created, got %+v", store.users)
	}
	if _, ok := store.oauthLinks["google:google-id-1"]; !ok {
		t.Error("expected oauth account to be linked")
	}
}

func TestGoogleCallbackFailureDoesNotSetSession(t *testing.T) {
	handler, _ := newTestOAuthHandler()
	handler.service.google = &fakeGoogleClient{exchangeErr: errors.New("boom")}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=abc&state=abc123", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "abc123"})
	handler.GoogleCallback(rec, req)

	if got := rec.Header().Get("Location"); got != "http://localhost:3000/login?google=error" {
		t.Errorf("expected error redirect, got %q", got)
	}
	if strings.Contains(rec.Header().Get("Set-Cookie"), session.CookieName+"=") {
		t.Error("must not set a session cookie on failure")
	}
}
