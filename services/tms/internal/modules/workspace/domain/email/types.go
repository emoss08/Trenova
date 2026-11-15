package email

import "slices"

type Kind string

const (
	KindSystem         Kind = "System"
	KindCustomerFacing Kind = "CustomerFacing"
)

func (k Kind) IsValid() bool {
	switch k {
	case KindSystem, KindCustomerFacing:
		return true
	default:
		return false
	}
}

type Sender string

const (
	SenderOrganization Sender = "Organization"
	SenderTrenova      Sender = "Trenova"
)

func (s Sender) IsValid() bool {
	switch s {
	case SenderOrganization, SenderTrenova:
		return true
	default:
		return false
	}
}

type Type string

type typeDefinition struct {
	kind       Kind
	invitation bool
}

var definitions = map[Type]typeDefinition{}

func defineType(name string, kind Kind) Type {
	t := Type(name)
	definitions[t] = typeDefinition{kind: kind}
	return t
}

func defineInvitation(name string) Type {
	t := Type(name)
	definitions[t] = typeDefinition{kind: KindSystem, invitation: true}
	return t
}

var (
	TypePasswordReset          = defineType("PasswordReset", KindSystem)
	TypeAccountInvitation      = defineInvitation("AccountInvitation")
	TypeDriverPortalInvitation = defineInvitation("DriverPortalInvitation")
	TypeNotification           = defineType("Notification", KindSystem)
	TypeNotificationDigest     = defineType("NotificationDigest", KindSystem)
	TypeProposalReminder       = defineType("ProposalReminder", KindSystem)
	TypeRecordShare            = defineType("RecordShare", KindSystem)
	TypeReportDelivery         = defineType("ReportDelivery", KindSystem)

	TypeInvoice                     = defineType("Invoice", KindCustomerFacing)
	TypeCustomerStatement           = defineType("CustomerStatement", KindCustomerFacing)
	TypePaymentReminder             = defineType("PaymentReminder", KindCustomerFacing)
	TypeFactoringNoticeOfAssignment = defineType(
		"FactoringNoticeOfAssignment",
		KindCustomerFacing,
	)
	TypeFactoringDocumentRequest = defineType("FactoringDocumentRequest", KindCustomerFacing)
	TypeFactoringSubmission      = defineType("FactoringSubmission", KindCustomerFacing)
	TypeCustomerRequest          = defineType("CustomerRequest", KindCustomerFacing)
	TypeCustomerUpdate           = defineType("CustomerUpdate", KindCustomerFacing)
	TypeCustomerTrackingLink     = defineType("CustomerTrackingLink", KindCustomerFacing)
	TypeCustomerStatusUpdate     = defineType("CustomerStatusUpdate", KindCustomerFacing)
	TypeCarrierTrackingLink      = defineType("CarrierTrackingLink", KindCustomerFacing)
	TypeTenderOffer              = defineType("TenderOffer", KindCustomerFacing)
	TypeRateConfirmation         = defineType("RateConfirmation", KindCustomerFacing)
	TypeDetentionNotice          = defineType("DetentionNotice", KindCustomerFacing)
	TypeInboundReply             = defineType("InboundReply", KindCustomerFacing)
	TypeMailboxTest              = defineType("MailboxTest", KindCustomerFacing)
	TypeTemplateTest             = defineType("TemplateTest", KindCustomerFacing)
	TypeProfileTest              = defineType("ProfileTest", KindCustomerFacing)
)

func Types() []Type {
	out := make([]Type, 0, len(definitions))
	for t := range definitions {
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

func (t Type) IsValid() bool {
	_, ok := definitions[t]
	return ok
}

func (t Type) Kind() Kind {
	return definitions[t].kind
}

func (t Type) IsSystem() bool {
	return t.Kind() == KindSystem
}

func (t Type) IsInvitation() bool {
	return definitions[t].invitation
}
