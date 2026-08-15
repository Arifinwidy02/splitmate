package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
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

type Handler struct {
	service       *Service
	tokens        *session.TokenService
	secureCookies bool
	oauth         *OAuthConfig
}

func NewHandler(service *Service, tokens *session.TokenService, secureCookies bool, oauth *OAuthConfig) *Handler {
	return &Handler{service: service, tokens: tokens, secureCookies: secureCookies, oauth: oauth}
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

	token, expiresAt, err := h.tokens.Issue(u.ID)
	if err != nil {
		slog.Error("failed to issue session token", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	h.setSessionCookie(w, token, expiresAt)
	response.WriteJSON(w, http.StatusOK, envelope{Data: userData{User: toUserResponse(u)}})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.clearSessionCookie(w)
	response.WriteJSON(w, http.StatusOK, logoutResponse{})
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

	u, err := h.service.GoogleLogin(r.Context(), code, h.oauth.RedirectURL, h.oauth.ClientID, h.oauth.ClientSecret)
	if err != nil {
		h.redirectOAuthFailure(w, r, err.Error())
		return
	}

	token, expiresAt, err := h.tokens.Issue(u.ID)
	if err != nil {
		slog.Error("failed to issue session token", "error", err)
		h.redirectOAuthFailure(w, r, "session token failed")
		return
	}

	h.setSessionCookie(w, token, expiresAt)
	http.Redirect(w, r, h.oauth.AppBaseURL+"/", http.StatusFound)
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

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookies,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookies,
	})
}

func toUserResponse(u *user.User) userResponse {
	return userResponse{ID: u.ID, Name: u.Name, Email: u.Email}
}
