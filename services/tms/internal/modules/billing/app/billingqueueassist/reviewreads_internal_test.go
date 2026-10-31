package billingqueueassist

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueassist/reviewprompt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type contentByID struct {
	documentports.DocumentContentService

	texts map[pulid.ID]string
}

func (c contentByID) GetContent(
	_ context.Context,
	id pulid.ID,
	_ pagination.TenantInfo,
) (*documentcontent.Content, error) {
	return &documentcontent.Content{ContentText: c.texts[id]}, nil
}

func pastInvoice(number string, charges ...*billingqueueports.RecentCharge) *billingqueueports.RecentInvoice {
	return &billingqueueports.RecentInvoice{
		Number:      number,
		InvoiceDate: 1_791_000_000,
		Total:       decimal.RequireFromString("4000"),
		Charges:     charges,
	}
}

func TestChargeHistory_ListsOnlyThePayersInvoicesThatCarriedTheCharge(t *testing.T) {
	t.Parallel()

	payerID := pulid.MustNew("cus_")
	repo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	repo.EXPECT().RecentCustomerInvoices(mock.Anything, mock.Anything, payerID, historyInvoices).Return(
		[]*billingqueueports.RecentInvoice{
			pastInvoice("INV-2",
				&billingqueueports.RecentCharge{
					Type: invoice.InvoiceLineTypeFreight, Description: "Linehaul",
					Amount: decimal.RequireFromString("3300"),
				},
				&billingqueueports.RecentCharge{
					Type: invoice.InvoiceLineTypeAccessorial, Code: "FSC", Description: "Fuel",
					Amount: decimal.RequireFromString("410"),
				},
			),
			pastInvoice("INV-1", &billingqueueports.RecentCharge{
				Type: invoice.InvoiceLineTypeFreight, Description: "Linehaul",
				Amount: decimal.RequireFromString("3300"),
			}),
		}, nil,
	).Twice()
	log := &readLog{}
	history := &chargeHistory{repo: repo, payerID: payerID, payer: "Acme Foods", log: log}

	fuel := history.Call(t.Context(), map[string]any{"chargeCode": "fsc"})
	require.False(t, fuel.Failed, fuel.Content)
	assert.Equal(t, "1 of the last 2 posted invoices to Acme Foods carried fsc:\n"+
		"INV-2 · 2026-10-03 · total $4,000.00: FSC Fuel $410.00", fuel.Content)

	freight := history.Call(t.Context(), map[string]any{"chargeCode": "FREIGHT"})
	assert.Equal(t, "2 of the last 2 posted invoices to Acme Foods carried FREIGHT:\n"+
		"INV-2 · 2026-10-03 · total $4,000.00: Linehaul $3,300.00\n"+
		"INV-1 · 2026-10-03 · total $4,000.00: Linehaul $3,300.00", freight.Content)
	assert.Equal(t, []reviewprompt.ChargeLookup{
		{Code: "FSC", Carried: 1, Of: 2},
		{Code: "FREIGHT", Carried: 2, Of: 2},
	}, log.taken().Charges, "each charge looked up, as the invoice names it, with the invoices that carried it")
}

func TestChargeHistory_SaysWhenNoInvoiceCarriedTheCharge(t *testing.T) {
	t.Parallel()

	repo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	repo.EXPECT().RecentCustomerInvoices(mock.Anything, mock.Anything, mock.Anything, historyInvoices).
		Return([]*billingqueueports.RecentInvoice{pastInvoice("INV-1")}, nil)
	log := &readLog{}
	history := &chargeHistory{repo: repo, payer: "Acme Foods", log: log}

	result := history.Call(t.Context(), map[string]any{"chargeCode": "DET"})

	assert.False(t, result.Failed)
	assert.Equal(t, "None of the last 1 posted invoices to Acme Foods carried DET.", result.Content)
	assert.True(t, history.Call(t.Context(), map[string]any{}).Failed)
	assert.Equal(t, []reviewprompt.ChargeLookup{{Code: "DET", Carried: 0, Of: 1}}, log.taken().Charges,
		"a refused look-up read nothing")
}

func TestReadDocument_ReadsTheAskedPartOfALongDocument(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("doc_")
	text := strings.Repeat("a", reviewprompt.ExcerptRunes) + strings.Repeat("b", 10)
	log := &readLog{}
	reader := &documentReader{
		content: contentByID{texts: map[pulid.ID]string{id: text}},
		ids:     []string{id.String()},
		log:     log,
	}

	result := reader.Call(t.Context(), map[string]any{"documentId": id.String(), "part": float64(2)})

	require.False(t, result.Failed, result.Content)
	assert.Equal(t, "Part 2 of 2 of document "+id.String()+":\nbbbbbbbbbb", result.Content)
	assert.Equal(t, "Document "+id.String()+" has 2 parts.",
		reader.Call(t.Context(), map[string]any{"documentId": id.String(), "part": float64(3)}).Content)
	assert.True(t, reader.Call(t.Context(), map[string]any{"documentId": id.String(), "part": 1.5}).Failed)
	assert.Equal(t, map[pulid.ID]int{id: 2}, log.taken().Parts, "the part read, not the ones refused")
}

func TestReadDocument_RefusesADocumentWithNothingMoreToRead(t *testing.T) {
	t.Parallel()

	reader := &documentReader{content: contentByID{}, ids: []string{pulid.MustNew("doc_").String()}}

	result := reader.Call(t.Context(), map[string]any{"documentId": "doc_other", "part": float64(2)})

	assert.True(t, result.Failed)
	assert.Contains(t, result.Content, "Only these documents have more to read")
}

func TestReviewReads_OffersTheDocumentReaderOnlyForADocumentLongerThanItsExcerpt(t *testing.T) {
	t.Parallel()

	short := pulid.MustNew("doc_")
	long := pulid.MustNew("doc_")
	s := &Service{content: contentByID{}}
	e := evidenceWithDocuments(map[pulid.ID]int{short: 1, long: 3})

	reads := s.reviewReads(pagination.TenantInfo{}, e, &readLog{})

	require.Len(t, reads, 2)
	assert.Equal(t, chargeHistoryTool, reads[0].Spec().Name)
	spec := reads[1].Spec()
	assert.Equal(t, readDocumentTool, spec.Name)
	properties, _ := spec.Parameters["properties"].(map[string]any)
	documentID, _ := properties["documentId"].(map[string]any)
	assert.Equal(t, []string{long.String()}, documentID["enum"])

	assert.Len(t, s.reviewReads(
		pagination.TenantInfo{},
		evidenceWithDocuments(map[pulid.ID]int{short: 1}),
		&readLog{},
	), 1)
}

func evidenceWithDocuments(parts map[pulid.ID]int) *reviewprompt.Evidence {
	e := &reviewprompt.Evidence{
		Item:     &billingqueue.BillingQueueItem{BillToCustomerID: pulid.MustNew("cus_")},
		Excerpts: map[pulid.ID]reviewprompt.Excerpt{},
	}
	for id, count := range parts {
		e.Documents = append(e.Documents, &document.Document{ID: id})
		e.Excerpts[id] = reviewprompt.Excerpt{Text: "text", Parts: count}
	}

	return e
}
