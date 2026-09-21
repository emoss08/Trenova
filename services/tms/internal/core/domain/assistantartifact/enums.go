package assistantartifact

// Kind is what an artifact is, which decides how the Desk renders it.
type Kind string

const (
	// KindReportPreview holds the rows a preview returned, rendered as a table.
	KindReportPreview Kind = "report_preview"
	// KindReportRun points at a finished run whose rows are read on demand.
	KindReportRun Kind = "report_run"
	// KindEmailDraft is an outbound email waiting on a person, a view over
	// the proposal that carries the decision.
	KindEmailDraft Kind = "email_draft"
	// KindPlan is several writes decided as one, a view over the plan.
	KindPlan Kind = "plan"
	// KindEntityCard is one record the assistant looked up.
	KindEntityCard Kind = "entity_card"
	// KindTableView is a filtered, sorted view of a table the person can open.
	KindTableView Kind = "table_view"
	// KindRateExplanation is the ledger behind a rate.
	KindRateExplanation Kind = "rate_explanation"
	// KindDashboardRef points at a dashboard the assistant made or found.
	KindDashboardRef Kind = "dashboard_ref"
	// KindBriefing is a day's briefing.
	KindBriefing Kind = "briefing"
	// KindInboundMessage is a message from the inbox the conversation is about.
	KindInboundMessage Kind = "inbound_message"
)

func (k Kind) IsValid() bool {
	switch k {
	case KindReportPreview, KindReportRun, KindEmailDraft, KindPlan, KindEntityCard,
		KindTableView, KindRateExplanation, KindDashboardRef, KindBriefing, KindInboundMessage:
		return true
	default:
		return false
	}
}

// Status is where an artifact is in its life. Most are Ready when made; a
// run is Pending until it finishes, and a draft is Sent once it went.
type Status string

const (
	StatusPending Status = "Pending"
	StatusReady   Status = "Ready"
	StatusFailed  Status = "Failed"
	StatusSent    Status = "Sent"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusReady, StatusFailed, StatusSent:
		return true
	default:
		return false
	}
}
