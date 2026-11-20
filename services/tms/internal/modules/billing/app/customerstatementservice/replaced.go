package customerstatementservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/shared/pulid"
)

func withoutReplacedInvoices(
	st *customerledgerports.ARCustomerStatement,
) *customerledgerports.ARCustomerStatement {
	net := make(map[pulid.ID]int64)
	for _, txn := range st.Transactions {
		if txn != nil && txn.SupersededInvoiceID.IsNotNil() {
			net[txn.SupersededInvoiceID] += txn.AmountMinor
		}
	}
	settled := func(txn *customerledgerports.ARStatementTransaction) bool {
		if txn.SupersededInvoiceID.IsNil() {
			return false
		}
		return net[txn.SupersededInvoiceID] == 0
	}
	if !slices.ContainsFunc(
		st.Transactions,
		func(txn *customerledgerports.ARStatementTransaction) bool {
			return txn != nil && settled(txn)
		},
	) {
		return st
	}

	out := *st
	out.Transactions = make([]*customerledgerports.ARStatementTransaction, 0, len(st.Transactions))
	out.TotalChargesMinor = 0
	out.TotalPaymentsMinor = 0
	running := st.OpeningBalanceMinor
	for _, txn := range st.Transactions {
		if txn == nil || settled(txn) {
			continue
		}
		kept := *txn
		running += kept.AmountMinor
		kept.RunningBalanceMinor = running
		if kept.AmountMinor >= 0 {
			out.TotalChargesMinor += kept.AmountMinor
		} else {
			out.TotalPaymentsMinor += -kept.AmountMinor
		}
		out.Transactions = append(out.Transactions, &kept)
	}
	return &out
}
