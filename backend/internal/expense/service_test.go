package expense

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/group"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
)

type fakeStore struct {
	expenses   map[uuid.UUID]*Expense
	splits     map[uuid.UUID][]Participant
	seq        []uuid.UUID
	failCreate error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		expenses: map[uuid.UUID]*Expense{},
		splits:   map[uuid.UUID][]Participant{},
	}
}

func (f *fakeStore) CreateExpenseWithSplits(ctx context.Context, e *Expense, splits []SplitAmount) (*Expense, error) {
	if f.failCreate != nil {
		return nil, f.failCreate
	}
	now := time.Now()
	cp := *e
	cp.ID = uuid.New()
	cp.CreatedAt = now
	cp.UpdatedAt = now
	f.expenses[cp.ID] = &cp
	f.splits[cp.ID] = nil
	for _, s := range splits {
		f.splits[cp.ID] = append(f.splits[cp.ID], Participant{UserID: s.UserID, AmountSen: s.AmountSen})
	}
	f.seq = append(f.seq, cp.ID)
	return &cp, nil
}

func (f *fakeStore) FindByID(ctx context.Context, id uuid.UUID) (*Expense, []Participant, error) {
	e, ok := f.expenses[id]
	if !ok {
		return nil, nil, ErrNotFound
	}
	cp := *e
	return &cp, f.splits[id], nil
}

func (f *fakeStore) UpdateExpenseWithSplits(ctx context.Context, e *Expense, splits []SplitAmount) error {
	existing, ok := f.expenses[e.ID]
	if !ok {
		return ErrNotFound
	}
	*existing = *e
	f.splits[e.ID] = nil
	for _, s := range splits {
		f.splits[e.ID] = append(f.splits[e.ID], Participant{UserID: s.UserID, AmountSen: s.AmountSen})
	}
	return nil
}

func (f *fakeStore) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := f.expenses[id]; !ok {
		return ErrNotFound
	}
	delete(f.expenses, id)
	delete(f.splits, id)
	return nil
}

func (f *fakeStore) ListByGroup(ctx context.Context, groupID uuid.UUID, page, limit int, category string, from, to *time.Time) ([]*ExpenseSummary, int, error) {
	summaries := []*ExpenseSummary{}
	for _, id := range f.seq {
		e := f.expenses[id]
		if e.GroupID != groupID {
			continue
		}
		if category != "" && e.Category != category {
			continue
		}
		summaries = append(summaries, &ExpenseSummary{
			Expense:          *e,
			PayerName:        "Payer",
			ParticipantCount: len(f.splits[id]),
		})
	}
	return summaries, len(summaries), nil
}

type fakeGroupStore struct {
	groups      map[uuid.UUID]*group.Group
	members     map[uuid.UUID][]*group.Member
	memberships map[string]string
}

func newFakeGroupStore() *fakeGroupStore {
	return &fakeGroupStore{
		groups:      map[uuid.UUID]*group.Group{},
		members:     map[uuid.UUID][]*group.Member{},
		memberships: map[string]string{},
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

func (f *fakeGroupStore) FindByID(ctx context.Context, id uuid.UUID) (*group.Group, error) {
	g, ok := f.groups[id]
	if !ok {
		return nil, group.ErrNotFound
	}
	cp := *g
	return &cp, nil
}

func newTestService() (*Service, *fakeStore, *fakeGroupStore) {
	store := newFakeStore()
	gs := newFakeGroupStore()
	return NewService(store, gs), store, gs
}

func setupGroup(t *testing.T, gs *fakeGroupStore, userIDs ...uuid.UUID) *group.Group {
	t.Helper()
	g := &group.Group{ID: uuid.New(), Name: "Trip", Currency: "IDR"}
	gs.groups[g.ID] = g
	now := time.Now()
	for _, id := range userIDs {
		gs.memberships[g.ID.String()+":"+id.String()] = group.RoleMember
		gs.members[g.ID] = append(gs.members[g.ID], &group.Member{UserID: id, Name: "User", Email: "u@test.com", Role: group.RoleMember, JoinedAt: now})
	}
	return g
}

func baseInput(g *group.Group, paidBy uuid.UUID) CreateExpenseInput {
	return CreateExpenseInput{
		Description: "Dinner",
		AmountSen:   60000000,
		Currency:    g.Currency,
		PaidBy:      paidBy,
		Category:    "Food & Drinks",
		ExpenseDate: time.Now(),
		SplitType:   SplitEqual,
	}
}

func asValidationError(err error) *apperror.Validation {
	var valErr *apperror.Validation
	if errors.As(err, &valErr) {
		return valErr
	}
	return nil
}

func TestCreateExpenseEqualSplit(t *testing.T) {
	svc, store, gs := newTestService()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b, c)

	input := baseInput(g, a)
	input.AmountSen = 60000000
	input.EqualIDs = []uuid.UUID{a, b, c}

	e, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}

	splits := store.splits[e.ID]
	if len(splits) != 3 {
		t.Fatalf("expected 3 splits, got %d", len(splits))
	}

	var total int64
	for _, s := range splits {
		if s.AmountSen != 20000000 {
			t.Errorf("expected equal share 20000000, got %d", s.AmountSen)
		}
		total += s.AmountSen
	}
	if total != e.AmountSen {
		t.Errorf("splits must sum to amount: got %d, want %d", total, e.AmountSen)
	}
}

func TestCreateExpenseEqualSplitRounding(t *testing.T) {
	svc, store, gs := newTestService()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b, c)

	input := baseInput(g, a)
	input.AmountSen = 10001 // 100.01 / 3 → 3334, 3334, 3333
	input.EqualIDs = []uuid.UUID{a, b, c}

	e, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}

	splits := store.splits[e.ID]
	var total int64
	for _, s := range splits {
		total += s.AmountSen
	}
	if total != 10001 {
		t.Errorf("rounded splits must sum to amount: got %d", total)
	}

	sorted := map[uuid.UUID]int64{}
	for _, s := range splits {
		sorted[s.UserID] = s.AmountSen
	}
	values := []int64{sorted[a], sorted[b], sorted[c]}
	if !(values[0] == 3334 || values[1] == 3334 || values[2] == 3334) {
		t.Errorf("expected two 3334 shares, got %v", values)
	}
}

func TestCreateExpenseEqualSplitTooSmall(t *testing.T) {
	svc, _, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	input := baseInput(g, a)
	input.AmountSen = 1 // 0.01 for 2 people impossible
	input.EqualIDs = []uuid.UUID{a, b}

	_, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if asValidationError(err) == nil {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestCreateExpenseCustomSplit(t *testing.T) {
	svc, store, gs := newTestService()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b, c)

	input := baseInput(g, a)
	input.AmountSen = 60000000
	input.SplitType = SplitCustom
	input.Splits = []SplitAmount{
		{UserID: a, AmountSen: 10000000},
		{UserID: b, AmountSen: 25000000},
		{UserID: c, AmountSen: 25000000},
	}

	e, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}

	var total int64
	for _, s := range store.splits[e.ID] {
		total += s.AmountSen
	}
	if total != 60000000 {
		t.Errorf("expected total 60000000, got %d", total)
	}
}

func TestCreateExpenseCustomSplitMismatch(t *testing.T) {
	svc, _, gs := newTestService()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b, c)

	input := baseInput(g, a)
	input.AmountSen = 60000000
	input.SplitType = SplitCustom
	input.Splits = []SplitAmount{
		{UserID: a, AmountSen: 10000000},
		{UserID: b, AmountSen: 20000000},
		{UserID: c, AmountSen: 25000000},
	}

	_, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if !errors.Is(err, ErrInvalidSplit) {
		t.Fatalf("expected ErrInvalidSplit, got %v", err)
	}
}

func TestCreateExpenseValidation(t *testing.T) {
	svc, _, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	tests := []struct {
		name string
		mut  func(*CreateExpenseInput)
	}{
		{"empty description", func(i *CreateExpenseInput) { i.Description = "  " }},
		{"zero amount", func(i *CreateExpenseInput) { i.AmountSen = 0 }},
		{"negative amount", func(i *CreateExpenseInput) { i.AmountSen = -5 }},
		{"amount too large", func(i *CreateExpenseInput) { i.AmountSen = 100_000_000_000_000 }},
		{"currency mismatch", func(i *CreateExpenseInput) { i.Currency = "USD" }},
		{"bad category", func(i *CreateExpenseInput) { i.Category = "Pets" }},
		{"payer not member", func(i *CreateExpenseInput) { i.PaidBy = uuid.New() }},
		{"participant not member", func(i *CreateExpenseInput) { i.EqualIDs = []uuid.UUID{a, b, uuid.New()} }},
		{"duplicate participants", func(i *CreateExpenseInput) { i.EqualIDs = []uuid.UUID{a, a} }},
		{"no participants", func(i *CreateExpenseInput) { i.EqualIDs = nil }},
		{"unknown split type", func(i *CreateExpenseInput) { i.SplitType = "percentage" }},
		{"zero split amount", func(i *CreateExpenseInput) {
			i.SplitType = SplitCustom
			i.Splits = []SplitAmount{{UserID: a, AmountSen: 0}}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := baseInput(g, a)
			input.EqualIDs = []uuid.UUID{a, b}
			tt.mut(&input)

			_, err := svc.CreateExpense(context.Background(), a, g.ID, input)
			if asValidationError(err) == nil {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestCreateExpenseNonMemberRejected(t *testing.T) {
	svc, _, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	input := baseInput(g, a)
	input.EqualIDs = []uuid.UUID{a, b}

	outsider := uuid.New()
	_, err := svc.CreateExpense(context.Background(), outsider, g.ID, input)
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestGetExpenseAuthorization(t *testing.T) {
	svc, _, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	input := baseInput(g, a)
	input.EqualIDs = []uuid.UUID{a, b}

	e, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}

	t.Run("member can view", func(t *testing.T) {
		got, err := svc.GetExpense(context.Background(), b, e.ID)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got.Participants) != 2 {
			t.Errorf("expected 2 participants, got %d", len(got.Participants))
		}
	})

	t.Run("non-member rejected", func(t *testing.T) {
		_, err := svc.GetExpense(context.Background(), uuid.New(), e.ID)
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})

	t.Run("unknown expense", func(t *testing.T) {
		_, err := svc.GetExpense(context.Background(), a, uuid.New())
		if !errors.Is(err, ErrExpenseNotFound) {
			t.Fatalf("expected ErrExpenseNotFound, got %v", err)
		}
	})
}

func TestUpdateExpenseAuthorization(t *testing.T) {
	svc, _, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	input := baseInput(g, a)
	input.EqualIDs = []uuid.UUID{a, b}

	e, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}

	update := baseInput(g, a)
	update.AmountSen = 70000000
	update.EqualIDs = []uuid.UUID{a, b}

	t.Run("creator can update", func(t *testing.T) {
		got, err := svc.UpdateExpense(context.Background(), a, e.ID, update)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if got.AmountSen != 70000000 {
			t.Errorf("expected updated amount, got %d", got.AmountSen)
		}
	})

	t.Run("non-creator member forbidden", func(t *testing.T) {
		_, err := svc.UpdateExpense(context.Background(), b, e.ID, update)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("non-member rejected", func(t *testing.T) {
		_, err := svc.UpdateExpense(context.Background(), uuid.New(), e.ID, update)
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})
}

func TestDeleteExpenseAuthorization(t *testing.T) {
	svc, store, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	input := baseInput(g, a)
	input.EqualIDs = []uuid.UUID{a, b}

	e, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}

	t.Run("non-creator member forbidden", func(t *testing.T) {
		err := svc.DeleteExpense(context.Background(), b, e.ID)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("creator can delete", func(t *testing.T) {
		if err := svc.DeleteExpense(context.Background(), a, e.ID); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if _, ok := store.expenses[e.ID]; ok {
			t.Errorf("expected expense to be deleted")
		}
	})

	t.Run("deleting again not found", func(t *testing.T) {
		err := svc.DeleteExpense(context.Background(), a, e.ID)
		if !errors.Is(err, ErrExpenseNotFound) {
			t.Fatalf("expected ErrExpenseNotFound, got %v", err)
		}
	})
}

func TestListExpensesAuthorization(t *testing.T) {
	svc, _, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	input := baseInput(g, a)
	input.EqualIDs = []uuid.UUID{a, b}

	if _, err := svc.CreateExpense(context.Background(), a, g.ID, input); err != nil {
		t.Fatalf("create expense: %v", err)
	}

	t.Run("member can list", func(t *testing.T) {
		expenses, total, err := svc.ListExpenses(context.Background(), b, g.ID, 1, 20, "", nil, nil)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if total != 1 || len(expenses) != 1 {
			t.Errorf("expected 1 expense, got %d (total %d)", len(expenses), total)
		}
	})

	t.Run("non-member rejected", func(t *testing.T) {
		_, _, err := svc.ListExpenses(context.Background(), uuid.New(), g.ID, 1, 20, "", nil, nil)
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})
}

func TestEqualSplitDeterministic(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	s1, err := equalSplit(100000, []uuid.UUID{a, b, c})
	if err != nil {
		t.Fatalf("equalSplit: %v", err)
	}
	s2, err := equalSplit(100000, []uuid.UUID{c, a, b})
	if err != nil {
		t.Fatalf("equalSplit: %v", err)
	}

	byUser := func(splits []SplitAmount) map[uuid.UUID]int64 {
		m := map[uuid.UUID]int64{}
		for _, s := range splits {
			m[s.UserID] = s.AmountSen
		}
		return m
	}

	m1, m2 := byUser(s1), byUser(s2)
	for _, id := range []uuid.UUID{a, b, c} {
		if m1[id] != m2[id] {
			t.Errorf("equal split not deterministic for %v: %d vs %d", id, m1[id], m2[id])
		}
	}

	var total int64
	for _, v := range m1 {
		total += v
	}
	if total != 100000 {
		t.Errorf("expected total 100000, got %d", total)
	}
}

func TestEqualSplitInsufficientFunds(t *testing.T) {
	_, err := equalSplit(2, []uuid.UUID{uuid.New(), uuid.New(), uuid.New()})
	if !errors.Is(err, ErrInvalidSplit) {
		t.Fatalf("expected ErrInvalidSplit, got %v", err)
	}
}

func TestCreateExpenseStoreErrorWrapped(t *testing.T) {
	svc, store, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	store.failCreate = errors.New("db down")

	input := baseInput(g, a)
	input.EqualIDs = []uuid.UUID{a, b}

	_, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if err == nil || !errors.Is(err, store.failCreate) {
		t.Fatalf("expected wrapped db error, got %v", err)
	}
}

func TestEqualSplitSingleParticipant(t *testing.T) {
	a := uuid.New()
	splits, err := equalSplit(50000, []uuid.UUID{a})
	if err != nil {
		t.Fatalf("equalSplit: %v", err)
	}
	if len(splits) != 1 || splits[0].AmountSen != 50000 {
		t.Errorf("expected single split of 50000, got %+v", splits)
	}
}

func TestCreateExpenseWithReceipt(t *testing.T) {
	svc, store, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	input := baseInput(g, a)
	input.EqualIDs = []uuid.UUID{a, b}
	input.Receipt = &Receipt{Image: []byte("fake-jpeg-bytes"), ContentType: "image/jpeg"}

	created, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}
	if !bytes.Equal(created.ReceiptImage, []byte("fake-jpeg-bytes")) {
		t.Error("expected receipt image to be persisted")
	}
	if created.ReceiptContentType != "image/jpeg" {
		t.Errorf("expected content type image/jpeg, got %q", created.ReceiptContentType)
	}

	stored := store.expenses[created.ID]
	if !bytes.Equal(stored.ReceiptImage, []byte("fake-jpeg-bytes")) {
		t.Error("expected receipt stored in fake store")
	}
}

func TestCreateExpenseReceiptValidation(t *testing.T) {
	cases := []struct {
		name    string
		receipt *Receipt
		wantMsg string
	}{
		{"empty image", &Receipt{ContentType: "image/jpeg"}, "Receipt image is empty"},
		{"oversized", &Receipt{Image: make([]byte, maxReceiptBytes+1), ContentType: "image/jpeg"}, "Receipt image must be at most 5MB"},
		{"unsupported type", &Receipt{Image: []byte("x"), ContentType: "application/pdf"}, "Receipt must be a JPEG, PNG, WebP or GIF image"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, gs := newTestService()
			a := uuid.New()
			g := setupGroup(t, gs, a)

			input := baseInput(g, a)
			input.EqualIDs = []uuid.UUID{a}
			input.Receipt = tc.receipt

			_, err := svc.CreateExpense(context.Background(), a, g.ID, input)
			valErr := asValidationError(err)
			if valErr == nil || valErr.Message != tc.wantMsg {
				t.Fatalf("expected validation %q, got %v", tc.wantMsg, err)
			}
		})
	}
}

func TestGetReceipt(t *testing.T) {
	svc, _, gs := newTestService()
	a, b := uuid.New(), uuid.New()
	g := setupGroup(t, gs, a, b)

	input := baseInput(g, a)
	input.EqualIDs = []uuid.UUID{a, b}
	input.Receipt = &Receipt{Image: []byte("img-bytes"), ContentType: "image/png"}
	created, err := svc.CreateExpense(context.Background(), a, g.ID, input)
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}

	image, contentType, err := svc.GetReceipt(context.Background(), b, created.ID)
	if err != nil {
		t.Fatalf("get receipt: %v", err)
	}
	if !bytes.Equal(image, []byte("img-bytes")) || contentType != "image/png" {
		t.Errorf("expected receipt bytes and image/png, got %q %q", image, contentType)
	}

	noReceiptInput := baseInput(g, a)
	noReceiptInput.EqualIDs = []uuid.UUID{a}
	noReceipt, err := svc.CreateExpense(context.Background(), a, g.ID, noReceiptInput)
	if err != nil {
		t.Fatalf("create expense without receipt: %v", err)
	}
	if _, _, err := svc.GetReceipt(context.Background(), a, noReceipt.ID); !errors.Is(err, ErrNoReceipt) {
		t.Errorf("expected ErrNoReceipt, got %v", err)
	}

	outsider := uuid.New()
	if _, _, err := svc.GetReceipt(context.Background(), outsider, created.ID); !errors.Is(err, ErrGroupNotFound) {
		t.Errorf("expected ErrGroupNotFound for non-member, got %v", err)
	}

	if _, _, err := svc.GetReceipt(context.Background(), a, uuid.New()); !errors.Is(err, ErrExpenseNotFound) {
		t.Errorf("expected ErrExpenseNotFound, got %v", err)
	}
}
