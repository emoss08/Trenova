package bankreceipt

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type CandidateKind string

const (
	CandidateCustomerPayment = CandidateKind("CustomerPayment")
	CandidateInvoice         = CandidateKind("Invoice")
)

type Standing string

const (
	StandingExact      = Standing("Exact")
	StandingSameAmount = Standing("SameAmount")
	StandingShort      = Standing("Short")
	StandingOver       = Standing("Over")
)

type Evidence string

const (
	EvidenceReferenceExact    = Evidence("ReferenceExact")
	EvidenceReferencePartial  = Evidence("ReferencePartial")
	EvidenceNumberInReference = Evidence("NumberInReference")
	EvidenceNumberInMemo      = Evidence("NumberInMemo")
	EvidenceAmountExact       = Evidence("AmountExact")
	EvidenceAmountNear        = Evidence("AmountNear")
	EvidenceDateNear          = Evidence("DateNear")
	EvidencePayer             = Evidence("Payer")
)

type Candidate struct {
	keel.Table `keel:"bank_receipt_candidates"`

	OrganizationID    pulid.ID      `json:"organizationId"    keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID      `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	BankReceiptID     pulid.ID      `json:"bankReceiptId"     keel:"bank_receipt_id,pk"`
	Rank              int           `json:"rank"              keel:"rank,pk"`
	Kind              CandidateKind `json:"kind"              keel:"candidate_kind"`
	CustomerPaymentID pulid.ID      `json:"customerPaymentId" keel:"customer_payment_id,nullzero"`
	InvoiceID         pulid.ID      `json:"invoiceId"         keel:"invoice_id,nullzero"`
	CustomerID        pulid.ID      `json:"customerId"        keel:"customer_id,nullzero"`
	Score             int           `json:"score"             keel:"score"`
	Evidence          []string      `json:"evidence"          keel:"evidence"`
	Standing          Standing      `json:"standing"          keel:"standing"`
	OpenMinor         int64         `json:"openMinor"         keel:"open_minor"`
	DifferenceMinor   int64         `json:"differenceMinor"   keel:"difference_minor"`
	ComputedAt        int64         `json:"computedAt"        keel:"computed_at"`
}

func (c *Candidate) Has(evidence Evidence) bool {
	return slices.Contains(c.Evidence, string(evidence))
}

func (c *Candidate) ItemID() pulid.ID {
	if c.Kind == CandidateInvoice {
		return c.InvoiceID
	}

	return c.CustomerPaymentID
}

func (c *Candidate) PaymentReason() string {
	_, reason := paymentScore(paymentFacts{
		refExact:    c.Has(EvidenceReferenceExact),
		refContains: c.Has(EvidenceReferencePartial) || c.Has(EvidenceReferenceExact),
		amountExact: c.Has(EvidenceAmountExact),
		amountNear:  c.Has(EvidenceAmountNear),
		dateNear:    c.Has(EvidenceDateNear),
	})

	return reason
}
