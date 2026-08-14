package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/balance"
	"github.com/Arifinwidy02/splitmate-backend/internal/group"
	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
)

func doRequest(t *testing.T, fn func(http.ResponseWriter, *http.Request), userID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), userID))

	rec := httptest.NewRecorder()
	fn(rec, r)
	return rec
}

func TestGetHandler(t *testing.T) {
	svc, dstore, gs, bs := newTestService()
	h := NewHandler(svc)

	me, a := uuid.New(), uuid.New()
	g := uuid.New()
	gs.groups = []*group.Group{
		{ID: g, Name: "Bali", Currency: "IDR", MemberCount: 2},
	}
	gs.memberships[g.String()+":"+me.String()] = true
	bs.expenses[g] = []balance.Expense{
		{PaidBy: me, AmountSen: 60000000, Splits: []balance.Split{
			{UserID: me, AmountSen: 30000000},
			{UserID: a, AmountSen: 30000000},
		}},
	}
	dstore.totalExpense = 60000000

	rec := doRequest(t, h.Get, me)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Summary struct {
				OwedToUser    string `json:"owedToUser"`
				UserOwes      string `json:"userOwes"`
				NetBalance    string `json:"netBalance"`
				TotalExpense  string `json:"totalExpense"`
				SettledAmount string `json:"settledAmount"`
			} `json:"summary"`
			Groups         []any `json:"groups"`
			RecentExpenses []any `json:"recentExpenses"`
			Categories     []any `json:"categories"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Data.Summary.OwedToUser != "300000.00" {
		t.Errorf("expected owedToUser 300000.00, got %s", resp.Data.Summary.OwedToUser)
	}
	if resp.Data.Summary.NetBalance != "300000.00" {
		t.Errorf("expected netBalance 300000.00, got %s", resp.Data.Summary.NetBalance)
	}
	if resp.Data.Summary.TotalExpense != "600000.00" {
		t.Errorf("expected totalExpense 600000.00, got %s", resp.Data.Summary.TotalExpense)
	}
	if len(resp.Data.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(resp.Data.Groups))
	}
	if resp.Data.RecentExpenses == nil || resp.Data.Categories == nil {
		t.Errorf("expected non-nil arrays, got %v %v", resp.Data.RecentExpenses, resp.Data.Categories)
	}
}

func TestGetHandlerRequiresUserID(t *testing.T) {
	h := NewHandler(&Service{})

	r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestGetHandlerInternalError(t *testing.T) {
	svc, dstore, _, _ := newTestService()
	h := NewHandler(svc)
	dstore.failTotal = &errFail{}

	rec := doRequest(t, h.Get, uuid.New())
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var eb struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &eb); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if eb.Error.Code != "INTERNAL" {
		t.Errorf("expected INTERNAL, got %q", eb.Error.Code)
	}
}

type errFail struct{}

func (e *errFail) Error() string { return "boom" }
