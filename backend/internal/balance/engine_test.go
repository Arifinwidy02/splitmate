package balance

import (
	"testing"

	"github.com/google/uuid"
)

func exp(paidBy uuid.UUID, amount int64, splits ...Split) Expense {
	return Expense{PaidBy: paidBy, AmountSen: amount, Splits: splits}
}

func split(userID uuid.UUID, amount int64) Split {
	return Split{UserID: userID, AmountSen: amount}
}

func setl(payer, receiver uuid.UUID, amount int64) Settlement {
	return Settlement{PayerID: payer, ReceiverID: receiver, AmountSen: amount}
}

func balanceOf(t *testing.T, balances map[uuid.UUID]int64, user uuid.UUID) int64 {
	t.Helper()
	if balances == nil {
		t.Fatalf("balances map is nil")
	}
	return balances[user]
}

func transferTotal(transfers []Transfer) int64 {
	var total int64
	for _, tr := range transfers {
		total += tr.AmountSen
	}
	return total
}

func TestCalculateBalancesBasic(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()

	// A paid 600000, each of A/B/C owes 200000.
	expenses := []Expense{
		exp(a, 60000000, split(a, 20000000), split(b, 20000000), split(c, 20000000)),
	}

	balances := CalculateBalances(expenses, nil)

	if got := balanceOf(t, balances, a); got != 40000000 {
		t.Errorf("expected A balance 40000000, got %d", got)
	}
	if got := balanceOf(t, balances, b); got != -20000000 {
		t.Errorf("expected B balance -20000000, got %d", got)
	}
	if got := balanceOf(t, balances, c); got != -20000000 {
		t.Errorf("expected C balance -20000000, got %d", got)
	}
}

func TestCalculateBalancesPayerNotParticipant(t *testing.T) {
	a, b := uuid.New(), uuid.New()

	// A paid 600000, B owes everything.
	expenses := []Expense{exp(a, 60000000, split(b, 60000000))}

	balances := CalculateBalances(expenses, nil)

	if got := balanceOf(t, balances, a); got != 60000000 {
		t.Errorf("expected A balance 60000000, got %d", got)
	}
	if got := balanceOf(t, balances, b); got != -60000000 {
		t.Errorf("expected B balance -60000000, got %d", got)
	}
}

func TestCalculateBalancesSettlements(t *testing.T) {
	a, b := uuid.New(), uuid.New()

	// A paid 600000, B owes 600000. B repays 400000.
	expenses := []Expense{exp(a, 60000000, split(b, 60000000))}
	settlements := []Settlement{setl(b, a, 40000000)}

	balances := CalculateBalances(expenses, settlements)

	if got := balanceOf(t, balances, a); got != 20000000 {
		t.Errorf("expected A balance 20000000 after settlement, got %d", got)
	}
	if got := balanceOf(t, balances, b); got != -20000000 {
		t.Errorf("expected B balance -20000000 after settlement, got %d", got)
	}
}

func TestCalculateBalancesFullSettlement(t *testing.T) {
	a, b := uuid.New(), uuid.New()

	expenses := []Expense{exp(a, 60000000, split(b, 60000000))}
	settlements := []Settlement{setl(b, a, 60000000)}

	balances := CalculateBalances(expenses, settlements)

	if got := balanceOf(t, balances, a); got != 0 {
		t.Errorf("expected A balance 0, got %d", got)
	}
	if got := balanceOf(t, balances, b); got != 0 {
		t.Errorf("expected B balance 0, got %d", got)
	}
}

func TestSimplifyDebtsPRDExample(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	balances := map[uuid.UUID]int64{
		a: 70000000,
		b: -40000000,
		c: -30000000,
	}

	transfers := SimplifyDebts(balances)

	if len(transfers) != 2 {
		t.Fatalf("expected 2 transfers, got %d: %+v", len(transfers), transfers)
	}
	// B -> A 400, C -> A 300.
	got := map[string]int64{}
	for _, tr := range transfers {
		got[tr.FromUserID.String()+">"+tr.ToUserID.String()] = tr.AmountSen
	}
	if got[b.String()+">"+a.String()] != 40000000 {
		t.Errorf("expected B -> A 40000000, got %+v", got)
	}
	if got[c.String()+">"+a.String()] != 30000000 {
		t.Errorf("expected C -> A 30000000, got %+v", got)
	}
}

func TestSimplifyDebtsTwoPeople(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	balances := map[uuid.UUID]int64{a: 5000000, b: -5000000}

	transfers := SimplifyDebts(balances)

	if len(transfers) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(transfers))
	}
	if transfers[0].FromUserID != b || transfers[0].ToUserID != a || transfers[0].AmountSen != 5000000 {
		t.Errorf("unexpected transfer: %+v", transfers[0])
	}
}

func TestSimplifyDebtsZeroBalance(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	balances := map[uuid.UUID]int64{a: 0, b: 0}

	transfers := SimplifyDebts(balances)
	if len(transfers) != 0 {
		t.Errorf("expected no transfers, got %d", len(transfers))
	}
}

func TestSimplifyDebtsEmpty(t *testing.T) {
	transfers := SimplifyDebts(map[uuid.UUID]int64{})
	if len(transfers) != 0 {
		t.Errorf("expected no transfers, got %d", len(transfers))
	}
}

func TestSimplifyDebtsMultipleCreditors(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	// A +700, B +300, C -1000.
	balances := map[uuid.UUID]int64{a: 70000000, b: 30000000, c: -100000000}

	transfers := SimplifyDebts(balances)

	if len(transfers) != 2 {
		t.Fatalf("expected 2 transfers, got %d: %+v", len(transfers), transfers)
	}

	// C must pay both A and B, largest creditor first.
	if transfers[0].FromUserID != c || transfers[0].ToUserID != a || transfers[0].AmountSen != 70000000 {
		t.Errorf("unexpected first transfer: %+v", transfers[0])
	}
	if transfers[1].FromUserID != c || transfers[1].ToUserID != b || transfers[1].AmountSen != 30000000 {
		t.Errorf("unexpected second transfer: %+v", transfers[1])
	}
}

func TestSimplifyDebtsMultipleDebtors(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	// A +700, B -400, C -300.
	balances := map[uuid.UUID]int64{a: 70000000, b: -40000000, c: -30000000}

	transfers := SimplifyDebts(balances)

	if len(transfers) != 2 {
		t.Fatalf("expected 2 transfers, got %d", len(transfers))
	}
	// Largest debtor pays first.
	if transfers[0].FromUserID != b {
		t.Errorf("expected B as first payer, got %v", transfers[0].FromUserID)
	}
}

func TestSimplifyDebtsTiesDeterministic(t *testing.T) {
	// Two creditors with equal claims: A +500, B +500, C -1000.
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	balances := map[uuid.UUID]int64{a: 5000000, b: 5000000, c: -10000000}

	first := SimplifyDebts(balances)
	second := SimplifyDebts(balances)

	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("expected 2 transfers, got %d and %d", len(first), len(second))
	}
	if first[0] != second[0] || first[1] != second[1] {
		t.Errorf("result not deterministic: %+v vs %+v", first, second)
	}

	// The first creditor must be the one with the smaller user id.
	firstCreditor := first[0].ToUserID
	var want uuid.UUID
	if a.String() < b.String() {
		want = a
	} else {
		want = b
	}
	if firstCreditor != want {
		t.Errorf("expected deterministic tie-break, got %v want %v", firstCreditor, want)
	}
}

func TestSimplifyDebtsLargeGroup(t *testing.T) {
	// 11 users with mixed balances summing to zero.
	balances := map[uuid.UUID]int64{}
	for i := 0; i < 5; i++ {
		balances[uuid.New()] = 1000000 + int64(i)*100000
	}
	for i := 0; i < 5; i++ {
		balances[uuid.New()] = -1000000 - int64(i)*100000
	}
	balances[uuid.New()] = -5000000 // extra debtor to balance the creditors

	var sum int64
	for _, v := range balances {
		sum += v
	}
	// Make sure the test data is actually balanced.
	balances[uuid.New()] = -sum

	transfers := SimplifyDebts(balances)

	if len(transfers) == 0 {
		t.Fatal("expected transfers for non-zero balances")
	}

	var transferTotalSen int64
	for _, tr := range transfers {
		if tr.AmountSen <= 0 {
			t.Errorf("transfer amount must be positive: %+v", tr)
		}
		if tr.FromUserID == tr.ToUserID {
			t.Errorf("transfer from == to: %+v", tr)
		}
		transferTotalSen += tr.AmountSen
	}

	var positiveSum int64
	for _, v := range balances {
		if v > 0 {
			positiveSum += v
		}
	}
	if transferTotalSen != positiveSum {
		t.Errorf("transfer total %d != positive balance total %d", transferTotalSen, positiveSum)
	}
}

func TestSimplifyDebtsVerySmallAmounts(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	// 1 sen amounts must work with integer math.
	balances := map[uuid.UUID]int64{a: 3, b: -2, c: -1}

	transfers := SimplifyDebts(balances)

	if transferTotal(transfers) != 3 {
		t.Errorf("expected transfer total 3, got %d", transferTotal(transfers))
	}
}

func TestSimplifyDebtsExactSettlement(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	// After a full settlement both balances are zero.
	expenses := []Expense{exp(a, 60000000, split(b, 60000000))}
	settlements := []Settlement{setl(b, a, 60000000)}

	balances := CalculateBalances(expenses, settlements)
	transfers := SimplifyDebts(balances)

	if len(transfers) != 0 {
		t.Errorf("expected no transfers after exact settlement, got %d", len(transfers))
	}
}

func TestSimplifyDebtsPartialSettlement(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	// A paid 600000, B owes 600000, B repaid 400000 → B still owes 200000.
	expenses := []Expense{exp(a, 60000000, split(b, 60000000))}
	settlements := []Settlement{setl(b, a, 40000000)}

	balances := CalculateBalances(expenses, settlements)
	transfers := SimplifyDebts(balances)

	if len(transfers) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(transfers))
	}
	if transfers[0].FromUserID != b || transfers[0].ToUserID != a || transfers[0].AmountSen != 20000000 {
		t.Errorf("unexpected transfer: %+v", transfers[0])
	}
}

func TestSimplifyDebtsCycles(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	// A +100, B -50, C -50.
	balances := map[uuid.UUID]int64{a: 1000000, b: -500000, c: -500000}

	transfers := SimplifyDebts(balances)

	if len(transfers) != 2 {
		t.Fatalf("expected 2 transfers, got %d", len(transfers))
	}
	for _, tr := range transfers {
		if tr.ToUserID != a {
			t.Errorf("expected all transfers to A, got %+v", tr)
		}
	}
	if transferTotal(transfers) != 1000000 {
		t.Errorf("expected total 1000000, got %d", transferTotal(transfers))
	}
}

func TestCalculateBalancesDeterministic(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	first := CalculateBalances([]Expense{exp(a, 10000000, split(b, 10000000))}, nil)
	second := CalculateBalances([]Expense{exp(a, 10000000, split(b, 10000000))}, nil)

	if first[a] != second[a] || first[b] != second[b] {
		t.Errorf("balance calculation not deterministic")
	}
}
