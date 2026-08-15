package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Arifinwidy02/splitmate-backend/internal/auth"
	"github.com/Arifinwidy02/splitmate-backend/internal/database"
	"github.com/Arifinwidy02/splitmate-backend/internal/server"
	"github.com/Arifinwidy02/splitmate-backend/internal/session"
	"github.com/Arifinwidy02/splitmate-backend/internal/user"
	"github.com/Arifinwidy02/splitmate-backend/migrations"
)

const (
	mainURL  = "postgres://splitmate:splitmate@localhost:5433/splitmate?sslmode=disable"
	testDB   = "splitmate_integration"
	adminURL = "postgres://splitmate:splitmate@localhost:5433/postgres?sslmode=disable"
)

var (
	handler   http.Handler
	serverURL string
)

func TestMain(m *testing.M) {
	main := os.Getenv("SPLITMATE_TEST_DATABASE_URL")
	if main == "" {
		main = mainURL
	}

	admin := os.Getenv("SPLITMATE_TEST_ADMIN_URL")
	if admin == "" {
		admin = adminURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cleanup, testURL, err := setupTestDB(ctx, admin, main)
	if err != nil {
		fmt.Printf("SKIP: integration tests require PostgreSQL (%v)\n", err)
		os.Exit(0)
	}
	defer cleanup()

	pool, err := database.Connect(ctx, testURL)
	if err != nil {
		fmt.Printf("SKIP: cannot connect to test database (%v)\n", err)
		os.Exit(0)
	}
	defer pool.Close()

	tokenService := session.NewTokenService([]byte("integration-test-secret"), session.DefaultTokenTTL)
	handler = server.New(server.Dependencies{
		Pool:          pool,
		TokenService:  tokenService,
		SecureCookies: false,
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()
	serverURL = ts.URL

	os.Exit(m.Run())
}

// setupTestDB creates a dedicated test database and applies migrations.
func setupTestDB(ctx context.Context, adminURL, mainURL string) (func(), string, error) {
	u, err := url.Parse(mainURL)
	if err != nil {
		return nil, "", err
	}
	u.Path = "/" + testDB
	testURL := u.String()

	adminPool, err := database.Connect(ctx, adminURL)
	if err != nil {
		return nil, "", err
	}
	defer adminPool.Close()

	if _, err := adminPool.Exec(ctx, "DROP DATABASE IF EXISTS "+testDB); err != nil {
		return nil, "", fmt.Errorf("drop test db: %w", err)
	}
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+testDB); err != nil {
		return nil, "", fmt.Errorf("create test db: %w", err)
	}

	pool, err := database.Connect(ctx, testURL)
	if err != nil {
		return nil, "", fmt.Errorf("connect test db: %w", err)
	}

	list, err := database.ParseMigrations(migrations.Files)
	if err != nil {
		pool.Close()
		return nil, "", fmt.Errorf("parse migrations: %w", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := database.Migrate(ctx, pool, list, logger); err != nil {
		pool.Close()
		return nil, "", fmt.Errorf("migrate test db: %w", err)
	}
	pool.Close()

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		adminPool, err := database.Connect(ctx, adminURL)
		if err == nil {
			defer adminPool.Close()
			_, _ = adminPool.Exec(ctx, "DROP DATABASE IF EXISTS "+testDB)
		}
	}, testURL, nil
}

type apiClient struct {
	t      *testing.T
	jar    *cookiejar.Jar
	client *http.Client
}

func newClient(t *testing.T) *apiClient {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &apiClient{t: t, jar: jar, client: &http.Client{Jar: jar}}
}

func (c *apiClient) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, serverURL+path, reader)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		c.t.Fatalf("invalid JSON response for %s %s: %s", method, path, raw)
	}
	return resp.StatusCode, parsed
}

// doRaw performs a request and returns the raw response for non-JSON
// endpoints (e.g. file downloads).
func (c *apiClient) doRaw(method, path string) (int, http.Header, []byte) {
	c.t.Helper()

	req, err := http.NewRequest(method, serverURL+path, nil)
	if err != nil {
		c.t.Fatal(err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	return resp.StatusCode, resp.Header.Clone(), raw
}

func (c *apiClient) expect(method, path string, body any, wantStatus int) map[string]any {
	c.t.Helper()
	status, parsed := c.do(method, path, body)
	if status != wantStatus {
		c.t.Fatalf("%s %s: expected status %d, got %d (body: %v)", method, path, wantStatus, status, parsed)
	}
	return parsed
}

func (c *apiClient) register(name, email string) string {
	c.t.Helper()
	resp := c.expect(http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     name,
		"email":    email,
		"password": "password123",
	}, http.StatusCreated)
	c.login(email)
	return resp["data"].(map[string]any)["user"].(map[string]any)["id"].(string)
}

func (c *apiClient) login(email string) {
	c.t.Helper()
	c.expect(http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": "password123",
	}, http.StatusOK)
}

func (c *apiClient) createGroup(name string) string {
	c.t.Helper()
	resp := c.expect(http.MethodPost, "/api/v1/groups", map[string]any{
		"name":     name,
		"currency": "IDR",
	}, http.StatusCreated)
	group := resp["data"].(map[string]any)["group"].(map[string]any)
	return group["id"].(string)
}

func (c *apiClient) inviteMember(groupID, email string) string {
	c.t.Helper()
	resp := c.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/invitations", map[string]any{
		"email": email,
	}, http.StatusCreated)
	inv := resp["data"].(map[string]any)["invitation"].(map[string]any)
	return inv["token"].(string)
}

func (c *apiClient) bulkInvite(groupID string, emails []string) ([]string, map[string]string) {
	c.t.Helper()
	resp := c.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/invitations/bulk", map[string]any{
		"emails": emails,
	}, http.StatusCreated)
	data := resp["data"].(map[string]any)

	var tokens []string
	for _, inv := range data["invitations"].([]any) {
		tokens = append(tokens, inv.(map[string]any)["token"].(string))
	}
	failures := map[string]string{}
	for _, f := range data["failed"].([]any) {
		fail := f.(map[string]any)
		failures[fail["email"].(string)] = fail["reason"].(string)
	}
	return tokens, failures
}

func (c *apiClient) acceptInvitation(token string) string {
	c.t.Helper()
	resp := c.expect(http.MethodPost, "/api/v1/groups/invitations/"+token+"/accept", nil, http.StatusOK)
	group := resp["data"].(map[string]any)["group"].(map[string]any)
	return group["id"].(string)
}

func (c *apiClient) createExpense(groupID string, e map[string]any) map[string]any {
	c.t.Helper()
	resp := c.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/expenses", e, http.StatusCreated)
	return resp["data"].(map[string]any)["expense"].(map[string]any)
}

func (c *apiClient) balances(groupID string) map[string]string {
	c.t.Helper()
	resp := c.expect(http.MethodGet, "/api/v1/groups/"+groupID+"/balances", nil, http.StatusOK)
	out := map[string]string{}
	for _, m := range resp["data"].(map[string]any)["members"].([]any) {
		member := m.(map[string]any)
		out[member["name"].(string)] = member["balance"].(string)
	}
	return out
}

func (c *apiClient) createSettlement(groupID string, s map[string]any) {
	c.t.Helper()
	c.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/settlements", s, http.StatusCreated)
}

func (c *apiClient) memberIDs(groupID string) map[string]string {
	c.t.Helper()
	resp := c.expect(http.MethodGet, "/api/v1/groups/"+groupID+"/members", nil, http.StatusOK)
	out := map[string]string{}
	for _, m := range resp["data"].(map[string]any)["members"].([]any) {
		member := m.(map[string]any)
		out[member["email"].(string)] = member["id"].(string)
	}
	return out
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@test.local", prefix, time.Now().UnixNano())
}

// TestOAuthFindOrCreate exercises the Google find-or-create path against the
// real database: new oauth user (NULL password_hash), re-login, email linking,
// and password login rejection for oauth-only users.
func TestOAuthFindOrCreate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	u, err := url.Parse(mainURL)
	if err != nil {
		t.Fatalf("parse main url: %v", err)
	}
	u.Path = "/" + testDB
	pool, err := database.Connect(ctx, u.String())
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	defer pool.Close()

	userRepo := user.NewRepository(pool)
	authService := auth.NewService(userRepo)

	email := uniqueEmail("oauth")
	providerID := fmt.Sprintf("google-acc-%d", time.Now().UnixNano())

	// New user created via oauth must succeed (regression: NULL password_hash).
	created, err := authService.FindOrCreateByOAuth(ctx, "google", providerID, email, "OAuth User", nil)
	if err != nil {
		t.Fatalf("find or create oauth user: %v", err)
	}
	if created.Email != email {
		t.Errorf("expected email %q, got %q", email, created.Email)
	}
	if created.PasswordHash != "" {
		t.Error("oauth user must not have a password hash")
	}

	// Same account again resolves to the same user.
	again, err := authService.FindOrCreateByOAuth(ctx, "google", providerID, email, "OAuth User", nil)
	if err != nil {
		t.Fatalf("second find or create: %v", err)
	}
	if again.ID != created.ID {
		t.Errorf("expected same user, got %s and %s", created.ID, again.ID)
	}

	// OAuth-only user cannot sign in with a password.
	if _, err := authService.Login(ctx, email, "password123"); err != auth.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for oauth-only user, got %v", err)
	}

	// A password user with the same email gets linked to the oauth account.
	registered, err := authService.Register(ctx, "Password User", email, "password123")
	if err != nil {
		t.Fatalf("register password user: %v", err)
	}
	linked, err := authService.FindOrCreateByOAuth(ctx, "google", providerID, email, "OAuth User", nil)
	if err != nil {
		t.Fatalf("find or create after register: %v", err)
	}
	if linked.ID != registered.ID {
		t.Errorf("expected linked user %s, got %s", registered.ID, linked.ID)
	}
}

// TestFullUserJourney exercises the complete flow against the real database:
// register → login → group → invite → accept → expense → balance → settle → dashboard.
func TestFullUserJourney(t *testing.T) {
	alice := newClient(t)
	bob := newClient(t)

	aliceEmail := uniqueEmail("alice")
	bobEmail := uniqueEmail("bob")

	alice.register("Alice", aliceEmail)
	bob.register("Bob", bobEmail)

	// A user who is not logged in must be rejected.
	anon := newClient(t)
	anon.expect(http.MethodGet, "/api/v1/groups", nil, http.StatusUnauthorized)

	groupID := alice.createGroup("M9 Trip")
	token := alice.inviteMember(groupID, bobEmail)
	bob.acceptInvitation(token)

	// Expense: Alice paid 200000, equal split between both.
	ids := alice.memberIDs(groupID)
	alice.createExpense(groupID, map[string]any{
		"description":  "Dinner",
		"amount":       "200000.00",
		"currency":     "IDR",
		"paidBy":       ids[aliceEmail],
		"expenseDate":  "2026-08-14T19:00:00+07:00",
		"splitType":    "equal",
		"category":     "Food & Drinks",
		"participants": []string{ids[aliceEmail], ids[bobEmail]},
	})

	balances := alice.balances(groupID)
	if balances["Alice"] != "100000.00" || balances["Bob"] != "-100000.00" {
		t.Fatalf("unexpected balances after expense: %v", balances)
	}

	// Bob settles 100000 to Alice; both should be settled.
	bob.createSettlement(groupID, map[string]any{
		"payerId":    ids[bobEmail],
		"receiverId": ids[aliceEmail],
		"amount":     "100000.00",
	})

	balances = alice.balances(groupID)
	if balances["Alice"] != "0.00" || balances["Bob"] != "0.00" {
		t.Fatalf("unexpected balances after settlement: %v", balances)
	}

	// Settlement history is visible to members.
	resp := bob.expect(http.MethodGet, "/api/v1/groups/"+groupID+"/settlements", nil, http.StatusOK)
	settlements := resp["data"].(map[string]any)["settlements"].([]any)
	if len(settlements) != 1 {
		t.Fatalf("expected 1 settlement, got %d", len(settlements))
	}
	s := settlements[0].(map[string]any)
	if s["payerName"] != "Bob" || s["receiverName"] != "Alice" || s["amount"] != "100000.00" {
		t.Fatalf("unexpected settlement record: %v", s)
	}

	// Dashboard aggregates group + expense + settlement totals.
	dash := alice.expect(http.MethodGet, "/api/v1/dashboard", nil, http.StatusOK)
	summary := dash["data"].(map[string]any)["summary"].(map[string]any)
	if summary["totalExpense"] != "200000.00" {
		t.Fatalf("expected totalExpense 200000.00, got %v", summary["totalExpense"])
	}
	if summary["settledAmount"] != "100000.00" {
		t.Fatalf("expected settledAmount 100000.00, got %v", summary["settledAmount"])
	}
	recent := dash["data"].(map[string]any)["recentExpenses"].([]any)
	if len(recent) != 1 {
		t.Fatalf("expected 1 recent expense, got %d", len(recent))
	}
	if recent[0].(map[string]any)["description"] != "Dinner" {
		t.Fatalf("unexpected recent expense: %v", recent[0])
	}

	// Expense list exposes createdBy (used by UI to gate delete).
	expenses := alice.expect(http.MethodGet, "/api/v1/groups/"+groupID+"/expenses", nil, http.StatusOK)
	exp := expenses["data"].(map[string]any)["expenses"].([]any)[0].(map[string]any)
	if exp["createdBy"] != ids[aliceEmail] {
		t.Fatalf("expected createdBy %s, got %v", ids[aliceEmail], exp["createdBy"])
	}

	// Deleting the expense leaves the settlement record, so Bob is now owed
	// 100000 by Alice (settlements are independent of expenses).
	alice.expect(http.MethodDelete, "/api/v1/expenses/"+exp["id"].(string), nil, http.StatusOK)
	balances = alice.balances(groupID)
	if balances["Alice"] != "-100000.00" || balances["Bob"] != "100000.00" {
		t.Fatalf("unexpected balances after delete: %v", balances)
	}

	// Admin deletes the group; members then get 404.
	alice.expect(http.MethodDelete, "/api/v1/groups/"+groupID, nil, http.StatusOK)
	alice.expect(http.MethodGet, "/api/v1/groups/"+groupID, nil, http.StatusNotFound)
	bob.expect(http.MethodGet, "/api/v1/groups/"+groupID, nil, http.StatusNotFound)
}

// TestAuthorization verifies membership and ownership rules end to end.
func TestAuthorization(t *testing.T) {
	owner := newClient(t)
	invitee := newClient(t)
	outside := newClient(t)

	ownerEmail := uniqueEmail("owner")
	inviteeEmail := uniqueEmail("invitee")

	owner.register("Owner", ownerEmail)
	invitee.register("Invitee", inviteeEmail)
	outside.register("Outside", uniqueEmail("outside"))

	groupID := owner.createGroup("Authz Group")
	token := owner.inviteMember(groupID, inviteeEmail)
	invitee.acceptInvitation(token)

	ids := owner.memberIDs(groupID)
	ownerID := ids[ownerEmail]
	inviteeID := ids[inviteeEmail]
	outsideID := outside.register("Outside2", uniqueEmail("outside2"))

	// Non-member cannot see the group, its members, balances, or expenses.
	for _, path := range []string{
		"/api/v1/groups/" + groupID,
		"/api/v1/groups/" + groupID + "/members",
		"/api/v1/groups/" + groupID + "/balances",
		"/api/v1/groups/" + groupID + "/expenses",
		"/api/v1/groups/" + groupID + "/settlement-suggestions",
		"/api/v1/groups/" + groupID + "/settlements",
	} {
		outside.expect(http.MethodGet, path, nil, http.StatusNotFound)
	}

	// Non-member cannot create expenses in the group.
	outside.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/expenses", map[string]any{
		"description":  "Sneaky",
		"amount":       "1000.00",
		"currency":     "IDR",
		"paidBy":       outsideID,
		"expenseDate":  "2026-08-14T19:00:00+07:00",
		"splitType":    "equal",
		"category":     "Other",
		"participants": []string{outsideID},
	}, http.StatusNotFound)

	// Expense created by owner; invitee cannot delete it, owner can.
	exp := owner.createExpense(groupID, map[string]any{
		"description":  "Lunch",
		"amount":       "60000.00",
		"currency":     "IDR",
		"paidBy":       ownerID,
		"expenseDate":  "2026-08-14T12:00:00+07:00",
		"splitType":    "equal",
		"category":     "Food & Drinks",
		"participants": []string{ownerID, inviteeID},
	})
	invitee.expect(http.MethodDelete, "/api/v1/expenses/"+exp["id"].(string), nil, http.StatusForbidden)
	owner.expect(http.MethodDelete, "/api/v1/expenses/"+exp["id"].(string), nil, http.StatusOK)

	// Only the payer (current user) may record a settlement.
	owner.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/settlements", map[string]any{
		"payerId":    inviteeID,
		"receiverId": ownerID,
		"amount":     "30000.00",
	}, http.StatusForbidden)
	invitee.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/settlements", map[string]any{
		"payerId":    inviteeID,
		"receiverId": ownerID,
		"amount":     "30000.00",
	}, http.StatusCreated)

	// A member cannot delete the group; only the admin can.
	invitee.expect(http.MethodDelete, "/api/v1/groups/"+groupID, nil, http.StatusForbidden)
	owner.expect(http.MethodDelete, "/api/v1/groups/"+groupID, nil, http.StatusOK)
}

// TestBulkInvite verifies bulk invitation creation and the mixed flow where
// some invitees already have accounts and others register after.
func TestBulkInvite(t *testing.T) {
	alice := newClient(t)
	aliceEmail := uniqueEmail("alice")
	alice.register("Alice", aliceEmail)

	bobEmail := uniqueEmail("bob")
	charlieEmail := uniqueEmail("charlie")

	groupID := alice.createGroup("Bulk Group")

	// Charlie already has an account; Bob registers after the invite.
	charlie := newClient(t)
	charlie.register("Charlie", charlieEmail)

	tokens, failures := alice.bulkInvite(groupID, []string{bobEmail, charlieEmail, aliceEmail, bobEmail})
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if failures[bobEmail] != "DUPLICATE" || failures[aliceEmail] != "MEMBER_EXISTS" {
		t.Fatalf("unexpected failures: %v", failures)
	}

	// Bob registers after being invited, then joins with his token.
	bob := newClient(t)
	bob.register("Bob", bobEmail)
	bob.acceptInvitation(tokens[0])

	// Charlie already had an account and joins with his token.
	charlie.acceptInvitation(tokens[1])

	ids := alice.memberIDs(groupID)
	if _, ok := ids[bobEmail]; !ok {
		t.Fatalf("bob not a member")
	}
	if _, ok := ids[charlieEmail]; !ok {
		t.Fatalf("charlie not a member")
	}

	// A regular member cannot bulk invite.
	bob.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/invitations/bulk", map[string]any{
		"emails": []string{"x@test.com"},
	}, http.StatusForbidden)
}

// TestFinancialValidation verifies the expense invariant end to end.
func TestFinancialValidation(t *testing.T) {
	user := newClient(t)
	me := user.register("Val", uniqueEmail("val"))

	groupID := user.createGroup("Validation Group")

	// Zero amount is rejected.
	user.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/expenses", map[string]any{
		"description":  "Free",
		"amount":       "0.00",
		"currency":     "IDR",
		"paidBy":       me,
		"expenseDate":  "2026-08-14T19:00:00+07:00",
		"splitType":    "equal",
		"category":     "Other",
		"participants": []string{me},
	}, http.StatusUnprocessableEntity)

	// Custom splits whose sum does not match the amount are rejected.
	user.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/expenses", map[string]any{
		"description": "Bad splits",
		"amount":      "100000.00",
		"currency":    "IDR",
		"paidBy":      me,
		"expenseDate": "2026-08-14T19:00:00+07:00",
		"splitType":   "custom",
		"category":    "Other",
		"splits": []map[string]string{
			{"userId": me, "amount": "40000.00"},
		},
	}, http.StatusUnprocessableEntity)

	// A settlement where payer == receiver is rejected.
	user.expect(http.MethodPost, "/api/v1/groups/"+groupID+"/settlements", map[string]any{
		"payerId":    me,
		"receiverId": me,
		"amount":     "10000.00",
	}, http.StatusUnprocessableEntity)
}

// TestGroupReportExport verifies the Excel report download end to end:
// content type, attachment headers, a valid xlsx (zip) body, and
// authorization (member only).
func TestGroupReportExport(t *testing.T) {
	alice := newClient(t)
	bob := newClient(t)

	aliceEmail := uniqueEmail("alice")
	bobEmail := uniqueEmail("bob")
	alice.register("Alice", aliceEmail)
	bob.register("Bob", bobEmail)

	groupID := alice.createGroup("Bali Trip")
	token := alice.inviteMember(groupID, bobEmail)
	bob.acceptInvitation(token)

	ids := alice.memberIDs(groupID)
	alice.createExpense(groupID, map[string]any{
		"description":  "Dinner",
		"amount":       "200000.00",
		"currency":     "IDR",
		"paidBy":       ids[aliceEmail],
		"expenseDate":  "2026-08-14T19:00:00+07:00",
		"splitType":    "equal",
		"category":     "Food & Drinks",
		"participants": []string{ids[aliceEmail], ids[bobEmail]},
	})
	bob.createSettlement(groupID, map[string]any{
		"payerId":    ids[bobEmail],
		"receiverId": ids[aliceEmail],
		"amount":     "100000.00",
	})

	status, header, body := alice.doRaw(http.MethodGet, "/api/v1/groups/"+groupID+"/export")
	if status != http.StatusOK {
		t.Fatalf("export: expected 200, got %d", status)
	}
	if ct := header.Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("unexpected Content-Type %q", ct)
	}
	if cd := header.Get("Content-Disposition"); !strings.Contains(cd, `filename="Bali-Trip-report.xlsx"`) {
		t.Errorf("unexpected Content-Disposition %q", cd)
	}

	if len(body) < 4 || !bytes.Equal(body[:4], []byte("PK\x03\x04")) {
		t.Fatalf("export body is not a zip/xlsx (first bytes %v)", body[:min(len(body), 4)])
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("open xlsx as zip: %v", err)
	}
	var sharedStrings string
	for _, f := range zr.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			raw, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatal(err)
			}
			sharedStrings = string(raw)
		}
	}
	if sharedStrings == "" {
		t.Fatal("xlsx has no shared strings")
	}
	for _, want := range []string{"Dinner", "Food &amp; Drinks", "Alice", "TOTAL"} {
		if !strings.Contains(sharedStrings, want) {
			t.Errorf("workbook missing %q", want)
		}
	}

	// Non-members cannot download the report.
	status, _, _ = bob.doRaw(http.MethodGet, "/api/v1/groups/"+groupID+"/export")
	if status != http.StatusOK {
		t.Fatalf("member export: expected 200, got %d", status)
	}

	outside := newClient(t)
	outside.register("Outside", uniqueEmail("outside"))
	status, _, _ = outside.doRaw(http.MethodGet, "/api/v1/groups/"+groupID+"/export")
	if status != http.StatusNotFound {
		t.Fatalf("non-member export: expected 404, got %d", status)
	}

	anon := newClient(t)
	status, _, _ = anon.doRaw(http.MethodGet, "/api/v1/groups/"+groupID+"/export")
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous export: expected 401, got %d", status)
	}
}
