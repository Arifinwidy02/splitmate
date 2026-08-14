package auth

import (
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

type Handler struct {
	service       *Service
	tokens        *session.TokenService
	secureCookies bool
}

func NewHandler(service *Service, tokens *session.TokenService, secureCookies bool) *Handler {
	return &Handler{service: service, tokens: tokens, secureCookies: secureCookies}
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
