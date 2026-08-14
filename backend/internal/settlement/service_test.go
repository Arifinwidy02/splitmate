package settlement

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/group"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
)

type fakeStore struct {
	settlements []*Settlement
	failCreate  error
}

func newFakeStore() *fakeStore {
	return &fakeStore{settlements: []*Settlement{}}
}

func (f *fakeStore) Create(ctx context.Context, s *Settlement) error {
	if f.failCreate != nil {
		return f.failCreate
	}
	cp := *s
	cp.ID = uuid.New()
	cp.PayerName = "Payer"
	cp.ReceiverName = "Receiver"
	f.settlements = append(f.settlements, &cp)
	*s = cp
	return nil
}

func (f *fakeStore) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]*Settlement, error) {
	result := []*Settlement{}
	for _, st := range f.settlements {
		if st.GroupID == groupID {
			cp := *st
			result = append(result, &cp)
		}
	}
	return result, nil
}

type fakeGroupStore struct {
	memberships map[string]string
}

func newFakeGroupStore() *fakeGroupStore {
	return &fakeGroupStore{memberships: map[string]string{}}
}

func (f *fakeGroupStore) FindMembership(ctx context.Context, groupID, userID uuid.UUID) (*group.Membership, error) {
	role, ok := f.memberships[groupID.String()+":"+userID.String()]
	if !ok {
		return nil, group.ErrNotFound
	}
	return &group.Membership{GroupID: groupID, UserID: userID, Role: role}, nil
}

func newTestService() (*Service, *fakeStore, *fakeGroupStore) {
	store := newFakeStore()
	gs := newFakeGroupStore()
	return NewService(store, gs), store, gs
}

func TestCreateSettlement(t *testing.T) {
	svc, store, gs := newTestService()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	gs.memberships[g.String()+":"+a.String()] = group.RoleMember
	gs.memberships[g.String()+":"+b.String()] = group.RoleMember

	st, err := svc.CreateSettlement(context.Background(), a, g, CreateSettlementInput{
		PayerID:    a,
		ReceiverID: b,
		AmountSen:  40000000,
	})
	if err != nil {
		t.Fatalf("create settlement: %v", err)
	}

	if st.ID == uuid.Nil {
		t.Error("expected settlement id")
	}
	if st.AmountSen != 40000000 {
		t.Errorf("expected amount 40000000, got %d", st.AmountSen)
	}
	if st.SettledAt.IsZero() {
		t.Error("expected settledAt to default to now")
	}
	if len(store.settlements) != 1 {
		t.Errorf("expected 1 settlement stored, got %d", len(store.settlements))
	}
}

func TestCreateSettlementWithSettledAt(t *testing.T) {
	svc, _, gs := newTestService()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	gs.memberships[g.String()+":"+a.String()] = group.RoleMember
	gs.memberships[g.String()+":"+b.String()] = group.RoleMember

	when := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	st, err := svc.CreateSettlement(context.Background(), a, g, CreateSettlementInput{
		PayerID:    a,
		ReceiverID: b,
		AmountSen:  50000,
		SettledAt:  when,
	})
	if err != nil {
		t.Fatalf("create settlement: %v", err)
	}
	if !st.SettledAt.Equal(when) {
		t.Errorf("expected settledAt %v, got %v", when, st.SettledAt)
	}
}

func TestCreateSettlementAuthz(t *testing.T) {
	svc, _, gs := newTestService()
	g := uuid.New()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	gs.memberships[g.String()+":"+a.String()] = group.RoleMember
	gs.memberships[g.String()+":"+b.String()] = group.RoleMember

	t.Run("non-member rejected", func(t *testing.T) {
		_, err := svc.CreateSettlement(context.Background(), c, g, CreateSettlementInput{
			PayerID:    a,
			ReceiverID: b,
			AmountSen:  1000,
		})
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("expected ErrGroupNotFound, got %v", err)
		}
	})

	t.Run("user not payer forbidden", func(t *testing.T) {
		_, err := svc.CreateSettlement(context.Background(), b, g, CreateSettlementInput{
			PayerID:    a,
			ReceiverID: b,
			AmountSen:  1000,
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("receiver not member rejected", func(t *testing.T) {
		_, err := svc.CreateSettlement(context.Background(), a, g, CreateSettlementInput{
			PayerID:    a,
			ReceiverID: c,
			AmountSen:  1000,
		})
		var valErr *apperror.Validation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})
}

func TestCreateSettlementValidation(t *testing.T) {
	svc, _, gs := newTestService()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	gs.memberships[g.String()+":"+a.String()] = group.RoleMember
	gs.memberships[g.String()+":"+b.String()] = group.RoleMember

	tests := []struct {
		name  string
		input CreateSettlementInput
	}{
		{"payer equals receiver", CreateSettlementInput{PayerID: a, ReceiverID: a, AmountSen: 1000}},
		{"zero amount", CreateSettlementInput{PayerID: a, ReceiverID: b, AmountSen: 0}},
		{"negative amount", CreateSettlementInput{PayerID: a, ReceiverID: b, AmountSen: -5}},
		{"amount too large", CreateSettlementInput{PayerID: a, ReceiverID: b, AmountSen: 100_000_000_000_000}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateSettlement(context.Background(), a, g, tt.input)
			var valErr *apperror.Validation
			if !errors.As(err, &valErr) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestListSettlements(t *testing.T) {
	svc, _, gs := newTestService()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	gs.memberships[g.String()+":"+a.String()] = group.RoleMember
	gs.memberships[g.String()+":"+b.String()] = group.RoleMember

	if _, err := svc.CreateSettlement(context.Background(), a, g, CreateSettlementInput{
		PayerID:    a,
		ReceiverID: b,
		AmountSen:  1000000,
	}); err != nil {
		t.Fatalf("create settlement: %v", err)
	}

	settlements, err := svc.ListSettlements(context.Background(), a, g)
	if err != nil {
		t.Fatalf("list settlements: %v", err)
	}

	if len(settlements) != 1 {
		t.Fatalf("expected 1 settlement, got %d", len(settlements))
	}
	if settlements[0].AmountSen != 1000000 {
		t.Errorf("expected amount 1000000, got %d", settlements[0].AmountSen)
	}
}

func TestListSettlementsNonMember(t *testing.T) {
	svc, _, gs := newTestService()
	g := uuid.New()
	a := uuid.New()
	gs.memberships[g.String()+":"+a.String()] = group.RoleMember

	_, err := svc.ListSettlements(context.Background(), uuid.New(), g)
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestListSettlementsEmpty(t *testing.T) {
	svc, _, gs := newTestService()
	g := uuid.New()
	a := uuid.New()
	gs.memberships[g.String()+":"+a.String()] = group.RoleMember

	settlements, err := svc.ListSettlements(context.Background(), a, g)
	if err != nil {
		t.Fatalf("list settlements: %v", err)
	}
	if len(settlements) != 0 {
		t.Errorf("expected empty list, got %d", len(settlements))
	}
}

func TestCreateSettlementStoreErrorWrapped(t *testing.T) {
	svc, store, gs := newTestService()
	g := uuid.New()
	a, b := uuid.New(), uuid.New()
	gs.memberships[g.String()+":"+a.String()] = group.RoleMember
	gs.memberships[g.String()+":"+b.String()] = group.RoleMember

	store.failCreate = errors.New("db down")

	_, err := svc.CreateSettlement(context.Background(), a, g, CreateSettlementInput{
		PayerID:    a,
		ReceiverID: b,
		AmountSen:  1000,
	})
	if err == nil || !errors.Is(err, store.failCreate) {
		t.Fatalf("expected wrapped db error, got %v", err)
	}
}
