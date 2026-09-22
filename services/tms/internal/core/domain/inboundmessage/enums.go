// Package inboundmessage is what arrives without anybody asking: a tender, a
// rate confirmation, a POD, an invoice, a customer wanting to know where their
// load is.
//
// All of it lands in somebody's inbox today and is retyped into the system by
// hand. The point of this domain is that it lands here instead, classified and
// matched to the records it is about, so the work is reviewing a decision
// rather than doing the data entry.
package inboundmessage

// Provider is who delivered the message to us.
type Provider string

const (
	ProviderPostmark = Provider("Postmark")
	ProviderResend   = Provider("Resend")
)

func (p Provider) IsValid() bool {
	switch p {
	case ProviderPostmark, ProviderResend:
		return true
	default:
		return false
	}
}

func AllProviders() []Provider {
	return []Provider{ProviderPostmark, ProviderResend}
}

// Classification is what the message turned out to be.
//
// The set is deliberately short. A classifier with thirty categories is one
// that picks the wrong one confidently; these are the kinds that lead to
// different work, and everything else is Other, which goes to a person.
type Classification string

const (
	ClassificationTender           = Classification("Tender")
	ClassificationRateConfirmation = Classification("RateConfirmation")
	ClassificationProofOfDelivery  = Classification("ProofOfDelivery")
	ClassificationInvoice          = Classification("Invoice")
	ClassificationStatusRequest    = Classification("StatusRequest")
	ClassificationDetentionDispute = Classification("DetentionDispute")
	ClassificationOther            = Classification("Other")
)

func (c Classification) IsValid() bool {
	switch c {
	case ClassificationTender, ClassificationRateConfirmation,
		ClassificationProofOfDelivery, ClassificationInvoice,
		ClassificationStatusRequest, ClassificationDetentionDispute,
		ClassificationOther:
		return true
	default:
		return false
	}
}

func AllClassifications() []Classification {
	return []Classification{
		ClassificationTender, ClassificationRateConfirmation,
		ClassificationProofOfDelivery, ClassificationInvoice,
		ClassificationStatusRequest, ClassificationDetentionDispute,
		ClassificationOther,
	}
}

// Status is where a message is in being dealt with.
type Status string

const (
	StatusReceived    = Status("Received")
	StatusProcessing  = Status("Processing")
	StatusClassified  = Status("Classified")
	StatusInReview    = Status("InReview")
	StatusActioned    = Status("Actioned")
	StatusIgnored     = Status("Ignored")
	StatusQuarantined = Status("Quarantined")
)

func (s Status) IsValid() bool {
	switch s {
	case StatusReceived, StatusProcessing, StatusClassified, StatusInReview,
		StatusActioned, StatusIgnored, StatusQuarantined:
		return true
	default:
		return false
	}
}

func AllStatuses() []Status {
	return []Status{
		StatusReceived, StatusProcessing, StatusClassified, StatusInReview,
		StatusActioned, StatusIgnored, StatusQuarantined,
	}
}

// Terminal reports whether a message is finished with, so a sweep knows what
// it may leave alone.
func (s Status) Terminal() bool {
	switch s {
	case StatusActioned, StatusIgnored, StatusQuarantined:
		return true
	default:
		return false
	}
}

// ReviewPolicy is how much a mailbox is trusted to act without a person.
//
// It is per mailbox rather than per organization because trust is not uniform:
// the address customers send status questions to can answer them; the one
// carriers send invoices to should not pay anything unread.
type ReviewPolicy string

const (
	// ReviewAlways puts every message in front of a person. It is the default,
	// and it is what a mailbox stays on until somebody decides otherwise.
	ReviewAlways = ReviewPolicy("AlwaysReview")
	// ReviewBelowConfidence handles what the classifier was sure about and
	// sends the rest to a person.
	ReviewBelowConfidence = ReviewPolicy("ReviewBelowConfidence")
	ReviewAutoHandle      = ReviewPolicy("AutoHandle")
)

func (p ReviewPolicy) IsValid() bool {
	switch p {
	case ReviewAlways, ReviewBelowConfidence, ReviewAutoHandle:
		return true
	default:
		return false
	}
}

func AllReviewPolicies() []ReviewPolicy {
	return []ReviewPolicy{ReviewAlways, ReviewBelowConfidence, ReviewAutoHandle}
}

// MailboxStatus is whether an address is still listening.
type MailboxStatus string

const (
	MailboxActive   = MailboxStatus("Active")
	MailboxInactive = MailboxStatus("Inactive")
)

func (s MailboxStatus) IsValid() bool {
	switch s {
	case MailboxActive, MailboxInactive:
		return true
	default:
		return false
	}
}

func AllMailboxStatuses() []MailboxStatus {
	return []MailboxStatus{MailboxActive, MailboxInactive}
}

// AttachmentKind is what a file turned out to be, once the document pipeline
// has read it.
type AttachmentKind string

const (
	AttachmentUnknown          = AttachmentKind("Unknown")
	AttachmentRateConfirmation = AttachmentKind("RateConfirmation")
	AttachmentProofOfDelivery  = AttachmentKind("ProofOfDelivery")
	AttachmentInvoice          = AttachmentKind("Invoice")
	AttachmentBillOfLading     = AttachmentKind("BillOfLading")
	AttachmentOther            = AttachmentKind("Other")
)

func (k AttachmentKind) IsValid() bool {
	switch k {
	case AttachmentUnknown, AttachmentRateConfirmation, AttachmentProofOfDelivery,
		AttachmentInvoice, AttachmentBillOfLading, AttachmentOther:
		return true
	default:
		return false
	}
}
