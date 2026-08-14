package balance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/group"
)

type fakeStore struct {
	expenses     map[uuid.UUID][]Expense
	settlements  map[uuid.UUID][]Settlement
	failExpenses error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		expenses:    map[uuid.UUID][]Expense{},
		settlements: map[uuid.UUID][]Settlement{},
	}
}

func (f *fakeStore) ExpensesWithSplits(ctx context.Context, groupID uuid.UUID) ([]Expense, error) {
	if f.failExpenses != nil {
		return nil, f.failExpenses
	}
	return append([]Expense{}, f.expenses[groupID]...), nil
}

func (f *fakeStore) Settlements(ctx context.Context, groupID uuid.UUID) ([]Settlement, error) {
	return append([]Settlement{}, f.settlements[groupID]...), nil
}

type fakeGroupStore struct {
	groups      map[uuid.UUID]*group.Group
	memberships map[string]string
	members     map[uuid.UUID][]*group.Member
}

func newFakeGroupStore() *fakeGroupStore {
	return &fakeGroupStore{
		groups:      map[uuid.UUID]*group.Group{},
		memberships: map[string]string{},
		members:     map[uuid.UUID][]*group.Member{},
	}
}

func (f *fakeGroupStore) FindMembership(ctx context.Context, groupID, userID uuid.UUID) (*group.Membership, error) {
	role, ok := f.memberships[groupID.String()+":"+userID.String()]
	if !ok {
		return nil, group.ErrNotFound
	}
	return &group.Membership{GroupID: groupID, UserID: userID, Role: role}, nil
}

func (f *fakeGroupStore) ListMembers(ctx context.Context, groupID uuid.UUID) ([]*group.Member, error) {
	return append([]*group.Member{}, f.members[groupID]...), nil
}

func (f *fakeGroupStore) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*group.Group, error) {
	groups := []*group.Group{}
	for _, g := range f.groups {
		if _, ok := f.memberships[g.ID.String()+":"+userID.String()]; ok {
			cp := *g
			groups = append(groups, &cp)
		}
	}
	return groups, nil
}

func newTestService() (*Service, *fakeStore, *fakeGroupStore) {
	store := newFakeStore()
	gs := newFakeGroupStore()
	return NewService(store, gs), store, gs
}

func setupGroup(t *testing.T, gs *fakeGroupStore, users map[uuid.UUID]string, groupID uuid.UUID) {
	t.Helper()
	gs.groups[groupID] = &group.Group{ID: groupID, Name: "Trip", Currency: "IDR"}
	for userID, name := range users {
		gs.memberships[groupID.String()+":"+userID.String()] = group.RoleMember
		gs.members[groupID] = append(gs.members[groupID], &group.Member{UserID: userID, Name: name})
	}
}

func TestGroupBalances(t *testing.T) {
	svc, store, gs := newTestService()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin", b: "Budi", c: "Citra"}, g)

	// A paid 600000, split equal among A/B/C (200000 each).
	store.expenses[g] = []Expense{
		{PaidBy: a, AmountSen: 60000000, Splits: []Split{
			{UserID: a, AmountSen: 20000000},
			{UserID: b, AmountSen: 20000000},
			{UserID: c, AmountSen: 20000000},
		}},
	}

	balances, err := svc.GroupBalances(context.Background(), b, g)
	if err != nil {
		t.Fatalf("group balances: %v", err)
	}

	if len(balances) != 3 {
		t.Fatalf("expected 3 members, got %d", len(balances))
	}

	byUser := map[uuid.UUID]int64{}
	for _, mb := range balances {
		byUser[mb.UserID] = mb.BalanceSen
	}
	if byUser[a] != 40000000 {
		t.Errorf("expected A +40000000, got %d", byUser[a])
	}
	if byUser[b] != -20000000 {
		t.Errorf("expected B -20000000, got %d", byUser[b])
	}
	if byUser[c] != -20000000 {
		t.Errorf("expected C -20000000, got %d", byUser[c])
	}
}

func TestGroupBalancesIncludesZeroMembers(t *testing.T) {
	svc, store, gs := newTestService()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin", b: "Budi", c: "Citra"}, g)

	// Only A and B are involved in expenses; C has zero balance.
	store.expenses[g] = []Expense{
		{PaidBy: a, AmountSen: 1000000, Splits: []Split{{UserID: b, AmountSen: 1000000}}},
	}

	balances, err := svc.GroupBalances(context.Background(), a, g)
	if err != nil {
		t.Fatalf("group balances: %v", err)
	}

	if len(balances) != 3 {
		t.Fatalf("expected 3 members, got %d", len(balances))
	}

	var foundC bool
	for _, mb := range balances {
		if mb.UserID == c && mb.BalanceSen != 0 {
			t.Errorf("expected C balance 0, got %d", mb.BalanceSen)
		}
		if mb.UserID == c {
			foundC = true
		}
	}
	if !foundC {
		t.Error("member with zero balance missing from response")
	}
}

func TestGroupBalancesSettlementsApplied(t *testing.T) {
	svc, store, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin", b: "Budi"}, g)

	store.expenses[g] = []Expense{
		{PaidBy: a, AmountSen: 60000000, Splits: []Split{{UserID: b, AmountSen: 60000000}}},
	}
	store.settlements[g] = []Settlement{{PayerID: b, ReceiverID: a, AmountSen: 40000000}}

	balances, err := svc.GroupBalances(context.Background(), a, g)
	if err != nil {
		t.Fatalf("group balances: %v", err)
	}

	byUser := map[uuid.UUID]int64{}
	for _, mb := range balances {
		byUser[mb.UserID] = mb.BalanceSen
	}
	if byUser[a] != 20000000 {
		t.Errorf("expected A +20000000 after settlement, got %d", byUser[a])
	}
	if byUser[b] != -20000000 {
		t.Errorf("expected B -20000000 after settlement, got %d", byUser[b])
	}
}

func TestGroupBalancesNonMember(t *testing.T) {
	svc, _, gs := newTestService()
	a := uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin"}, g)

	_, err := svc.GroupBalances(context.Background(), uuid.New(), g)
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestSettlementSuggestions(t *testing.T) {
	svc, store, gs := newTestService()
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

	transfers, err := svc.SettlementSuggestions(context.Background(), b, g)
	if err != nil {
		t.Fatalf("settlement suggestions: %v", err)
	}

	if len(transfers) != 2 {
		t.Fatalf("expected 2 transfers, got %d", len(transfers))
	}
	if transfers[0].FromUserID != b || transfers[0].ToUserID != a || transfers[0].AmountSen != 40000000 {
		t.Errorf("unexpected transfer: %+v", transfers[0])
	}
	if transfers[1].FromUserID != c || transfers[1].ToUserID != a || transfers[1].AmountSen != 30000000 {
		t.Errorf("unexpected transfer: %+v", transfers[1])
	}
}

func TestSettlementSuggestionsNonMember(t *testing.T) {
	svc, _, gs := newTestService()
	a := uuid.New()
	g := uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin"}, g)

	_, err := svc.SettlementSuggestions(context.Background(), uuid.New(), g)
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestPersonalBalanceAcrossGroups(t *testing.T) {
	svc, store, gs := newTestService()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g1, g2 := uuid.New(), uuid.New()
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin", b: "Budi"}, g1)
	setupGroup(t, gs, map[uuid.UUID]string{a: "Arifin", c: "Citra"}, g2)

	// In g1: expense1 A paid 50000000 (A share 40000000, B share 10000000) → A +10000000.
	// expense2 A paid 10000000, B share 10000000 → A +10000000. g1 total for A: +20000000.
	store.expenses[g1] = []Expense{
		{PaidBy: a, AmountSen: 50000000, Splits: []Split{
			{UserID: a, AmountSen: 40000000},
			{UserID: b, AmountSen: 10000000},
		}},
		{PaidBy: a, AmountSen: 10000000, Splits: []Split{{UserID: b, AmountSen: 10000000}}},
	}
	// In g2: A owes C 300000.
	store.expenses[g2] = []Expense{
		{PaidBy: c, AmountSen: 30000000, Splits: []Split{{UserID: a, AmountSen: 30000000}}},
	}

	summary, err := svc.PersonalBalance(context.Background(), a)
	if err != nil {
		t.Fatalf("personal balance: %v", err)
	}

	if summary.OwedToUserSen != 20000000 {
		t.Errorf("expected owed to user 20000000, got %d", summary.OwedToUserSen)
	}
	if summary.UserOwesSen != 30000000 {
		t.Errorf("expected user owes 30000000, got %d", summary.UserOwesSen)
	}
	if summary.NetBalanceSen != -10000000 {
		t.Errorf("expected net -10000000, got %d", summary.NetBalanceSen)
	}
}

func TestPersonalBalanceEmpty(t *testing.T) {
	svc, _, _ := newTestService()
	a := uuid.New()

	summary, err := svc.PersonalBalance(context.Background(), a)
	if err != nil {
		t.Fatalf("personal balance: %v", err)
	}

	if summary.OwedToUserSen != 0 || summary.UserOwesSen != 0 || summary.NetBalanceSen != 0 {
		t.Errorf("expected zero summary, got %+v", summary)
	}
}
