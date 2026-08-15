package group

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
	"github.com/Arifinwidy02/splitmate-backend/internal/user"
)

func newTestHandler() (*Handler, *fakeStore, *fakeUsers) {
	svc, store, users := newTestService()
	return NewHandler(svc), store, users
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
	if len(segs) < 2 || segs[0] != "groups" {
		return
	}
	switch {
	case len(segs) == 2:
		r.SetPathValue("groupId", segs[1])
	case len(segs) == 3 && segs[1] == "invitations":
		r.SetPathValue("token", segs[2])
	case len(segs) == 3:
		r.SetPathValue("groupId", segs[1])
	case len(segs) == 4 && segs[1] == "invitations":
		r.SetPathValue("token", segs[2])
	case len(segs) == 4:
		r.SetPathValue("groupId", segs[1])
		r.SetPathValue("userId", segs[3])
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

func createGroupIn(t *testing.T, h *Handler, store *fakeStore, users *fakeUsers, admin uuid.UUID) *Group {
	t.Helper()
	g, err := h.service.CreateGroup(t.Context(), admin, "Trip", "", "IDR", nil)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	return g
}

func TestCreateGroupHandler(t *testing.T) {
	h, _, _ := newTestHandler()
	userID := uuid.New()

	rec := doRequest(t, h.Create, http.MethodPost, "/groups", map[string]any{
		"name":        "Trip",
		"description": "summer",
		"currency":    "idr",
	}, userID)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Group struct {
				ID          uuid.UUID `json:"id"`
				Name        string    `json:"name"`
				Currency    string    `json:"currency"`
				Role        string    `json:"role"`
				MemberCount int       `json:"memberCount"`
			} `json:"group"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)
	if resp.Data.Group.Name != "Trip" {
		t.Errorf("expected name Trip, got %q", resp.Data.Group.Name)
	}
	if resp.Data.Group.Currency != "IDR" {
		t.Errorf("expected normalized currency IDR, got %q", resp.Data.Group.Currency)
	}
	if resp.Data.Group.Role != RoleAdmin {
		t.Errorf("expected role admin, got %q", resp.Data.Group.Role)
	}
	if resp.Data.Group.MemberCount != 1 {
		t.Errorf("expected member count 1, got %d", resp.Data.Group.MemberCount)
	}
}

func TestCreateGroupHandlerValidation(t *testing.T) {
	h, _, _ := newTestHandler()

	rec := doRequest(t, h.Create, http.MethodPost, "/groups", map[string]any{
		"name":     "",
		"currency": "IDR",
	}, uuid.New())

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateGroupHandlerBadBody(t *testing.T) {
	h, _, _ := newTestHandler()

	rec := doRequest(t, h.Create, http.MethodPost, "/groups", map[string]any{
		"name":     "Trip",
		"currency": "IDR",
		"unknown":  "field",
	}, uuid.New())

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestListGroupsHandler(t *testing.T) {
	h, _, _ := newTestHandler()
	userID := uuid.New()

	if _, err := h.service.CreateGroup(t.Context(), userID, "Trip", "", "IDR", nil); err != nil {
		t.Fatalf("create group: %v", err)
	}

	rec := doRequest(t, h.List, http.MethodGet, "/groups", nil, userID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Groups []struct {
				ID   uuid.UUID `json:"id"`
				Role string    `json:"role"`
			} `json:"groups"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)
	if len(resp.Data.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(resp.Data.Groups))
	}
	if resp.Data.Groups[0].Role != RoleAdmin {
		t.Errorf("expected admin role, got %q", resp.Data.Groups[0].Role)
	}
}

func TestGetGroupHandlerInvalidUUID(t *testing.T) {
	h, _, _ := newTestHandler()

	rec := doRequest(t, h.Get, http.MethodGet, "/groups/not-a-uuid", nil, uuid.New())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestGetGroupHandlerNotFoundForNonMember(t *testing.T) {
	h, store, users := newTestHandler()
	owner := uuid.New()
	outsider := uuid.New()

	g := createGroupIn(t, h, store, users, owner)

	rec := doRequest(t, h.Get, http.MethodGet, "/groups/"+g.ID.String(), nil, outsider)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var eb errorBody
	decodeData(t, rec, &eb)
	if eb.Error.Code != "GROUP_NOT_FOUND" {
		t.Errorf("expected GROUP_NOT_FOUND, got %q", eb.Error.Code)
	}
}

func TestUpdateGroupHandlerForbiddenForMember(t *testing.T) {
	h, store, users := newTestHandler()
	owner := uuid.New()
	member := uuid.New()

	g := createGroupIn(t, h, store, users, owner)

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := h.service.store.AddMember(t.Context(), g.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	rec := doRequest(t, h.Update, http.MethodPatch, "/groups/"+g.ID.String(), map[string]any{
		"name": "Renamed",
	}, member)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var eb errorBody
	decodeData(t, rec, &eb)
	if eb.Error.Code != "FORBIDDEN" {
		t.Errorf("expected FORBIDDEN, got %q", eb.Error.Code)
	}
}

func TestUpdateGroupHandlerAdminSuccess(t *testing.T) {
	h, _, _ := newTestHandler()
	owner := uuid.New()

	rec := doRequest(t, h.Create, http.MethodPost, "/groups", map[string]any{
		"name":     "Trip",
		"currency": "IDR",
	}, owner)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var created struct {
		Data struct {
			Group struct {
				ID uuid.UUID `json:"id"`
			} `json:"group"`
		} `json:"data"`
	}
	decodeData(t, rec, &created)

	rec = doRequest(t, h.Update, http.MethodPatch, "/groups/"+created.Data.Group.ID.String(), map[string]any{
		"name":     "Roadtrip",
		"currency": "USD",
	}, owner)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var updated struct {
		Data struct {
			Group struct {
				Name     string `json:"name"`
				Currency string `json:"currency"`
			} `json:"group"`
		} `json:"data"`
	}
	decodeData(t, rec, &updated)
	if updated.Data.Group.Name != "Roadtrip" || updated.Data.Group.Currency != "USD" {
		t.Errorf("unexpected update: %+v", updated.Data.Group)
	}
}

func TestDeleteGroupHandlerForbiddenForMember(t *testing.T) {
	h, store, users := newTestHandler()
	owner := uuid.New()
	member := uuid.New()

	g := createGroupIn(t, h, store, users, owner)

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := h.service.store.AddMember(t.Context(), g.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	rec := doRequest(t, h.Delete, http.MethodDelete, "/groups/"+g.ID.String(), nil, member)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestDeleteGroupHandlerAdminSuccess(t *testing.T) {
	h, _, _ := newTestHandler()
	owner := uuid.New()

	rec := doRequest(t, h.Create, http.MethodPost, "/groups", map[string]any{
		"name":     "Trip",
		"currency": "IDR",
	}, owner)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var created struct {
		Data struct {
			Group struct {
				ID uuid.UUID `json:"id"`
			} `json:"group"`
		} `json:"data"`
	}
	decodeData(t, rec, &created)

	rec = doRequest(t, h.Delete, http.MethodDelete, "/groups/"+created.Data.Group.ID.String(), nil, owner)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestListMembersHandler(t *testing.T) {
	h, _, _ := newTestHandler()
	owner := uuid.New()

	rec := doRequest(t, h.Create, http.MethodPost, "/groups", map[string]any{
		"name":     "Trip",
		"currency": "IDR",
	}, owner)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var created struct {
		Data struct {
			Group struct {
				ID uuid.UUID `json:"id"`
			} `json:"group"`
		} `json:"data"`
	}
	decodeData(t, rec, &created)

	rec = doRequest(t, h.ListMembers, http.MethodGet, "/groups/"+created.Data.Group.ID.String()+"/members", nil, owner)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Members []struct {
				ID   uuid.UUID `json:"id"`
				Role string    `json:"role"`
			} `json:"members"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)
	if len(resp.Data.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(resp.Data.Members))
	}
	if resp.Data.Members[0].ID != owner {
		t.Errorf("expected owner member, got %v", resp.Data.Members[0].ID)
	}
}

func TestRemoveMemberHandler(t *testing.T) {
	h, store, users := newTestHandler()
	owner := uuid.New()
	member := uuid.New()

	g := createGroupIn(t, h, store, users, owner)

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := h.service.store.AddMember(t.Context(), g.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	t.Run("member cannot remove", func(t *testing.T) {
		rec := doRequest(t, h.RemoveMember, http.MethodDelete, "/groups/"+g.ID.String()+"/members/"+owner.String(), nil, member)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin cannot remove self", func(t *testing.T) {
		rec := doRequest(t, h.RemoveMember, http.MethodDelete, "/groups/"+g.ID.String()+"/members/"+owner.String(), nil, owner)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin removes member", func(t *testing.T) {
		rec := doRequest(t, h.RemoveMember, http.MethodDelete, "/groups/"+g.ID.String()+"/members/"+member.String(), nil, owner)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})
}

func TestCreateInvitationHandler(t *testing.T) {
	h, _, _ := newTestHandler()
	owner := uuid.New()

	rec := doRequest(t, h.Create, http.MethodPost, "/groups", map[string]any{
		"name":     "Trip",
		"currency": "IDR",
	}, owner)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var created struct {
		Data struct {
			Group struct {
				ID uuid.UUID `json:"id"`
			} `json:"group"`
		} `json:"data"`
	}
	decodeData(t, rec, &created)

	rec = doRequest(t, h.CreateInvitation, http.MethodPost, "/groups/"+created.Data.Group.ID.String()+"/invitations", map[string]any{
		"email": "budi@test.com",
	}, owner)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Invitation struct {
				ID        uuid.UUID `json:"id"`
				Email     string    `json:"email"`
				Status    string    `json:"status"`
				ExpiresAt time.Time `json:"expiresAt"`
				Token     string    `json:"token"`
			} `json:"invitation"`
		} `json:"data"`
	}
	decodeData(t, rec, &resp)
	if resp.Data.Invitation.Token == "" {
		t.Errorf("expected token to be returned once, got empty")
	}
	if resp.Data.Invitation.Email != "budi@test.com" {
		t.Errorf("expected email budi@test.com, got %q", resp.Data.Invitation.Email)
	}
	if !resp.Data.Invitation.ExpiresAt.After(time.Now()) {
		t.Errorf("expected future expiry")
	}
}

func TestCreateInvitationHandlerNonAdmin(t *testing.T) {
	h, store, users := newTestHandler()
	owner := uuid.New()
	member := uuid.New()

	g := createGroupIn(t, h, store, users, owner)

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := h.service.store.AddMember(t.Context(), g.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	rec := doRequest(t, h.CreateInvitation, http.MethodPost, "/groups/"+g.ID.String()+"/invitations", map[string]any{
		"email": "x@test.com",
	}, member)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateBulkInvitationsHandler(t *testing.T) {
	h, store, users := newTestHandler()
	owner := uuid.New()
	member := uuid.New()

	g := createGroupIn(t, h, store, users, owner)

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := h.service.store.AddMember(t.Context(), g.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	t.Run("non-admin 403", func(t *testing.T) {
		rec := doRequest(t, h.CreateBulkInvitations, http.MethodPost, "/groups/"+g.ID.String()+"/invitations/bulk", map[string]any{
			"emails": []string{"x@test.com"},
		}, member)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid emails 422", func(t *testing.T) {
		rec := doRequest(t, h.CreateBulkInvitations, http.MethodPost, "/groups/"+g.ID.String()+"/invitations/bulk", map[string]any{
			"emails": []string{"good@test.com", "not-an-email"},
		}, owner)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("creates invitations and reports skips", func(t *testing.T) {
		rec := doRequest(t, h.CreateBulkInvitations, http.MethodPost, "/groups/"+g.ID.String()+"/invitations/bulk", map[string]any{
			"emails": []string{"a@test.com", "member@test.com", "a@test.com", "b@Test.com"},
		}, owner)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Invitations []struct {
					Email     string    `json:"email"`
					Status    string    `json:"status"`
					ExpiresAt time.Time `json:"expiresAt"`
					Token     string    `json:"token"`
				} `json:"invitations"`
				Failed []struct {
					Email  string `json:"email"`
					Reason string `json:"reason"`
				} `json:"failed"`
			} `json:"data"`
		}
		decodeData(t, rec, &resp)

		if len(resp.Data.Invitations) != 2 {
			t.Fatalf("expected 2 invitations, got %d", len(resp.Data.Invitations))
		}
		byEmail := map[string]string{}
		for _, inv := range resp.Data.Invitations {
			byEmail[inv.Email] = inv.Token
			if inv.Status != "pending" {
				t.Errorf("expected pending, got %q", inv.Status)
			}
			if inv.Token == "" {
				t.Errorf("expected token for %s", inv.Email)
			}
			if !inv.ExpiresAt.After(time.Now()) {
				t.Errorf("expected future expiry for %s", inv.Email)
			}
		}
		if _, ok := byEmail["a@test.com"]; !ok {
			t.Errorf("expected a@test.com in results")
		}
		if _, ok := byEmail["b@test.com"]; !ok {
			t.Errorf("expected normalized b@test.com in results")
		}

		expectedFailures := map[string]string{
			"member@test.com": ReasonMemberExists,
			"a@test.com":      ReasonDuplicate,
		}
		if len(resp.Data.Failed) != len(expectedFailures) {
			t.Fatalf("expected %d failures, got %v", len(expectedFailures), resp.Data.Failed)
		}
		for _, f := range resp.Data.Failed {
			if expectedFailures[f.Email] != f.Reason {
				t.Errorf("expected %s=%s, got %s=%s", f.Email, expectedFailures[f.Email], f.Email, f.Reason)
			}
		}
	})
}

func TestAcceptInvitationHandler(t *testing.T) {
	h, store, users := newTestHandler()
	owner := uuid.New()
	invitee := uuid.New()

	g := createGroupIn(t, h, store, users, owner)

	inv, token, err := h.service.CreateInvitation(t.Context(), owner, g.ID, "invitee@test.com")
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	users.users[invitee] = &user.User{ID: invitee, Name: "Invitee", Email: "invitee@test.com"}
	store.emails[invitee] = "invitee@test.com"

	t.Run("unknown token 404", func(t *testing.T) {
		rec := doRequest(t, h.AcceptInvitation, http.MethodPost, "/groups/invitations/nope/accept", nil, invitee)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("wrong email 403", func(t *testing.T) {
		evil := uuid.New()
		users.users[evil] = &user.User{ID: evil, Name: "Evil", Email: "evil@test.com"}
		store.emails[evil] = "evil@test.com"

		rec := doRequest(t, h.AcceptInvitation, http.MethodPost, "/groups/invitations/"+token+"/accept", nil, evil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("invitee accepts 200", func(t *testing.T) {
		rec := doRequest(t, h.AcceptInvitation, http.MethodPost, "/groups/invitations/"+token+"/accept", nil, invitee)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Group struct {
					ID   uuid.UUID `json:"id"`
					Role string    `json:"role"`
				} `json:"group"`
			} `json:"data"`
		}
		decodeData(t, rec, &resp)
		if resp.Data.Group.ID != g.ID {
			t.Errorf("expected group %s, got %s", g.ID, resp.Data.Group.ID)
		}
	})

	t.Run("already used 409", func(t *testing.T) {
		rec := doRequest(t, h.AcceptInvitation, http.MethodPost, "/groups/invitations/"+token+"/accept", nil, invitee)
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	_ = inv
}

func TestAcceptInvitationHandlerExpired(t *testing.T) {
	h, store, users := newTestHandler()
	owner := uuid.New()
	invitee := uuid.New()

	g := createGroupIn(t, h, store, users, owner)

	_, token, err := h.service.CreateInvitation(t.Context(), owner, g.ID, "invitee@test.com")
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	users.users[invitee] = &user.User{ID: invitee, Name: "Invitee", Email: "invitee@test.com"}
	store.emails[invitee] = "invitee@test.com"

	for _, i := range store.invites {
		i.ExpiresAt = time.Now().Add(-time.Hour)
	}

	rec := doRequest(t, h.AcceptInvitation, http.MethodPost, "/groups/invitations/"+token+"/accept", nil, invitee)
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var eb errorBody
	decodeData(t, rec, &eb)
	if eb.Error.Code != "INVITATION_EXPIRED" {
		t.Errorf("expected INVITATION_EXPIRED, got %q", eb.Error.Code)
	}
}

func TestHandlerRequiresUserID(t *testing.T) {
	h, _, _ := newTestHandler()

	r := httptest.NewRequest(http.MethodGet, "/groups", nil)
	rec := httptest.NewRecorder()
	h.List(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestCreateInvitationHandlerErrorCodeMemberExists(t *testing.T) {
	h, store, users := newTestHandler()
	owner := uuid.New()
	member := uuid.New()

	g := createGroupIn(t, h, store, users, owner)

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := h.service.store.AddMember(t.Context(), g.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	rec := doRequest(t, h.CreateInvitation, http.MethodPost, "/groups/"+g.ID.String()+"/invitations", map[string]any{
		"email": "member@test.com",
	}, owner)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var eb errorBody
	decodeData(t, rec, &eb)
	if eb.Error.Code != "MEMBER_EXISTS" {
		t.Errorf("expected MEMBER_EXISTS, got %q", eb.Error.Code)
	}
}

func TestCreateGroupHandlerMultipartWithLogo(t *testing.T) {
	h, store, _ := newTestHandler()
	owner := uuid.New()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range map[string]string{
		"name":        "Trip",
		"description": "Beach trip",
		"currency":    "IDR",
	} {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="logo"; filename="logo.png"`)
	partHeader.Set("Content-Type", "image/png")
	part, err := mw.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("fake-logo-png")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/groups", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r = r.WithContext(middleware.WithUserID(r.Context(), owner))
	setPathValues(r)

	rec := httptest.NewRecorder()
	h.Create(rec, r)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Data struct {
			Group groupResponse `json:"group"`
		} `json:"data"`
	}
	decodeData(t, rec, &body)
	if !body.Data.Group.HasLogo {
		t.Error("expected response hasLogo=true")
	}

	stored := store.groups[0]
	if stored == nil {
		t.Fatal("expected group to be stored")
	}
	if !bytes.Equal(stored.LogoImage, []byte("fake-logo-png")) {
		t.Errorf("expected logo bytes stored, got %q", stored.LogoImage)
	}
	if stored.LogoContentType != "image/png" {
		t.Errorf("expected content type image/png, got %q", stored.LogoContentType)
	}
}

func TestCreateGroupHandlerMultipartRejectsInvalidLogo(t *testing.T) {
	h, _, _ := newTestHandler()
	owner := uuid.New()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("name", "Trip"); err != nil {
		t.Fatalf("write field: %v", err)
	}
	if err := mw.WriteField("currency", "IDR"); err != nil {
		t.Fatalf("write field: %v", err)
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="logo"; filename="logo.pdf"`)
	partHeader.Set("Content-Type", "application/pdf")
	part, err := mw.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("%PDF")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/groups", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r = r.WithContext(middleware.WithUserID(r.Context(), owner))
	setPathValues(r)

	rec := httptest.NewRecorder()
	h.Create(rec, r)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestGetLogoHandler(t *testing.T) {
	h, _, _ := newTestHandler()
	owner := uuid.New()

	g, err := h.service.CreateGroup(t.Context(), owner, "Trip", "", "IDR",
		&Logo{Image: []byte("img-bytes"), ContentType: "image/jpeg"})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	rec := doRequest(t, h.GetLogo, http.MethodGet, "/groups/"+g.ID.String()+"/logo", nil, owner)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "img-bytes" {
		t.Errorf("expected logo body, got %q", rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected image/jpeg content type, got %q", rec.Header().Get("Content-Type"))
	}

	noLogo, err := h.service.CreateGroup(t.Context(), owner, "Trip 2", "", "IDR", nil)
	if err != nil {
		t.Fatalf("create group without logo: %v", err)
	}
	rec = doRequest(t, h.GetLogo, http.MethodGet, "/groups/"+noLogo.ID.String()+"/logo", nil, owner)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var eb errorBody
	decodeData(t, rec, &eb)
	if eb.Error.Code != "LOGO_NOT_FOUND" {
		t.Errorf("expected LOGO_NOT_FOUND, got %q", eb.Error.Code)
	}
}
