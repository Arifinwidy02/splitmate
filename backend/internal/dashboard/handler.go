package dashboard

import (
	"log/slog"
	"net/http"

	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
	"github.com/Arifinwidy02/splitmate-backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type envelope struct {
	Data any `json:"data"`
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	dashboard, err := h.service.GetDashboard(r.Context(), userID)
	if err != nil {
		slog.Error("dashboard request failed", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: dashboard})
}
