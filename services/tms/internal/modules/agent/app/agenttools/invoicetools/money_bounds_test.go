package invoicetools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func boundsOf(t *testing.T, tool agenttoolports.AgentTool) func(
	context.Context,
	agenttoolports.ToolExecuteParams,
) agenttoolports.MoneyBound {
	t.Helper()

	policy, ok := agenttoolports.PolicyOf(tool)
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
		override map[string]any
		within   bool
		held     string
	}{
		{
			name: "a remittance naming each invoice paid to its balance is within bounds",
			memo: "Remit INV-1001 INV-1002",
			invoices: []*invoice.Invoice{
				openInvoice(customerID, "INV-1001", 100_000),
				openInvoice(customerID, "INV-1002", 50_000),
			},
			applied: []string{"1000.00", "500.00"},
			amount:  "1500.00",
			receipt: true,
			within:  true,
		},
		{
			name:     "a payment with no bank receipt has no remittance to match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			held:     agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name: "an invoice the remittance does not name is not an exact match",
			memo: "Remit INV-1001",
			invoices: []*invoice.Invoice{
				openInvoice(customerID, "INV-1001", 100_000),
				openInvoice(customerID, "INV-1002", 50_000),
			},
			applied: []string{"1000.00", "500.00"},
			amount:  "1500.00",
			receipt: true,
			held:    agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:     "a part payment is not an exact match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"900.00"},
			amount:   "900.00",
			receipt:  true,
			held:     agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:     "an unapplied remainder is not an exact match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1100.00",
			receipt:  true,
			held:     agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:     "a short pay is not an exact match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			shortPay: true,
			receipt:  true,
			held:     agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:     "a payment dated otherwise than its receipt is not the receipt",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			receipt:  true,
			override: map[string]any{"paymentDate": "2026-09-30"},
			held:     agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:     "a reference other than the receipt's is not the receipt",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			receipt:  true,
			override: map[string]any{"referenceNumber": "INV-1001"},
			held:     agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:     "a memo other than the receipt's is not the receipt",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			receipt:  true,
			override: map[string]any{"memo": "Paid in full, thanks"},
			held:     agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:     "a method a bank receipt does not arrive by is not the receipt",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			receipt:  true,
			override: map[string]any{"paymentMethod": "Check"},
			held:     agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:     "the receipt's own reference and memo, written out, are the receipt",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(customerID, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			receipt:  true,
			override: map[string]any{"referenceNumber": "ach-88213", "memo": "Remit INV-1001"},
			within:   true,
		},
		{
			name:     "another customer's invoice is not an exact match",
			memo:     "Remit INV-1001",
			invoices: []*invoice.Invoice{openInvoice(other, "INV-1001", 100_000)},
			applied:  []string{"1000.00"},
			amount:   "1000.00",
			receipt:  true,
			held:     agenttooltrust.HeldByMoneyExactMatch,
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
				ReceiptDate:     time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Unix(),
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
				"customerId":   customerID.String(),
				"amount":       tc.amount,
				"paymentDate":  "2026-10-01",
				"applications": applications,
			}
			if tc.receipt {
				args["bankReceiptId"] = receipt.ID.String()
			}
			for key, value := range tc.override {
				args[key] = value
			}

			bound := boundsOf(t, tool)(t.Context(), agenttoolstest.MemoryParams(args))
			assert.Equal(t, tc.within, bound.Within)
			if tc.within {
				assert.Equal(t, amountMinor, bound.AmountMinor)
				assert.Equal(t, "USD", bound.Currency)
				assert.Equal(t, []agent.SourcedRef{{
					Source: agent.TaintSourceBankReceipt,
					Ref: agentrecord.RecordRef{
						EntityType: agent.TaintEntityBankReceipt,
						ID:         receipt.ID.String(),
					},
				}}, bound.Reread, "the receipt it records was read again from the organization's records")
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
			held:       agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:       "a second payment that matches exactly makes it ambiguous",
			paymentRef: "ACH-88213",
			candidates: []*customerpayment.Payment{
				exact,
				postedPayment(pulid.MustNew("cpay_"), "ACH-88213", 120_000),
			},
			held: agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:       "a second payment writing the reference another way makes it ambiguous",
			paymentRef: "ACH-88213",
			candidates: []*customerpayment.Payment{
				exact,
				postedPayment(pulid.MustNew("cpay_"), "ach.88213", 120_000),
			},
			held: agenttooltrust.HeldByMoneyExactMatch,
		},
		{
			name:       "a payment the candidates do not include is not exact",
			paymentRef: "ACH-88213",
			candidates: nil,
			held:       agenttooltrust.HeldByMoneyExactMatch,
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
			held:       agenttooltrust.HeldByMoneyUnproven,
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
				assert.Equal(t, []agent.SourcedRef{{
					Source: agent.TaintSourceBankReceipt,
					Ref: agentrecord.RecordRef{
						EntityType: agent.TaintEntityBankReceipt,
						ID:         receipt.ID.String(),
					},
				}}, bound.Reread)
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
	assert.Equal(t, agenttooltrust.HeldByMoneyUnproven, bound.Held)
}

func TestBankReceiptTools_VouchOnlyForTheReceiptsTheyReread(t *testing.T) {
	t.Parallel()

	for _, tool := range []agenttoolports.AgentTool{
		NewMatchBankReceiptTool(&fakeReceiptMatcher{}),
		NewPostCustomerPaymentTool(&remittancePoster{}, &fakeReceiptMatcher{}, &allowingPermissions{}),
	} {
		policy, ok := agenttoolports.PolicyOf(tool)
		require.True(t, ok)
		assert.Equal(t, []agent.TaintSource{agent.TaintSourceBankReceipt}, policy.Condition.Vouches,
			policy.Name)
	}

	policy, ok := agenttoolports.PolicyOf(NewCorrectChargeCodeTool(nil, nil))
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

type payerBooks struct {
	currency string
	err      error
	asked    pulid.ID
}

func (b *payerBooks) GetBillingProfile(
	_ context.Context,
	req customerports.GetCustomerBillingProfileRequest,
) (*customer.CustomerBillingProfile, error) {
	b.asked = req.CustomerID
	if b.err != nil {
		return nil, b.err
	}

	return &customer.CustomerBillingProfile{
		CustomerID:      req.CustomerID,
		BillingCurrency: b.currency,
	}, nil
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
	books *payerBooks,
	before, after []*shipment.AdditionalCharge,
) agenttoolports.MoneyBound {
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

	return boundsOf(t, NewCorrectChargeCodeTool(billing, books))(
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
	payer := pulid.MustNew("cus_")
	cases := []struct {
		name   string
		item   *billingqueue.BillingQueueItem
		books  *payerBooks
		within bool
		held   string
	}{
		{
			name: "an item in review with nothing invoiced is within bounds",
			item: &billingqueue.BillingQueueItem{
				Status:           billingqueue.StatusInReview,
				BillToCustomerID: payer,
			},
			within: true,
		},
		{
			name: "an item already invoiced is not",
			item: &billingqueue.BillingQueueItem{
				Status: billingqueue.StatusInReview, InvoiceID: invoiceID, BillToCustomerID: payer,
			},
			held: agenttooltrust.HeldByMoneyRecordState,
		},
		{
			name: "a correction of a posted invoice is not",
			item: &billingqueue.BillingQueueItem{
				Status: billingqueue.StatusInReview, SourceInvoiceID: &invoiceID, BillToCustomerID: payer,
			},
			held: agenttooltrust.HeldByMoneyRecordState,
		},
		{
			name: "a posted item is not",
			item: &billingqueue.BillingQueueItem{
				Status:           billingqueue.StatusPosted,
				BillToCustomerID: payer,
			},
			held: agenttooltrust.HeldByMoneyRecordState,
		},
		{
			name: "an item with no payer proves nothing",
			item: &billingqueue.BillingQueueItem{Status: billingqueue.StatusInReview},
			held: agenttooltrust.HeldByMoneyUnproven,
		},
		{
			name: "a payer whose billing profile cannot be read proves nothing",
			item: &billingqueue.BillingQueueItem{
				Status:           billingqueue.StatusInReview,
				BillToCustomerID: payer,
			},
			books: &payerBooks{err: errors.New("database down")},
			held:  agenttooltrust.HeldByMoneyUnproven,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			books := tc.books
			if books == nil {
				books = &payerBooks{currency: "USD"}
			}
			bound := chargeBound(t, tc.item, books,
				[]*shipment.AdditionalCharge{charge(lumper, "150")},
				[]*shipment.AdditionalCharge{charge(lumper, "100.50")},
			)
			assert.Equal(t, tc.within, bound.Within)
			if tc.within {
				assert.Equal(t, int64(4_950), bound.AmountMinor)
				assert.Equal(t, "USD", bound.Currency)
				assert.Equal(t, payer, books.asked, "the payer's own billing profile is read")
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
		Status:           billingqueue.StatusInReview,
		BillToCustomerID: pulid.MustNew("cus_"),
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

			bound := chargeBound(t, inReview, &payerBooks{currency: "cad"}, tc.before, tc.after)
			require.True(t, bound.Within)
			assert.Equal(t, tc.gross, bound.AmountMinor)
			assert.Equal(t, "CAD", bound.Currency, "the payer's billing currency is what moves")
		})
	}
}
