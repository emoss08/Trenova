package reviewprompt_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueassist/reviewprompt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestReads_ListWhatTheReviewWasShownWithWhatItReadFurther(t *testing.T) {
	t.Parallel()

	e := evidence()
	docID := e.Documents[0].ID

	reads := reviewprompt.Reads(e, reviewprompt.Taken{
		Parts:   map[pulid.ID]int{docID: 2},
		Charges: []reviewprompt.ChargeLookup{{Code: "FSC", Carried: 3, Of: 40}},
	})

	assert.Equal(t, []billingqueue.AgentRead{
		{Kind: billingqueue.AgentReadInvoice, Count: 2},
		{Kind: billingqueue.AgentReadStops, Count: 1},
		{Kind: billingqueue.AgentReadTerms},
		{Kind: billingqueue.AgentReadBillingNotes},
		{Kind: billingqueue.AgentReadComments, Count: 1},
		{Kind: billingqueue.AgentReadPastInvoices, Count: 1},
		{Kind: billingqueue.AgentReadChargeHistory, Code: "FSC", Count: 3, Of: 40},
		{
			Kind:       billingqueue.AgentReadDocument,
			DocumentID: docID,
			Name:       "Proof of Delivery",
			FileName:   "pod.pdf",
			Part:       2,
			Parts:      3,
		},
	}, reads)
}

func TestReads_CountADocumentReadOnlyAsFarAsItsExcerptAndOneWithNoTextAsUnread(t *testing.T) {
	t.Parallel()

	e := evidence()
	e.Item.Review.Charges.Lines[1].Removed = true
	e.Item.BillToCustomer.BillingProfile.BillingNotes = " "
	e.Comments = nil
	e.Recent = nil
	unread := &document.Document{ID: pulid.MustNew("doc_"), OriginalName: "scan.jpg"}
	e.Documents = append(e.Documents, unread)

	reads := reviewprompt.Reads(e, reviewprompt.Taken{})

	assert.Equal(t, []billingqueue.AgentRead{
		{Kind: billingqueue.AgentReadInvoice, Count: 1},
		{Kind: billingqueue.AgentReadStops, Count: 1},
		{Kind: billingqueue.AgentReadTerms},
		{
			Kind:       billingqueue.AgentReadDocument,
			DocumentID: e.Documents[0].ID,
			Name:       "Proof of Delivery",
			FileName:   "pod.pdf",
			Part:       1,
			Parts:      3,
		},
		{Kind: billingqueue.AgentReadDocument, DocumentID: unread.ID, Name: "scan.jpg", FileName: "scan.jpg"},
	}, reads)
}
