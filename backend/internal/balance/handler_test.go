package balance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
)

func newTestHandler() (*Handler, *fakeStore, *fakeGroupStore) {
	svc, store, gs := newTestService()
	return NewHandler(svc), store, gs
}

func doRequest(t *testing.T, fn func(http.ResponseWriter, *http.Request), path string, userID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, path, nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), userID))
	setPathValues(r)

	rec := httptest.NewRecorder()
	fn(rec, r)
	return rec
}

func setPathValues(r *http.Request) {
	segs := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case len(segs) == 3 && segs[0] == "groups" && (segs[2] == "balances" || segs[2] == "settlement-suggestions"):
		r.SetPathValue("groupId", segs[1])
	}
}

func decodeData(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rec.Body.String())
	}
}

func TestGroupBalancesHandler(t *testing.T) {
	h, store, gs := newTestHandler()
	a, b := uuid.New(), uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin", b: "Budi"}, g)

	store.expenses[g] = []Expense{
		{PaidBy: a, AmountSen: 60000000, Splits: []Split{
			{UserID: a, AmountSen: 30000000},
			{UserID: b, AmountSen: 30000000},
		}},
	}

	rec := doRequest(t, h.GroupBalances, "/groups/"+g.String()+"/balances", b)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Members []struct {
				UserID  string `json:"userId"`
				Name    string `json:"name"`
				Balance string `json:"balance"`
			} `json:"members"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)

	if len(resp.Data.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(resp.Data.Members))
	}

	byName := map[string]string{}
	for _, m := range resp.Data.Members {
		byName[m.Name] = m.Balance
	}
	if byName["Arifin"] != "300000.00" {
		t.Errorf("expected Arifin 300000.00, got %q", byName["Arifin"])
	}
	if byName["Budi"] != "-300000.00" {
		t.Errorf("expected Budi -300000.00, got %q", byName["Budi"])
	}
}

func TestGroupBalancesHandlerNonMember(t *testing.T) {
	h, _, gs := newTestHandler()
	a := uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin"}, g)

	rec := doRequest(t, h.GroupBalances, "/groups/"+g.String()+"/balances", uuid.New())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var eb struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeData(t, rec, &eb)
	if eb.Error.Code != "GROUP_NOT_FOUND" {
		t.Errorf("expected GROUP_NOT_FOUND, got %q", eb.Error.Code)
	}
}

func TestGroupBalancesHandlerInvalidGroupID(t *testing.T) {
	h, _, _ := newTestHandler()

	rec := doRequest(t, h.GroupBalances, "/groups/not-a-uuid/balances", uuid.New())
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestSettlementSuggestionsHandler(t *testing.T) {
	h, store, gs := newTestHandler()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin", b: "Budi", c: "Citra"}, g)

	store.expenses[g] = []Expense{
		{PaidBy: a, AmountSen: 70000000, Splits: []Split{
			{UserID: a, AmountSen: 0},
			{UserID: b, AmountSen: 40000000},
			{UserID: c, AmountSen: 30000000},
		}},
	}

	rec := doRequest(t, h.SettlementSuggestions, "/groups/"+g.String()+"/settlement-suggestions", b)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Settlements []struct {
				FromUserID string `json:"fromUserId"`
				ToUserID   string `json:"toUserId"`
				Amount     string `json:"amount"`
			} `json:"settlements"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)

	if len(resp.Data.Settlements) != 2 {
		t.Fatalf("expected 2 settlements, got %d", len(resp.Data.Settlements))
	}
	if resp.Data.Settlements[0].FromUserID != b.String() ||
		resp.Data.Settlements[0].ToUserID != a.String() ||
		resp.Data.Settlements[0].Amount != "400000.00" {
		t.Errorf("unexpected first settlement: %+v", resp.Data.Settlements[0])
	}
}

func TestSettlementSuggestionsHandlerEmpty(t *testing.T) {
	h, _, gs := newTestHandler()
	a := uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin"}, g)

	rec := doRequest(t, h.SettlementSuggestions, "/groups/"+g.String()+"/settlement-suggestions", a)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Settlements []any `json:"settlements"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)
	if resp.Data.Settlements == nil || len(resp.Data.Settlements) != 0 {
		t.Errorf("expected empty settlements array, got %v", resp.Data.Settlements)
	}
}

func TestPersonalBalanceHandler(t *testing.T) {
	h, store, gs := newTestHandler()
	a, b := uuid.New(), uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin", b: "Budi"}, g)

	store.expenses[g] = []Expense{
		{PaidBy: a, AmountSen: 60000000, Splits: []Split{
			{UserID: a, AmountSen: 30000000},
			{UserID: b, AmountSen: 30000000},
		}},
	}

	rec := doRequest(t, h.PersonalBalance, "/me/balance", a)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			OwedToUser string `json:"owedToUser"`
			UserOwes   string `json:"userOwes"`
			NetBalance string `json:"netBalance"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)

	if resp.Data.OwedToUser != "300000.00" || resp.Data.UserOwes != "0.00" || resp.Data.NetBalance != "300000.00" {
		t.Errorf("unexpected summary: %+v", resp.Data)
	}
}

func TestHandlerRequiresUserID(t *testing.T) {
	h, _, _ := newTestHandler()

	r := httptest.NewRequest(http.MethodGet, "/groups/x/balances", nil)
	rec := httptest.NewRecorder()
	h.GroupBalances(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}
