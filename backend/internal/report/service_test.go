package report

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeStore struct {
	group       *Group
	memberships map[string]bool
	expenses    []ExpenseRow
	settlements []SettlementRow
	members     []MemberRow
}

func (f *fakeStore) FindGroup(ctx context.Context, groupID uuid.UUID) (*Group, error) {
	if f.group == nil {
		return nil, ErrNotFound
	}
	return f.group, nil
}

func (f *fakeStore) FindMembership(ctx context.Context, groupID, userID uuid.UUID) error {
	if !f.memberships[groupID.String()+":"+userID.String()] {
		return ErrNotFound
	}
	return nil
}

func (f *fakeStore) ExpensesWithSplits(ctx context.Context, groupID uuid.UUID) ([]ExpenseRow, error) {
	return f.expenses, nil
}

func (f *fakeStore) Settlements(ctx context.Context, groupID uuid.UUID) ([]SettlementRow, error) {
	return f.settlements, nil
}

func (f *fakeStore) Members(ctx context.Context, groupID uuid.UUID) ([]MemberRow, error) {
	return f.members, nil
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		memberships: map[string]bool{},
	}
}

func newTestService() (*Service, *fakeStore) {
	store := newFakeStore()
	return NewService(store), store
}

func TestBuildReportRejectsNonMember(t *testing.T) {
	svc, store := newTestService()
	store.group = &Group{Name: "Bali Trip", Currency: "IDR"}
	store.members = []MemberRow{{UserID: uuid.New(), Name: "Arifin"}}

	groupID := uuid.New()
	outsider := uuid.New()

	_, err := svc.BuildReport(context.Background(), outsider, groupID)
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound for non-member, got %v", err)
	}
}

func TestBuildReportRejectsMissingGroup(t *testing.T) {
	svc, store := newTestService()
	userID := uuid.New()
	store.memberships["missing:"+userID.String()] = true

	_, err := svc.BuildReport(context.Background(), userID, uuid.New())
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound for missing group, got %v", err)
	}
}

func TestBuildReportHappyPath(t *testing.T) {
	svc, store := newTestService()

	groupID := uuid.New()
	arifin := uuid.New()
	budi := uuid.New()
	citra := uuid.New()

	store.group = &Group{Name: "Bali Trip", Currency: "IDR"}
	store.memberships[groupID.String()+":"+arifin.String()] = true
	store.members = []MemberRow{
		{UserID: arifin, Name: "Arifin"},
		{UserID: budi, Name: "Budi"},
		{UserID: citra, Name: "Citra"},
	}

	store.expenses = []ExpenseRow{
		{
			ID:          uuid.New(),
			Description: "Dinner",
			Category:    "Food & Drinks",
			PaidBy:      arifin,
			PaidByName:  "Arifin",
			ExpenseDate: time.Date(2026, 8, 10, 19, 0, 0, 0, time.UTC),
			AmountSen:   60000000,
			Participants: []ParticipantRow{
				{UserID: arifin, Name: "Arifin", AmountSen: 20000000},
				{UserID: budi, Name: "Budi", AmountSen: 20000000},
				{UserID: citra, Name: "Citra", AmountSen: 20000000},
			},
		},
	}
	store.settlements = []SettlementRow{
		{
			PayerID:      budi,
			ReceiverID:   arifin,
			PayerName:    "Budi",
			ReceiverName: "Arifin",
			AmountSen:    10000000,
			SettledAt:    time.Date(2026, 8, 12, 15, 0, 0, 0, time.UTC),
		},
	}

	report, err := svc.BuildReport(context.Background(), arifin, groupID)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}

	if report.GroupName != "Bali Trip" || report.GroupCurrency != "IDR" {
		t.Errorf("group header = %q/%q, want Bali Trip/IDR", report.GroupName, report.GroupCurrency)
	}
	if report.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}

	// A: +600000 - 200000 (own share) - 100000 (settlement received) = +300000
	// B: -200000 + 100000 (settlement paid) = -100000
	// C: -200000
	balanceByID := map[uuid.UUID]int64{}
	for _, b := range report.Balances {
		balanceByID[b.UserID] = b.BalanceSen
	}
	if balanceByID[arifin] != 30000000 {
		t.Errorf("Arifin balance = %d, want 30000000", balanceByID[arifin])
	}
	if balanceByID[budi] != -10000000 {
		t.Errorf("Budi balance = %d, want -10000000", balanceByID[budi])
	}
	if balanceByID[citra] != -20000000 {
		t.Errorf("Citra balance = %d, want -20000000", balanceByID[citra])
	}

	if len(report.Balances) != 3 {
		t.Fatalf("balances = %d rows, want 3", len(report.Balances))
	}
	for i := 1; i < len(report.Balances); i++ {
		if report.Balances[i].Name < report.Balances[i-1].Name {
			t.Errorf("balances not sorted by name: %v", report.Balances)
		}
	}

	if len(report.Suggestions) != 2 {
		t.Fatalf("suggestions = %d rows, want 2", len(report.Suggestions))
	}
	suggestionByFrom := map[string]SuggestionRow{}
	for _, s := range report.Suggestions {
		suggestionByFrom[s.FromName] = s
	}
	if s := suggestionByFrom["Budi"]; s.ToName != "Arifin" || s.AmountSen != 10000000 {
		t.Errorf("Budi suggestion = %+v, want -> Arifin 10000000", s)
	}
	if s := suggestionByFrom["Citra"]; s.ToName != "Arifin" || s.AmountSen != 20000000 {
		t.Errorf("Citra suggestion = %+v, want -> Arifin 20000000", s)
	}

	if len(report.Expenses) != 1 || report.Expenses[0].Description != "Dinner" {
		t.Errorf("expenses = %+v, want Dinner", report.Expenses)
	}
	if len(report.Settlements) != 1 || report.Settlements[0].PayerName != "Budi" {
		t.Errorf("settlements = %+v, want Budi", report.Settlements)
	}
}

func TestBuildReportNoExpenses(t *testing.T) {
	svc, store := newTestService()

	groupID := uuid.New()
	userID := uuid.New()
	store.group = &Group{Name: "New Group", Currency: "IDR"}
	store.memberships[groupID.String()+":"+userID.String()] = true
	store.members = []MemberRow{{UserID: userID, Name: "Arifin"}}

	report, err := svc.BuildReport(context.Background(), userID, groupID)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}

	if len(report.Expenses) != 0 || len(report.Settlements) != 0 {
		t.Errorf("expected empty expenses/settlements, got %d/%d", len(report.Expenses), len(report.Settlements))
	}
	if len(report.Balances) != 0 {
		t.Errorf("expected no balances, got %d", len(report.Balances))
	}
	if len(report.Suggestions) != 0 {
		t.Errorf("expected no suggestions, got %d", len(report.Suggestions))
	}
}
