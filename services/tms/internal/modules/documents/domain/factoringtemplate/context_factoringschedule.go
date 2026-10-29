package factoringtemplate

import (
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

// FactoringScheduleContext is a schedule of accounts: the list of invoices an
// organization sells to its factor in one delivery, with what each should bring
// in under the agreement in force.
type FactoringScheduleContext struct {
	Number       string
	ScheduleDate string
	CurrencyCode string
	Organization templatecatalog.AddressBlock
	Factor       templatecatalog.AddressBlock
	// LogoDataURI is typed as trusted for the reason InvoiceContext gives: it
	// is produced by the asset inliner from bytes it has already decoded.
	LogoDataURI template.URL

	AdvanceRate string
	ReserveRate string
	Recourse    string

	Items []FactoringScheduleItem

	InvoiceCount    int
	TotalFace       string
	TotalAdvance    string
	TotalFee        string
	TotalReserve    string
	ExpectedPayment string
}

// FactoringScheduleItem is one invoice on a schedule of accounts.
type FactoringScheduleItem struct {
	InvoiceNumber string
	InvoiceDate   string
	DueDate       string
	Debtor        string
	DebtorCode    string
	DebtorAddress string
	Reference     string
	Amount        string
	Advance       string
	Fee           string
	Reserve       string
}

// FactoringScheduleEmailContext is the message a schedule of accounts is sent in.
type FactoringScheduleEmailContext struct {
	FactorName      string
	CompanyName     string
	Number          string
	ScheduleDate    string
	InvoiceCount    int
	TotalFace       string
	ExpectedPayment string
	Attachments     []string
	Part            int
	Parts           int
	ReplyTo         string
}

const (
	scheduleNumberPath    = "Number"
	sampleScheduleNo      = "FS26100004"
	sampleScheduleDate    = "2026-10-14"
	sampleScheduleAdvance = "$3,267.00"
)

func newFactoringScheduleSampleContext() any {
	return FactoringScheduleContext{
		Number:       sampleScheduleNo,
		ScheduleDate: sampleScheduleDate,
		CurrencyCode: templatecatalog.SampleCurrency,
		Organization: sampleCarrierOrganization(),
		Factor: templatecatalog.AddressBlock{
			Name:  sampleFactorName,
			Lines: []string{"1700 Pacific Ave, Suite 2100", sampleDallasCityLine, sampleCountry},
		},
		LogoDataURI: templatecatalog.SampleLogoDataURI,
		AdvanceRate: "90%",
		ReserveRate: "10%",
		Recourse:    "With recourse",
		Items: []FactoringScheduleItem{
			{
				InvoiceNumber: "INV-104221",
				InvoiceDate:   "2026-10-10",
				DueDate:       "2026-11-09",
				Debtor:        templatecatalog.SampleCustomerName,
				DebtorCode:    "HALSTEAD",
				DebtorAddress: "410 Market Ave, Fort Worth, TX 76102",
				Reference:     "PRO 5501829",
				Amount:        "$2,450.00",
				Advance:       "$2,205.00",
				Fee:           "$73.50",
				Reserve:       "$245.00",
			},
			{
				InvoiceNumber: "INV-104227",
				InvoiceDate:   "2026-10-12",
				DueDate:       "2026-11-11",
				Debtor:        "Brazos Building Supply",
				DebtorCode:    "BRAZOS",
				DebtorAddress: "88 Industrial Blvd, Waco, TX 76701",
				Reference:     "PRO 5501844",
				Amount:        "$1,180.00",
				Advance:       "$1,062.00",
				Fee:           "$35.40",
				Reserve:       "$118.00",
			},
		},
		InvoiceCount:    2,
		TotalFace:       "$3,630.00",
		TotalAdvance:    sampleScheduleAdvance,
		TotalFee:        "$108.90",
		TotalReserve:    "$363.00",
		ExpectedPayment: sampleScheduleAdvance,
	}
}

func newFactoringScheduleEmailSampleContext() any {
	return FactoringScheduleEmailContext{
		FactorName:      sampleFactorName,
		CompanyName:     templatecatalog.SampleCompanyName,
		Number:          sampleScheduleNo,
		ScheduleDate:    sampleScheduleDate,
		InvoiceCount:    2,
		TotalFace:       "$3,630.00",
		ExpectedPayment: sampleScheduleAdvance,
		Attachments: []string{
			"FS26100004.pdf",
			"FS26100004.csv",
			"INV-104221.pdf",
			"INV-104227.pdf",
		},
		Part:    1,
		Parts:   1,
		ReplyTo: "billing@trenova.example",
	}
}

func scheduleMoney(path, description string) templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        path,
		Type:        templatecatalog.VariableMoney,
		Description: description,
	}
}

//nolint:funlen // A variable catalog is one declaration; splitting it only hides it.
func RegisterFactoringScheduleKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:          documenttemplate.KindFactoringSchedulePDF,
		DisplayName:   "Factoring Schedule of Accounts",
		Description:   "The schedule sent to your factoring company with each batch of invoices you sell to it: every invoice, its debtor and amount, and what the agreement in force says each should bring in.",
		Category:      templatecatalog.BillingCategory,
		Channels:      []documenttemplate.Channel{documenttemplate.ChannelPDF},
		Paged:         true,
		SampleFactory: newFactoringScheduleSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        scheduleNumberPath,
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The schedule's number. Required: the factor files funding against it.",
			},
			{
				Path:        "ScheduleDate",
				Type:        templatecatalog.VariableDate,
				Required:    true,
				Description: "The day the schedule is submitted, in the organization timezone. Required: the factor's fees run from it.",
			},
			{
				Path:        "CurrencyCode",
				Type:        templatecatalog.VariableString,
				Description: "The currency every amount on the schedule is in.",
			},
			{
				Path:        organizationPath,
				Type:        templatecatalog.VariableObject,
				Required:    true,
				Description: "Your organization, as the seller of the invoices. Required: the factor needs to know whose invoices these are.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        "Factor",
				Type:        templatecatalog.VariableObject,
				Description: "The factoring company buying the invoices.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			templatecatalog.LogoVariable(),
			{
				Path:        "AdvanceRate",
				Type:        templatecatalog.VariableString,
				Description: "The share of each invoice the factor advances, as a percentage.",
			},
			{
				Path:        "ReserveRate",
				Type:        templatecatalog.VariableString,
				Description: "The share of each invoice the factor holds back until the customer pays, as a percentage.",
			},
			{
				Path:        "Recourse",
				Type:        templatecatalog.VariableString,
				Description: "Whether the agreement is with or without recourse.",
			},
			{
				Path:        "Items",
				Type:        templatecatalog.VariableCollection,
				Required:    true,
				Description: "Every invoice on the schedule, in submission order. Required: a schedule that lists no invoices assigns nothing.",
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
						Description: "When the customer must pay.",
					},
					{
						Path:        "Debtor",
						Type:        templatecatalog.VariableString,
						Description: "The customer who owes the invoice.",
					},
					{
						Path:        "DebtorCode",
						Type:        templatecatalog.VariableString,
						Description: "The customer's code.",
					},
					{
						Path:        "DebtorAddress",
						Type:        templatecatalog.VariableString,
						Description: "The customer's billing address on one line.",
					},
					{
						Path:        "Reference",
						Type:        templatecatalog.VariableString,
						Description: "The load reference: PRO and BOL numbers.",
					},
					scheduleMoney("Amount", "The invoice amount being sold."),
					scheduleMoney("Advance", "What the factor is expected to advance on it."),
					scheduleMoney("Fee", "The factor's expected fee on it."),
					scheduleMoney("Reserve", "What the factor is expected to hold in reserve."),
				},
			},
			{
				Path:        "InvoiceCount",
				Type:        templatecatalog.VariableInt,
				Description: "How many invoices are on the schedule.",
			},
			{
				Path:        "TotalFace",
				Type:        templatecatalog.VariableMoney,
				Required:    true,
				Description: "The total of every invoice on the schedule. Required: the factor checks the batch against it.",
			},
			scheduleMoney(
				"TotalAdvance",
				"What the factor is expected to advance on the whole schedule.",
			),
			scheduleMoney("TotalFee", "The factor's expected fees on the whole schedule."),
			scheduleMoney("TotalReserve", "What the factor is expected to hold in reserve."),
			scheduleMoney(
				"ExpectedPayment",
				"What you expect the factor to pay you for the schedule.",
			),
		},
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindFactoringScheduleEmail,
		DisplayName: "Factoring Schedule Email",
		Description: "The message a schedule of accounts and its invoice packets are sent to your factoring company in.",
		Category:    templatecatalog.BillingCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newFactoringScheduleEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "FactorName",
				Type:        templatecatalog.VariableString,
				Description: "The factoring company the schedule is sent to.",
			},
			templatecatalog.CompanyNameVariable(),
			{
				Path:        scheduleNumberPath,
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The schedule's number. Required: the factor files the message against it.",
			},
			{
				Path:        "ScheduleDate",
				Type:        templatecatalog.VariableDate,
				Description: "The day the schedule is submitted.",
			},
			{
				Path:        "InvoiceCount",
				Type:        templatecatalog.VariableInt,
				Description: "How many invoices are on the schedule.",
			},
			scheduleMoney("TotalFace", "The total of every invoice on the schedule."),
			scheduleMoney("ExpectedPayment", "What you expect the factor to pay for the schedule."),
			{
				Path:        "Attachments",
				Type:        templatecatalog.VariableStringList,
				Description: "The files attached to this message.",
			},
			{
				Path:        "Part",
				Type:        templatecatalog.VariableInt,
				Description: "Which message this is when the packets are too large for one, starting at 1.",
			},
			{
				Path:        "Parts",
				Type:        templatecatalog.VariableInt,
				Required:    true,
				Description: "How many messages the schedule is sent in. Required: the factor must know when it has every packet.",
			},
			{
				Path:        "ReplyTo",
				Type:        templatecatalog.VariableString,
				Description: "The address the factor's questions go to.",
			},
		},
	})
}
