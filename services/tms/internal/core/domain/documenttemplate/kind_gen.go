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
	// KindRateConfirmationPDF is the outbound rate confirmation a carrier signs
	// before hauling a brokered move.
	KindRateConfirmationPDF Kind = "rateconfirmation.pdf"
	// KindRateConfirmationEmail is the message that delivers a rate confirmation.
	KindRateConfirmationEmail Kind = "rateconfirmation.email"

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

	// Notifications.
	//
	// The key is "notification." plus the exact event type already stored in
	// notifications.event_type, so the registry and the column cannot drift. The
	// events themselves are named by the services that emit them; do not rename
	// one here without renaming it there.

	// KindNotificationLoadAssigned tells a driver a load is on their schedule.
	KindNotificationLoadAssigned Kind = "notification.dash.load_assigned"
	// KindNotificationLoadUnassigned tells a driver a load was removed.
	KindNotificationLoadUnassigned Kind = "notification.dash.load_unassigned"
	// KindNotificationPTOReviewed answers a time-off request.
	KindNotificationPTOReviewed Kind = "notification.dash.pto_reviewed"
	// KindNotificationCredentialExpiring warns about a lapsing credential.
	KindNotificationCredentialExpiring Kind = "notification.dash.credential_expiring"
	// KindNotificationHOSAlert wraps an hours-of-service warning.
	KindNotificationHOSAlert Kind = "notification.dash.hos_alert"

	// KindNotificationSettlementPosted announces an issued settlement statement.
	KindNotificationSettlementPosted Kind = "notification.dash.settlement_posted"
	// KindNotificationSettlementPaid announces a paid settlement.
	KindNotificationSettlementPaid Kind = "notification.dash.settlement_paid"
	// KindNotificationPayHeld tells a driver pay was held, and why.
	KindNotificationPayHeld Kind = "notification.dash.pay_held"
	// KindNotificationExpenseReviewed answers a submitted expense.
	KindNotificationExpenseReviewed Kind = "notification.dash.expense_reviewed"
	// KindNotificationDisputeResolved reports the outcome of a pay dispute.
	KindNotificationDisputeResolved Kind = "notification.dash.dispute_resolved"

	// KindNotificationReportRunCompleted announces a finished report.
	KindNotificationReportRunCompleted Kind = "notification.report_run_completed"
	// KindNotificationReportRunFailed reports a report that could not be built.
	KindNotificationReportRunFailed Kind = "notification.report_run_failed"
	// KindNotificationReportRunCanceled reports a canceled run.
	KindNotificationReportRunCanceled Kind = "notification.report_run_canceled"
	// KindNotificationReportDelivered announces a delivered scheduled report.
	KindNotificationReportDelivered Kind = "notification.report_run_delivered"
	// KindNotificationReportDeliveryFailed reports a scheduled report that was
	// produced but could not be emailed.
	KindNotificationReportDeliveryFailed Kind = "notification.report_delivery_email_failed"
	// KindNotificationReportScheduleSkipped reports a skipped or disabled schedule.
	KindNotificationReportScheduleSkipped Kind = "notification.report_schedule_skipped"

	// KindNotificationCommentMention tells someone they were named in a comment.
	KindNotificationCommentMention Kind = "notification.shipment_comment_mention"
	// KindNotificationCommentReply tells someone their comment got a reply.
	KindNotificationCommentReply Kind = "notification.shipment_comment_reply"
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
		KindRateConfirmationPDF,
		KindRateConfirmationEmail,
		KindReportPDF,
		KindReportDeliveryEmail,
		KindDriverPortalInvitationEmail,
		KindAgentRequestMissingDocsEmail,
		KindNotificationLoadAssigned,
		KindNotificationLoadUnassigned,
		KindNotificationPTOReviewed,
		KindNotificationCredentialExpiring,
		KindNotificationHOSAlert,
		KindNotificationSettlementPosted,
		KindNotificationSettlementPaid,
		KindNotificationPayHeld,
		KindNotificationExpenseReviewed,
		KindNotificationDisputeResolved,
		KindNotificationReportRunCompleted,
		KindNotificationReportRunFailed,
		KindNotificationReportRunCanceled,
		KindNotificationReportDelivered,
		KindNotificationReportDeliveryFailed,
		KindNotificationReportScheduleSkipped,
		KindNotificationCommentMention,
		KindNotificationCommentReply,
	}
}
