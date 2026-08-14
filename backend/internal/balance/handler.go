package balance

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
	"github.com/Arifinwidy02/splitmate-backend/pkg/money"
	"github.com/Arifinwidy02/splitmate-backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type memberBalanceResponse struct {
	UserID  uuid.UUID `json:"userId"`
	Name    string    `json:"name"`
	Balance string    `json:"balance"`
}

type settlementResponse struct {
	FromUserID uuid.UUID `json:"fromUserId"`
	ToUserID   uuid.UUID `json:"toUserId"`
	Amount     string    `json:"amount"`
}

type personalBalanceResponse struct {
	OwedToUser string `json:"owedToUser"`
	UserOwes   string `json:"userOwes"`
	NetBalance string `json:"netBalance"`
}

type envelope struct {
	Data any `json:"data"`
}

func (h *Handler) GroupBalances(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid group id")
		return
	}

	balances, err := h.service.GroupBalances(r.Context(), userID, groupID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	members := make([]memberBalanceResponse, 0, len(balances))
	for _, b := range balances {
		members = append(members, memberBalanceResponse{
			UserID:  b.UserID,
			Name:    b.Name,
			Balance: money.FormatMajor(b.BalanceSen),
		})
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: map[string]any{"members": members}})
}

func (h *Handler) SettlementSuggestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid group id")
		return
	}

	transfers, err := h.service.SettlementSuggestions(r.Context(), userID, groupID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	settlements := make([]settlementResponse, 0, len(transfers))
	for _, tr := range transfers {
		settlements = append(settlements, settlementResponse{
			FromUserID: tr.FromUserID,
			ToUserID:   tr.ToUserID,
			Amount:     money.FormatMajor(tr.AmountSen),
		})
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: map[string]any{"settlements": settlements}})
}

func (h *Handler) PersonalBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	summary, err := h.service.PersonalBalance(r.Context(), userID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: personalBalanceResponse{
		OwedToUser: money.FormatMajor(summary.OwedToUserSen),
		UserOwes:   money.FormatMajor(summary.UserOwesSen),
		NetBalance: money.FormatMajor(summary.NetBalanceSen),
	}})
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrGroupNotFound):
		response.WriteError(w, http.StatusNotFound, "GROUP_NOT_FOUND", "Group not found")
	default:
		slog.Error("balance request failed", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
	}
}

func pathUUID(r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
