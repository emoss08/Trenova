package invoicetemplate

import (
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

// The context types live in the domain rather than inside the service that builds
// them because a template context is a published contract: it is the variable list
// an administrator sees in the editor and writes templates against. Hiding it in a
// service would mean the catalog and the struct could drift apart with nothing to
// catch it. registry_context_test.go asserts they match exactly, in both directions.

// InvoiceContext is the data a customer invoice renders against.
//
// Every field is pre-formatted text rather than a raw value. That is deliberate:
// currency rounding, timezone resolution, and address composition are decisions the
// billing domain has already made, and re-making them inside a template is how two
// documents come to disagree about the same invoice.
type InvoiceContext struct {
	InvoiceNumber string
	InvoiceDate   string
	DueDate       string
	PaymentTerm   string
	CurrencyCode  string

	Organization templatecatalog.AddressBlock
	BillTo       templatecatalog.AddressBlock
	RemitTo      templatecatalog.AddressBlock
	Shipper      templatecatalog.AddressBlock
	Consignee    templatecatalog.AddressBlock

	// LogoDataURI is the organization logo, already inlined as a data: URI.
	//
	// The type is template.URL rather than string because html/template's URL filter
	// allows only http, https, and mailto: a data: URI supplied as a plain string is
	// replaced with "#ZgotmplZ", which would leave every document with a broken logo.
	//
	// Typing it as trusted is safe here and only here: the value is produced by the
	// asset inliner from image bytes it has already decoded and validated, so it is
	// base64 and cannot contain a quote or an angle bracket. It is never user text.
	LogoDataURI template.URL

	HeaderRows    []templatecatalog.KeyValue
	CommodityRows []CommodityRow
	ChargeRows    []ChargeRow

	// ShipmentRows lists the freight on an invoice covering more than one
	// shipment, and is empty on a single-shipment invoice. Shipper, Consignee and
	// CommodityRows are the inverse — populated for one shipment, empty for many —
	// so a template branches on whichever is present.
	ShipmentRows []ShipmentRow
	// ShipmentCount is how many shipments this invoice bills, blank unless more
	// than one.
	ShipmentCount string
	// Period is the billing window a consolidated invoice covers, blank otherwise.
	Period string

	Subtotal   string
	Other      string
	Total      string
	BalanceDue string

	Terms         []string
	InvoiceTerms  []string
	InvoiceFooter string
	Notes         []string
	Attachments   []string

	// NoticeOfAssignment is set when the invoice is assigned to a factor:
	// RemitTo is then the factor, and the notice must be printed.
	NoticeOfAssignment string
}

// ChargeRow is one billed line on an invoice.
type ChargeRow struct {
	Line        string
	Description string
	Quantity    string
	UnitPrice   string
	Amount      string
	// ProNumber attributes the charge to its shipment on an invoice that bills
	// several, and is blank on one that bills a single shipment.
	ProNumber string
}

// ShipmentRow is one shipment on an invoice that bills several.
//
// It is a line about a shipment rather than a block describing it: a month of
// LTL is dozens of shipments, and repeating a shipper, a consignee and a
// commodity table for each produces a document nobody reads. What a customer
// checks at statement level is which shipments are on the bill, the lane, and
// what each cost.
type ShipmentRow struct {
	ProNumber   string
	BOL         string
	PONumber    string
	ServiceDate string
	Origin      string
	Destination string
	Amount      string
}

// CommodityRow is one freight line on an invoice.
type CommodityRow struct {
	Quantity         string
	Type             string
	DescriptionLines []string
	Weight           string
	NMFC             string
	Class            string
}

// InvoiceEmailContext is the message that delivers an invoice.
type InvoiceEmailContext struct {
	InvoiceNumber string
	InvoiceDate   string
	DueDate       string
	PaymentTerm   string
	Total         string
	CurrencyCode  string
	Memo          string

	CustomerName string
	CustomerCode string
	CompanyName  string

	ShipmentProNumber   string
	ShipmentBOL         string
	ShipmentOrigin      string
	ShipmentDestination string
	ShipmentServiceDate string

	RemittanceInstructions string
	// NoticeOfAssignment is set when the invoice is assigned to a factor.
	NoticeOfAssignment string

	// PartLabel is set when a large invoice is split across messages, so the
	// template can say "2 of 3" rather than sending three identical-looking emails.
	PartLabel string

	AttachmentName string
}

//nolint:funlen // Sample fixture data; splitting it would only scatter one document.
func newInvoiceSampleContext() any {
	return InvoiceContext{
		InvoiceNumber: templatecatalog.SampleInvoiceNo,
		InvoiceDate:   "2026-07-15",
		DueDate:       "2026-08-14",
		PaymentTerm:   "Net30",
		CurrencyCode:  "USD",
		Organization: templatecatalog.AddressBlock{
			Name:  templatecatalog.SampleCompanyName,
			Lines: []string{"4400 Commerce Way", "Suite 220", "Austin, TX 78744"},
			Details: []templatecatalog.KeyValue{
				{Label: "Phone", Value: "(512) 555-0142"},
				{Label: "MC", Value: "MC-884120"},
			},
		},
		BillTo: templatecatalog.AddressBlock{
			Name:  templatecatalog.SampleCustomerName,
			Lines: []string{"Accounts Payable", "900 Distribution Pkwy", "Fort Worth, TX 76106"},
			Details: []templatecatalog.KeyValue{
				{Label: "Account", Value: "HALSTEAD-4471"},
				{Label: "AP Contact", Value: "ap@halstead.example"},
			},
		},
		RemitTo: templatecatalog.AddressBlock{
			Name:  templatecatalog.SampleCompanyName,
			Lines: []string{"PO Box 41220", "Austin, TX 78704"},
			Details: []templatecatalog.KeyValue{
				{Label: "ACH", Value: "Routing 111000614 / Account 8842091"},
			},
		},
		Shipper: templatecatalog.AddressBlock{
			Name:  templatecatalog.SampleShipperName,
			Lines: []string{"1200 Airport Rd", "Pharr, TX 78577"},
			Details: []templatecatalog.KeyValue{
				{Label: "Loaded", Value: "2026-07-11 06:40 CDT"},
			},
		},
		Consignee: templatecatalog.AddressBlock{
			Name:  "Halstead DC 14",
			Lines: []string{"900 Distribution Pkwy", "Fort Worth, TX 76106"},
			Details: []templatecatalog.KeyValue{
				{Label: "Delivered", Value: "2026-07-12 14:05 CDT"},
			},
		},
		//nolint:gosec // A compile-time constant data: URI; see the field's doc comment.
		LogoDataURI: template.URL(templatecatalog.SampleLogoDataURI),
		HeaderRows: []templatecatalog.KeyValue{
			{Label: "Pro Number", Value: templatecatalog.SampleProNumber},
			{Label: "BOL", Value: sampleBOL},
			{Label: "Customer PO", Value: "PO-5591004"},
		},
		CommodityRows: []CommodityRow{
			{
				Quantity:         "22",
				Type:             "Pallets",
				DescriptionLines: []string{"Fresh produce, refrigerated", "Keep at 34-38F"},
				Weight:           "41,200 lb",
				NMFC:             "73320",
				Class:            "70",
			},
			{
				Quantity:         "4",
				Type:             "Crates",
				DescriptionLines: []string{"Packaging materials"},
				Weight:           "820 lb",
				NMFC:             "156600",
				Class:            "55",
			},
		},
		ChargeRows: []ChargeRow{
			{
				Line:        "1",
				Description: "Linehaul",
				Quantity:    "612 mi",
				UnitPrice:   "2.85",
				Amount:      "1,744.20",
			},
			{
				Line:        "2",
				Description: "Fuel surcharge",
				Quantity:    "612 mi",
				UnitPrice:   "0.42",
				Amount:      "257.04",
			},
			{
				Line:        "3",
				Description: "Detention — Halstead DC 14",
				Quantity:    "2.5 hr",
				UnitPrice:   "65.00",
				Amount:      "162.50",
			},
			{
				Line:        "4",
				Description: "Lumper",
				Quantity:    "1",
				UnitPrice:   "175.00",
				Amount:      "175.00",
			},
		},
		// The sample carries both freight shapes, which no real invoice does: one
		// shipment fills Shipper, Consignee and CommodityRows, and several fill
		// ShipmentRows instead. It is the variable exerciser, and a template author
		// previewing a consolidated layout has to see rows in it — an empty
		// collection would render nothing and hide a typo inside the range.
		ShipmentRows: []ShipmentRow{
			{
				ProNumber:   templatecatalog.SampleProNumber,
				BOL:         sampleBOL,
				PONumber:    "PO-5591004",
				ServiceDate: "2026-07-12",
				Origin:      "Austin, TX",
				Destination: "Fort Worth, TX",
				Amount:      "2,338.74",
			},
			{
				ProNumber:   "PRO-77402",
				BOL:         "BOL-77402",
				PONumber:    "PO-5591118",
				ServiceDate: "2026-07-19",
				Origin:      "Austin, TX",
				Destination: "Waco, TX",
				Amount:      "1,104.60",
			},
		},
		ShipmentCount: "2",
		Period:        "2026-07-01 - 2026-08-01",
		Subtotal:      "USD 2,001.24",
		Other:         "USD 337.50",
		Total:         templatecatalog.SampleTotalAmount,
		BalanceDue:    templatecatalog.SampleTotalAmount,
		Terms: []string{
			"Payment due within 30 days of the invoice date.",
			"Claims must be filed within 9 months of delivery.",
		},
		InvoiceTerms: []string{
			"Remit by ACH or check to the address above.",
		},
		InvoiceFooter: "Thank you for shipping with Trenova.",
		Notes:         []string{"Reefer set to 36F continuous per customer instruction."},
		Attachments:   []string{"BOL-77401.pdf", "POD-77401.pdf", "Lumper-receipt.pdf"},
	}
}

func newInvoiceEmailSampleContext() any {
	return InvoiceEmailContext{
		InvoiceNumber:          templatecatalog.SampleInvoiceNo,
		InvoiceDate:            "2026-07-15",
		DueDate:                "2026-08-14",
		PaymentTerm:            "Net30",
		Total:                  templatecatalog.SampleTotalAmount,
		CurrencyCode:           templatecatalog.SampleCurrency,
		Memo:                   "Reefer set to 36F continuous per customer instruction.",
		CustomerName:           templatecatalog.SampleCustomerName,
		CustomerCode:           "HALSTEAD",
		CompanyName:            templatecatalog.SampleCompanyName,
		ShipmentProNumber:      templatecatalog.SampleProNumber,
		ShipmentBOL:            sampleBOL,
		ShipmentOrigin:         "Pharr, TX",
		ShipmentDestination:    "Fort Worth, TX",
		ShipmentServiceDate:    "2026-07-12",
		RemittanceInstructions: "Remit by ACH to routing 111000614, account 8842091.",
		PartLabel:              "",
		AttachmentName:         "Invoice-INV-2026-1042.pdf",
	}
}

//nolint:funlen // A variable catalog is one declaration; splitting it only hides it.
func RegisterBillingKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:           documenttemplate.KindInvoicePDF,
		DisplayName:    "Customer Invoice",
		Description:    "The invoice document sent to a customer and attached to the delivery email.",
		Category:       "Billing",
		Channels:       []documenttemplate.Channel{documenttemplate.ChannelPDF},
		Paged:          true,
		CustomerScoped: true,
		SampleFactory:  newInvoiceSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			templatecatalog.InvoiceNumberVariable(
				true,
				"The invoice number. Customers reference it on payment, so an invoice without it cannot be matched to a remittance.",
			),
			invoiceDateVariable(),
			dueDateVariable(),
			paymentTermVariable(),
			currencyCodeVariable(),
			templatecatalog.LogoVariable(),
			{
				Path:        "Subtotal",
				Type:        templatecatalog.VariableMoney,
				Description: "Charges before other amounts, with currency.",
			},
			{
				Path:        "Other",
				Type:        templatecatalog.VariableMoney,
				Description: "Accessorial and adjustment amounts, with currency.",
			},
			{
				Path:        templatecatalog.SampleTotalsLabel,
				Type:        templatecatalog.VariableMoney,
				Required:    true,
				Description: "The invoice total, with currency. An invoice that does not state its total is one a customer can refuse to pay.",
			},
			{
				Path:        "BalanceDue",
				Type:        templatecatalog.VariableMoney,
				Description: "Amount still owed. Blank when the billing settings hide it.",
			},
			{
				Path:        "InvoiceFooter",
				Type:        templatecatalog.VariableString,
				Description: "The footer text from the organization's billing settings.",
			},
			{
				Path:        "Organization",
				Type:        templatecatalog.VariableObject,
				Description: "The carrier issuing the invoice.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        "BillTo",
				Type:        templatecatalog.VariableObject,
				Required:    true,
				Description: "The party being billed. Required: an invoice with no addressee cannot be collected on.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        "RemitTo",
				Type:        templatecatalog.VariableObject,
				Required:    true,
				Description: "Where payment should be sent. Required: without it a customer has nowhere to pay.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        "Shipper",
				Type:        templatecatalog.VariableObject,
				Description: "Origin party and pickup detail.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        "Consignee",
				Type:        templatecatalog.VariableObject,
				Description: "Destination party and delivery detail.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        "HeaderRows",
				Type:        templatecatalog.VariableCollection,
				Description: "Reference numbers for the shipment: pro number, BOL, customer PO.",
				Fields:      templatecatalog.KeyValueFields(),
			},
			{
				Path:        "CommodityRows",
				Type:        templatecatalog.VariableCollection,
				Description: "The freight billed, one row per commodity.",
				Fields: []templatecatalog.VariableDefinition{
					{
						Path:        "Quantity",
						Type:        templatecatalog.VariableString,
						Description: "Piece count.",
					},
					{
						Path:        "Type",
						Type:        templatecatalog.VariableString,
						Description: "Packaging, such as Pallets.",
					},
					{
						Path:        "DescriptionLines",
						Type:        templatecatalog.VariableStringList,
						Description: "Commodity description, one entry per line.",
					},
					{
						Path:        "Weight",
						Type:        templatecatalog.VariableString,
						Description: "Billed weight.",
					},
					{
						Path:        "NMFC",
						Type:        templatecatalog.VariableString,
						Description: "NMFC freight classification code.",
					},
					{
						Path:        "Class",
						Type:        templatecatalog.VariableString,
						Description: "Freight class.",
					},
				},
			},
			{
				Path:        "ChargeRows",
				Type:        templatecatalog.VariableCollection,
				Required:    true,
				Description: "The billed lines. Required: a customer cannot verify a total with no lines behind it.",
				Fields: []templatecatalog.VariableDefinition{
					{
						Path:        "Line",
						Type:        templatecatalog.VariableString,
						Description: "Line number.",
					},
					{
						Path:        "Description",
						Type:        templatecatalog.VariableString,
						Description: "What is being charged.",
					},
					{
						Path:        "Quantity",
						Type:        templatecatalog.VariableString,
						Description: "Billed quantity and unit.",
					},
					{
						Path:        "UnitPrice",
						Type:        templatecatalog.VariableString,
						Description: "Rate per unit.",
					},
					{
						Path:        "Amount",
						Type:        templatecatalog.VariableString,
						Description: "Extended amount for the line.",
					},
					{
						Path:        "ProNumber",
						Type:        templatecatalog.VariableString,
						Description: "Pro number of the shipment this charge belongs to. Blank when the invoice bills only one shipment.",
					},
				},
			},
			{
				Path:        "ShipmentRows",
				Type:        templatecatalog.VariableCollection,
				Description: "The shipments on a consolidated invoice, one row each. Empty when the invoice bills a single shipment, where Shipper, Consignee and CommodityRows describe the freight instead.",
				Fields: []templatecatalog.VariableDefinition{
					{
						Path:        "ProNumber",
						Type:        templatecatalog.VariableString,
						Description: "Pro number.",
					},
					{
						Path:        "BOL",
						Type:        templatecatalog.VariableString,
						Description: "Bill of lading number.",
					},
					{
						Path:        "PONumber",
						Type:        templatecatalog.VariableString,
						Description: "Customer purchase order number.",
					},
					{
						Path:        "ServiceDate",
						Type:        templatecatalog.VariableString,
						Description: "Delivery date.",
					},
					{
						Path:        "Origin",
						Type:        templatecatalog.VariableString,
						Description: "City and state the freight was picked up in.",
					},
					{
						Path:        "Destination",
						Type:        templatecatalog.VariableString,
						Description: "City and state the freight was delivered to.",
					},
					{
						Path:        "Amount",
						Type:        templatecatalog.VariableString,
						Description: "What this shipment contributes to the invoice total.",
					},
				},
			},
			{
				Path:        "ShipmentCount",
				Type:        templatecatalog.VariableString,
				Description: "How many shipments this invoice bills. Blank unless there is more than one.",
			},
			{
				Path:        "Period",
				Type:        templatecatalog.VariableString,
				Description: "The billing period a consolidated invoice covers. Blank on an invoice that is not billed on a cycle.",
			},
			{
				Path:        "Terms",
				Type:        templatecatalog.VariableStringList,
				Description: "Terms and conditions, one entry per paragraph.",
			},
			{
				Path:        "InvoiceTerms",
				Type:        templatecatalog.VariableStringList,
				Description: "Additional terms from the organization's billing settings.",
			},
			{
				Path:        "Notes",
				Type:        templatecatalog.VariableStringList,
				Description: "The invoice memo, one entry per paragraph.",
			},
			{
				Path:        "Attachments",
				Type:        templatecatalog.VariableStringList,
				Description: "Filenames of the documents accompanying this invoice.",
			},
			templatecatalog.NoticeOfAssignmentVariable(),
		},
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindInvoiceEmail,
		DisplayName: "Invoice Delivery Email",
		Description: "The message a customer receives with their invoice attached.",
		Category:    "Billing",
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		CustomerScoped: true,
		SampleFactory:  newInvoiceEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			templatecatalog.InvoiceNumberVariable(
				true,
				"The invoice number. A delivery email that does not name the invoice forces the recipient to open the attachment to find out what arrived.",
			),
			invoiceDateVariable(),
			dueDateVariable(),
			paymentTermVariable(),
			{
				Path:        "Total",
				Type:        templatecatalog.VariableMoney,
				Description: "The invoice total, with currency.",
			},
			currencyCodeVariable(),
			{
				Path:        "Memo",
				Type:        templatecatalog.VariableString,
				Description: "The invoice memo, if one was entered.",
			},
			templatecatalog.CustomerNameVariable(false, "The billed customer's name."),
			{
				Path:        "CustomerCode",
				Type:        templatecatalog.VariableString,
				Description: "The billed customer's code.",
			},
			templatecatalog.CompanyNameVariable(),
			templatecatalog.ProNumberVariable(false, "Pro number of the shipment billed."),
			{
				Path:        "ShipmentBOL",
				Type:        templatecatalog.VariableString,
				Description: "Bill of lading number.",
			},
			{
				Path:        "ShipmentOrigin",
				Type:        templatecatalog.VariableString,
				Description: "Origin city and state.",
			},
			{
				Path:        "ShipmentDestination",
				Type:        templatecatalog.VariableString,
				Description: "Destination city and state.",
			},
			{
				Path:        "ShipmentServiceDate",
				Type:        templatecatalog.VariableDate,
				Description: "Delivery date.",
			},
			{
				Path:        "RemittanceInstructions",
				Type:        templatecatalog.VariableString,
				Description: "How to pay, from the invoice. For an invoice assigned to a factor, the factor's name, address and payment details.",
			},
			templatecatalog.NoticeOfAssignmentVariable(),
			{
				Path:        "PartLabel",
				Type:        templatecatalog.VariableString,
				Description: "Set to \"2 of 3\" when an invoice is too large for one message and had to be split. Blank otherwise.",
			},
			{
				Path:        "AttachmentName",
				Type:        templatecatalog.VariableString,
				Description: "Filename of the attached invoice PDF.",
			},
		},
	})
}

type InvoiceShareEmailContext struct {
	RecipientFirstName string
	SharedByName       string
	InvoiceNumber      string
	CustomerName       string
	Note               string
	InvoiceURL         template.URL
	CompanyName        string
}

func newInvoiceShareEmailSampleContext() any {
	return InvoiceShareEmailContext{
		RecipientFirstName: "Dana",
		SharedByName:       "Marcus Bell",
		InvoiceNumber:      templatecatalog.SampleInvoiceNo,
		CustomerName:       templatecatalog.SampleCustomerName,
		Note:               "Halstead disputed the detention line on this one. Can you check it against the gate times before we send?",
		InvoiceURL: template.URL(
			"https://app.example.com/billing/invoices?item=inv_01J2Y8Z3Q4R5S6T7V8W9X0Y1Z2&tab=charges",
		),
		CompanyName: templatecatalog.SampleCompanyName,
	}
}

func RegisterInvoiceShareKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindInvoiceShareEmail,
		DisplayName: "Invoice Shared With a Teammate",
		Description: "Tells a teammate someone shared an invoice with them, with the sharer's note and a link that opens it. Not customer-scoped: the recipient is a user who can already view invoices.",
		Category:    "Billing",
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newInvoiceShareEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "RecipientFirstName",
				Type:        templatecatalog.VariableString,
				Description: "The teammate's first name.",
			},
			{
				Path:        "SharedByName",
				Type:        templatecatalog.VariableString,
				Description: "Who shared the invoice. Name them: a share from nobody gives the reader no reason to open it.",
			},
			templatecatalog.InvoiceNumberVariable(true, "The invoice that was shared."),
			templatecatalog.CustomerNameVariable(false, "The billed customer's name."),
			{
				Path:        "Note",
				Type:        templatecatalog.VariableString,
				Description: "What the sharer wrote, if anything. Print it: it is the reason the invoice was shared.",
			},
			{
				Path:        "InvoiceURL",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The link that opens the invoice, on the tab the sharer was looking at. Required: without it the message has no purpose.",
			},
			templatecatalog.CompanyNameVariable(),
		},
	})
}

// The sample fixtures below describe one shipment consistently across every kind,
// so an administrator previewing an invoice, the email that delivers it, and the
// detention notice that preceded it sees the same load rather than three unrelated
// inventions.
const (
	sampleBOL = "BOL-77401"
)

func invoiceDateVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "InvoiceDate",
		Type:        templatecatalog.VariableDate,
		Description: "The date the invoice was issued, in the organization timezone.",
	}
}

func dueDateVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "DueDate",
		Type:        templatecatalog.VariableDate,
		Description: "When payment is due. Blank when the billing settings hide it.",
	}
}

func paymentTermVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "PaymentTerm",
		Type:        templatecatalog.VariableString,
		Description: "The payment term code, such as Net30.",
	}
}

func currencyCodeVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "CurrencyCode",
		Type:        templatecatalog.VariableString,
		Description: "The invoice currency.",
	}
}
