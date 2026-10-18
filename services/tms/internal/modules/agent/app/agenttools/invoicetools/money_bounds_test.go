package invoicetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func boundsOf(t *testing.T, tool agentports.AgentTool) func(
	context.Context,
	agentports.ToolExecuteParams,
) agentports.MoneyBound {
	t.Helper()

	policy, ok := agentports.PolicyOf(tool)
	require.True(t, ok)
	require.True(t, policy.Bounded(), "%s proves its bounds", policy.Name)

	return policy.Condition.Bounds
}

type remittancePoster struct {
	invoices []*invoice.Invoice
	shortPay bool
}

func (p *remittancePoster) PreviewPostAndApply(
	_ context.Context,
	req *customerpaymentports.PostCustomerPaymentRequest,
	_ *permissionports.RequestActor,
) (*customerpaymentports.CustomerPaymentPostPreview, error) {
	payment := &customerpayment.Payment{
		CustomerID:  req.CustomerID,
		AmountMinor: req.AmountMinor,
		Status:      customerpayment.StatusPosted,
	}
	for _, application := range req.Applications {
		line := &customerpayment.Application{
			InvoiceID:          application.InvoiceID,
			AppliedAmountMinor: application.AppliedAmountMinor,
		}
		if p.shortPay {
			line.ShortPayAmountMinor = 100
		}
		payment.Applications = append(payment.Applications, line)
	}
	payment.SyncAmounts()

	return &customerpaymentports.CustomerPaymentPostPreview{
		Payment:        payment,
		InvoicesBefore: p.invoices,
	}, nil
}

func (p *remittancePoster) PostAndApply(
	context.Context,
	*customerpaymentports.PostCustomerPaymentRequest,
	*permissionports.RequestActor,
) (*customerpayment.Payment, error) {
	return nil, nil
}

func openInvoice(customerID pulid.ID, number string, totalMinor int64) *invoice.Invoice {
	return &invoice.Invoice{
		ID:               pulid.MustNew("inv_"),
		CustomerID:       customerID,
		Number:           number,
		Status:           invoice.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
		TotalAmountMinor: totalMinor,
	}
}

func TestPostCustomerPaymentBounds(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	other := pulid.MustNew("cus_")
	cases := []struct {
		name     string
		memo     string
		invoices []*invoice.Invoice
		applied  []string
		amount   string
		shortPay bool
		receipt  bool
		within   bool
		held     string
	}{
		{
			name:     "a remittance naming each invoice paid to its balance is within bounds",
			memo:     "Remit INV-1001 INV-1002",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000), openInvoice(customerID, "INV-1002", 50_000)},
			applied:  []string{"1000.00", "500.00"},
			amount:   "1500.00",
			receipt:  true,
			within:   true,
		},
		{
			name:     "a payment with no bank receipt has no remittance to match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			held:     agent.HeldByMoneyExactMatch,
		},
		{
			name:     "an invoice the remittance does not name is not an exact match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000), openInvoice(customerID, "INV-1002", 50_000)},
			applied:  []string{"1000.00", "500.00"},
			amount:   "1500.00",
			receipt:  true,
			held:     agent.HeldByMoneyExactMatch,
		},
		{
			name:     "a part payment is not an exact match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"900.00"},
			amount:   "900.00",
			receipt:  true,
			held:     agent.HeldByMoneyExactMatch,
		},
		{
			name:     "an unapplied remainder is not an exact match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1100.00",
			receipt:  true,
			held:     agent.HeldByMoneyExactMatch,
		},
		{
			name:     "a short pay is not an exact match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			shortPay: true,
			receipt:  true,
			held:     agent.HeldByMoneyExactMatch,
		},
		{
			name:     "another customer's invoice is not an exact match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(other, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			receipt:  true,
			held:     agent.HeldByMoneyExactMatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			amountMinor := decimal.RequireFromString(tc.amount).Shift(2).IntPart()
			receipt := &bankreceipt.BankReceipt{
				ID:              pulid.MustNew("brcpt_"),
				AmountMinor:     amountMinor,
				ReferenceNumber: "ACH-88213",
				Memo:            tc.memo,
				Status:          bankreceipt.StatusException,
			}
			tool := NewPostCustomerPaymentTool(
				&remittancePoster{invoices: tc.invoices, shortPay: tc.shortPay},
				&fakeReceiptMatcher{receipt: receipt},
				&allowingPermissions{allowed: true},
			)
			applications := make([]any, 0, len(tc.applied))
			for idx, applied := range tc.applied {
				applications = append(applications, map[string]any{
					"invoiceId": tc.invoices[idx].ID.String(),
					"amount":    applied,
				})
			}
			args := map[string]any{
				"customerId":      customerID.String(),
				"amount":          tc.amount,
				"paymentDate":     "2026-10-01",
				"referenceNumber": "INV-1001 INV-1002",
				"memo":            "INV-1001 INV-1002",
				"applications":    applications,
			}
			if tc.receipt {
				args["bankReceiptId"] = receipt.ID.String()
			}

			bound := boundsOf(t, tool)(t.Context(), agenttoolstest.MemoryParams(args))
			assert.Equal(t, tc.within, bound.Within)
			if tc.within {
				assert.Equal(t, amountMinor, bound.AmountMinor)
				return
			}
			assert.Equal(t, tc.held, bound.Held)
		})
	}
}

func TestMatchBankReceiptBounds(t *testing.T) {
	t.Parallel()

	paymentID := pulid.MustNew("cpay_")
	exact := &bankreceiptports.BankReceiptMatchSuggestion{
		CustomerPaymentID: paymentID,
		ReferenceNumber:   "ACH-88213",
		AmountMinor:       120_000,
	}
	cases := []struct {
		name        string
		paymentRef  string
		suggestions []*bankreceiptports.BankReceiptMatchSuggestion
		within      bool
		held        string
	}{
		{
			name:        "the one posted payment with the receipt's amount and reference",
			paymentRef:  "ach-88213",
			suggestions: []*bankreceiptports.BankReceiptMatchSuggestion{exact},
			within:      true,
		},
		{
			name:        "a payment with another reference is not exact",
			paymentRef:  "ACH-99999",
			suggestions: []*bankreceiptports.BankReceiptMatchSuggestion{exact},
			held:        agent.HeldByMoneyExactMatch,
		},
		{
			name:       "a second payment that matches exactly makes it ambiguous",
			paymentRef: "ACH-88213",
			suggestions: []*bankreceiptports.BankReceiptMatchSuggestion{exact, {
				CustomerPaymentID: pulid.MustNew("cpay_"),
				ReferenceNumber:   "ACH-88213",
				AmountMinor:       120_000,
			}},
			held: agent.HeldByMoneyExactMatch,
		},
		{
			name:        "a payment the candidates do not include is unproven",
			paymentRef:  "ACH-88213",
			suggestions: nil,
			held:        agent.HeldByMoneyExactMatch,
		},
		{
			name:        "a full candidate list cannot prove there is only one",
			paymentRef:  "ACH-88213",
			suggestions: fullCandidates(exact),
			held:        agent.HeldByMoneyUnproven,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			receipt := &bankreceipt.BankReceipt{
				ID:              pulid.MustNew("brcpt_"),
				AmountMinor:     120_000,
				ReferenceNumber: "ACH-88213",
				Status:          bankreceipt.StatusException,
			}
			tool := NewMatchBankReceiptTool(&fakeReceiptMatcher{
				receipt: receipt,
				payment: &customerpayment.Payment{
					ID:              paymentID,
					AmountMinor:     120_000,
					ReferenceNumber: tc.paymentRef,
					Status:          customerpayment.StatusPosted,
				},
				suggestions: tc.suggestions,
			})

			bound := boundsOf(t, tool)(t.Context(), agenttoolstest.MemoryParams(map[string]any{
				"bankReceiptId":     receipt.ID.String(),
				"customerPaymentId": paymentID.String(),
			}))
			assert.Equal(t, tc.within, bound.Within)
			if tc.within {
				assert.Equal(t, int64(120_000), bound.AmountMinor)
				return
			}
			assert.Equal(t, tc.held, bound.Held)
		})
	}
}

func fullCandidates(
	exact *bankreceiptports.BankReceiptMatchSuggestion,
) []*bankreceiptports.BankReceiptMatchSuggestion {
	out := []*bankreceiptports.BankReceiptMatchSuggestion{exact}
	for len(out) < matchCandidateLimit {
		out = append(out, &bankreceiptports.BankReceiptMatchSuggestion{
			CustomerPaymentID: pulid.MustNew("cpay_"),
			ReferenceNumber:   "ACH-88213-B",
			AmountMinor:       120_050,
		})
	}

	return out
}

func TestCorrectChargeCodeBounds(t *testing.T) {
	t.Parallel()

	invoiceID := pulid.MustNew("inv_")
	cases := []struct {
		name   string
		item   *billingqueue.BillingQueueItem
		within bool
		held   string
	}{
		{
			name:   "an item in review with nothing invoiced is within bounds",
			item:   &billingqueue.BillingQueueItem{Status: billingqueue.StatusInReview},
			within: true,
		},
		{
			name: "an item already invoiced is not",
			item: &billingqueue.BillingQueueItem{Status: billingqueue.StatusInReview, InvoiceID: invoiceID},
			held: agent.HeldByMoneyRecordState,
		},
		{
			name: "a correction of a posted invoice is not",
			item: &billingqueue.BillingQueueItem{Status: billingqueue.StatusInReview, SourceInvoiceID: &invoiceID},
			held: agent.HeldByMoneyRecordState,
		},
		{
			name: "a posted item is not",
			item: &billingqueue.BillingQueueItem{Status: billingqueue.StatusPosted},
			held: agent.HeldByMoneyRecordState,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			before := chargedShipment()
			after := *before
			after.TotalChargeAmount = decimal.NewNullDecimal(decimal.RequireFromString("2100.50"))
			billing := billingqueueportsmock.NewMockBillingQueueService(t)
			billing.EXPECT().
				PreviewUpdateCharges(mock.Anything, mock.Anything, mock.Anything).
				Return(&billingqueueports.ChargeUpdatePreview{
					Item:   tc.item,
					Before: before,
					After:  &after,
				}, nil).
				Once()

			tool := NewCorrectChargeCodeTool(billing)
			bound := boundsOf(t, tool)(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"billingQueueItemId": pulid.MustNew("bqi_").String(),
				"additionalCharges":  []any{},
			}))
			assert.Equal(t, tc.within, bound.Within)
			if tc.within {
				assert.Equal(t, int64(-4_950), bound.AmountMinor)
				return
			}
			assert.Equal(t, tc.held, bound.Held)
		})
	}
}
