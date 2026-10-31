package reviewprompt

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/shared/pulid"
)

// Taken is what a review read beyond what it was shown: the furthest part it
// read of each long document, and each charge it looked up across the payer's
// invoices, in the order it read them.
type Taken struct {
	Parts   map[pulid.ID]int
	Charges []ChargeLookup
}

// ChargeLookup is one charge looked up across the payer's invoices: how many
// carried it, of how many.
type ChargeLookup struct {
	Code    string
	Carried int
	Of      int
}

// Reads are what the review of e read, in the order its sections show them,
// with what it took further folded in, as the biller is shown them.
func Reads(e *Evidence, taken Taken) []billingqueue.AgentRead {
	reads := make([]billingqueue.AgentRead, 0, 6+len(taken.Charges)+len(e.Documents))
	if charges := chargeLines(e); charges > 0 {
		reads = append(reads, billingqueue.AgentRead{Kind: billingqueue.AgentReadInvoice, Count: charges})
	}
	if stops := stopCount(e); stops > 0 {
		reads = append(reads, billingqueue.AgentRead{Kind: billingqueue.AgentReadStops, Count: stops})
	}
	if termsText(e) != "" {
		reads = append(reads, billingqueue.AgentRead{Kind: billingqueue.AgentReadTerms})
	}
	if profile := billingProfile(e); profile != nil && strings.TrimSpace(profile.BillingNotes) != "" {
		reads = append(reads, billingqueue.AgentRead{Kind: billingqueue.AgentReadBillingNotes})
	}
	if len(e.Comments) > 0 {
		reads = append(reads, billingqueue.AgentRead{Kind: billingqueue.AgentReadComments, Count: len(e.Comments)})
	}
	if len(e.Recent) > 0 {
		reads = append(reads, billingqueue.AgentRead{
			Kind:  billingqueue.AgentReadPastInvoices,
			Count: len(e.Recent),
		})
	}
	for _, lookup := range taken.Charges {
		reads = append(reads, billingqueue.AgentRead{
			Kind:  billingqueue.AgentReadChargeHistory,
			Code:  lookup.Code,
			Count: lookup.Carried,
			Of:    lookup.Of,
		})
	}
	for _, doc := range e.Documents {
		excerpt := e.Excerpts[doc.ID]
		read := billingqueue.AgentRead{
			Kind:       billingqueue.AgentReadDocument,
			DocumentID: doc.ID,
			Name:       DocumentKind(doc),
			FileName:   doc.OriginalName,
			Parts:      excerpt.Parts,
		}
		if excerpt.Parts > 0 {
			read.Part = min(max(1, taken.Parts[doc.ID]), excerpt.Parts)
		}
		reads = append(reads, read)
	}

	return reads
}

func chargeLines(e *Evidence) int {
	if e.Item.Review == nil || e.Item.Review.Charges == nil {
		return 0
	}
	count := 0
	for _, line := range e.Item.Review.Charges.Lines {
		if !line.Removed {
			count++
		}
	}

	return count
}

func stopCount(e *Evidence) int {
	if e.Item.Shipment == nil {
		return 0
	}
	count := 0
	for _, move := range e.Item.Shipment.Moves {
		count += len(move.Stops)
	}

	return count
}
