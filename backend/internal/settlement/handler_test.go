package settlement

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/group"
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
	if len(segs) == 3 && segs[0] == "groups" && segs[2] == "settlements" {
		r.SetPathValue("groupId", segs[1])
	}
}

func decodeData(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rec.Body.String())
	}
}

func memberMap(gs *fakeGroupStore, groupID uuid.UUID, userIDs ...uuid.UUID) {
	for _, id := range userIDs {
		gs.memberships[groupID.String()+":"+id.String()] = group.RoleMember
	}
}

func TestCreateSettlementHandler(t *testing.T) {
	h, _, gs := newTestHandler()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	memberMap(gs, g, a, b)

	body := map[string]any{
		"payerId":    a.String(),
		"receiverId": b.String(),
		"amount":     "400000.00",
		"settledAt":  "2026-08-14T19:00:00+07:00",
	}

	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.String()+"/settlements", body, a)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			ID         uuid.UUID `json:"id"`
			PayerID    uuid.UUID `json:"payerId"`
			PayerName  string    `json:"payerName"`
			ReceiverID uuid.UUID `json:"receiverId"`
			Amount     string    `json:"amount"`
			SettledAt  string    `json:"settledAt"`
			CreatedAt  string    `json:"createdAt"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)

	if resp.Data.ID == uuid.Nil {
		t.Error("expected settlement id")
	}
	if resp.Data.PayerID != a {
		t.Errorf("expected payer %v, got %v", a, resp.Data.PayerID)
	}
	if resp.Data.ReceiverID != b {
		t.Errorf("expected receiver %v, got %v", b, resp.Data.ReceiverID)
	}
	if resp.Data.Amount != "400000.00" {
		t.Errorf("expected amount 400000.00, got %q", resp.Data.Amount)
	}
	if resp.Data.SettledAt == "" || resp.Data.CreatedAt == "" {
		t.Error("expected timestamps in response")
	}
}

func TestCreateSettlementHandlerForbidden(t *testing.T) {
	h, _, gs := newTestHandler()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	memberMap(gs, g, a, b)

	body := map[string]any{
		"payerId":    a.String(),
		"receiverId": b.String(),
		"amount":     "1000.00",
	}

	// b tries to record a settlement where a is the payer.
	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.String()+"/settlements", body, b)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateSettlementHandlerNonMember(t *testing.T) {
	h, _, gs := newTestHandler()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	memberMap(gs, g, a, b)

	body := map[string]any{
		"payerId":    a.String(),
		"receiverId": b.String(),
		"amount":     "1000.00",
	}

	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.String()+"/settlements", body, uuid.New())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateSettlementHandlerBadAmount(t *testing.T) {
	h, _, gs := newTestHandler()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	memberMap(gs, g, a, b)

	body := map[string]any{
		"payerId":    a.String(),
		"receiverId": b.String(),
		"amount":     "abc",
	}

	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.String()+"/settlements", body, a)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateSettlementHandlerMissingFields(t *testing.T) {
	h, _, gs := newTestHandler()
	g := uuid.New()
	a := uuid.New()
	memberMap(gs, g, a)

	body := map[string]any{
		"payerId": a.String(),
		"amount":  "1000.00",
	}

	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.String()+"/settlements", body, a)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateSettlementHandlerBadSettledAt(t *testing.T) {
	h, _, gs := newTestHandler()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	memberMap(gs, g, a, b)

	body := map[string]any{
		"payerId":    a.String(),
		"receiverId": b.String(),
		"amount":     "1000.00",
		"settledAt":  "yesterday",
	}

	rec := doRequest(t, h.Create, http.MethodPost, "/groups/"+g.String()+"/settlements", body, a)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestListSettlementsHandler(t *testing.T) {
	h, store, gs := newTestHandler()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	memberMap(gs, g, a, b)

	st := &Settlement{
		GroupID:      g,
		PayerID:      a,
		PayerName:    "Budi",
		ReceiverID:   b,
		ReceiverName: "Ani",
		AmountSen:    40000000,
	}
	if err := store.Create(nil, st); err != nil {
		t.Fatalf("seed settlement: %v", err)
	}

	rec := doRequest(t, h.List, http.MethodGet, "/groups/"+g.String()+"/settlements", nil, a)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Settlements []struct {
				PayerName    string `json:"payerName"`
				ReceiverName string `json:"receiverName"`
				Amount       string `json:"amount"`
			} `json:"settlements"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)

	if len(resp.Data.Settlements) != 1 {
		t.Fatalf("expected 1 settlement, got %d", len(resp.Data.Settlements))
	}
	if resp.Data.Settlements[0].Amount != "400000.00" {
		t.Errorf("expected amount 400000.00, got %q", resp.Data.Settlements[0].Amount)
	}
}

func TestListSettlementsHandlerEmpty(t *testing.T) {
	h, _, gs := newTestHandler()
	g := uuid.New()
	a := uuid.New()
	memberMap(gs, g, a)

	rec := doRequest(t, h.List, http.MethodGet, "/groups/"+g.String()+"/settlements", nil, a)
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

func TestListSettlementsHandlerNonMember(t *testing.T) {
	h, _, gs := newTestHandler()
	g := uuid.New()
	a := uuid.New()
	memberMap(gs, g, a)

	rec := doRequest(t, h.List, http.MethodGet, "/groups/"+g.String()+"/settlements", nil, uuid.New())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateSettlementHandlerInvalidGroupID(t *testing.T) {
	h, _, _ := newTestHandler()

	rec := doRequest(t, h.Create, http.MethodPost, "/groups/xyz/settlements", map[string]any{}, uuid.New())
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandlerRequiresUserID(t *testing.T) {
	h, _, _ := newTestHandler()

	r := httptest.NewRequest(http.MethodGet, "/groups/x/settlements", nil)
	rec := httptest.NewRecorder()
	h.List(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}
