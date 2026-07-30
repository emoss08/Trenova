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

	ShipmentProNumber string
	StopCount         int

	CredentialName string
	ExpiresInDays  int
	ExpiresAt      string

	// AlertTitle and AlertMessage carry text composed elsewhere — an hours-of-
	// service rule evaluated against a driver's logs, where the wording is the
	// rule's, not the organization's. A template wraps them rather than replacing
	// them, the same inversion the agent email uses.
	AlertTitle   string
	AlertMessage string
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
		ShipmentProNumber:     sampleProNumber,
		StopCount:             4,
		CredentialName:        "Commercial Driver's License",
		ExpiresInDays:         21,
		ExpiresAt:             "2026-08-19",
		AlertTitle:            "Drive time almost up",
		AlertMessage:          "You have 45 minutes of drive time left today.",
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
// expiresAtPath is named once because the catalog, the struct, and the starter
// all have to agree on it.
const expiresAtPath = "ExpiresAt"

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
			Path:        "Reason",
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
			Path:        "AlertTitle",
			Type:        VariableString,
			Description: "The headline an hours-of-service rule produced. Wrap it rather than replacing it: the rule knows what it measured.",
		},
		VariableDefinition{
			Path:        "AlertMessage",
			Type:        VariableString,
			Description: "The detail behind the alert, including the time remaining.",
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
			Path:        "Reason",
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
		kind:        KindNotificationCredentialExpiring,
		displayName: "Credential Expiring",
		description: "Warns a driver that a licence or certificate is about to lapse, while there is still time to renew it.",
	},
	{
		kind:        KindNotificationHOSAlert,
		displayName: "Hours of Service Alert",
		description: "Wraps an hours-of-service warning. The rule composes the text; this decides how it is presented.",
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
