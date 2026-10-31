package billingqueue

import "github.com/emoss08/trenova/shared/pulid"

type AgentReadKind string

const (
	AgentReadInvoice       = AgentReadKind("Invoice")
	AgentReadStops         = AgentReadKind("Stops")
	AgentReadTerms         = AgentReadKind("Terms")
	AgentReadBillingNotes  = AgentReadKind("BillingNotes")
	AgentReadComments      = AgentReadKind("Comments")
	AgentReadPastInvoices  = AgentReadKind("PastInvoices")
	AgentReadChargeHistory = AgentReadKind("ChargeHistory")
	AgentReadDocument      = AgentReadKind("Document")
)

// AgentRead is one thing Nova read to review an item, as the biller is shown
// it: what the review was given and what it read further. A charge looked up
// counts the invoices that carried it (Count) of those it looked across (Of);
// a document, the parts read (Part) of its parts (Parts).
type AgentRead struct {
	Kind       AgentReadKind `json:"kind"`
	Count      int           `json:"count,omitempty"`
	Code       string        `json:"code,omitempty"`
	DocumentID pulid.ID      `json:"documentId,omitempty"`
	Name       string        `json:"name,omitempty"`
	FileName   string        `json:"fileName,omitempty"`
	Part       int           `json:"part,omitempty"`
	Parts      int           `json:"parts,omitempty"`
	Of         int           `json:"of,omitempty"`
}
