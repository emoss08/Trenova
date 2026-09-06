package documenttemplate

// In-app notifications are a title and a message. They are the highest-volume
// thing this system renders and the lowest-stakes individually, which shapes two
// decisions:
//
//   - Kinds are per event type, so the registry and notifications.event_type stay
//     in lockstep and an administrator rewording "load assigned" cannot touch
//     "load removed".
//   - Contexts are shared per *family*, because thirty structs holding a driver's
//     first name is thirty places for that field to drift. A family is drawn where
//     the data differs, not where the event names do: everything a driver is told
//     about their schedule and their credentials reads from one shape, everything
//     about their money reads from another.

// NotificationRecipient is who is being told. Every notification family embeds it,
// and the notification hub fills it from the worker it already loaded, so no call
// site restates the driver's name.
type NotificationRecipient struct {
	FirstName string
	LastName  string
	FullName  string
}

// SetRecipient satisfies RecipientAware.
func (r *NotificationRecipient) SetRecipient(first, last, full string) {
	r.FirstName = first
	r.LastName = last
	r.FullName = full
}

// RecipientAware is how the hub fills the recipient without knowing which family
// a context belongs to.
//
// The alternative — a switch over every context type in the hub — would need
// editing every time a family is added, which is exactly the coupling the
// registry exists to avoid.
type RecipientAware interface {
	SetRecipient(first, last, full string)
}

// DriverNotificationContext is what a driver is told about their work: the loads
// on their schedule, their time off, their credentials, and their hours.
type DriverNotificationContext struct {
	NotificationRecipient

	// Approved carries the decision for the events that are an answer to a
	// request. A template that does not branch on it will tell a driver their
	// time off was approved when it was denied.
	Approved bool

	// Reason is the reviewer's note, when there is one. Denials usually have one
	// and approvals usually do not.
	Reason string

	// PolicyTitle and PolicyVersion name what a driver is being asked to sign.
	PolicyTitle   string
	PolicyVersion string

	ShipmentProNumber string
	StopCount         int

	CredentialName string
	ExpiresInDays  int
	ExpiresAt      string

	// CourseName, DueInDays and DueAt describe a training assignment. Renewal is
	// set when the reminder is about a lapsing certification rather than a course
	// the driver has not taken yet.
	CourseName string
	DueInDays  int
	DueAt      string
	Renewal    bool

	// ReviewTitle and ReviewerName describe a performance review waiting for the
	// driver's sign-off.
	ReviewTitle  string
	ReviewerName string

	// RecognitionTitle and RecognitionMessage carry praise the office recorded.
	RecognitionTitle   string
	RecognitionMessage string

	// DisciplineLevel and DisciplineReason describe a disciplinary action the
	// driver must acknowledge.
	DisciplineLevel  string
	DisciplineReason string

	// Digest is everything a driver owes, bundled into one notice when the
	// carrier has chosen a daily or weekly cadence. A template ranges over it;
	// the individual reminders are suppressed while it is in use, so a driver
	// is told once rather than twice.
	Digest []DriverObligation
	// DigestPeriod names the round-up — "today" or "this week" — so the
	// wording can say which without the template having to guess.
	DigestPeriod string

	// AlertTitle and AlertMessage carry text composed elsewhere — an hours-of-
	// service rule evaluated against a driver's logs, where the wording is the
	// rule's, not the organization's. A template wraps them rather than replacing
	// them, the same inversion the agent email uses.
	AlertTitle   string
	AlertMessage string
}

// DriverObligation is one line of an obligations digest: a credential to renew
// or a course to finish, with when it falls due.
type DriverObligation struct {
	// What is the credential or course by name, as the driver knows it.
	What string
	// Kind is "credential" or "training", so a template can group or icon them
	// without parsing the name.
	Kind string
	// DueAt is the calendar date, already formatted.
	DueAt string
	// DueInDays is negative once the date has passed.
	DueInDays int
	// Overdue is set rather than inferred from DueInDays, so a template that
	// does not do arithmetic still gets the distinction right.
	Overdue bool
}

// SettlementNotificationContext is what a driver is told about their pay.
//
// Money notifications are separated from the rest because they are the ones a
// driver reads carefully and the ones that carry numbers. Nothing here is a
// currency-formatted amount by accident: the amounts arrive already formatted, so
// the notification and the settlement statement quote money identically.
type SettlementNotificationContext struct {
	NotificationRecipient

	// Approved is the decision on an expense or a dispute.
	Approved bool

	// Reason is the review note, the hold reason, or the dispute resolution note.
	// It is the sentence a driver actually acts on.
	Reason string

	SettlementNumber string
	PaymentMethod    string
	Amount           string
	Currency         string
}

func newDriverNotificationSampleContext() any {
	return DriverNotificationContext{
		NotificationRecipient: sampleRecipient(),
		Approved:              true,
		Reason:                "Covered by another driver.",
		PolicyTitle:           "Driver Handbook",
		PolicyVersion:         "2026.1",
		ShipmentProNumber:     sampleProNumber,
		StopCount:             4,
		CredentialName:        "Commercial Driver's License",
		ExpiresInDays:         21,
		ExpiresAt:             "2026-08-19",
		CourseName:            "Defensive Driving",
		DueInDays:             5,
		DueAt:                 "2026-08-03",
		Renewal:               false,
		ReviewTitle:           "H1 2026 review",
		ReviewerName:          "Dana Ortiz",
		Digest: []DriverObligation{
			{
				What:      "Commercial Driver's License",
				Kind:      "credential",
				DueAt:     "2026-08-19",
				DueInDays: 21,
			},
			{
				What:      "Defensive Driving",
				Kind:      "training",
				DueAt:     "2026-08-03",
				DueInDays: -2,
				Overdue:   true,
			},
		},
		DigestPeriod:       "this week",
		RecognitionTitle:   "Six months accident-free",
		RecognitionMessage: "Clean inspections all spring. Thank you.",
		DisciplineLevel:    "WrittenWarning",
		DisciplineReason:   "Second late departure this month",
		AlertTitle:         "Drive time almost up",
		AlertMessage:       "You have 45 minutes of drive time left today.",
	}
}

func newSettlementNotificationSampleContext() any {
	return SettlementNotificationContext{
		NotificationRecipient: sampleRecipient(),
		Approved:              true,
		Reason:                "Receipt did not match the amount claimed.",
		SettlementNumber:      "STL-004182",
		PaymentMethod:         "ACH",
		Amount:                "USD 1,842.60",
		Currency:              "USD",
	}
}

// recipientVariables is the catalog for the embedded recipient block. Every
// family shares it, so the description an administrator reads is identical
// wherever the field appears.
// These paths are named once because the catalog, the struct, and the starter all
// have to agree on them.
const (
	expiresAtPath = "ExpiresAt"
	reasonPath    = "Reason"
)

func recipientVariables() []VariableDefinition {
	return []VariableDefinition{
		{
			Path:        "FirstName",
			Type:        VariableString,
			Description: "The driver's first name. Filled in for you from their worker record.",
		},
		{
			Path:        "LastName",
			Type:        VariableString,
			Description: "The driver's last name.",
		},
		{
			Path:        "FullName",
			Type:        VariableString,
			Description: "The driver's full name, for a greeting that reads naturally.",
		},
	}
}

// driverNotificationVariables is one catalog shared by every kind in the driver
// family.
//
// It is built by a function rather than copied per kind because
// registry_context_test.go asserts the catalog and the struct match in both
// directions for every kind: copies would have to be corrected in ten places
// whenever a field is added, and the test would only tell you about the first one.
func driverNotificationVariables() []VariableDefinition {
	return append(recipientVariables(),
		VariableDefinition{
			Path:        "Approved",
			Type:        VariableBool,
			Description: "Whether the request was approved. Branch on this or a denial will read as an approval.",
		},
		VariableDefinition{
			Path:        reasonPath,
			Type:        VariableString,
			Description: "The reviewer's note. Usually present on a denial and absent on an approval.",
		},
		proNumberVariable(false, "The load this is about, when it is about a load."),
		VariableDefinition{
			Path:        "StopCount",
			Type:        VariableInt,
			Description: "How many stops the load has, so a driver can size the day before opening it.",
		},
		VariableDefinition{
			Path:        "PolicyTitle",
			Type:        VariableString,
			Description: "The policy or handbook a driver is being asked to read and sign.",
		},
		VariableDefinition{
			Path:        "PolicyVersion",
			Type:        VariableString,
			Description: "Which version of it, so a driver who signed an earlier one knows this is new.",
		},
		VariableDefinition{
			Path:        "CredentialName",
			Type:        VariableString,
			Description: "The licence or certificate this is about — a CDL, a medical card.",
		},
		VariableDefinition{
			Path:        "ExpiresInDays",
			Type:        VariableInt,
			Description: "Days until it lapses. Zero or negative means it already has.",
		},
		VariableDefinition{
			Path:        expiresAtPath,
			Type:        VariableDate,
			Description: "The expiry date, in the organization's timezone.",
		},
		VariableDefinition{
			Path:        "CourseName",
			Type:        VariableString,
			Description: "The training course this is about.",
		},
		VariableDefinition{
			Path:        "DueInDays",
			Type:        VariableInt,
			Description: "Days until the course is due, or until the certification lapses. Zero is today; negative is already past.",
		},
		VariableDefinition{
			Path:        "DueAt",
			Type:        VariableDate,
			Description: "The due or expiry date.",
		},
		VariableDefinition{
			Path:        "Renewal",
			Type:        VariableBool,
			Description: "True when a completed certification is lapsing and needs renewing, false for a course not yet taken.",
		},
		VariableDefinition{
			Path:        "ReviewTitle",
			Type:        VariableString,
			Description: "The performance review waiting for the driver's sign-off.",
		},
		VariableDefinition{
			Path:        "ReviewerName",
			Type:        VariableString,
			Description: "Who wrote the review.",
		},
		VariableDefinition{
			Path:        "RecognitionTitle",
			Type:        VariableString,
			Description: "The headline of the praise being shared.",
		},
		VariableDefinition{
			Path:        "RecognitionMessage",
			Type:        VariableString,
			Description: "The note that goes with it, when there is one.",
		},
		VariableDefinition{
			Path:        "DisciplineLevel",
			Type:        VariableString,
			Description: "The rung of the ladder that was issued: Coaching, VerbalWarning, WrittenWarning, FinalWarning, Suspension, Termination.",
		},
		VariableDefinition{
			Path:        "DisciplineReason",
			Type:        VariableString,
			Description: "Why the action was taken.",
		},
		VariableDefinition{
			Path:        "AlertTitle",
			Type:        VariableString,
			Description: "The headline an hours-of-service rule produced. Wrap it rather than replacing it: the rule knows what it measured.",
		},
		VariableDefinition{
			Path:        "AlertMessage",
			Type:        VariableString,
			Description: "The detail behind the alert, including the time remaining.",
		},
		VariableDefinition{
			Path:        "Digest",
			Type:        VariableCollection,
			Description: "Everything the driver owes, bundled into one notice. Range over it; empty on every kind but the digest.",
			Fields: []VariableDefinition{
				{
					Path:        "What",
					Type:        VariableString,
					Description: "The credential or course by name, as the driver knows it.",
				},
				{
					Path:        "Kind",
					Type:        VariableString,
					Description: "Either \"credential\" or \"training\", for grouping or an icon.",
				},
				{
					Path:        "DueAt",
					Type:        VariableDate,
					Description: "The date it falls due.",
				},
				{
					Path:        "DueInDays",
					Type:        VariableInt,
					Description: "Days until it falls due; negative once the date has passed.",
				},
				{
					Path:        "Overdue",
					Type:        VariableBool,
					Description: "True once the date has passed. Set rather than inferred, so a template that does no arithmetic still gets it right.",
				},
			},
		},
		VariableDefinition{
			Path:        "DigestPeriod",
			Type:        VariableString,
			Description: "Which round-up this is — \"today\" or \"this week\" — so the wording can say so without guessing.",
		},
	)
}

// settlementNotificationVariables is the settlement family's shared catalog.
func settlementNotificationVariables() []VariableDefinition {
	return append(recipientVariables(),
		VariableDefinition{
			Path:        "Approved",
			Type:        VariableBool,
			Description: "Whether the expense or dispute was approved. Branch on this or a denial will read as an approval.",
		},
		VariableDefinition{
			Path:        reasonPath,
			Type:        VariableString,
			Description: "The review note, hold reason, or resolution note — the sentence the driver acts on.",
		},
		VariableDefinition{
			Path:        "SettlementNumber",
			Type:        VariableString,
			Description: "The settlement this concerns, so a driver can find it in their history.",
		},
		VariableDefinition{
			Path:        "PaymentMethod",
			Type:        VariableString,
			Description: "How the money moved — ACH, cheque, fuel card.",
		},
		VariableDefinition{
			Path:        "Amount",
			Type:        VariableMoney,
			Description: "The amount, already formatted with its currency, so it reads the same here as on the statement.",
		},
		VariableDefinition{
			Path:        "Currency",
			Type:        VariableString,
			Description: "The currency code behind the amount.",
		},
	)
}

// notificationCategory groups every notification kind in the admin catalog. It is
// a constant because the registry tests key an exemption off it.
const notificationCategory = "Notifications"

// sampleRecipient is the same driver in every family's sample, so an
// administrator comparing two notification templates side by side is not
// distracted by two different names.
func sampleRecipient() NotificationRecipient {
	const (
		firstName = "Marcus"
		lastName  = "Dell"
	)

	return NotificationRecipient{
		FirstName: firstName,
		LastName:  lastName,
		FullName:  firstName + " " + lastName,
	}
}

// ReportNotificationContext is what the person who asked for a report, or owns
// the schedule that produced it, is told about it.
//
// These notifications go to internal users rather than to drivers, so there is no
// recipient block: the reader is whoever requested the run, and naming them back
// to themselves reads oddly.
type ReportNotificationContext struct {
	ReportName string
	Format     string

	RowCount  int64
	FileSize  string
	Truncated bool

	// Reason is the failure. It is the sentence the reader needs: a report that
	// "could not be generated" with no cause is a support ticket.
	Reason string

	// Disabled marks the schedule that has now been turned off rather than merely
	// skipped, which is a different message with the same event type.
	Disabled            bool
	ConsecutiveFailures int

	ExpiresAt string
}

func newReportNotificationSampleContext() any {
	return ReportNotificationContext{
		ReportName:          sampleReportTitle,
		Format:              "xlsx",
		RowCount:            67,
		FileSize:            "184 KB",
		Truncated:           false,
		Reason:              "The email profile has no recipients configured",
		Disabled:            false,
		ConsecutiveFailures: 3,
		ExpiresAt:           "2026-07-22",
	}
}

// reportNotificationVariables is the report family's shared catalog.
func reportNotificationVariables() []VariableDefinition {
	return []VariableDefinition{
		{
			Path:        "ReportName",
			Type:        VariableString,
			Description: "The report this is about. A notification that does not name it is one nobody can act on.",
		},
		{
			Path:        "Format",
			Type:        VariableString,
			Description: "The export format — xlsx, csv, pdf. Pipe it through upper if you want it shouted.",
		},
		rowCountVariable(),
		{
			Path:        "FileSize",
			Type:        VariableString,
			Description: "The artifact's size, already humanized.",
		},
		truncatedVariable(),
		{
			Path:        reasonPath,
			Type:        VariableString,
			Description: "Why it failed, when it failed. Print it: a failure with no cause becomes a support ticket.",
		},
		{
			Path:        "Disabled",
			Type:        VariableBool,
			Description: "Whether the schedule was turned off rather than merely skipped this once.",
		},
		{
			Path:        "ConsecutiveFailures",
			Type:        VariableInt,
			Description: "How many runs in a row failed, which is what justified disabling the schedule.",
		},
		{
			Path:        expiresAtPath,
			Type:        VariableDate,
			Description: "When the download stops being available.",
		},
	}
}

// CommentNotificationContext is what a user is told when a teammate writes
// something they need to see.
//
// The excerpt is the substance: a notification that says only "you were
// mentioned" makes the reader open the shipment to find out whether it mattered.
type CommentNotificationContext struct {
	AuthorName        string
	ShipmentProNumber string

	// Excerpt is the comment, already truncated to a notification-sized length.
	Excerpt string
}

func newCommentNotificationSampleContext() any {
	return CommentNotificationContext{
		AuthorName:        "Dana Whitfield",
		ShipmentProNumber: sampleProNumber,
		Excerpt:           "Receiver moved the appointment to 14:00 — can we still make it?",
	}
}

// commentNotificationVariables is the comment family's shared catalog.
func commentNotificationVariables() []VariableDefinition {
	return []VariableDefinition{
		{
			Path:        "AuthorName",
			Type:        VariableString,
			Description: "Who wrote it. A notification from nobody is one the reader cannot judge the urgency of.",
		},
		proNumberVariable(false, "The shipment the comment is on."),
		{
			Path:        "Excerpt",
			Type:        VariableString,
			Description: "The comment itself, already shortened. Print it: without it the reader has to open the shipment to find out whether it mattered.",
		},
	}
}

// notificationKind is one registered event, sharing its family's context.
type notificationKind struct {
	kind        Kind
	displayName string
	description string
}

var driverNotificationKinds = []notificationKind{
	{
		kind:        KindNotificationLoadAssigned,
		displayName: "Load Assigned",
		description: "Tells a driver a load is now on their schedule.",
	},
	{
		kind:        KindNotificationLoadUnassigned,
		displayName: "Load Removed",
		description: "Tells a driver a load has been taken off their schedule, so they do not drive to a stop that is no longer theirs.",
	},
	{
		kind:        KindNotificationPTOReviewed,
		displayName: "Time Off Reviewed",
		description: "The answer to a time-off request. Branch on Approved: this kind carries both outcomes.",
	},
	{
		kind:        KindNotificationPTOCancelled,
		displayName: "Time Off Cancelled",
		description: "Tells a driver that a pending or approved time-off request was cancelled by the office, with the reason when one was given.",
	},
	{
		kind:        KindNotificationProfileChangeReviewed,
		displayName: "Profile Change Reviewed",
		description: "The answer to a driver's request to change their own record. Branch on Approved: this kind carries both outcomes.",
	},
	{
		kind:        KindNotificationPolicyPublished,
		displayName: "Policy Published",
		description: "Asks a driver to read and sign a policy the carrier has published or revised.",
	},
	{
		kind:        KindNotificationCredentialExpiring,
		displayName: "Credential Expiring",
		description: "Warns a driver that a licence or certificate is about to lapse, while there is still time to renew it.",
	},
	{
		kind:        KindNotificationTrainingDue,
		displayName: "Training Due",
		description: "Nudges a driver about a course that is due, or a certification that is about to lapse. Branch on Renewal: this kind carries both.",
	},
	{
		kind:        KindNotificationReviewSubmitted,
		displayName: "Review Ready to Sign",
		description: "Asks a driver to read a performance review and sign off on it in Dash.",
	},
	{
		kind:        KindNotificationRecognition,
		displayName: "Recognition",
		description: "Shares praise the office recorded for the driver.",
	},
	{
		kind:        KindNotificationDisciplinaryIssued,
		displayName: "Disciplinary Action Issued",
		description: "Tells a driver a disciplinary action was issued and asks them to acknowledge it.",
	},
	{
		kind:        KindNotificationHOSAlert,
		displayName: "Hours of Service Alert",
		description: "Wraps an hours-of-service warning. The rule composes the text; this decides how it is presented.",
	},
	{
		kind:        KindNotificationObligationsDigest,
		displayName: "Obligations Digest",
		description: "Everything a driver owes, in one notice. Range over Digest: each entry carries What, DueAt and DueInDays, and Overdue is set on the ones already past. DigestPeriod says whether this is the daily or the weekly round-up.",
	},
}

var settlementNotificationKinds = []notificationKind{
	{
		kind:        KindNotificationSettlementPosted,
		displayName: "Settlement Statement Ready",
		description: "Tells a driver their settlement statement has been issued and can be reviewed.",
	},
	{
		kind:        KindNotificationSettlementPaid,
		displayName: "Settlement Paid",
		description: "Tells a driver their settlement has been paid, and how.",
	},
	{
		kind:        KindNotificationPayHeld,
		displayName: "Pay Held",
		description: "Tells a driver pay for a load was held, and why. Reason carries the explanation they will ask for.",
	},
	{
		kind:        KindNotificationExpenseReviewed,
		displayName: "Expense Reviewed",
		description: "The answer to a submitted expense. Branch on Approved: this kind carries both outcomes.",
	},
	{
		kind:        KindNotificationDisputeResolved,
		displayName: "Pay Dispute Resolved",
		description: "The outcome of a pay dispute. Branch on Approved: a driver reads this one closely either way.",
	},
}

var reportNotificationKinds = []notificationKind{
	{
		kind:        KindNotificationReportRunCompleted,
		displayName: "Report Ready",
		description: "Tells whoever asked for a report that it finished and can be downloaded.",
	},
	{
		kind:        KindNotificationReportRunFailed,
		displayName: "Report Failed",
		description: "Tells whoever asked for a report that it could not be generated, and why.",
	},
	{
		kind:        KindNotificationReportRunCanceled,
		displayName: "Report Canceled",
		description: "Tells whoever asked for a report that the run was canceled.",
	},
	{
		kind:        KindNotificationReportDelivered,
		displayName: "Scheduled Report Ready",
		description: "Tells a recipient that a scheduled report has been produced for them.",
	},
	{
		kind:        KindNotificationReportDeliveryFailed,
		displayName: "Scheduled Report Email Failed",
		description: "Tells a schedule owner the report was produced but could not be emailed. Reason carries the cause.",
	},
	{
		kind:        KindNotificationReportScheduleSkipped,
		displayName: "Report Schedule Skipped",
		description: "Tells a schedule owner a run was skipped. Branch on Disabled: the same event covers a schedule that has now been turned off after repeated failures.",
	},
}

var commentNotificationKinds = []notificationKind{
	{
		kind:        KindNotificationCommentMention,
		displayName: "Mentioned in a Comment",
		description: "Tells someone a teammate named them in a shipment comment.",
	},
	{
		kind:        KindNotificationCommentReply,
		displayName: "Reply to Your Comment",
		description: "Tells someone a teammate replied to their shipment comment.",
	},
}

func (r *Registry) registerNotificationKinds() {
	r.registerNotificationFamily(
		driverNotificationKinds,
		driverNotificationVariables(),
		newDriverNotificationSampleContext,
	)
	r.registerNotificationFamily(
		settlementNotificationKinds,
		settlementNotificationVariables(),
		newSettlementNotificationSampleContext,
	)
	r.registerNotificationFamily(
		reportNotificationKinds,
		reportNotificationVariables(),
		newReportNotificationSampleContext,
	)
	r.registerNotificationFamily(
		commentNotificationKinds,
		commentNotificationVariables(),
		newCommentNotificationSampleContext,
	)
}

// registerNotificationFamily registers every event in a family against one
// catalog and one sample context.
func (r *Registry) registerNotificationFamily(
	kinds []notificationKind,
	variables []VariableDefinition,
	sample func() any,
) {
	for _, entry := range kinds {
		_ = r.Register(&KindDefinition{
			Kind:        entry.kind,
			DisplayName: entry.displayName,
			Description: entry.description,
			Category:    notificationCategory,
			// An in-app notification is a title and a body. There is no HTML
			// channel: the client renders both as text into its own layout, and a
			// template that emitted markup would show it as characters.
			Channels:      []Channel{ChannelNotificationTitle, ChannelNotificationBody},
			sampleFactory: sample,
			Variables:     variables,
		})
	}
}
