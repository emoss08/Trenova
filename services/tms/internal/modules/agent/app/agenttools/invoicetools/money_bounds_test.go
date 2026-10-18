package invoicetools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
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
		CustomerID:   req.CustomerID,
		AmountMinor:  req.AmountMinor,
		Status:       customerpayment.StatusPosted,
		CurrencyCode: "USD",
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
				assert.Equal(t, "USD", bound.Currency)
				assert.Equal(t, []agent.TaintSource{agent.TaintSourceBankReceipt}, bound.Reread,
					"the receipt was read again from the organization's records")
				return
			}
			assert.Equal(t, tc.held, bound.Held)
			assert.Empty(t, bound.Reread)
		})
	}
}

func postedPayment(id pulid.ID, reference string, amountMinor int64) *customerpayment.Payment {
	return &customerpayment.Payment{
		ID:              id,
		AmountMinor:     amountMinor,
		ReferenceNumber: reference,
		Status:          customerpayment.StatusPosted,
		CurrencyCode:    "USD",
	}
}

func TestMatchBankReceiptBounds(t *testing.T) {
	t.Parallel()

	paymentID := pulid.MustNew("cpay_")
	exact := postedPayment(paymentID, "ACH-88213", 120_000)
	cases := []struct {
		name       string
		paymentRef string
		candidates []*customerpayment.Payment
		within     bool
		held       string
	}{
		{
			name:       "the one posted payment with the receipt's amount and reference",
			paymentRef: "ach-88213",
			candidates: []*customerpayment.Payment{exact},
			within:     true,
		},
		{
			name:       "a payment with another reference is not exact",
			paymentRef: "ACH-99999",
			candidates: []*customerpayment.Payment{exact},
			held:       agent.HeldByMoneyExactMatch,
		},
		{
			name:       "a second payment that matches exactly makes it ambiguous",
			paymentRef: "ACH-88213",
			candidates: []*customerpayment.Payment{
				exact,
				postedPayment(pulid.MustNew("cpay_"), "ACH-88213", 120_000),
			},
			held: agent.HeldByMoneyExactMatch,
		},
		{
			name:       "a second payment writing the reference another way makes it ambiguous",
			paymentRef: "ACH-88213",
			candidates: []*customerpayment.Payment{
				exact,
				postedPayment(pulid.MustNew("cpay_"), "ach.88213", 120_000),
			},
			held: agent.HeldByMoneyExactMatch,
		},
		{
			name:       "a payment the candidates do not include is not exact",
			paymentRef: "ACH-88213",
			candidates: nil,
			held:       agent.HeldByMoneyExactMatch,
		},
		{
			name:       "a list holding exactly the limit is complete",
			paymentRef: "ACH-88213",
			candidates: candidatesOf(exact, matchCandidateLimit),
			within:     true,
		},
		{
			name:       "a list past the limit cannot prove there is only one",
			paymentRef: "ACH-88213",
			candidates: candidatesOf(exact, matchCandidateLimit+5),
			held:       agent.HeldByMoneyUnproven,
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
			matcher := &fakeReceiptMatcher{
				receipt:    receipt,
				payment:    postedPayment(paymentID, tc.paymentRef, 120_000),
				candidates: tc.candidates,
			}
			tool := NewMatchBankReceiptTool(matcher)

			bound := boundsOf(t, tool)(t.Context(), agenttoolstest.MemoryParams(map[string]any{
				"bankReceiptId":     receipt.ID.String(),
				"customerPaymentId": paymentID.String(),
			}))
			assert.Equal(t, tc.within, bound.Within)
			if tc.paymentRef == "ACH-88213" || tc.paymentRef == "ach-88213" {
				assert.Equal(t, matchCandidateLimit+1, matcher.askedLimit,
					"the bound asks for one more than it can judge, to see a full list")
			}
			if tc.within {
				assert.Equal(t, int64(120_000), bound.AmountMinor)
				assert.Equal(t, "USD", bound.Currency)
				assert.Equal(t, []agent.TaintSource{agent.TaintSourceBankReceipt}, bound.Reread)
				return
			}
			assert.Equal(t, tc.held, bound.Held)
		})
	}
}

func TestMatchBankReceiptBounds_CandidatesThatCannotBeReadProveNothing(t *testing.T) {
	t.Parallel()

	paymentID := pulid.MustNew("cpay_")
	receipt := &bankreceipt.BankReceipt{
		ID:              pulid.MustNew("brcpt_"),
		AmountMinor:     120_000,
		ReferenceNumber: "ACH-88213",
		Status:          bankreceipt.StatusException,
	}
	tool := NewMatchBankReceiptTool(&fakeReceiptMatcher{
		receipt:       receipt,
		payment:       postedPayment(paymentID, "ACH-88213", 120_000),
		candidatesErr: errors.New("database down"),
	})

	bound := boundsOf(t, tool)(t.Context(), agenttoolstest.MemoryParams(map[string]any{
		"bankReceiptId":     receipt.ID.String(),
		"customerPaymentId": paymentID.String(),
	}))
	assert.False(t, bound.Within)
	assert.Equal(t, agent.HeldByMoneyUnproven, bound.Held)
}

func TestBankReceiptTools_VouchOnlyForTheReceiptsTheyReread(t *testing.T) {
	t.Parallel()

	for _, tool := range []agentports.AgentTool{
		NewMatchBankReceiptTool(&fakeReceiptMatcher{}),
		NewPostCustomerPaymentTool(&remittancePoster{}, &fakeReceiptMatcher{}, &allowingPermissions{}),
	} {
		policy, ok := agentports.PolicyOf(tool)
		require.True(t, ok)
		assert.Equal(t, []agent.TaintSource{agent.TaintSourceBankReceipt}, policy.Condition.Vouches,
			policy.Name)
	}

	policy, ok := agentports.PolicyOf(NewCorrectChargeCodeTool(nil))
	require.True(t, ok)
	assert.Empty(t, policy.Condition.Vouches, "a charge correction re-reads no outside record")
}

func candidatesOf(exact *customerpayment.Payment, count int) []*customerpayment.Payment {
	out := []*customerpayment.Payment{exact}
	for len(out) < count {
		out = append(out, postedPayment(pulid.MustNew("cpay_"), "ACH-88213-B", 120_050))
	}

	return out
}

func billedTo(currency string) *customer.Customer {
	return &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		BillingProfile: &customer.CustomerBillingProfile{BillingCurrency: currency},
	}
}

func charge(id pulid.ID, amount string) *shipment.AdditionalCharge {
	return &shipment.AdditionalCharge{
		ID:     id,
		Method: accessorialcharge.MethodFlat,
		Amount: decimal.RequireFromString(amount),
		Unit:   1,
	}
}

func chargeBound(
	t *testing.T,
	item *billingqueue.BillingQueueItem,
	before, after []*shipment.AdditionalCharge,
) agentports.MoneyBound {
	t.Helper()

	freight := decimal.NewNullDecimal(decimal.NewFromInt(2000))
	billing := billingqueueportsmock.NewMockBillingQueueService(t)
	billing.EXPECT().
		PreviewUpdateCharges(mock.Anything, mock.Anything, mock.Anything).
		Return(&billingqueueports.ChargeUpdatePreview{
			Item:   item,
			Before: &shipment.Shipment{FreightChargeAmount: freight, AdditionalCharges: before},
			After:  &shipment.Shipment{FreightChargeAmount: freight, AdditionalCharges: after},
		}, nil).
		Once()

	return boundsOf(t, NewCorrectChargeCodeTool(billing))(
		t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			"billingQueueItemId": pulid.MustNew("bqi_").String(),
			"additionalCharges":  []any{},
		}),
	)
}

func TestCorrectChargeCodeBounds(t *testing.T) {
	t.Parallel()

	invoiceID := pulid.MustNew("inv_")
	lumper := pulid.MustNew("ac_")
	cases := []struct {
		name   string
		item   *billingqueue.BillingQueueItem
		within bool
		held   string
	}{
		{
			name:   "an item in review with nothing invoiced is within bounds",
			item:   &billingqueue.BillingQueueItem{Status: billingqueue.StatusInReview, BillToCustomer: billedTo("USD")},
			within: true,
		},
		{
			name: "an item already invoiced is not",
			item: &billingqueue.BillingQueueItem{
				Status: billingqueue.StatusInReview, InvoiceID: invoiceID, BillToCustomer: billedTo("USD"),
			},
			held: agent.HeldByMoneyRecordState,
		},
		{
			name: "a correction of a posted invoice is not",
			item: &billingqueue.BillingQueueItem{
				Status: billingqueue.StatusInReview, SourceInvoiceID: &invoiceID, BillToCustomer: billedTo("USD"),
			},
			held: agent.HeldByMoneyRecordState,
		},
		{
			name: "a posted item is not",
			item: &billingqueue.BillingQueueItem{Status: billingqueue.StatusPosted, BillToCustomer: billedTo("USD")},
			held: agent.HeldByMoneyRecordState,
		},
		{
			name: "an item whose payer was not read proves nothing",
			item: &billingqueue.BillingQueueItem{Status: billingqueue.StatusInReview},
			held: agent.HeldByMoneyUnproven,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bound := chargeBound(t, tc.item,
				[]*shipment.AdditionalCharge{charge(lumper, "150")},
				[]*shipment.AdditionalCharge{charge(lumper, "100.50")},
			)
			assert.Equal(t, tc.within, bound.Within)
			if tc.within {
				assert.Equal(t, int64(4_950), bound.AmountMinor)
				assert.Equal(t, "USD", bound.Currency)
				assert.Empty(t, bound.Reread)
				return
			}
			assert.Equal(t, tc.held, bound.Held)
		})
	}
}

func TestCorrectChargeCodeBounds_CountEveryChargeItChangesInFull(t *testing.T) {
	t.Parallel()

	lumper, detention := pulid.MustNew("ac_"), pulid.MustNew("ac_")
	inReview := &billingqueue.BillingQueueItem{
		Status:         billingqueue.StatusInReview,
		BillToCustomer: billedTo("CAD"),
	}
	cases := []struct {
		name          string
		before, after []*shipment.AdditionalCharge
		gross         int64
	}{
		{
			name:   "swapping one charge for another of the same amount moves both",
			before: []*shipment.AdditionalCharge{charge(lumper, "500")},
			after:  []*shipment.AdditionalCharge{charge(pulid.Nil, "500")},
			gross:  100_000,
		},
		{
			name:   "moving an amount between two charges moves it twice",
			before: []*shipment.AdditionalCharge{charge(lumper, "500"), charge(detention, "100")},
			after:  []*shipment.AdditionalCharge{charge(lumper, "200"), charge(detention, "400")},
			gross:  60_000,
		},
		{
			name:   "removing a charge counts it in full",
			before: []*shipment.AdditionalCharge{charge(lumper, "500"), charge(detention, "100")},
			after:  []*shipment.AdditionalCharge{charge(lumper, "500")},
			gross:  10_000,
		},
		{
			name:   "adding a charge counts it in full",
			before: nil,
			after:  []*shipment.AdditionalCharge{charge(pulid.Nil, "75.25")},
			gross:  7_525,
		},
		{
			name:   "an edit that changes nothing moves nothing",
			before: []*shipment.AdditionalCharge{charge(lumper, "500")},
			after:  []*shipment.AdditionalCharge{charge(lumper, "500")},
			gross:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bound := chargeBound(t, inReview, tc.before, tc.after)
			require.True(t, bound.Within)
			assert.Equal(t, tc.gross, bound.AmountMinor)
			assert.Equal(t, "CAD", bound.Currency, "the payer's billing currency is what moves")
		})
	}
}
