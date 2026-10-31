package billingqueueassist

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueassist/reviewprompt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fuelInvoiceEvidence is an item billing freight and a fuel surcharge to one
// payer, with an uncoded charge the history cannot look up.
func fuelInvoiceEvidence(payerID pulid.ID) *reviewprompt.Evidence {
	fuelID, uncodedID := pulid.MustNew("ac_"), pulid.MustNew("ac_")

	return &reviewprompt.Evidence{
		Item: &billingqueue.BillingQueueItem{
			BillToCustomerID: payerID,
			BillToCustomer:   &customer.Customer{Name: "Acme Foods"},
			Shipment: &shipment.Shipment{AdditionalCharges: []*shipment.AdditionalCharge{
				{ID: fuelID, AccessorialCharge: &accessorialcharge.AccessorialCharge{Code: "FSC"}},
				{ID: uncodedID},
			}},
			Review: &billingqueue.Review{Charges: &billingqueue.ChargeReview{Lines: []*billingqueue.ChargeLine{
				{Key: "freight-0", Label: "Freight"},
				{Key: fuelID.String(), AdditionalChargeID: fuelID, Label: "Fuel Surcharge"},
				{Key: uncodedID.String(), AdditionalChargeID: uncodedID, Label: "Misc"},
			}}},
		},
		Excerpts: map[pulid.ID]reviewprompt.Excerpt{},
	}
}

func TestReadAhead_GivesABatchedReviewEachChargeAcrossThePayersInvoicesReadOncePerPayer(t *testing.T) {
	t.Parallel()

	payerID := pulid.MustNew("cus_")
	repo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	repo.EXPECT().RecentCustomerInvoices(mock.Anything, mock.Anything, payerID, historyInvoices).Return(
		[]*billingqueueports.RecentInvoice{pastInvoice("INV-1",
			&billingqueueports.RecentCharge{
				Type: invoice.InvoiceLineTypeFreight, Description: "Linehaul",
				Amount: decimal.RequireFromString("3300"),
			},
			&billingqueueports.RecentCharge{
				Type: invoice.InvoiceLineTypeAccessorial, Code: "FSC", Description: "Fuel",
				Amount: decimal.RequireFromString("410"),
			},
		)}, nil,
	).Once()
	ahead := (&Service{l: zap.NewNop(), repo: repo}).newReadAhead(pagination.TenantInfo{})

	sections := ahead.sections(t.Context(), fuelInvoiceEvidence(payerID))
	again := ahead.sections(t.Context(), fuelInvoiceEvidence(payerID))

	require.Len(t, sections, 2, "freight and the fuel surcharge; the uncoded charge has no code to look up")
	assert.Equal(t, "What charge_history found for FREIGHT", sections[0].Title)
	assert.Equal(t, "What charge_history found for FSC", sections[1].Title)
	assert.Equal(t, "1 of the last 1 posted invoices to Acme Foods carried FSC:\n"+
		"INV-1 · 2026-10-03 · total $4,000.00: FSC Fuel $410.00", sections[1].Content)
	assert.False(t, sections[1].Trusted, "past invoices carry descriptions people typed")
	assert.Equal(t, sections, again)
}

func TestReadAhead_GivesABatchedReviewTheNextPartsOfALongDocument(t *testing.T) {
	t.Parallel()

	long := pulid.MustNew("doc_")
	text := strings.Repeat("a", reviewprompt.ExcerptRunes) + strings.Repeat("b", reviewprompt.ExcerptRunes) +
		strings.Repeat("c", reviewprompt.ExcerptRunes) + "d"
	repo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	s := &Service{l: zap.NewNop(), repo: repo, content: contentByID{texts: map[pulid.ID]string{long: text}}}
	e := evidenceWithDocuments(map[pulid.ID]int{long: 4, pulid.MustNew("doc_"): 1})

	sections := s.newReadAhead(pagination.TenantInfo{}).sections(t.Context(), e)

	require.Len(t, sections, readAheadParts-1, "the parts after the excerpt, up to readAheadParts")
	assert.Equal(t, "What read_document found in document "+long.String(), sections[0].Title)
	assert.True(t, strings.HasPrefix(sections[0].Content, "Part 2 of 4 of document "+long.String()+":\nbbb"))
	assert.True(t, strings.HasPrefix(sections[1].Content, "Part 3 of 4 of document "+long.String()+":\nccc"))
	assert.False(t, sections[0].Trusted)
}

func TestReadAheadTaken_RecordsWhatABatchedReviewWasGivenAsTheReadsItWouldHaveMade(t *testing.T) {
	t.Parallel()

	payerID := pulid.MustNew("cus_")
	long, longer := pulid.MustNew("doc_"), pulid.MustNew("doc_")
	e := fuelInvoiceEvidence(payerID)
	e.Documents = evidenceWithDocuments(map[pulid.ID]int{long: 2, longer: 6, pulid.MustNew("doc_"): 1}).Documents
	e.Excerpts = map[pulid.ID]reviewprompt.Excerpt{long: {Parts: 2}, longer: {Parts: 6}}
	repo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	repo.EXPECT().RecentCustomerInvoices(mock.Anything, mock.Anything, payerID, historyInvoices).Return(
		[]*billingqueueports.RecentInvoice{
			pastInvoice("INV-2", &billingqueueports.RecentCharge{Type: invoice.InvoiceLineTypeFreight}),
			pastInvoice("INV-1",
				&billingqueueports.RecentCharge{Type: invoice.InvoiceLineTypeFreight},
				&billingqueueports.RecentCharge{Type: invoice.InvoiceLineTypeAccessorial, Code: "FSC"},
			),
		}, nil,
	).Once()
	ahead := (&Service{l: zap.NewNop(), repo: repo, content: contentByID{}}).newReadAhead(pagination.TenantInfo{})

	taken := ahead.taken(t.Context(), e)

	assert.Equal(t, []reviewprompt.ChargeLookup{
		{Code: reviewprompt.FreightCode, Carried: 2, Of: 2},
		{Code: "FSC", Carried: 1, Of: 2},
	}, taken.Charges)
	assert.Equal(t, map[pulid.ID]int{long: 2, longer: readAheadParts}, taken.Parts)
	assert.Equal(t, taken, ahead.taken(t.Context(), e), "the payer's invoices are read once for the page")
}

func TestReadAheadTaken_LooksUpNoChargeForAPayerWithNoInvoices(t *testing.T) {
	t.Parallel()

	repo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	repo.EXPECT().RecentCustomerInvoices(mock.Anything, mock.Anything, mock.Anything, historyInvoices).
		Return(nil, nil)
	ahead := (&Service{l: zap.NewNop(), repo: repo}).newReadAhead(pagination.TenantInfo{})

	assert.Empty(t, ahead.taken(t.Context(), fuelInvoiceEvidence(pulid.MustNew("cus_"))).Charges)
}
