package settlement

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
	"github.com/Arifinwidy02/splitmate-backend/pkg/money"
	"github.com/Arifinwidy02/splitmate-backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createSettlementRequest struct {
	PayerID    uuid.UUID `json:"payerId"`
	ReceiverID uuid.UUID `json:"receiverId"`
	Amount     string    `json:"amount"`
	SettledAt  string    `json:"settledAt"`
}

type settlementResponse struct {
	ID           uuid.UUID `json:"id"`
	PayerID      uuid.UUID `json:"payerId"`
	PayerName    string    `json:"payerName"`
	ReceiverID   uuid.UUID `json:"receiverId"`
	ReceiverName string    `json:"receiverName"`
	Amount       string    `json:"amount"`
	SettledAt    time.Time `json:"settledAt"`
	CreatedAt    time.Time `json:"createdAt"`
}

type envelope struct {
	Data any `json:"data"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req createSettlementRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		return
	}

	if req.PayerID == uuid.Nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Payer is required")
		return
	}
	if req.ReceiverID == uuid.Nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Receiver is required")
		return
	}

	amountSen, err := money.ParseMajor(req.Amount)
	if err != nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid amount")
		return
	}

	var settledAt time.Time
	if req.SettledAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.SettledAt)
		if err != nil {
			response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid settledAt, expected RFC3339")
			return
		}
		settledAt = parsed
	}

	st, err := h.service.CreateSettlement(r.Context(), userID, groupID, CreateSettlementInput{
		PayerID:    req.PayerID,
		ReceiverID: req.ReceiverID,
		AmountSen:  amountSen,
		SettledAt:  settledAt,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, envelope{Data: toSettlementResponse(st)})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
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

	settlements, err := h.service.ListSettlements(r.Context(), userID, groupID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	resp := make([]settlementResponse, 0, len(settlements))
	for _, st := range settlements {
		resp = append(resp, toSettlementResponse(st))
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: map[string]any{"settlements": resp}})
}

func toSettlementResponse(st *Settlement) settlementResponse {
	return settlementResponse{
		ID:           st.ID,
		PayerID:      st.PayerID,
		PayerName:    st.PayerName,
		ReceiverID:   st.ReceiverID,
		ReceiverName: st.ReceiverName,
		Amount:       money.FormatMajor(st.AmountSen),
		SettledAt:    st.SettledAt,
		CreatedAt:    st.CreatedAt,
	}
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	var valErr *apperror.Validation
	switch {
	case errors.Is(err, ErrGroupNotFound):
		response.WriteError(w, http.StatusNotFound, "GROUP_NOT_FOUND", "Group not found")
	case errors.Is(err, ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to do this")
	case errors.As(err, &valErr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", valErr.Message)
	default:
		slog.Error("settlement request failed", "error", err)
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
