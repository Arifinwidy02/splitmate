package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
	"github.com/Arifinwidy02/splitmate-backend/internal/session"
	"github.com/Arifinwidy02/splitmate-backend/internal/user"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
	"github.com/Arifinwidy02/splitmate-backend/pkg/response"
)

const oauthStateCookie = "oauth_state"
const oauthStateTTL = 10 * time.Minute

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AppBaseURL   string
}

// sessionManager is the subset of session.Service used by the auth handler.
// It is an interface so the handler can be tested without a database.
type sessionManager interface {
	CreateTokenPair(ctx context.Context, userID uuid.UUID) (*session.TokenPair, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (*session.TokenPair, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
	ParseAccessToken(token string) (uuid.UUID, error)
}

type Handler struct {
	service       *Service
	session       sessionManager
	secureCookies bool
	oauth         *OAuthConfig
}

func NewHandler(service *Service, sessions sessionManager, secureCookies bool, oauth *OAuthConfig) *Handler {
	return &Handler{service: service, session: sessions, secureCookies: secureCookies, oauth: oauth}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type userData struct {
	User userResponse `json:"user"`
}

type envelope struct {
	Data userData `json:"data"`
}

type logoutResponse struct {
	Data struct{} `json:"data"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		return
	}

	u, err := h.service.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	tokens, err := h.session.CreateTokenPair(r.Context(), u.ID)
	if err != nil {
		slog.Error("failed to create token pair", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	h.setAccessCookie(w, tokens.AccessToken, tokens.AccessExpiresAt)
	h.setRefreshCookie(w, tokens.RefreshToken, tokens.RefreshExpiresAt)
	response.WriteJSON(w, http.StatusCreated, envelope{Data: userData{User: toUserResponse(u)}})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		return
	}

	u, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	tokens, err := h.session.CreateTokenPair(r.Context(), u.ID)
	if err != nil {
		slog.Error("failed to create token pair", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	h.setAccessCookie(w, tokens.AccessToken, tokens.AccessExpiresAt)
	h.setRefreshCookie(w, tokens.RefreshToken, tokens.RefreshExpiresAt)
	response.WriteJSON(w, http.StatusOK, envelope{Data: userData{User: toUserResponse(u)}})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear both cookies
	h.clearAccessCookie(w)
	h.clearRefreshCookie(w)
	
	// Try to revoke refresh token if present
	if refreshCookie, err := r.Cookie(session.RefreshTokenCookie); err == nil {
		_ = h.session.RevokeRefreshToken(r.Context(), refreshCookie.Value)
	}
	
	response.WriteJSON(w, http.StatusOK, logoutResponse{})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshCookie, err := r.Cookie(session.RefreshTokenCookie)
	if err != nil {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Refresh token required")
		return
	}

	tokens, err := h.session.RefreshAccessToken(r.Context(), refreshCookie.Value)
	if err != nil {
		slog.Warn("failed to refresh access token", "error", err)
		h.clearAccessCookie(w)
		h.clearRefreshCookie(w)
		
		switch {
		case errors.Is(err, session.ErrRefreshTokenNotFound), 
		     errors.Is(err, session.ErrRefreshTokenRevoked),
		     errors.Is(err, session.ErrRefreshTokenExpired):
			response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired refresh token")
		default:
			response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	h.setAccessCookie(w, tokens.AccessToken, tokens.AccessExpiresAt)
	h.setRefreshCookie(w, tokens.RefreshToken, tokens.RefreshExpiresAt)
	
	// Return user data with new tokens
	userID, err := h.session.ParseAccessToken(tokens.AccessToken)
	if err != nil {
		slog.Error("failed to parse new access token", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	u, err := h.service.GetUser(r.Context(), userID)
	if errors.Is(err, user.ErrNotFound) {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	if err != nil {
		slog.Error("failed to load user", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: userData{User: toUserResponse(u)}})
}

func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "GOOGLE_NOT_CONFIGURED", "Google sign in is not configured")
		return
	}

	state, err := randomState()
	if err != nil {
		slog.Error("failed to generate oauth state", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   int(oauthStateTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookies,
	})

	http.Redirect(w, r, googleAuthURL(h.oauth.ClientID, h.oauth.RedirectURL, state), http.StatusFound)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil {
		h.redirectOAuthFailure(w, r, "missing oauth state")
		return
	}
	h.clearOAuthStateCookie(w)

	if h.oauth == nil {
		h.redirectOAuthFailure(w, r, "oauth not configured")
		return
	}

	state := r.URL.Query().Get("state")
	if subtle.ConstantTimeCompare([]byte(state), []byte(stateCookie.Value)) != 1 {
		h.redirectOAuthFailure(w, r, "oauth state mismatch")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		h.redirectOAuthFailure(w, r, "missing authorization code")
		return
	}

	// Read optional `next` query param for post-login redirect
	nextURL := r.URL.Query().Get("next")
	if nextURL == "" || !strings.HasPrefix(nextURL, "/") {
		nextURL = "/"
	}

	u, err := h.service.GoogleLogin(r.Context(), code, h.oauth.RedirectURL, h.oauth.ClientID, h.oauth.ClientSecret)
	if err != nil {
		h.redirectOAuthFailure(w, r, err.Error())
		return
	}

	tokens, err := h.session.CreateTokenPair(r.Context(), u.ID)
	if err != nil {
		slog.Error("failed to create token pair", "error", err)
		h.redirectOAuthFailure(w, r, "session token failed")
		return
	}

	h.setAccessCookie(w, tokens.AccessToken, tokens.AccessExpiresAt)
	h.setRefreshCookie(w, tokens.RefreshToken, tokens.RefreshExpiresAt)
	http.Redirect(w, r, h.oauth.AppBaseURL+nextURL, http.StatusFound)
}

func (h *Handler) redirectOAuthFailure(w http.ResponseWriter, r *http.Request, reason string) {
	slog.Warn("google oauth callback failed", "reason", reason)
	if h.oauth == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "GOOGLE_NOT_CONFIGURED", "Google sign in is not configured")
		return
	}
	http.Redirect(w, r, h.oauth.AppBaseURL+"/login?google=error", http.StatusFound)
}

func (h *Handler) clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookies,
	})
}

func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	u, err := h.service.GetUser(r.Context(), userID)
	if errors.Is(err, user.ErrNotFound) {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	if err != nil {
		slog.Error("failed to load user", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: userData{User: toUserResponse(u)}})
}

func (h *Handler) writeAuthError(w http.ResponseWriter, err error) {
	var valErr *apperror.Validation
	switch {
	case errors.As(err, &valErr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", valErr.Message)
	case errors.Is(err, ErrEmailTaken):
		response.WriteError(w, http.StatusConflict, "EMAIL_TAKEN", "An account with this email already exists")
	case errors.Is(err, ErrInvalidCredentials):
		response.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect")
	default:
		slog.Error("auth request failed", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
	}
}

func (h *Handler) setAccessCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.AccessTokenCookie,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookies,
	})
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.RefreshTokenCookie,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.secureCookies,
	})
}

func (h *Handler) clearAccessCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.AccessTokenCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookies,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.RefreshTokenCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.secureCookies,
	})
}

// Deprecated: use setAccessCookie
func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	h.setAccessCookie(w, token, expiresAt)
}

// Deprecated: use clearAccessCookie
func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	h.clearAccessCookie(w)
}

func toUserResponse(u *user.User) userResponse {
	return userResponse{ID: u.ID, Name: u.Name, Email: u.Email}
}
