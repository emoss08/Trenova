package notificationtemplate

import (
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

/*
Sharing is one thing whatever is shared: an invoice, a shipment, a worker's
case. So it is one notification and one email, told what the record is in
words (RecordTitle, "Invoice INV-2026-1042") rather than one pair per kind of
record. Nothing here is specific to a kind, which is what lets a new kind be
shared without a new template.
*/

// RecordShareNotificationContext is the in-app notice of a share. The sharer's
// note is not in it: in-app notifications are broadcast to the organization's
// live clients, so the note is only sent by email.
type RecordShareNotificationContext struct {
	SharedByName string
	// RecordTitle names the record inside a sentence: its kind, then its label
	// ("invoice INV-2026-1042").
	RecordTitle string
	// RecordSubtitle is what the record says of itself beside its name, such
	// as the customer an invoice bills; empty when it says nothing.
	RecordSubtitle string
}

func newRecordShareNotificationSampleContext() any {
	return RecordShareNotificationContext{
		SharedByName:   "Marcus Bell",
		RecordTitle:    "invoice " + templatecatalog.SampleInvoiceNo,
		RecordSubtitle: templatecatalog.SampleCustomerName,
	}
}

func recordShareNotificationVariables() []templatecatalog.VariableDefinition {
	return []templatecatalog.VariableDefinition{
		{
			Path:        "SharedByName",
			Type:        templatecatalog.VariableString,
			Description: "Who shared the record.",
		},
		{
			Path:        "RecordTitle",
			Type:        templatecatalog.VariableString,
			Required:    true,
			Description: "What was shared, as a heading names it: the kind of record and its label, such as \"invoice INV-2026-1042\"; it is written to sit inside a sentence.",
		},
		{
			Path:        "RecordSubtitle",
			Type:        templatecatalog.VariableString,
			Description: "What the record says beside its name, such as the customer an invoice bills. Empty for a record with nothing to add.",
		},
	}
}

var recordShareNotificationKinds = []notificationKind{
	{
		kind:        documenttemplate.KindNotificationRecordShared,
		displayName: "Record Shared With You",
		description: "Tells someone a teammate shared a record with them: an invoice, a shipment, a case. The sharer's note is not available here: in-app notifications are broadcast to the organization's live clients, so it is only sent by email.",
	},
}

// RecordShareEmailContext is the email a share sends, with the sharer's note
// and the link that opens what was shared.
type RecordShareEmailContext struct {
	RecipientFirstName string
	SharedByName       string
	RecordTitle        string
	RecordSubtitle     string
	// RecordNoun is the kind of record in a word ("invoice"), for the label
	// above the heading.
	RecordNoun  string
	Note        string
	RecordURL   template.URL
	CompanyName string
}

func newRecordShareEmailSampleContext() any {
	return RecordShareEmailContext{
		RecipientFirstName: "Dana",
		SharedByName:       "Marcus Bell",
		RecordTitle:        "invoice " + templatecatalog.SampleInvoiceNo,
		RecordSubtitle:     templatecatalog.SampleCustomerName,
		RecordNoun:         "invoice",
		Note:               "Halstead disputed the detention line on this one. Can you check it against the gate times before we send?",
		RecordURL: template.URL(
			"https://app.example.com/billing/invoices?item=inv_01J2Y8Z3Q4R5S6T7V8W9X0Y1Z2&tab=charges",
		),
		CompanyName: templatecatalog.SampleCompanyName,
	}
}

// RegisterRecordShareKinds registers the email a share sends. Its in-app
// notification is registered with the other notification families.
func RegisterRecordShareKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindRecordShareEmail,
		DisplayName: "Record Shared With a Teammate",
		Description: "Tells a teammate someone shared a record with them (an invoice, a shipment, a case), with the sharer's note and a link that opens it. The recipient is a user who can already view that kind of record.",
		Category:    notificationCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newRecordShareEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "RecipientFirstName",
				Type:        templatecatalog.VariableString,
				Description: "The teammate's first name.",
			},
			{
				Path:        "SharedByName",
				Type:        templatecatalog.VariableString,
				Description: "Who shared the record. Name them: a share from nobody gives the reader no reason to open it.",
			},
			{
				Path:        "RecordTitle",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "What was shared, as a heading names it: the kind of record and its label, such as \"invoice INV-2026-1042\"; it is written to sit inside a sentence.",
			},
			{
				Path:        "RecordSubtitle",
				Type:        templatecatalog.VariableString,
				Description: "What the record says beside its name, such as the customer an invoice bills.",
			},
			{
				Path:        "RecordNoun",
				Type:        templatecatalog.VariableString,
				Description: "The kind of record in a word, such as \"invoice\" or \"shipment\".",
			},
			{
				Path:        "Note",
				Type:        templatecatalog.VariableString,
				Description: "What the sharer wrote, if anything. Print it: it is the reason the record was shared.",
			},
			{
				Path:        "RecordURL",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The link that opens what was shared, where the sharer was looking. Required: without it the message has no purpose.",
			},
			templatecatalog.CompanyNameVariable(),
		},
	})
}
