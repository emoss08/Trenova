package documenttemplate

import "html/template"

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

	Organization AddressBlock
	Customer     AddressBlock
	RemitTo      AddressBlock

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
	billingCategory            = "Billing"
	customerCodePath           = "CustomerCode"
	notePath                   = "Note"
	currencyCodePath           = "CurrencyCode"
	customerPath               = "Customer"
	descriptionPath            = "Description"
	invoiceDatePath            = "InvoiceDate"
	dueDatePath                = "DueDate"
	remittanceInstructionsPath = "RemittanceInstructions"
	sampleCustomerCode         = "HALSTEAD"
	sampleRemittanceText       = "Remit by ACH to routing 111000614, account 8842091."
	sampleAmountPartial        = "USD 1,763.62"
	sampleAmountOverdue        = "USD 4,102.36"
	sampleAmountZero           = "USD 0.00"
	sampleDateJul15            = "2026-07-15"
	sampleDateAug01            = "2026-08-01"
	sampleDateAug14            = "2026-08-14"
)

func newCustomerStatementSampleContext() any {
	return CustomerStatementContext{
		StatementDate: "2026-08-31",
		PeriodStart:   sampleDateAug01,
		CurrencyCode:  sampleCurrency,
		CustomerCode:  sampleCustomerCode,
		Organization: AddressBlock{
			Name:  sampleCompanyName,
			Lines: []string{"2200 Commerce St", "Dallas, TX 75201", "United States"},
		},
		Customer: AddressBlock{
			Name:  sampleCustomerName,
			Lines: []string{"410 Market Ave", "Fort Worth, TX 76102", "United States"},
		},
		RemitTo: AddressBlock{
			Name: sampleCompanyName,
			Lines: []string{
				"PO Box 2290",
				"Dallas, TX 75221",
				sampleRemittanceText,
			},
		},
		LogoDataURI:    "",
		OpeningBalance: sampleTotalAmount,
		TotalCharges:   "USD 3,527.24",
		TotalPayments:  sampleAmountPartial,
		EndingBalance:  sampleAmountOverdue,
		PastDue:        sampleTotalAmount,
		Aging: StatementAging{
			Current:    sampleAmountPartial,
			Days1To30:  sampleTotalAmount,
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
				Payment:     sampleAmountPartial,
				Balance:     "USD 575.12",
			},
			{
				Date:        sampleDateAug14,
				Document:    sampleInvoiceNo,
				Description: "Invoice",
				Charge:      sampleAmountPartial,
				Payment:     "",
				Balance:     sampleTotalAmount,
			},
			{
				Date:        "2026-08-23",
				Document:    "INV-2026-1077",
				Description: "Invoice",
				Charge:      sampleAmountPartial,
				Payment:     "",
				Balance:     sampleAmountOverdue,
			},
		},
		OpenItems: []StatementOpenItem{
			{
				InvoiceNumber: "INV-2026-0988",
				InvoiceDate:   "2026-07-02",
				DueDate:       sampleDateAug01,
				DaysPastDue:   30,
				Total:         sampleTotalAmount,
				Open:          "USD 575.12",
				Disputed:      false,
				AssignedTo:    "Lonestar Freight Capital",
			},
			{
				InvoiceNumber: sampleInvoiceNo,
				InvoiceDate:   sampleDateAug14,
				DueDate:       "2026-09-13",
				DaysPastDue:   0,
				Total:         sampleAmountPartial,
				Open:          sampleAmountPartial,
				Disputed:      false,
			},
		},
		TotalOpen:          sampleAmountOverdue,
		Note:               "",
		InvoiceFooter:      "Thank you for your business.",
		NoticeOfAssignment: "",
	}
}

func newCustomerStatementEmailSampleContext() any {
	return CustomerStatementEmailContext{
		CustomerName:           sampleCustomerName,
		CustomerCode:           sampleCustomerCode,
		CompanyName:            sampleCompanyName,
		StatementDate:          "2026-08-31",
		PeriodStart:            sampleDateAug01,
		CurrencyCode:           sampleCurrency,
		EndingBalance:          sampleAmountOverdue,
		PastDue:                sampleTotalAmount,
		OpenItemCount:          2,
		IsMonthly:              true,
		RemittanceInstructions: sampleRemittanceText,
		Note:                   "",
		AttachmentName:         "Statement-HALSTEAD-2026-08-31.pdf",
	}
}

func statementDateVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "StatementDate",
		Type:        VariableDate,
		Required:    true,
		Description: "The day the statement is drawn up to, in the organization timezone. Required: a balance means nothing without the date it stood on.",
	}
}

func statementPeriodStartVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "PeriodStart",
		Type:        VariableDate,
		Description: "The first day the statement covers. Blank when it runs from the account's first transaction.",
	}
}

func statementEndingBalanceVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "EndingBalance",
		Type:        VariableMoney,
		Required:    true,
		Description: "What the customer owes on the statement date, with currency. Required: it is the figure a statement exists to give.",
	}
}

func statementPastDueVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "PastDue",
		Type:        VariableMoney,
		Description: "The part of the balance that is past its due date, with currency.",
	}
}

func statementCurrencyVariable() VariableDefinition {
	return VariableDefinition{
		Path:        currencyCodePath,
		Type:        VariableString,
		Description: "The currency the customer is billed in.",
	}
}

func statementCustomerCodeVariable() VariableDefinition {
	return VariableDefinition{
		Path:        customerCodePath,
		Type:        VariableString,
		Description: "The customer's code.",
	}
}

func statementNoteVariable() VariableDefinition {
	return VariableDefinition{
		Path:        notePath,
		Type:        VariableString,
		Description: "What the person sending the statement wrote. Print it when present.",
	}
}

//nolint:funlen // A variable catalog is one declaration; splitting it only hides it.
func (r *Registry) registerCustomerStatementKinds() {
	_ = r.Register(&KindDefinition{
		Kind:           KindCustomerStatementPDF,
		DisplayName:    "Customer Statement",
		Description:    "A customer's account statement: opening balance, every invoice, payment and credit in the period, the closing balance, and the invoices still open, aged. Downloaded from the statement page, emailed by hand, or sent each month by billing control.",
		Category:       billingCategory,
		Channels:       []Channel{ChannelPDF},
		Paged:          true,
		CustomerScoped: true,
		sampleFactory:  newCustomerStatementSampleContext,
		Variables: []VariableDefinition{
			statementDateVariable(),
			statementPeriodStartVariable(),
			statementCurrencyVariable(),
			statementCustomerCodeVariable(),
			{
				Path:        "Organization",
				Type:        VariableObject,
				Description: "Your organization, as the statement's issuer.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        customerPath,
				Type:        VariableObject,
				Required:    true,
				Description: "The customer the statement is for. Required: a statement with no addressee cannot be acted on.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        "RemitTo",
				Type:        VariableObject,
				Description: "Where payment should be sent: the factor when the customer's invoices are assigned to one, otherwise your organization with the remittance instructions from billing control.",
				Fields:      addressBlockFields(),
			},
			logoVariable(),
			{
				Path:        "OpeningBalance",
				Type:        VariableMoney,
				Description: "What the customer owed before the period began, with currency.",
			},
			{
				Path:        "TotalCharges",
				Type:        VariableMoney,
				Description: "Invoices and debit memos in the period, with currency.",
			},
			{
				Path:        "TotalPayments",
				Type:        VariableMoney,
				Description: "Payments and credits in the period, with currency.",
			},
			statementEndingBalanceVariable(),
			statementPastDueVariable(),
			{
				Path:        "Aging",
				Type:        VariableObject,
				Description: "What is open on the statement date, by how late it is.",
				Fields: []VariableDefinition{
					{Path: "Current", Type: VariableMoney, Description: "Not yet due."},
					{Path: "Days1To30", Type: VariableMoney, Description: "1 to 30 days past due."},
					{
						Path:        "Days31To60",
						Type:        VariableMoney,
						Description: "31 to 60 days past due.",
					},
					{
						Path:        "Days61To90",
						Type:        VariableMoney,
						Description: "61 to 90 days past due.",
					},
					{
						Path:        "Over90",
						Type:        VariableMoney,
						Description: "More than 90 days past due.",
					},
				},
			},
			{
				Path:        "Transactions",
				Type:        VariableCollection,
				Description: "Every document that moved the balance in the period, oldest first, with the running balance after each.",
				Fields: []VariableDefinition{
					{Path: "Date", Type: VariableDate, Description: "When it happened."},
					{
						Path:        "Document",
						Type:        VariableString,
						Description: "The invoice number or payment reference.",
					},
					{
						Path:        descriptionPath,
						Type:        VariableString,
						Description: "What it was: Invoice, Payment, Credit memo.",
					},
					{
						Path:        "Charge",
						Type:        VariableMoney,
						Description: "What it added to the balance. Blank for a payment or credit.",
					},
					{
						Path:        "Payment",
						Type:        VariableMoney,
						Description: "What it took off the balance. Blank for a charge.",
					},
					{Path: "Balance", Type: VariableMoney, Description: "The balance after it."},
				},
			},
			{
				Path:        "OpenItems",
				Type:        VariableCollection,
				Description: "Every invoice with money still owed on the statement date, oldest due date first.",
				Fields: []VariableDefinition{
					invoiceNumberVariable(false, "The invoice number."),
					{
						Path:        invoiceDatePath,
						Type:        VariableDate,
						Description: "When the invoice was issued.",
					},
					{Path: dueDatePath, Type: VariableDate, Description: "When payment is due."},
					{
						Path:        "DaysPastDue",
						Type:        VariableInt,
						Description: "Whole days past the due date; zero when it is not yet due.",
					},
					{Path: "Total", Type: VariableMoney, Description: "The invoice total."},
					{Path: "Open", Type: VariableMoney, Description: "What is still unpaid on it."},
					{
						Path:        "Disputed",
						Type:        VariableBool,
						Description: "True while the customer disputes the invoice.",
					},
					{
						Path:        "AssignedTo",
						Type:        VariableString,
						Description: "Who this invoice is paid to when that differs from RemitTo: the factor it was assigned to when it posted, or your organization for an invoice posted before the customer was factored. Blank when it is paid to RemitTo; print it whenever it is set.",
					},
				},
			},
			{
				Path:        "TotalOpen",
				Type:        VariableMoney,
				Description: "The open amount across every listed invoice, with currency.",
			},
			statementNoteVariable(),
			{
				Path:        "InvoiceFooter",
				Type:        VariableString,
				Description: "The footer text from the organization's billing settings.",
			},
			noticeOfAssignmentVariable(),
		},
	})

	_ = r.Register(&KindDefinition{
		Kind:           KindCustomerStatementEmail,
		DisplayName:    "Customer Statement Email",
		Description:    "The message a customer statement is attached to.",
		Category:       billingCategory,
		Channels:       []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		CustomerScoped: true,
		sampleFactory:  newCustomerStatementEmailSampleContext,
		Variables: []VariableDefinition{
			customerNameVariable(false, "The customer the statement is for."),
			statementCustomerCodeVariable(),
			companyNameVariable(),
			statementDateVariable(),
			statementPeriodStartVariable(),
			statementCurrencyVariable(),
			statementEndingBalanceVariable(),
			statementPastDueVariable(),
			{
				Path:        "OpenItemCount",
				Type:        VariableInt,
				Description: "How many invoices are still open.",
			},
			{
				Path:        "IsMonthly",
				Type:        VariableBool,
				Description: "True for the statement sent each month, false for one a person sent.",
			},
			{
				Path:        remittanceInstructionsPath,
				Type:        VariableString,
				Description: "How to pay, from billing control's default invoice terms.",
			},
			statementNoteVariable(),
			{
				Path:        "AttachmentName",
				Type:        VariableString,
				Description: "The file name of the attached statement.",
			},
		},
	})
}
