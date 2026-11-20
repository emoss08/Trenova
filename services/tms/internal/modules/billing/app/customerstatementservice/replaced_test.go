package customerstatementservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func txn(
	document string,
	amount int64,
	replaced pulid.ID,
) *customerledgerports.ARStatementTransaction {
	out := &customerledgerports.ARStatementTransaction{
		DocumentNumber:      document,
		AmountMinor:         amount,
		SupersededInvoiceID: replaced,
	}
	if amount >= 0 {
		out.ChargeMinor = amount
	} else {
		out.PaymentMinor = -amount
	}
	return out
}

func documents(st *customerledgerports.ARCustomerStatement) []string {
	out := make([]string, 0, len(st.Transactions))
	for _, t := range st.Transactions {
		out = append(out, t.DocumentNumber)
	}
	return out
}

func balances(st *customerledgerports.ARCustomerStatement) []int64 {
	out := make([]int64, 0, len(st.Transactions))
	for _, t := range st.Transactions {
		out = append(out, t.RunningBalanceMinor)
	}
	return out
}

func TestWithoutReplacedInvoices_LeavesOutAReplacedInvoiceAndTheCreditThatReversedIt(t *testing.T) {
	t.Parallel()

	replaced := pulid.ID("inv_replaced")
	st := &customerledgerports.ARCustomerStatement{
		OpeningBalanceMinor: 100_00,
		TotalChargesMinor:   1_250_00,
		TotalPaymentsMinor:  550_00,
		EndingBalanceMinor:  800_00,
		Transactions: []*customerledgerports.ARStatementTransaction{
			txn("INV-1", 500_00, replaced),
			txn("INV-2", 250_00, ""),
			txn("CM-1", -500_00, replaced),
			txn("INV-1-R1", 500_00, ""),
			txn("PMT-9", -50_00, ""),
		},
	}

	got := withoutReplacedInvoices(st)

	assert.Equal(t, []string{"INV-2", "INV-1-R1", "PMT-9"}, documents(got))
	assert.Equal(t, []int64{350_00, 850_00, 800_00}, balances(got))
	assert.Equal(t, int64(750_00), got.TotalChargesMinor)
	assert.Equal(t, int64(50_00), got.TotalPaymentsMinor)
	assert.Equal(t, int64(100_00), got.OpeningBalanceMinor)
	assert.Equal(t, int64(800_00), got.EndingBalanceMinor)
	assert.Len(t, st.Transactions, 5, "the statement it was given is left as it was")
}

func TestWithoutReplacedInvoices_KeepsAReplacedInvoiceWhoseRowsDoNotSettleInThePeriod(
	t *testing.T,
) {
	t.Parallel()

	paid := pulid.ID("inv_paid")
	earlier := pulid.ID("inv_earlier")
	st := &customerledgerports.ARCustomerStatement{
		OpeningBalanceMinor: 300_00,
		TotalChargesMinor:   400_00,
		TotalPaymentsMinor:  700_00,
		EndingBalanceMinor:  0,
		Transactions: []*customerledgerports.ARStatementTransaction{
			txn("INV-5", 400_00, paid),
			txn("PMT-1", -400_00, paid),
			txn("CM-5", -400_00, paid),
			txn("CM-3", -300_00, earlier),
			txn("INV-5-R1", 400_00, ""),
		},
	}

	got := withoutReplacedInvoices(st)

	assert.Same(t, st, got, "nothing settles inside the period, so every row stays")
}

func TestRender_LeavesOutReplacedInvoicesUnlessTheCustomerSeesThem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		policy    invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicy
		documents []string
		charges   string
		payments  string
	}{
		{
			name:      "current invoices only",
			policy:    invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicyShowCurrentOnlyExternally,
			documents: []string{"INV-1-R1"},
			charges:   "500.00 USD",
			payments:  "0.00 USD",
		},
		{
			name:      "current and replaced invoices",
			policy:    invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicyShowCurrentAndSupersededExternally,
			documents: []string{"INV-1", "CM-1", "INV-1-R1"},
			charges:   "1000.00 USD",
			payments:  "500.00 USD",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.adjustments.SupersededInvoiceVisibilityPolicy = tt.policy
			replaced := pulid.ID("inv_replaced")
			f.statements.statement = &customerledgerports.ARCustomerStatement{
				TotalChargesMinor:  1_000_00,
				TotalPaymentsMinor: 500_00,
				EndingBalanceMinor: 500_00,
				Transactions: []*customerledgerports.ARStatementTransaction{
					txn("INV-1", 500_00, replaced),
					txn("CM-1", -500_00, replaced),
					txn("INV-1-R1", 500_00, ""),
				},
			}
			cus := f.customer("HALSTEAD", "ap@halstead.test")

			_, err := f.svc.Render(t.Context(), &customerstatementports.StatementRequest{
				TenantInfo: f.tenant,
				CustomerID: cus.ID,
				AsOfDate:   time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC).Unix(),
			})
			require.NoError(t, err)

			require.Len(t, f.pdfs, 1)
			pdf := f.pdfs[0]
			got := make([]string, 0, len(pdf.Transactions))
			for _, row := range pdf.Transactions {
				got = append(got, row.Document)
			}
			assert.Equal(t, tt.documents, got)
			assert.Equal(t, tt.charges, pdf.TotalCharges)
			assert.Equal(t, tt.payments, pdf.TotalPayments)
			assert.Equal(t, "500.00 USD", pdf.EndingBalance)
		})
	}
}
