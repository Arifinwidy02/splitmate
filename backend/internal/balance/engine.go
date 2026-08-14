package balance

import (
	"sort"

	"github.com/google/uuid"
)

// Split is a single member's share of an expense.
type Split struct {
	UserID    uuid.UUID
	AmountSen int64
}

// Expense is the minimal financial data the engine needs.
type Expense struct {
	PaidBy    uuid.UUID
	AmountSen int64
	Splits    []Split
}

// Settlement is an actual repayment from PayerID to ReceiverID.
type Settlement struct {
	PayerID    uuid.UUID
	ReceiverID uuid.UUID
	AmountSen  int64
}

// Transfer is a suggested repayment.
type Transfer struct {
	FromUserID uuid.UUID
	ToUserID   uuid.UUID
	AmountSen  int64
}

// CalculateBalances computes the net balance of every user involved:
//
//	balance = payments made - personal shares + settlements paid - settlements received
//
// A positive balance means the user should receive money, a negative
// balance means the user should pay money, zero means settled.
//
// A settlement cancels part of the payer's debt, so the payer's balance
// increases and the receiver's balance decreases by the settlement amount.
func CalculateBalances(expenses []Expense, settlements []Settlement) map[uuid.UUID]int64 {
	balances := map[uuid.UUID]int64{}

	for _, e := range expenses {
		balances[e.PaidBy] += e.AmountSen
		for _, s := range e.Splits {
			balances[s.UserID] -= s.AmountSen
		}
	}

	for _, st := range settlements {
		balances[st.PayerID] += st.AmountSen
		balances[st.ReceiverID] -= st.AmountSen
	}

	return balances
}

type account struct {
	userID uuid.UUID
	amount int64
}

// SimplifyDebts converts net balances into a minimal or near-minimal set of
// transfers using a deterministic greedy algorithm: the largest debtor always
// pays the largest creditor first. Ties are broken by user id so the result
// is stable regardless of map iteration order.
//
// The output satisfies:
//
//	sum(transfer amounts) == sum(positive balances)
//	every transfer amount > 0
//	from != to
func SimplifyDebts(balances map[uuid.UUID]int64) []Transfer {
	creditors := []account{}
	debtors := []account{}

	for userID, amount := range balances {
		switch {
		case amount > 0:
			creditors = append(creditors, account{userID: userID, amount: amount})
		case amount < 0:
			debtors = append(debtors, account{userID: userID, amount: -amount})
		}
	}

	sortAccounts(creditors, true)
	sortAccounts(debtors, true)

	transfers := []Transfer{}
	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		debtor := debtors[i]
		creditor := creditors[j]

		amount := debtor.amount
		if creditor.amount < amount {
			amount = creditor.amount
		}

		transfers = append(transfers, Transfer{
			FromUserID: debtor.userID,
			ToUserID:   creditor.userID,
			AmountSen:  amount,
		})

		debtors[i].amount -= amount
		creditors[j].amount -= amount
		if debtors[i].amount == 0 {
			i++
		}
		if creditors[j].amount == 0 {
			j++
		}
	}

	return transfers
}

func sortAccounts(accounts []account, largestFirst bool) {
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].amount != accounts[j].amount {
			if largestFirst {
				return accounts[i].amount > accounts[j].amount
			}
			return accounts[i].amount < accounts[j].amount
		}
		return accounts[i].userID.String() < accounts[j].userID.String()
	})
}
