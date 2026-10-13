package customerpaymentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationLedgerEntriesNameEachInvoice(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("inv_")
	second := pulid.MustNew("inv_")
	payment := &customerpayment.Payment{
		ID:             pulid.MustNew("cpay_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		CustomerID:     pulid.MustNew("cus_"),
	}
	applications := []*customerpayment.Application{
		{InvoiceID: first, AppliedAmountMinor: 10000, ShortPayAmountMinor: 500},
		nil,
		{InvoiceID: second, AppliedAmountMinor: 4000},
	}

	tests := []struct {
		name       string
		reversal   bool
		wantEvents []string
		wantSign   int64
	}{
		{
			name:     "applying credits each invoice",
			reversal: false,
			wantEvents: []string{
				tenant.JournalSourceEventCustomerPaymentPosted.String(),
				tenant.JournalSourceEventCustomerShortPayRecognized.String(),
				tenant.JournalSourceEventCustomerPaymentPosted.String(),
			},
			wantSign: -1,
		},
		{
			name:     "reversing debits each invoice back",
			reversal: true,
			wantEvents: []string{
				tenant.JournalSourceEventCustomerPaymentReversed.String(),
				tenant.JournalSourceEventCustomerPaymentReversed.String(),
				tenant.JournalSourceEventCustomerPaymentReversed.String(),
			},
			wantSign: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries := applicationLedgerEntries(&applicationLedgerParams{
				payment:         payment,
				applications:    applications,
				appliedEvent:    tenant.JournalSourceEventCustomerPaymentPosted.String(),
				transactionDate: 100,
				actorID:         pulid.MustNew("usr_"),
				reversal:        tt.reversal,
			})

			require.Len(t, entries, 3)
			wantInvoices := []pulid.ID{first, first, second}
			wantAmounts := []int64{10000, 500, 4000}
			for i, entry := range entries {
				assert.Equal(t, wantInvoices[i], entry.RelatedInvoiceID)
				assert.Equal(t, tt.wantSign*wantAmounts[i], entry.AmountMinor)
				assert.Equal(t, tt.wantEvents[i], entry.SourceEventType)
				assert.Equal(t, i+1, entry.LineNumber)
			}
		})
	}
}
