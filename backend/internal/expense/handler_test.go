package expense

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
)

func newTestHandler() (*Handler, *fakeStore, *fakeGroupStore) {
	svc, store, gs := newTestService()
	return NewHandler(svc), store, gs
}

func doRequest(t *testing.T, fn func(http.ResponseWriter, *http.Request), method, path string, body any, userID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	r := httptest.NewRequest(method, path, &buf)
	r = r.WithContext(middleware.WithUserID(r.Context(), userID))
	setPathValues(r)

	rec := httptest.NewRecorder()
	fn(rec, r)
	return rec
}

func setPathValues(r *http.Request) {
	segs := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case len(segs) == 2 && segs[0] == "expenses":
		r.SetPathValue("expenseId", segs[1])
	case len(segs) == 3 && segs[0] == "expenses" && segs[2] == "receipt":
		r.SetPathValue("expenseId", segs[1])
	case len(segs) == 3 && segs[0] == "groups" && segs[2] == "expenses":
		r.SetPathValue("groupId", segs[1])
	}
}

func decodeData(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rec.Body.String())
	}
}

type errorBody struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func newTestGroupWithMembers(t *testing.T, h *Handler, gs *fakeGroupStore, members ...uuid.UUID) *testGroup {
	t.Helper()
	g := setupGroup(t, gs, members...)
	return &testGroup{id: g.ID}
}

type testGroup struct {
	id uuid.UUID
}

func (g *testGroup) String() string {
	return g.id.String()
}

func equalBody(g *testGroup, payer uuid.UUID, amount string, participants ...uuid.UUID) map[string]any {
	if len(participants) == 0 {
		participants = []uuid.UUID{payer}
	}

	ids := make([]string, 0, len(participants))
	for _, p := range participants {
		ids = append(ids, p.String())
	}

	return map[string]any{
		"description":  "Dinner",
		"amount":       amount,
		"currency":     "IDR",
		"paidBy":       payer.String(),
		"category":     "Food & Drinks",
		"expenseDate":  "2026-08-14T19:00:00+07:00",
		"splitType":    "equal",
		"participants": ids,
	}
}

func TestCreateExpenseHandlerEqual(t *testing.T) {
	h, _, gs := newTestHandler()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.ID.String()+"/expenses", equalBody(&testGroup{id: g.ID}, a, "600000", a, b), a)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Expense struct {
				ID        uuid.UUID `json:"id"`
				Amount    string    `json:"amount"`
				PayerName string    `json:"payerName"`
			} `json:"expense"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)
	if resp.Data.Expense.Amount != "600000.00" {
		t.Errorf("expected amount 600000.00, got %q", resp.Data.Expense.Amount)
	}
}

func TestCreateExpenseHandlerCustom(t *testing.T) {
	h, _, gs := newTestHandler()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b, c)

	body := map[string]any{
		"description": "Dinner",
		"amount":      "600000.50",
		"currency":    "IDR",
		"paidBy":      a.String(),
		"category":    "Food & Drinks",
		"expenseDate": "2026-08-14T19:00:00+07:00",
		"splitType":   "custom",
		"splits": []map[string]any{
			{"userId": a.String(), "amount": "100000.50"},
			{"userId": b.String(), "amount": "250000"},
			{"userId": c.String(), "amount": "250000"},
		},
	}

	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.ID.String()+"/expenses", body, a)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateExpenseHandlerInvalidSplit(t *testing.T) {
	h, _, gs := newTestHandler()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b, c)

	body := map[string]any{
		"description": "Dinner",
		"amount":      "600000",
		"currency":    "IDR",
		"paidBy":      a.String(),
		"category":    "Food & Drinks",
		"expenseDate": "2026-08-14T19:00:00+07:00",
		"splitType":   "custom",
		"splits": []map[string]any{
			{"userId": a.String(), "amount": "100000"},
			{"userId": b.String(), "amount": "200000"},
			{"userId": c.String(), "amount": "200000"},
		},
	}

	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.ID.String()+"/expenses", body, a)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var eb errorBody
	decodeData(t, rec, &eb)
	if eb.Error.Code != "INVALID_SPLIT" {
		t.Errorf("expected INVALID_SPLIT, got %q", eb.Error.Code)
	}
}

func TestCreateExpenseHandlerBadAmount(t *testing.T) {
	h, _, gs := newTestHandler()
	a := uuid.New()
	g := setupGroup(t, gs, a)

	body := equalBody(&testGroup{id: g.ID}, a, "abc")
	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.ID.String()+"/expenses", body, a)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateExpenseHandlerNonMember(t *testing.T) {
	h, _, gs := newTestHandler()
	a := uuid.New()
	g := setupGroup(t, gs, a)

	outsider := uuid.New()
	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.ID.String()+"/expenses", equalBody(&testGroup{id: g.ID}, a, "1000"), outsider)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var eb errorBody
	decodeData(t, rec, &eb)
	if eb.Error.Code != "GROUP_NOT_FOUND" {
		t.Errorf("expected GROUP_NOT_FOUND, got %q", eb.Error.Code)
	}
}

func TestGetExpenseHandlerNotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	rec := doRequest(t, h.Get, http.MethodGet, "/expenses/"+uuid.New().String(), nil, uuid.New())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestGetExpenseHandler(t *testing.T) {
	h, _, gs := newTestHandler()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	created := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.ID.String()+"/expenses", equalBody(&testGroup{id: g.ID}, a, "600000", a, b), a)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", created.Code, created.Body.String())
	}

	var createdResp struct {
		Data struct {
			Expense struct {
				ID uuid.UUID `json:"id"`
			} `json:"expense"`
		} `json:"data"`
	}
	decodeData(t, created, &createdResp)

	rec := doRequest(t, h.Get, http.MethodGet, "/expenses/"+createdResp.Data.Expense.ID.String(), nil, b)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Expense struct {
				Participants []struct {
					UserID uuid.UUID `json:"userId"`
					Amount string    `json:"amount"`
				} `json:"participants"`
			} `json:"expense"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)
	if len(resp.Data.Expense.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(resp.Data.Expense.Participants))
	}
	for _, p := range resp.Data.Expense.Participants {
		if p.Amount != "300000.00" {
			t.Errorf("expected 300000.00 share, got %q", p.Amount)
		}
	}
}

func TestUpdateExpenseHandlerForbidden(t *testing.T) {
	h, _, gs := newTestHandler()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	created := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.ID.String()+"/expenses", equalBody(&testGroup{id: g.ID}, a, "600000", a, b), a)
	var createdResp struct {
		Data struct {
			Expense struct {
				ID uuid.UUID `json:"id"`
			} `json:"expense"`
		} `json:"data"`
	}
	decodeData(t, created, &createdResp)

	rec := doRequest(t, h.Update, http.MethodPatch, "/expenses/"+createdResp.Data.Expense.ID.String(), equalBody(&testGroup{id: g.ID}, a, "700000"), b)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestDeleteExpenseHandler(t *testing.T) {
	h, _, gs := newTestHandler()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	created := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.ID.String()+"/expenses", equalBody(&testGroup{id: g.ID}, a, "600000", a, b), a)
	var createdResp struct {
		Data struct {
			Expense struct {
				ID uuid.UUID `json:"id"`
			} `json:"expense"`
		} `json:"data"`
	}
	decodeData(t, created, &createdResp)

	rec := doRequest(t, h.Delete, http.MethodDelete, "/expenses/"+createdResp.Data.Expense.ID.String(), nil, b)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-creator, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, h.Delete, http.MethodDelete, "/expenses/"+createdResp.Data.Expense.ID.String(), nil, a)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for creator, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestListExpensesHandler(t *testing.T) {
	h, _, gs := newTestHandler()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	doRequest(t, h.Create, http.MethodPost, "/groups/"+g.ID.String()+"/expenses", equalBody(&testGroup{id: g.ID}, a, "600000", a, b), a)

	rec := doRequest(t, h.List, http.MethodGet, "/groups/"+g.ID.String()+"/expenses", nil, b)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Expenses []struct {
				ID               string `json:"id"`
				ParticipantCount int    `json:"participantCount"`
			} `json:"expenses"`
			Total int `json:"total"`
			Page  int `json:"page"`
			Limit int `json:"limit"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)
	if resp.Data.Total != 1 || len(resp.Data.Expenses) != 1 {
		t.Fatalf("expected 1 expense, got %d (total %d)", len(resp.Data.Expenses), resp.Data.Total)
	}
	if resp.Data.Page != 1 || resp.Data.Limit != 20 {
		t.Errorf("expected default pagination 1/20, got %d/%d", resp.Data.Page, resp.Data.Limit)
	}
	if resp.Data.Expenses[0].ParticipantCount != 2 {
		t.Errorf("expected 2 participants, got %d", resp.Data.Expenses[0].ParticipantCount)
	}
}

func TestListExpensesHandlerBadPagination(t *testing.T) {
	h, _, gs := newTestHandler()
	a := uuid.New()
	g := setupGroup(t, gs, a)

	rec := doRequest(t, h.List, http.MethodGet, "/groups/"+g.ID.String()+"/expenses?limit=1000", nil, a)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandlerRequiresUserID(t *testing.T) {
	h, _, _ := newTestHandler()

	r := httptest.NewRequest(http.MethodGet, "/groups/x/expenses", nil)
	rec := httptest.NewRecorder()
	h.List(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestListExpensesHandlerInvalidDateFilter(t *testing.T) {
	h, _, gs := newTestHandler()
	a := uuid.New()
	g := setupGroup(t, gs, a)

	rec := doRequest(t, h.List, http.MethodGet, "/groups/"+g.ID.String()+"/expenses?from=notadate", nil, a)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateExpenseHandlerMultipartWithReceipt(t *testing.T) {
	h, store, gs := newTestHandler()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range map[string]string{
		"description": "Dinner",
		"amount":      "600000",
		"currency":    "IDR",
		"paidBy":      a.String(),
		"category":    "Food & Drinks",
		"expenseDate": "2026-08-14T19:00:00+07:00",
		"splitType":   "equal",
	} {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	for _, id := range []uuid.UUID{a, b} {
		if err := mw.WriteField("participant", id.String()); err != nil {
			t.Fatalf("write participant: %v", err)
		}
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="receipt"; filename="receipt.jpg"`)
	partHeader.Set("Content-Type", "image/jpeg")
	part, err := mw.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("fake-jpeg")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/groups/"+g.ID.String()+"/expenses", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r = r.WithContext(middleware.WithUserID(r.Context(), a))
	setPathValues(r)

	rec := httptest.NewRecorder()
	h.Create(rec, r)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var stored *Expense
	for _, e := range store.expenses {
		stored = e
	}
	if stored == nil {
		t.Fatal("expected expense to be stored")
	}
	if !bytes.Equal(stored.ReceiptImage, []byte("fake-jpeg")) {
		t.Errorf("expected receipt bytes stored, got %q", stored.ReceiptImage)
	}
	if stored.ReceiptContentType != "image/jpeg" {
		t.Errorf("expected content type image/jpeg, got %q", stored.ReceiptContentType)
	}
}

func TestGetReceiptHandler(t *testing.T) {
	h, store, gs := newTestHandler()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	withReceipt := &Expense{
		ID:                 uuid.New(),
		GroupID:            g.ID,
		Description:        "Dinner",
		AmountSen:          10000,
		Currency:           "IDR",
		PaidBy:             a,
		Category:           "Food & Drinks",
		ExpenseDate:        time.Now(),
		CreatedBy:          a,
		ReceiptImage:       []byte("img-bytes"),
		ReceiptContentType: "image/jpeg",
	}
	store.expenses[withReceipt.ID] = withReceipt

	rec := doRequest(t, h.GetReceipt, http.MethodGet, "/expenses/"+withReceipt.ID.String()+"/receipt", nil, b)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "img-bytes" {
		t.Errorf("expected receipt body, got %q", rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected image/jpeg content type, got %q", rec.Header().Get("Content-Type"))
	}

	noReceipt := &Expense{
		ID:          uuid.New(),
		GroupID:     g.ID,
		Description: "No receipt",
		AmountSen:   10000,
		Currency:    "IDR",
		PaidBy:      a,
		Category:    "Other",
		ExpenseDate: time.Now(),
		CreatedBy:   a,
	}
	store.expenses[noReceipt.ID] = noReceipt

	rec = doRequest(t, h.GetReceipt, http.MethodGet, "/expenses/"+noReceipt.ID.String()+"/receipt", nil, b)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing receipt, got %d", rec.Code)
	}
	var errResp errorBody
	decodeData(t, rec, &errResp)
	if errResp.Error.Code != "RECEIPT_NOT_FOUND" {
		t.Errorf("expected RECEIPT_NOT_FOUND, got %q", errResp.Error.Code)
	}

	outsider := uuid.New()
	rec = doRequest(t, h.GetReceipt, http.MethodGet, "/expenses/"+withReceipt.ID.String()+"/receipt", nil, outsider)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-member, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, h.GetReceipt, http.MethodGet, "/expenses/"+uuid.New().String()+"/receipt", nil, a)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing expense, got %d", rec.Code)
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		query     string
		wantPage  int
		wantLimit int
		wantErr   bool
	}{
		{"", 1, 20, false},
		{"?page=3", 3, 20, false},
		{"?limit=50", 1, 50, false},
		{"?page=2&limit=10", 2, 10, false},
		{"?page=0", 0, 0, true},
		{"?page=abc", 0, 0, true},
		{"?limit=0", 0, 0, true},
		{"?limit=101", 0, 0, true},
		{"?page=-1", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/groups/x/expenses"+tt.query, nil)
			page, limit, err := parsePagination(r)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %d/%d", page, limit)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if page != tt.wantPage || limit != tt.wantLimit {
				t.Errorf("expected %d/%d, got %d/%d", tt.wantPage, tt.wantLimit, page, limit)
			}
		})
	}
}
