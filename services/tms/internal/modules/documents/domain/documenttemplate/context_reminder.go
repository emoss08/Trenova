package documenttemplate

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
		CustomerName:      sampleCustomerName,
		CustomerCode:      sampleCustomerCode,
		CompanyName:       sampleCompanyName,
		ReminderNumber:    2,
		IsFinalReminder:   false,
		CurrencyCode:      sampleCurrency,
		TotalOverdue:      sampleAmountOverdue,
		OldestDaysPastDue: 21,
		Invoices: []PaymentReminderInvoice{
			{
				InvoiceNumber: sampleInvoiceNo,
				InvoiceDate:   sampleDateJul15,
				DueDate:       sampleDateAug14,
				DaysPastDue:   21,
				OpenAmount:    sampleTotalAmount,
			},
			{
				InvoiceNumber: "INV-2026-1077",
				InvoiceDate:   "2026-07-24",
				DueDate:       "2026-08-23",
				DaysPastDue:   12,
				OpenAmount:    sampleAmountPartial,
			},
		},
		RemittanceInstructions: sampleRemittanceText,
		Note:                   "",
	}
}

func (r *Registry) registerPaymentReminderKinds() {
	_ = r.Register(&KindDefinition{
		Kind:           KindPaymentReminderEmail,
		DisplayName:    "Payment Reminder",
		Description:    "Reminds a customer that invoices are past due, listing each one with what is still open. Sent on the days past due billing control schedules, or by hand from collections.",
		Category:       billingCategory,
		Channels:       []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		CustomerScoped: true,
		sampleFactory:  newPaymentReminderEmailSampleContext,
		Variables: []VariableDefinition{
			customerNameVariable(false, "The customer being reminded."),
			statementCustomerCodeVariable(),
			companyNameVariable(),
			{
				Path:        "ReminderNumber",
				Type:        VariableInt,
				Description: "Which scheduled reminder this is, counting from 1. Zero when someone sent it by hand.",
			},
			{
				Path:        "IsFinalReminder",
				Type:        VariableBool,
				Description: "True on the last reminder the schedule sends. Say what happens next when it is.",
			},
			{
				Path:        "Invoices",
				Type:        VariableCollection,
				Required:    true,
				Description: "Every overdue invoice, oldest first. Required: a reminder that names no invoice cannot be acted on.",
				Fields: []VariableDefinition{
					invoiceNumberVariable(false, "The invoice number."),
					{
						Path:        invoiceDatePath,
						Type:        VariableDate,
						Description: "When the invoice was issued.",
					},
					{Path: dueDatePath, Type: VariableDate, Description: "When payment was due."},
					{
						Path:        "DaysPastDue",
						Type:        VariableInt,
						Description: "Whole days since the due date.",
					},
					{
						Path:        "OpenAmount",
						Type:        VariableMoney,
						Description: "What is still unpaid on the invoice, with its currency.",
					},
				},
			},
			{
				Path:        "TotalOverdue",
				Type:        VariableMoney,
				Required:    true,
				Description: "The overdue total across the listed invoices. Required: the customer needs the figure to pay.",
			},
			{
				Path:        currencyCodePath,
				Type:        VariableString,
				Description: "The currency the invoices are billed in.",
			},
			{
				Path:        "OldestDaysPastDue",
				Type:        VariableInt,
				Description: "Days past due of the oldest invoice listed.",
			},
			{
				Path:        remittanceInstructionsPath,
				Type:        VariableString,
				Description: "How to pay, from billing control's default invoice terms.",
			},
			{
				Path:        notePath,
				Type:        VariableString,
				Description: "What the sender wrote, when the reminder was sent by hand. Print it: it is why this one went out.",
			},
		},
	})
}
