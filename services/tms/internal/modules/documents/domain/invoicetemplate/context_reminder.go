package invoicetemplate

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

// PaymentReminderEmailContext is a reminder to a customer that invoices are
// past due. One reminder covers every overdue invoice the customer has, so a
// customer with five late invoices receives one message rather than five.
type PaymentReminderEmailContext struct {
	CustomerName string
	CustomerCode string
	CompanyName  string

	// ReminderNumber is which scheduled reminder this is, counting from one;
	// zero for a reminder a person sent by hand.
	ReminderNumber  int
	IsFinalReminder bool

	Invoices          []PaymentReminderInvoice
	TotalOverdue      string
	CurrencyCode      string
	OldestDaysPastDue int

	RemittanceInstructions string
	// Note is what the person sending a reminder by hand wrote.
	Note string
}

// PaymentReminderInvoice is one overdue invoice in a reminder.
type PaymentReminderInvoice struct {
	InvoiceNumber string
	InvoiceDate   string
	DueDate       string
	DaysPastDue   int
	OpenAmount    string
}

func newPaymentReminderEmailSampleContext() any {
	return PaymentReminderEmailContext{
		CustomerName:      templatecatalog.SampleCustomerName,
		CustomerCode:      templatecatalog.SampleCustomerCode,
		CompanyName:       templatecatalog.SampleCompanyName,
		ReminderNumber:    2,
		IsFinalReminder:   false,
		CurrencyCode:      templatecatalog.SampleCurrency,
		TotalOverdue:      templatecatalog.SampleAmountOverdue,
		OldestDaysPastDue: 21,
		Invoices: []PaymentReminderInvoice{
			{
				InvoiceNumber: templatecatalog.SampleInvoiceNo,
				InvoiceDate:   templatecatalog.SampleDateJul15,
				DueDate:       templatecatalog.SampleDateAug14,
				DaysPastDue:   21,
				OpenAmount:    templatecatalog.SampleTotalAmount,
			},
			{
				InvoiceNumber: "INV-2026-1077",
				InvoiceDate:   "2026-07-24",
				DueDate:       "2026-08-23",
				DaysPastDue:   12,
				OpenAmount:    templatecatalog.SampleAmountPartial,
			},
		},
		RemittanceInstructions: templatecatalog.SampleRemittanceText,
		Note:                   "",
	}
}

func RegisterPaymentReminderKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindPaymentReminderEmail,
		DisplayName: "Payment Reminder",
		Description: "Reminds a customer that invoices are past due, listing each one with what is still open. Sent on the days past due billing control schedules, or by hand from collections.",
		Category:    templatecatalog.BillingCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		CustomerScoped: true,
		SampleFactory:  newPaymentReminderEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			templatecatalog.CustomerNameVariable(false, "The customer being reminded."),
			templatecatalog.StatementCustomerCodeVariable(),
			templatecatalog.CompanyNameVariable(),
			{
				Path:        "ReminderNumber",
				Type:        templatecatalog.VariableInt,
				Description: "Which scheduled reminder this is, counting from 1. Zero when someone sent it by hand.",
			},
			{
				Path:        "IsFinalReminder",
				Type:        templatecatalog.VariableBool,
				Description: "True on the last reminder the schedule sends. Say what happens next when it is.",
			},
			{
				Path:        "Invoices",
				Type:        templatecatalog.VariableCollection,
				Required:    true,
				Description: "Every overdue invoice, oldest first. Required: a reminder that names no invoice cannot be acted on.",
				Fields: []templatecatalog.VariableDefinition{
					templatecatalog.InvoiceNumberVariable(false, "The invoice number."),
					{
						Path:        templatecatalog.InvoiceDatePath,
						Type:        templatecatalog.VariableDate,
						Description: templatecatalog.InvoiceIssuedDescription,
					},
					{
						Path:        templatecatalog.DueDatePath,
						Type:        templatecatalog.VariableDate,
						Description: "When payment was due.",
					},
					{
						Path:        "DaysPastDue",
						Type:        templatecatalog.VariableInt,
						Description: "Whole days since the due date.",
					},
					{
						Path:        "OpenAmount",
						Type:        templatecatalog.VariableMoney,
						Description: "What is still unpaid on the invoice, with its currency.",
					},
				},
			},
			{
				Path:        "TotalOverdue",
				Type:        templatecatalog.VariableMoney,
				Required:    true,
				Description: "The overdue total across the listed invoices. Required: the customer needs the figure to pay.",
			},
			{
				Path:        templatecatalog.CurrencyCodePath,
				Type:        templatecatalog.VariableString,
				Description: "The currency the invoices are billed in.",
			},
			{
				Path:        "OldestDaysPastDue",
				Type:        templatecatalog.VariableInt,
				Description: "Days past due of the oldest invoice listed.",
			},
			{
				Path:        templatecatalog.RemittanceInstructionsPath,
				Type:        templatecatalog.VariableString,
				Description: "How to pay, from billing control's default invoice terms.",
			},
			{
				Path:        templatecatalog.NotePath,
				Type:        templatecatalog.VariableString,
				Description: "What the sender wrote, when the reminder was sent by hand. Print it: it is why this one went out.",
			},
		},
	})
}
