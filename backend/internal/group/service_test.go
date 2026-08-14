package group

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/user"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
)

type fakeStore struct {
	groups          []*Group
	members         map[uuid.UUID][]*Member
	membership      map[string]string
	invites         []*Invitation
	emails          map[uuid.UUID]string
	failCreateGroup error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		members:    map[uuid.UUID][]*Member{},
		membership: map[string]string{},
		emails:     map[uuid.UUID]string{},
	}
}

func membershipKey(groupID, userID uuid.UUID) string {
	return groupID.String() + ":" + userID.String()
}

func (f *fakeStore) CreateGroupWithAdmin(ctx context.Context, g *Group, adminUserID uuid.UUID) (*Group, error) {
	if f.failCreateGroup != nil {
		return nil, f.failCreateGroup
	}
	now := time.Now()
	g.ID = uuid.New()
	g.CreatedBy = adminUserID
	g.CreatedAt = now
	g.UpdatedAt = now
	f.groups = append(f.groups, g)
	f.addMember(g.ID, adminUserID, RoleAdmin, now)
	return g, nil
}

func (f *fakeStore) addMember(groupID, userID uuid.UUID, role string, joinedAt time.Time) {
	f.membership[membershipKey(groupID, userID)] = role
	email, ok := f.emails[userID]
	if !ok {
		email = "test@example.com"
	}
	member := &Member{UserID: userID, Name: "Test " + userID.String()[:8], Email: email, Role: role, JoinedAt: joinedAt}
	members := f.members[groupID]
	for _, m := range members {
		if m.UserID == userID {
			return
		}
	}
	f.members[groupID] = append(members, member)
}

func (f *fakeStore) FindByID(ctx context.Context, id uuid.UUID) (*Group, error) {
	for _, g := range f.groups {
		if g.ID == id {
			cp := *g
			cp.MemberCount = len(f.members[g.ID])
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeStore) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Group, error) {
	groups := []*Group{}
	for _, g := range f.groups {
		if role, ok := f.membership[membershipKey(g.ID, userID)]; ok {
			cp := *g
			cp.Role = role
			cp.MemberCount = len(f.members[g.ID])
			groups = append(groups, &cp)
		}
	}
	return groups, nil
}

func (f *fakeStore) ListMembers(ctx context.Context, groupID uuid.UUID) ([]*Member, error) {
	return append([]*Member{}, f.members[groupID]...), nil
}

func (f *fakeStore) FindMembership(ctx context.Context, groupID, userID uuid.UUID) (*Membership, error) {
	role, ok := f.membership[membershipKey(groupID, userID)]
	if !ok {
		return nil, ErrNotFound
	}
	return &Membership{GroupID: groupID, UserID: userID, Role: role}, nil
}

func (f *fakeStore) FindMembershipByEmail(ctx context.Context, groupID uuid.UUID, email string) (*Membership, error) {
	for _, m := range f.members[groupID] {
		if m.Email == email {
			return &Membership{GroupID: groupID, UserID: m.UserID, Role: m.Role}, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeStore) AddMember(ctx context.Context, groupID, userID uuid.UUID, role string) error {
	if _, ok := f.membership[membershipKey(groupID, userID)]; ok {
		return ErrMemberExists
	}
	f.addMember(groupID, userID, role, time.Now())
	return nil
}

func (f *fakeStore) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	key := membershipKey(groupID, userID)
	if _, ok := f.membership[key]; !ok {
		return ErrNotFound
	}
	delete(f.membership, key)
	members := f.members[groupID]
	for i, m := range members {
		if m.UserID == userID {
			f.members[groupID] = append(members[:i], members[i+1:]...)
			break
		}
	}
	return nil
}

func (f *fakeStore) Update(ctx context.Context, g *Group) error {
	for _, existing := range f.groups {
		if existing.ID == g.ID {
			*existing = *g
			return nil
		}
	}
	return ErrNotFound
}

func (f *fakeStore) Delete(ctx context.Context, id uuid.UUID) error {
	for i, g := range f.groups {
		if g.ID == id {
			f.groups = append(f.groups[:i], f.groups[i+1:]...)
			delete(f.members, id)
			return nil
		}
	}
	return ErrNotFound
}

func (f *fakeStore) CreateInvitation(ctx context.Context, inv *Invitation) error {
	inv.ID = uuid.New()
	inv.CreatedAt = time.Now()
	f.invites = append(f.invites, inv)
	return nil
}

func (f *fakeStore) FindInvitationByTokenHash(ctx context.Context, tokenHash string) (*Invitation, error) {
	for _, inv := range f.invites {
		if inv.TokenHash == tokenHash {
			return inv, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeStore) FindPendingInvitation(ctx context.Context, groupID uuid.UUID, email string) (*Invitation, error) {
	for _, inv := range f.invites {
		if inv.GroupID == groupID && inv.Email == email && inv.Status == statusPending {
			return inv, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeStore) AcceptInvitation(ctx context.Context, inv *Invitation, userID uuid.UUID) error {
	if _, ok := f.membership[membershipKey(inv.GroupID, userID)]; ok {
		return ErrMemberExists
	}
	f.addMember(inv.GroupID, userID, RoleMember, time.Now())
	for _, i := range f.invites {
		if i.ID == inv.ID {
			i.Status = statusAccepted
		}
	}
	return nil
}

type fakeUsers struct {
	users map[uuid.UUID]*user.User
}

func (f *fakeUsers) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func newTestService() (*Service, *fakeStore, *fakeUsers) {
	store := newFakeStore()
	users := &fakeUsers{users: map[uuid.UUID]*user.User{}}
	return NewService(store, users), store, users
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	return uuid.New()
}

func asValidationError(err error) *apperror.Validation {
	var valErr *apperror.Validation
	if errors.As(err, &valErr) {
		return valErr
	}
	return nil
}

func TestCreateGroupValidation(t *testing.T) {
	svc, _, _ := newTestService()
	owner := mustUUID(t)

	tests := []struct {
		name      string
		groupName string
		currency  string
		desc      string
		wantErr   bool
	}{
		{"empty name", "", "IDR", "", true},
		{"name too long", strings.Repeat("a", 101), "IDR", "", true},
		{"bad currency two letters", "Trip", "ID", "", true},
		{"bad currency digits", "Trip", "12D", "", true},
		{"description too long", "Trip", "IDR", strings.Repeat("a", 501), true},
		{"valid", "Trip", "IDR", "", false},
		{"valid lowercase currency normalized", "Trip", "idr", "", false},
		{"valid with description", "Trip", "USD", "summer", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := svc.CreateGroup(context.Background(), owner, tt.groupName, tt.desc, tt.currency)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if g.Role != RoleAdmin {
				t.Errorf("expected creator role admin, got %q", g.Role)
			}
			if g.MemberCount != 1 {
				t.Errorf("expected member count 1, got %d", g.MemberCount)
			}
			if g.Currency != "IDR" && g.Currency != "USD" {
				t.Errorf("currency should be uppercased, got %q", g.Currency)
			}
		})
	}
}

func TestGetGroupAuthorization(t *testing.T) {
	svc, _, _ := newTestService()
	owner := mustUUID(t)
	other := mustUUID(t)

	group, err := svc.CreateGroup(context.Background(), owner, "Trip", "", "IDR")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	t.Run("member can view", func(t *testing.T) {
		g, err := svc.GetGroup(context.Background(), owner, group.ID)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if g.Name != "Trip" {
			t.Errorf("expected name Trip, got %q", g.Name)
		}
		if g.Role != RoleAdmin {
			t.Errorf("expected role admin, got %q", g.Role)
		}
	})

	t.Run("non-member gets group not found", func(t *testing.T) {
		_, err := svc.GetGroup(context.Background(), other, group.ID)
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})

	t.Run("unknown group gets group not found", func(t *testing.T) {
		_, err := svc.GetGroup(context.Background(), owner, uuid.New())
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})
}

func TestUpdateGroupAuthorization(t *testing.T) {
	svc, store, users := newTestService()
	owner := mustUUID(t)
	member := mustUUID(t)

	group, err := svc.CreateGroup(context.Background(), owner, "Trip", "", "IDR")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := svc.store.AddMember(context.Background(), group.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	name := "Renamed"

	t.Run("admin can update", func(t *testing.T) {
		g, err := svc.UpdateGroup(context.Background(), owner, group.ID, &name, nil, nil)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if g.Name != "Renamed" {
			t.Errorf("expected Renamed, got %q", g.Name)
		}
		if g.Role != RoleAdmin {
			t.Errorf("expected role admin, got %q", g.Role)
		}
	})

	t.Run("member cannot update", func(t *testing.T) {
		_, err := svc.UpdateGroup(context.Background(), member, group.ID, &name, nil, nil)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("non-member gets group not found", func(t *testing.T) {
		_, err := svc.UpdateGroup(context.Background(), uuid.New(), group.ID, &name, nil, nil)
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})

	t.Run("invalid currency rejected", func(t *testing.T) {
		bad := "xx"
		_, err := svc.UpdateGroup(context.Background(), owner, group.ID, nil, nil, &bad)
		if asValidationError(err) == nil {
			t.Fatalf("expected validation error, got %v", err)
		}
	})
}

func TestDeleteGroupAuthorization(t *testing.T) {
	svc, store, users := newTestService()
	owner := mustUUID(t)
	member := mustUUID(t)

	group, err := svc.CreateGroup(context.Background(), owner, "Trip", "", "IDR")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := svc.store.AddMember(context.Background(), group.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	t.Run("member cannot delete", func(t *testing.T) {
		err := svc.DeleteGroup(context.Background(), member, group.ID)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("non-member cannot delete", func(t *testing.T) {
		err := svc.DeleteGroup(context.Background(), uuid.New(), group.ID)
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})

	t.Run("admin can delete", func(t *testing.T) {
		if err := svc.DeleteGroup(context.Background(), owner, group.ID); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(store.groups) != 0 {
			t.Errorf("expected group to be deleted")
		}
	})
}

func TestListMembersAuthorization(t *testing.T) {
	svc, store, users := newTestService()
	owner := mustUUID(t)
	member := mustUUID(t)

	group, err := svc.CreateGroup(context.Background(), owner, "Trip", "", "IDR")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := svc.store.AddMember(context.Background(), group.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	t.Run("member can list", func(t *testing.T) {
		members, err := svc.ListMembers(context.Background(), member, group.ID)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(members) != 2 {
			t.Errorf("expected 2 members, got %d", len(members))
		}
	})

	t.Run("non-member cannot list", func(t *testing.T) {
		_, err := svc.ListMembers(context.Background(), uuid.New(), group.ID)
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})
}

func TestRemoveMember(t *testing.T) {
	svc, store, users := newTestService()
	owner := mustUUID(t)
	member := mustUUID(t)
	stranger := mustUUID(t)

	group, err := svc.CreateGroup(context.Background(), owner, "Trip", "", "IDR")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := svc.store.AddMember(context.Background(), group.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	t.Run("member cannot remove anyone", func(t *testing.T) {
		err := svc.RemoveMember(context.Background(), member, group.ID, owner)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("admin cannot remove self", func(t *testing.T) {
		err := svc.RemoveMember(context.Background(), owner, group.ID, owner)
		if asValidationError(err) == nil {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("admin cannot remove non-member", func(t *testing.T) {
		err := svc.RemoveMember(context.Background(), owner, group.ID, stranger)
		if !errors.Is(err, ErrMemberNotFound) {
			t.Fatalf("expected ErrMemberNotFound, got %v", err)
		}
	})

	t.Run("admin can remove member", func(t *testing.T) {
		if err := svc.RemoveMember(context.Background(), owner, group.ID, member); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if _, err := svc.store.FindMembership(context.Background(), group.ID, member); !errors.Is(err, ErrNotFound) {
			t.Errorf("expected member to be removed, got %v", err)
		}
	})
}

func TestCreateInvitation(t *testing.T) {
	svc, store, users := newTestService()
	owner := mustUUID(t)
	member := mustUUID(t)

	group, err := svc.CreateGroup(context.Background(), owner, "Trip", "", "IDR")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	users.users[member] = &user.User{ID: member, Name: "Member", Email: "member@test.com"}
	store.emails[member] = "member@test.com"
	if err := svc.store.AddMember(context.Background(), group.ID, member, RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	t.Run("member cannot invite", func(t *testing.T) {
		_, _, err := svc.CreateInvitation(context.Background(), member, group.ID, "x@test.com")
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("non-member cannot invite", func(t *testing.T) {
		_, _, err := svc.CreateInvitation(context.Background(), uuid.New(), group.ID, "x@test.com")
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})

	t.Run("invalid email rejected", func(t *testing.T) {
		_, _, err := svc.CreateInvitation(context.Background(), owner, group.ID, "not-an-email")
		if asValidationError(err) == nil {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("existing member rejected", func(t *testing.T) {
		_, _, err := svc.CreateInvitation(context.Background(), owner, group.ID, "member@test.com")
		if !errors.Is(err, ErrMemberExists) {
			t.Fatalf("expected ErrMemberExists, got %v", err)
		}
	})

	var token string
	t.Run("admin can invite", func(t *testing.T) {
		inv, tok, err := svc.CreateInvitation(context.Background(), owner, group.ID, "New@Test.com")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		token = tok
		if inv.Status != statusPending {
			t.Errorf("expected pending, got %q", inv.Status)
		}
		if inv.Email != "new@test.com" {
			t.Errorf("expected normalized email, got %q", inv.Email)
		}
		if !inv.ExpiresAt.After(time.Now()) {
			t.Errorf("expected future expiry, got %v", inv.ExpiresAt)
		}

		sum := sha256.Sum256([]byte(tok))
		if inv.TokenHash != hex.EncodeToString(sum[:]) {
			t.Errorf("token hash does not match token")
		}
	})

	t.Run("duplicate pending invitation rejected", func(t *testing.T) {
		_, _, err := svc.CreateInvitation(context.Background(), owner, group.ID, "new@test.com")
		if !errors.Is(err, ErrInvitationExists) {
			t.Fatalf("expected ErrInvitationExists, got %v", err)
		}
	})

	_ = token
}

func TestAcceptInvitation(t *testing.T) {
	svc, store, users := newTestService()
	owner := mustUUID(t)
	invitee := mustUUID(t)
	interloper := mustUUID(t)

	users.users[invitee] = &user.User{ID: invitee, Name: "Invitee", Email: "invitee@test.com"}
	users.users[interloper] = &user.User{ID: interloper, Name: "Interloper", Email: "evil@test.com"}
	store.emails[invitee] = "invitee@test.com"
	store.emails[interloper] = "evil@test.com"

	group, err := svc.CreateGroup(context.Background(), owner, "Trip", "", "IDR")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	inv, token, err := svc.CreateInvitation(context.Background(), owner, group.ID, "invitee@test.com")
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	t.Run("unknown token rejected", func(t *testing.T) {
		_, err := svc.AcceptInvitation(context.Background(), invitee, "does-not-exist")
		if !errors.Is(err, ErrInvitationNotFound) {
			t.Fatalf("expected ErrInvitationNotFound, got %v", err)
		}
	})

	t.Run("user with different email rejected", func(t *testing.T) {
		_, err := svc.AcceptInvitation(context.Background(), interloper, token)
		if !errors.Is(err, ErrInvitationForbidden) {
			t.Fatalf("expected ErrInvitationForbidden, got %v", err)
		}
	})

	t.Run("invitee can accept", func(t *testing.T) {
		g, err := svc.AcceptInvitation(context.Background(), invitee, token)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if g.Role != RoleMember {
			t.Errorf("expected role member, got %q", g.Role)
		}

		m, err := store.FindMembership(context.Background(), group.ID, invitee)
		if err != nil {
			t.Fatalf("expected membership, got %v", err)
		}
		if m.Role != RoleMember {
			t.Errorf("expected member role, got %q", m.Role)
		}

		for _, i := range store.invites {
			if i.ID == inv.ID && i.Status != statusAccepted {
				t.Errorf("expected invitation accepted, got %q", i.Status)
			}
		}
	})

	t.Run("already member cannot accept again", func(t *testing.T) {
		_, err := svc.AcceptInvitation(context.Background(), invitee, token)
		if !errors.Is(err, ErrInvitationUsed) {
			t.Fatalf("expected ErrInvitationUsed, got %v", err)
		}
	})

	t.Run("used invitation rejected for new user", func(t *testing.T) {
		other := mustUUID(t)
		users.users[other] = &user.User{ID: other, Name: "Other", Email: "invitee@test.com"}
		_, err := svc.AcceptInvitation(context.Background(), other, token)
		if !errors.Is(err, ErrInvitationUsed) {
			t.Fatalf("expected ErrInvitationUsed, got %v", err)
		}
	})
}

func TestAcceptInvitationExpired(t *testing.T) {
	svc, store, users := newTestService()
	owner := mustUUID(t)
	invitee := mustUUID(t)

	users.users[invitee] = &user.User{ID: invitee, Name: "Invitee", Email: "invitee@test.com"}

	group, err := svc.CreateGroup(context.Background(), owner, "Trip", "", "IDR")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	_, token, err := svc.CreateInvitation(context.Background(), owner, group.ID, "invitee@test.com")
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	for _, i := range store.invites {
		i.Status = statusPending
		i.ExpiresAt = time.Now().Add(-time.Hour)
	}

	_, err = svc.AcceptInvitation(context.Background(), invitee, token)
	if !errors.Is(err, ErrInvitationExpired) {
		t.Fatalf("expected ErrInvitationExpired, got %v", err)
	}
}

func TestAcceptInvitationEmptyToken(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.AcceptInvitation(context.Background(), uuid.New(), "")
	if !errors.Is(err, ErrInvitationNotFound) {
		t.Fatalf("expected ErrInvitationNotFound, got %v", err)
	}
}

func TestListGroupsOnlyMemberships(t *testing.T) {
	svc, _, _ := newTestService()
	owner := mustUUID(t)
	other := mustUUID(t)

	if _, err := svc.CreateGroup(context.Background(), owner, "Trip", "", "IDR"); err != nil {
		t.Fatalf("create group: %v", err)
	}

	groups, err := svc.ListGroups(context.Background(), other)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected no groups for non-member, got %d", len(groups))
	}

	groups, err = svc.ListGroups(context.Background(), owner)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Role != RoleAdmin {
		t.Errorf("expected admin role, got %q", groups[0].Role)
	}
}

func TestInvitationTokenUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		token, err := newInvitationToken()
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}
		if token == "" {
			t.Fatalf("expected non-empty token")
		}
		if seen[token] {
			t.Fatalf("duplicate token generated: %s", token)
		}
		seen[token] = true
	}
}

func TestCreateGroupErrorWrapping(t *testing.T) {
	svc, store, _ := newTestService()
	store.failCreateGroup = errors.New("db down")

	_, err := svc.CreateGroup(context.Background(), mustUUID(t), "Trip", "", "IDR")
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("expected wrapped db error, got %v", err)
	}
}
