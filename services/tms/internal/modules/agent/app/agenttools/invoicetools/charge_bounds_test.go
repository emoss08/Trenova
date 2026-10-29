package invoicetools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
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

	return agenttoolstest.BoundsOf(t, NewCorrectChargeCodeTool(billing, books))(
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
