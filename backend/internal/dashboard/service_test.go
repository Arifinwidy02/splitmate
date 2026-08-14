package dashboard

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/balance"
	"github.com/Arifinwidy02/splitmate-backend/internal/group"
	"github.com/Arifinwidy02/splitmate-backend/pkg/money"
)

type fakeDashboardStore struct {
	totalExpense int64
	settled      int64
	recent       []RecentExpense
	categories   []CategoryTotal
	failTotal    error
}

func (f *fakeDashboardStore) TotalExpense(ctx context.Context, userID uuid.UUID) (int64, error) {
	if f.failTotal != nil {
		return 0, f.failTotal
	}
	return f.totalExpense, nil
}

func (f *fakeDashboardStore) SettledAmount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return f.settled, nil
}

func (f *fakeDashboardStore) RecentExpenses(ctx context.Context, userID uuid.UUID, limit int) ([]RecentExpense, error) {
	return append([]RecentExpense{}, f.recent...), nil
}

func (f *fakeDashboardStore) CategoryTotals(ctx context.Context, userID uuid.UUID) ([]CategoryTotal, error) {
	return append([]CategoryTotal{}, f.categories...), nil
}

type fakeGroupStore struct {
	groups      []*group.Group
	memberships map[string]bool
}

func (f *fakeGroupStore) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*group.Group, error) {
	out := []*group.Group{}
	for _, g := range f.groups {
		if f.memberships[g.ID.String()+":"+userID.String()] {
			out = append(out, g)
		}
	}
	return out, nil
}

type fakeBalanceStore struct {
	expenses    map[uuid.UUID][]balance.Expense
	settlements map[uuid.UUID][]balance.Settlement
	failExpense error
}

func (f *fakeBalanceStore) ExpensesWithSplits(ctx context.Context, groupID uuid.UUID) ([]balance.Expense, error) {
	if f.failExpense != nil {
		return nil, f.failExpense
	}
	return append([]balance.Expense{}, f.expenses[groupID]...), nil
}

func (f *fakeBalanceStore) Settlements(ctx context.Context, groupID uuid.UUID) ([]balance.Settlement, error) {
	return append([]balance.Settlement{}, f.settlements[groupID]...), nil
}

func newTestService() (*Service, *fakeDashboardStore, *fakeGroupStore, *fakeBalanceStore) {
	dstore := &fakeDashboardStore{}
	gs := &fakeGroupStore{memberships: map[string]bool{}}
	bs := &fakeBalanceStore{
		expenses:    map[uuid.UUID][]balance.Expense{},
		settlements: map[uuid.UUID][]balance.Settlement{},
	}
	return NewService(dstore, gs, bs), dstore, gs, bs
}

func TestGetDashboardEmpty(t *testing.T) {
	svc, _, _, _ := newTestService()
	user := uuid.New()

	d, err := svc.GetDashboard(context.Background(), user)
	if err != nil {
		t.Fatalf("get dashboard: %v", err)
	}

	if d.Summary.OwedToUser != "0.00" || d.Summary.UserOwes != "0.00" || d.Summary.NetBalance != "0.00" {
		t.Errorf("unexpected summary: %+v", d.Summary)
	}
	if len(d.Groups) != 0 || len(d.RecentExpenses) != 0 || len(d.Categories) != 0 {
		t.Errorf("expected empty dashboard, got %+v", d)
	}
}

func TestGetDashboardSummaryAndGroups(t *testing.T) {
	svc, dstore, gs, bs := newTestService()

	me := uuid.New()
	a := uuid.New()
	b := uuid.New()
	g1 := uuid.New()
	g2 := uuid.New()

	gs.groups = []*group.Group{
		{ID: g1, Name: "Bali", Currency: "IDR", MemberCount: 3},
		{ID: g2, Name: "Kantor", Currency: "IDR", MemberCount: 2},
	}
	for _, g := range []uuid.UUID{g1, g2} {
		gs.memberships[g.String()+":"+me.String()] = true
	}

	// Group 1: me paid 600000, split among me/a/b (200000 each).
	// Balance: me +400000, a -200000, b -200000.
	bs.expenses[g1] = []balance.Expense{
		{PaidBy: me, AmountSen: 60000000, Splits: []balance.Split{
			{UserID: me, AmountSen: 20000000},
			{UserID: a, AmountSen: 20000000},
			{UserID: b, AmountSen: 20000000},
		}},
	}

	// Group 2: a paid 500000, split equal 2.
	// Balance before settlement: me -250000, a +250000.
	bs.expenses[g2] = []balance.Expense{
		{PaidBy: a, AmountSen: 50000000, Splits: []balance.Split{
			{UserID: me, AmountSen: 25000000},
			{UserID: a, AmountSen: 25000000},
		}},
	}

	// A settlement in group 2: me pays a 100000.
	// Balance after: me -150000, a +150000.
	bs.settlements[g2] = []balance.Settlement{
		{PayerID: me, ReceiverID: a, AmountSen: 10000000},
	}

	dstore.totalExpense = 110000000
	dstore.settled = 10000000
	dstore.categories = []CategoryTotal{{Category: "Food & Drinks", Total: money.FormatMajor(60000000)}}

	d, err := svc.GetDashboard(context.Background(), me)
	if err != nil {
		t.Fatalf("get dashboard: %v", err)
	}

	if got := d.Summary.OwedToUser; got != "400000.00" {
		t.Errorf("expected owedToUser 400000.00, got %s", got)
	}
	if got := d.Summary.UserOwes; got != "150000.00" {
		t.Errorf("expected userOwes 150000.00, got %s", got)
	}
	if got := d.Summary.NetBalance; got != "250000.00" {
		t.Errorf("expected netBalance 250000.00, got %s", got)
	}
	if got := d.Summary.TotalExpense; got != "1100000.00" {
		t.Errorf("expected totalExpense 1100000.00, got %s", got)
	}
	if got := d.Summary.SettledAmount; got != "100000.00" {
		t.Errorf("expected settledAmount 100000.00, got %s", got)
	}

	if len(d.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(d.Groups))
	}
	byID := map[uuid.UUID]string{}
	for _, g := range d.Groups {
		byID[g.ID] = g.Balance
	}
	if byID[g1] != "400000.00" {
		t.Errorf("expected group 1 balance 400000.00, got %s", byID[g1])
	}
	if byID[g2] != "-150000.00" {
		t.Errorf("expected group 2 balance -150000.00, got %s", byID[g2])
	}
	if d.Groups[0].MemberCount != 3 {
		t.Errorf("expected memberCount 3, got %d", d.Groups[0].MemberCount)
	}

	if len(d.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(d.Categories))
	}
	if d.Categories[0].Total != "600000.00" {
		t.Errorf("expected category total 600000.00, got %s", d.Categories[0].Total)
	}
}

func TestGetDashboardPropagatesErrors(t *testing.T) {
	svc, dstore, _, bs := newTestService()

	me := uuid.New()
	dstore.failTotal = errors.New("boom")

	if _, err := svc.GetDashboard(context.Background(), me); err == nil {
		t.Fatal("expected error from total expense store, got nil")
	}

	// Balance store failure should also surface.
	dstore.failTotal = nil
	g := uuid.New()
	bs.failExpense = errors.New("boom")
	svc.groupStore = &fakeGroupStore{
		groups:      []*group.Group{{ID: g, Name: "G", Currency: "IDR"}},
		memberships: map[string]bool{g.String() + ":" + me.String(): true},
	}

	if _, err := svc.GetDashboard(context.Background(), me); err == nil {
		t.Fatal("expected error from balance store, got nil")
	}
}
