package expense

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
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

type createExpenseRequest struct {
	Description  string                `json:"description"`
	Amount       string                `json:"amount"`
	Currency     string                `json:"currency"`
	PaidBy       string                `json:"paidBy"`
	Category     string                `json:"category"`
	ExpenseDate  string                `json:"expenseDate"`
	Note         string                `json:"note"`
	SplitType    string                `json:"splitType"`
	Participants []string              `json:"participants"`
	Splits       []expenseSplitRequest `json:"splits"`
}

type expenseSplitRequest struct {
	UserID string `json:"userId"`
	Amount string `json:"amount"`
}

type expenseResponse struct {
	ID           uuid.UUID             `json:"id"`
	GroupID      uuid.UUID             `json:"groupId"`
	Description  string                `json:"description"`
	Amount       string                `json:"amount"`
	Currency     string                `json:"currency"`
	PaidBy       uuid.UUID             `json:"paidBy"`
	PayerName    string                `json:"payerName"`
	CreatedBy    uuid.UUID             `json:"createdBy"`
	Category     string                `json:"category"`
	ExpenseDate  time.Time             `json:"expenseDate"`
	Note         *string               `json:"note"`
	Participants []participantResponse `json:"participants,omitempty"`
	CreatedAt    time.Time             `json:"createdAt"`
	UpdatedAt    time.Time             `json:"updatedAt"`
}

type participantResponse struct {
	UserID uuid.UUID `json:"userId"`
	Name   string    `json:"name,omitempty"`
	Amount string    `json:"amount"`
}

type summaryResponse struct {
	ID               uuid.UUID `json:"id"`
	GroupID          uuid.UUID `json:"groupId"`
	Description      string    `json:"description"`
	Amount           string    `json:"amount"`
	Currency         string    `json:"currency"`
	PaidBy           uuid.UUID `json:"paidBy"`
	PayerName        string    `json:"payerName"`
	CreatedBy        uuid.UUID `json:"createdBy"`
	Category         string    `json:"category"`
	ExpenseDate      time.Time `json:"expenseDate"`
	ParticipantCount int       `json:"participantCount"`
	HasReceipt       bool      `json:"hasReceipt"`
}

type expenseData struct {
	Expense *expenseResponse `json:"expense"`
}

type expensesData struct {
	Expenses []*summaryResponse `json:"expenses"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	Limit    int                `json:"limit"`
}

type emptyData struct{}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group id")
		return
	}

	var req createExpenseRequest
	receipt, err := decodeExpenseRequest(w, r, &req)
	if err != nil {
		return
	}

	input, err := parseExpenseInput(req)
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}
	input.Receipt = receipt

	e, err := h.service.CreateExpense(r.Context(), userID, groupID, input)
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, envelope{Data: expenseData{Expense: toExpenseResponse(e)}})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group id")
		return
	}

	page, limit, err := parsePagination(r)
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}

	category := r.URL.Query().Get("category")
	if category != "" && !isValidCategory(category) {
		h.writeExpenseError(w, &apperror.Validation{Message: "Unknown expense category"})
		return
	}

	from, err := parseTimeParam(r, "from")
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}
	to, err := parseTimeParam(r, "to")
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}

	expenses, total, err := h.service.ListExpenses(r.Context(), userID, groupID, page, limit, category, from, to)
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}

	resp := make([]*summaryResponse, 0, len(expenses))
	for _, e := range expenses {
		resp = append(resp, toSummaryResponse(e))
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: expensesData{Expenses: resp, Total: total, Page: page, Limit: limit}})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	expenseID, ok := pathUUID(r, "expenseId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid expense id")
		return
	}

	e, err := h.service.GetExpense(r.Context(), userID, expenseID)
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: expenseData{Expense: toExpenseResponse(e)}})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	expenseID, ok := pathUUID(r, "expenseId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid expense id")
		return
	}

	var req createExpenseRequest
	receipt, err := decodeExpenseRequest(w, r, &req)
	if err != nil {
		return
	}

	input, err := parseExpenseInput(req)
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}
	input.Receipt = receipt

	e, err := h.service.UpdateExpense(r.Context(), userID, expenseID, input)
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: expenseData{Expense: toExpenseResponse(e)}})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	expenseID, ok := pathUUID(r, "expenseId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid expense id")
		return
	}

	if err := h.service.DeleteExpense(r.Context(), userID, expenseID); err != nil {
		h.writeExpenseError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: emptyData{}})
}

func (h *Handler) GetReceipt(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	expenseID, ok := pathUUID(r, "expenseId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid expense id")
		return
	}

	image, contentType, err := h.service.GetReceipt(r.Context(), userID, expenseID)
	if err != nil {
		h.writeExpenseError(w, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(image); err != nil {
		slog.Error("failed to write receipt", "error", err)
	}
}

func (h *Handler) writeExpenseError(w http.ResponseWriter, err error) {
	var valErr *apperror.Validation
	switch {
	case errors.As(err, &valErr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", valErr.Message)
	case errors.Is(err, ErrInvalidSplit):
		response.WriteError(w, http.StatusUnprocessableEntity, "INVALID_SPLIT", "Expense split does not equal the total amount")
	case errors.Is(err, ErrExpenseNotFound):
		response.WriteError(w, http.StatusNotFound, "EXPENSE_NOT_FOUND", "Expense not found")
	case errors.Is(err, ErrNoReceipt):
		response.WriteError(w, http.StatusNotFound, "RECEIPT_NOT_FOUND", "Expense has no receipt")
	case errors.Is(err, ErrGroupNotFound):
		response.WriteError(w, http.StatusNotFound, "GROUP_NOT_FOUND", "Group not found")
	case errors.Is(err, ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to do this")
	default:
		slog.Error("expense request failed", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
	}
}

func decodeExpenseRequest(w http.ResponseWriter, r *http.Request, dst *createExpenseRequest) (*Receipt, error) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := response.DecodeJSON(w, r, dst); err != nil {
			return nil, err
		}
		return nil, nil
	}

	receipt, err := decodeMultipartRequest(r, dst)
	if err != nil {
		hErr := &apperror.Validation{Message: err.Error()}
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", hErr.Message)
		return nil, err
	}

	return receipt, nil
}

func decodeMultipartRequest(r *http.Request, dst *createExpenseRequest) (*Receipt, error) {
	if err := r.ParseMultipartForm(receiptFieldLimit); err != nil {
		return nil, errors.New("Invalid form data")
	}

	form := r.MultipartForm
	get := func(key string) string {
		if values := form.Value[key]; len(values) > 0 {
			return values[0]
		}
		return ""
	}
	dst.Description = get("description")
	dst.Amount = get("amount")
	dst.Currency = get("currency")
	dst.PaidBy = get("paidBy")
	dst.Category = get("category")
	dst.ExpenseDate = get("expenseDate")
	dst.Note = get("note")
	dst.SplitType = get("splitType")
	dst.Participants = form.Value["participant"]

	for key, values := range form.Value {
		if strings.HasPrefix(key, "split.") && len(values) > 0 {
			dst.Splits = append(dst.Splits, expenseSplitRequest{
				UserID: strings.TrimPrefix(key, "split."),
				Amount: values[0],
			})
		}
	}

	file, header, err := r.FormFile("receipt")
	if errors.Is(err, http.ErrMissingFile) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("Invalid receipt upload")
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxReceiptBytes+1))
	if err != nil {
		return nil, errors.New("Invalid receipt upload")
	}

	return &Receipt{
		Image:       data,
		ContentType: header.Header.Get("Content-Type"),
	}, nil
}

func parseExpenseInput(req createExpenseRequest) (CreateExpenseInput, error) {
	input := CreateExpenseInput{
		Description: req.Description,
		Currency:    strings.ToUpper(strings.TrimSpace(req.Currency)),
		Category:    strings.TrimSpace(req.Category),
		SplitType:   req.SplitType,
	}

	amountSen, err := money.ParseMajor(req.Amount)
	if err != nil {
		return input, &apperror.Validation{Message: "Amount must be a valid number with at most 2 decimal places"}
	}
	input.AmountSen = amountSen

	paidBy, err := uuid.Parse(req.PaidBy)
	if err != nil {
		return input, &apperror.Validation{Message: "Payer is required"}
	}
	input.PaidBy = paidBy

	expenseDate, err := time.Parse(time.RFC3339, req.ExpenseDate)
	if err != nil {
		return input, &apperror.Validation{Message: "Expense date must be a valid timestamp"}
	}
	input.ExpenseDate = expenseDate

	note := strings.TrimSpace(req.Note)
	if note != "" {
		input.Note = &note
	}

	switch req.SplitType {
	case SplitEqual:
		for _, id := range req.Participants {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return input, &apperror.Validation{Message: "Invalid participant id"}
			}
			input.EqualIDs = append(input.EqualIDs, parsed)
		}
	case SplitCustom:
		for _, sp := range req.Splits {
			parsed, err := uuid.Parse(sp.UserID)
			if err != nil {
				return input, &apperror.Validation{Message: "Invalid participant id"}
			}
			amountSen, err := money.ParseMajor(sp.Amount)
			if err != nil {
				return input, &apperror.Validation{Message: "Split amount must be a valid number with at most 2 decimal places"}
			}
			input.Splits = append(input.Splits, SplitAmount{UserID: parsed, AmountSen: amountSen})
		}
	}

	return input, nil
}

func parsePagination(r *http.Request) (int, int, error) {
	q := r.URL.Query()

	page := 1
	if v := q.Get("page"); v != "" {
		p, err := parsePositiveInt(v)
		if err != nil {
			return 0, 0, &apperror.Validation{Message: "page must be a positive integer"}
		}
		page = p
	}

	limit := 20
	if v := q.Get("limit"); v != "" {
		l, err := parsePositiveInt(v)
		if err != nil || l > 100 {
			return 0, 0, &apperror.Validation{Message: "limit must be between 1 and 100"}
		}
		limit = l
	}

	return page, limit, nil
}

func parsePositiveInt(s string) (int, error) {
	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(r-'0')
		if n > 1_000_000 {
			return 0, errors.New("too large")
		}
	}
	if n == 0 {
		return 0, errors.New("zero")
	}
	return n, nil
}

func parseTimeParam(r *http.Request, name string) (*time.Time, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, &apperror.Validation{Message: name + " must be a valid timestamp"}
	}
	return &t, nil
}

func toExpenseResponse(e *ExpenseWithSplits) *expenseResponse {
	resp := &expenseResponse{
		ID:          e.ID,
		GroupID:     e.GroupID,
		Description: e.Description,
		Amount:      money.FormatMajor(e.AmountSen),
		Currency:    e.Currency,
		PaidBy:      e.PaidBy,
		PayerName:   e.PayerName,
		CreatedBy:   e.CreatedBy,
		Category:    e.Category,
		ExpenseDate: e.ExpenseDate,
		Note:        e.Note,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}

	for _, p := range e.Participants {
		resp.Participants = append(resp.Participants, participantResponse{
			UserID: p.UserID,
			Name:   p.Name,
			Amount: money.FormatMajor(p.AmountSen),
		})
	}

	return resp
}

func toSummaryResponse(e *ExpenseSummary) *summaryResponse {
	return &summaryResponse{
		ID:               e.ID,
		GroupID:          e.GroupID,
		Description:      e.Description,
		Amount:           money.FormatMajor(e.AmountSen),
		Currency:         e.Currency,
		PaidBy:           e.PaidBy,
		PayerName:        e.PayerName,
		CreatedBy:        e.CreatedBy,
		Category:         e.Category,
		ExpenseDate:      e.ExpenseDate,
		ParticipantCount: e.ParticipantCount,
		HasReceipt:       e.HasReceipt,
	}
}

func pathUUID(r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

type envelope struct {
	Data any `json:"data"`
}
