package statementtemplate

import (
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

// CustomerStatementContext is a customer's account statement: the balance it
// opened the period with, every invoice, payment and credit since, the balance
// it closed with, and the invoices still open, aged.
type CustomerStatementContext struct {
	StatementDate string
	// PeriodStart is blank on a statement that runs from the account's first
	// transaction.
	PeriodStart  string
	CurrencyCode string
	CustomerCode string

	Organization templatecatalog.AddressBlock
	Customer     templatecatalog.AddressBlock
	RemitTo      templatecatalog.AddressBlock

	// LogoDataURI is typed as trusted for the reason InvoiceContext gives: it
	// is produced by the asset inliner from bytes it has already decoded.
	LogoDataURI template.URL

	OpeningBalance string
	TotalCharges   string
	TotalPayments  string
	EndingBalance  string
	PastDue        string

	Aging        StatementAging
	Transactions []StatementTransaction
	OpenItems    []StatementOpenItem
	TotalOpen    string

	Note          string
	InvoiceFooter string
	// NoticeOfAssignment is set when the customer's invoices are assigned to
	// a factor; RemitTo is then the factor.
	NoticeOfAssignment string
}

// StatementAging is what is open, by how late it is.
type StatementAging struct {
	Current    string
	Days1To30  string
	Days31To60 string
	Days61To90 string
	Over90     string
}

// StatementTransaction is one document that moved the balance.
type StatementTransaction struct {
	Date        string
	Document    string
	Description string
	Charge      string
	Payment     string
	Balance     string
}

// StatementOpenItem is one invoice with money still owed on it.
type StatementOpenItem struct {
	InvoiceNumber string
	InvoiceDate   string
	DueDate       string
	DaysPastDue   int
	Total         string
	Open          string
	Disputed      bool
	// AssignedTo is who this invoice is paid to when that is not the
	// statement's RemitTo: the factor it was assigned to when it posted, or
	// your organization for an invoice posted before the customer was factored.
	AssignedTo string
}

// CustomerStatementEmailContext is the message a statement is attached to.
type CustomerStatementEmailContext struct {
	CustomerName string
	CustomerCode string
	CompanyName  string

	StatementDate string
	PeriodStart   string
	CurrencyCode  string
	EndingBalance string
	PastDue       string
	OpenItemCount int

	// IsMonthly is true for the statement billing control sends each month,
	// false for one a person sent.
	IsMonthly bool

	RemittanceInstructions string
	Note                   string
	AttachmentName         string
}

const (
	descriptionPath  = "Description"
	sampleAmountZero = "USD 0.00"
	sampleDateAug01  = "2026-08-01"
)

func newCustomerStatementSampleContext() any {
	return CustomerStatementContext{
		StatementDate: "2026-08-31",
		PeriodStart:   sampleDateAug01,
		CurrencyCode:  templatecatalog.SampleCurrency,
		CustomerCode:  templatecatalog.SampleCustomerCode,
		Organization: templatecatalog.AddressBlock{
			Name:  templatecatalog.SampleCompanyName,
			Lines: []string{"2200 Commerce St", "Dallas, TX 75201", "United States"},
		},
		Customer: templatecatalog.AddressBlock{
			Name:  templatecatalog.SampleCustomerName,
			Lines: []string{"410 Market Ave", "Fort Worth, TX 76102", "United States"},
		},
		RemitTo: templatecatalog.AddressBlock{
			Name: templatecatalog.SampleCompanyName,
			Lines: []string{
				"PO Box 2290",
				"Dallas, TX 75221",
				templatecatalog.SampleRemittanceText,
			},
		},
		LogoDataURI:    "",
		OpeningBalance: templatecatalog.SampleTotalAmount,
		TotalCharges:   "USD 3,527.24",
		TotalPayments:  templatecatalog.SampleAmountPartial,
		EndingBalance:  templatecatalog.SampleAmountOverdue,
		PastDue:        templatecatalog.SampleTotalAmount,
		Aging: StatementAging{
			Current:    templatecatalog.SampleAmountPartial,
			Days1To30:  templatecatalog.SampleTotalAmount,
			Days31To60: sampleAmountZero,
			Days61To90: sampleAmountZero,
			Over90:     sampleAmountZero,
		},
		Transactions: []StatementTransaction{
			{
				Date:        "2026-08-04",
				Document:    "PMT-88213",
				Description: "Payment",
				Charge:      "",
				Payment:     templatecatalog.SampleAmountPartial,
				Balance:     "USD 575.12",
			},
			{
				Date:        templatecatalog.SampleDateAug14,
				Document:    templatecatalog.SampleInvoiceNo,
				Description: "Invoice",
				Charge:      templatecatalog.SampleAmountPartial,
				Payment:     "",
				Balance:     templatecatalog.SampleTotalAmount,
			},
			{
				Date:        "2026-08-23",
				Document:    "INV-2026-1077",
				Description: "Invoice",
				Charge:      templatecatalog.SampleAmountPartial,
				Payment:     "",
				Balance:     templatecatalog.SampleAmountOverdue,
			},
		},
		OpenItems: []StatementOpenItem{
			{
				InvoiceNumber: "INV-2026-0988",
				InvoiceDate:   "2026-07-02",
				DueDate:       sampleDateAug01,
				DaysPastDue:   30,
				Total:         templatecatalog.SampleTotalAmount,
				Open:          "USD 575.12",
				Disputed:      false,
				AssignedTo:    "Lonestar Freight Capital",
			},
			{
				InvoiceNumber: templatecatalog.SampleInvoiceNo,
				InvoiceDate:   templatecatalog.SampleDateAug14,
				DueDate:       "2026-09-13",
				DaysPastDue:   0,
				Total:         templatecatalog.SampleAmountPartial,
				Open:          templatecatalog.SampleAmountPartial,
				Disputed:      false,
			},
		},
		TotalOpen:          templatecatalog.SampleAmountOverdue,
		Note:               "",
		InvoiceFooter:      "Thank you for your business.",
		NoticeOfAssignment: "",
	}
}

func newCustomerStatementEmailSampleContext() any {
	return CustomerStatementEmailContext{
		CustomerName:           templatecatalog.SampleCustomerName,
		CustomerCode:           templatecatalog.SampleCustomerCode,
		CompanyName:            templatecatalog.SampleCompanyName,
		StatementDate:          "2026-08-31",
		PeriodStart:            sampleDateAug01,
		CurrencyCode:           templatecatalog.SampleCurrency,
		EndingBalance:          templatecatalog.SampleAmountOverdue,
		PastDue:                templatecatalog.SampleTotalAmount,
		OpenItemCount:          2,
		IsMonthly:              true,
		RemittanceInstructions: templatecatalog.SampleRemittanceText,
		Note:                   "",
		AttachmentName:         "Statement-HALSTEAD-2026-08-31.pdf",
	}
}

func statementDateVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "StatementDate",
		Type:        templatecatalog.VariableDate,
		Required:    true,
		Description: "The day the statement is drawn up to, in the organization timezone. Required: a balance means nothing without the date it stood on.",
	}
}

func statementPeriodStartVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "PeriodStart",
		Type:        templatecatalog.VariableDate,
		Description: "The first day the statement covers. Blank when it runs from the account's first transaction.",
	}
}

func statementEndingBalanceVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "EndingBalance",
		Type:        templatecatalog.VariableMoney,
		Required:    true,
		Description: "What the customer owes on the statement date, with currency. Required: it is the figure a statement exists to give.",
	}
}

func statementPastDueVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "PastDue",
		Type:        templatecatalog.VariableMoney,
		Description: "The part of the balance that is past its due date, with currency.",
	}
}

func statementCurrencyVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        templatecatalog.CurrencyCodePath,
		Type:        templatecatalog.VariableString,
		Description: "The currency the customer is billed in.",
	}
}

func statementNoteVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        templatecatalog.NotePath,
		Type:        templatecatalog.VariableString,
		Description: "What the person sending the statement wrote. Print it when present.",
	}
}

//nolint:funlen // A variable catalog is one declaration; splitting it only hides it.
func RegisterCustomerStatementKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:           documenttemplate.KindCustomerStatementPDF,
		DisplayName:    "Customer Statement",
		Description:    "A customer's account statement: opening balance, every invoice, payment and credit in the period, the closing balance, and the invoices still open, aged. Downloaded from the statement page, emailed by hand, or sent each month by billing control.",
		Category:       templatecatalog.BillingCategory,
		Channels:       []documenttemplate.Channel{documenttemplate.ChannelPDF},
		Paged:          true,
		CustomerScoped: true,
		SampleFactory:  newCustomerStatementSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			statementDateVariable(),
			statementPeriodStartVariable(),
			statementCurrencyVariable(),
			templatecatalog.StatementCustomerCodeVariable(),
			{
				Path:        "Organization",
				Type:        templatecatalog.VariableObject,
				Description: "Your organization, as the statement's issuer.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        templatecatalog.CustomerPath,
				Type:        templatecatalog.VariableObject,
				Required:    true,
				Description: "The customer the statement is for. Required: a statement with no addressee cannot be acted on.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        "RemitTo",
				Type:        templatecatalog.VariableObject,
				Description: "Where payment should be sent: the factor when the customer's invoices are assigned to one, otherwise your organization with the remittance instructions from billing control.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			templatecatalog.LogoVariable(),
			{
				Path:        "OpeningBalance",
				Type:        templatecatalog.VariableMoney,
				Description: "What the customer owed before the period began, with currency.",
			},
			{
				Path:        "TotalCharges",
				Type:        templatecatalog.VariableMoney,
				Description: "Invoices and debit memos in the period, with currency.",
			},
			{
				Path:        "TotalPayments",
				Type:        templatecatalog.VariableMoney,
				Description: "Payments and credits in the period, with currency.",
			},
			statementEndingBalanceVariable(),
			statementPastDueVariable(),
			{
				Path:        "Aging",
				Type:        templatecatalog.VariableObject,
				Description: "What is open on the statement date, by how late it is.",
				Fields: []templatecatalog.VariableDefinition{
					{
						Path:        "Current",
						Type:        templatecatalog.VariableMoney,
						Description: "Not yet due.",
					},
					{
						Path:        "Days1To30",
						Type:        templatecatalog.VariableMoney,
						Description: "1 to 30 days past due.",
					},
					{
						Path:        "Days31To60",
						Type:        templatecatalog.VariableMoney,
						Description: "31 to 60 days past due.",
					},
					{
						Path:        "Days61To90",
						Type:        templatecatalog.VariableMoney,
						Description: "61 to 90 days past due.",
					},
					{
						Path:        "Over90",
						Type:        templatecatalog.VariableMoney,
						Description: "More than 90 days past due.",
					},
				},
			},
			{
				Path:        "Transactions",
				Type:        templatecatalog.VariableCollection,
				Description: "Every document that moved the balance in the period, oldest first, with the running balance after each.",
				Fields: []templatecatalog.VariableDefinition{
					{
						Path:        "Date",
						Type:        templatecatalog.VariableDate,
						Description: "When it happened.",
					},
					{
						Path:        "Document",
						Type:        templatecatalog.VariableString,
						Description: "The invoice number or payment reference.",
					},
					{
						Path:        descriptionPath,
						Type:        templatecatalog.VariableString,
						Description: "What it was: Invoice, Payment, Credit memo.",
					},
					{
						Path:        "Charge",
						Type:        templatecatalog.VariableMoney,
						Description: "What it added to the balance. Blank for a payment or credit.",
					},
					{
						Path:        "Payment",
						Type:        templatecatalog.VariableMoney,
						Description: "What it took off the balance. Blank for a charge.",
					},
					{
						Path:        "Balance",
						Type:        templatecatalog.VariableMoney,
						Description: "The balance after it.",
					},
				},
			},
			{
				Path:        "OpenItems",
				Type:        templatecatalog.VariableCollection,
				Description: "Every invoice with money still owed on the statement date, oldest due date first.",
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
						Description: "When payment is due.",
					},
					{
						Path:        "DaysPastDue",
						Type:        templatecatalog.VariableInt,
						Description: "Whole days past the due date; zero when it is not yet due.",
					},
					{
						Path:        "Total",
						Type:        templatecatalog.VariableMoney,
						Description: "The invoice total.",
					},
					{
						Path:        "Open",
						Type:        templatecatalog.VariableMoney,
						Description: "What is still unpaid on it.",
					},
					{
						Path:        "Disputed",
						Type:        templatecatalog.VariableBool,
						Description: "True while the customer disputes the invoice.",
					},
					{
						Path:        "AssignedTo",
						Type:        templatecatalog.VariableString,
						Description: "Who this invoice is paid to when that differs from RemitTo: the factor it was assigned to when it posted, or your organization for an invoice posted before the customer was factored. Blank when it is paid to RemitTo; print it whenever it is set.",
					},
				},
			},
			{
				Path:        "TotalOpen",
				Type:        templatecatalog.VariableMoney,
				Description: "The open amount across every listed invoice, with currency.",
			},
			statementNoteVariable(),
			{
				Path:        "InvoiceFooter",
				Type:        templatecatalog.VariableString,
				Description: "The footer text from the organization's billing settings.",
			},
			templatecatalog.NoticeOfAssignmentVariable(),
		},
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindCustomerStatementEmail,
		DisplayName: "Customer Statement Email",
		Description: "The message a customer statement is attached to.",
		Category:    templatecatalog.BillingCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		CustomerScoped: true,
		SampleFactory:  newCustomerStatementEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			templatecatalog.CustomerNameVariable(false, "The customer the statement is for."),
			templatecatalog.StatementCustomerCodeVariable(),
			templatecatalog.CompanyNameVariable(),
			statementDateVariable(),
			statementPeriodStartVariable(),
			statementCurrencyVariable(),
			statementEndingBalanceVariable(),
			statementPastDueVariable(),
			{
				Path:        "OpenItemCount",
				Type:        templatecatalog.VariableInt,
				Description: "How many invoices are still open.",
			},
			{
				Path:        "IsMonthly",
				Type:        templatecatalog.VariableBool,
				Description: "True for the statement sent each month, false for one a person sent.",
			},
			{
				Path:        templatecatalog.RemittanceInstructionsPath,
				Type:        templatecatalog.VariableString,
				Description: "How to pay, from billing control's default invoice terms.",
			},
			statementNoteVariable(),
			{
				Path:        "AttachmentName",
				Type:        templatecatalog.VariableString,
				Description: "The file name of the attached statement.",
			},
		},
	})
}
