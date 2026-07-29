package documenttemplate

// The _gen suffix follows the convention set by permission/resource_gen.go: there
// is no generator, the suffix marks this as the flat declaration list that an
// AST-based coverage test parses. Every constant here must have a matching
// Register call, and registry_coverage_test.go fails the build if one does not.

// Kind identifies one authorable template.
//
// The value is stored in document_templates.kind as a plain varchar rather than a
// Postgres enum, precisely so adding a kind never requires a migration. This list
// is the single source of truth.
type Kind string

const (
	// Billing.

	// KindInvoicePDF is the customer invoice document.
	KindInvoicePDF Kind = "invoice.pdf"
	// KindInvoiceEmail is the message that delivers an invoice.
	KindInvoiceEmail Kind = "invoice.email"

	// Detention.

	// KindDetentionNoticeEmail is the notice sent while a driver is on a dock.
	KindDetentionNoticeEmail Kind = "detention.notice.email"
	// KindDetentionNoticePDF is the printable form of the same notice, which is
	// what turns it into billable evidence when a charge is disputed.
	KindDetentionNoticePDF Kind = "detention.notice.pdf"

	// Reporting.

	// KindReportPDF is the tabular report export.
	KindReportPDF Kind = "report.pdf"
	// KindReportDeliveryEmail is the scheduled-report delivery message.
	KindReportDeliveryEmail Kind = "report.delivery.email"

	// Portal.

	// KindDriverPortalInvitationEmail invites a driver to the portal.
	KindDriverPortalInvitationEmail Kind = "driverportal.invitation.email"

	// Agent.

	// KindAgentRequestMissingDocsEmail wraps agent-written prose in the
	// organization's own letterhead and signature.
	KindAgentRequestMissingDocsEmail Kind = "agent.request_missing_docs.email"
)

func (k Kind) String() string { return string(k) }

// AllKinds is the enumeration the coverage test walks. Keeping it beside the
// constants means a new kind is one edit away from being checked.
func AllKinds() []Kind {
	return []Kind{
		KindInvoicePDF,
		KindInvoiceEmail,
		KindDetentionNoticeEmail,
		KindDetentionNoticePDF,
		KindReportPDF,
		KindReportDeliveryEmail,
		KindDriverPortalInvitationEmail,
		KindAgentRequestMissingDocsEmail,
	}
}
