package notificationtemplate

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

type NotificationEmailContext struct {
	RecipientFirstName string
	Title              string
	Message            string
	Link               string
	SettingsURL        string
}

type NotificationDigestEmailContext struct {
	RecipientFirstName string
	ItemCount          int
	Items              []NotificationDigestItem
	BellURL            string
	SettingsURL        string
}

type NotificationDigestItem struct {
	Title   string
	Message string
	Link    string
}

const (
	sampleRecipientFirstName = "Dana"
	recipientFirstNamePath   = "RecipientFirstName"
	sampleSettingsURL        = "https://app.example.com/settings/notifications"
	sampleBellURL            = "https://app.example.com/"
)

func newNotificationEmailSampleContext() any {
	return NotificationEmailContext{
		RecipientFirstName: sampleRecipientFirstName,
		Title:              "Marcus Bell mentioned you on shipment " + templatecatalog.SampleProNumber,
		Message:            "Can you confirm the delivery appointment with the receiver before 3 PM?",
		Link:               "https://app.example.com/shipments/management?item=shp_01J2Y8Z3Q4R5S6T7V8W9X0Y1Z2",
		SettingsURL:        sampleSettingsURL,
	}
}

func newNotificationDigestEmailSampleContext() any {
	return NotificationDigestEmailContext{
		RecipientFirstName: sampleRecipientFirstName,
		ItemCount:          3,
		Items: []NotificationDigestItem{
			{
				Title:   "Report Revenue by Customer is ready",
				Message: "42 rows, delivered to your inbox.",
				Link:    "https://app.example.com/reports?item=rpt_01J2Y8Z3Q4R5S6T7V8W9X0Y1Z2",
			},
			{
				Title:   "Marcus Bell replied to your comment",
				Message: "The receiver moved the appointment to 9 AM.",
				Link:    "https://app.example.com/shipments/management?item=shp_01J2Y8Z3Q4R5S6T7V8W9X0Y1Z3",
			},
			{
				Title:   "Invoice " + templatecatalog.SampleInvoiceNo + " was shared with you",
				Message: templatecatalog.SampleCustomerName,
				Link:    "https://app.example.com/billing/invoices?item=inv_01J2Y8Z3Q4R5S6T7V8W9X0Y1Z2",
			},
		},
		BellURL:     sampleBellURL,
		SettingsURL: sampleSettingsURL,
	}
}

func notificationEmailChannels() []documenttemplate.Channel {
	return []documenttemplate.Channel{
		documenttemplate.ChannelSubject,
		documenttemplate.ChannelEmailHTML,
		documenttemplate.ChannelEmailText,
	}
}

func RegisterNotificationEmailKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:          documenttemplate.KindNotificationEmail,
		DisplayName:   "Notification Email",
		Description:   "Sends one notification by email to someone who asked to hear about it right away. Its title and message are the notification's own.",
		Category:      notificationCategory,
		Channels:      notificationEmailChannels(),
		SampleFactory: newNotificationEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        recipientFirstNamePath,
				Type:        templatecatalog.VariableString,
				Description: "The recipient's first name.",
			},
			{
				Path:        "Title",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The notification's title, which is also the subject.",
			},
			{
				Path:        "Message",
				Type:        templatecatalog.VariableString,
				Description: "The notification's message. Empty when the title says everything.",
			},
			{
				Path:        "Link",
				Type:        templatecatalog.VariableString,
				Description: "The link that opens what the notification is about. Empty when it is about nothing that can be opened.",
			},
			{
				Path:        "SettingsURL",
				Type:        templatecatalog.VariableString,
				Description: "The link to the recipient's notification settings, where they choose what reaches them by email.",
			},
		},
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:          documenttemplate.KindNotificationDigestEmail,
		DisplayName:   "Notification Digest",
		Description:   "Bundles a day of notifications into one email for someone who asked for a digest instead of an email per notification.",
		Category:      notificationCategory,
		Channels:      notificationEmailChannels(),
		SampleFactory: newNotificationDigestEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        recipientFirstNamePath,
				Type:        templatecatalog.VariableString,
				Description: "The recipient's first name.",
			},
			{
				Path:        "ItemCount",
				Type:        templatecatalog.VariableInt,
				Required:    true,
				Description: "How many notifications the digest holds.",
			},
			{
				Path:        "Items",
				Type:        templatecatalog.VariableCollection,
				Required:    true,
				Description: "The notifications, newest first.",
				Fields: []templatecatalog.VariableDefinition{
					{
						Path:        "Title",
						Type:        templatecatalog.VariableString,
						Description: "The notification's title.",
					},
					{
						Path:        "Message",
						Type:        templatecatalog.VariableString,
						Description: "The notification's message, if it has one.",
					},
					{
						Path:        "Link",
						Type:        templatecatalog.VariableString,
						Description: "The link that opens what it is about, if anything.",
					},
				},
			},
			{
				Path:        "BellURL",
				Type:        templatecatalog.VariableString,
				Description: "The link that opens every notification in the app.",
			},
			{
				Path:        "SettingsURL",
				Type:        templatecatalog.VariableString,
				Description: "The link to the recipient's notification settings, where they change the digest.",
			},
		},
	})
}
