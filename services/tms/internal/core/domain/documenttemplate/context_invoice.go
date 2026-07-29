package documenttemplate

import "html/template"

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

	Organization AddressBlock
	BillTo       AddressBlock
	RemitTo      AddressBlock
	Shipper      AddressBlock
	Consignee    AddressBlock

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

	HeaderRows    []KeyValue
	CommodityRows []CommodityRow
	ChargeRows    []ChargeRow

	Subtotal   string
	Other      string
	Total      string
	BalanceDue string

	Terms         []string
	InvoiceTerms  []string
	InvoiceFooter string
	Notes         []string
	Attachments   []string
}

// AddressBlock is a named party with its address lines and any extra detail.
type AddressBlock struct {
	Name    string
	Lines   []string
	Details []KeyValue
}

// KeyValue is a labelled detail line.
type KeyValue struct {
	Label string
	Value string
}

// ChargeRow is one billed line on an invoice.
type ChargeRow struct {
	Line        string
	Description string
	Quantity    string
	UnitPrice   string
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

	// PartLabel is set when a large invoice is split across messages, so the
	// template can say "2 of 3" rather than sending three identical-looking emails.
	PartLabel string

	AttachmentName string
}

//nolint:funlen // Sample fixture data; splitting it would only scatter one document.
func newInvoiceSampleContext() any {
	return InvoiceContext{
		InvoiceNumber: sampleInvoiceNo,
		InvoiceDate:   "2026-07-15",
		DueDate:       "2026-08-14",
		PaymentTerm:   "Net30",
		CurrencyCode:  "USD",
		Organization: AddressBlock{
			Name:  sampleCompanyName,
			Lines: []string{"4400 Commerce Way", "Suite 220", "Austin, TX 78744"},
			Details: []KeyValue{
				{Label: "Phone", Value: "(512) 555-0142"},
				{Label: "MC", Value: "MC-884120"},
			},
		},
		BillTo: AddressBlock{
			Name:  sampleCustomerName,
			Lines: []string{"Accounts Payable", "900 Distribution Pkwy", "Fort Worth, TX 76106"},
			Details: []KeyValue{
				{Label: "Account", Value: "HALSTEAD-4471"},
				{Label: "AP Contact", Value: "ap@halstead.example"},
			},
		},
		RemitTo: AddressBlock{
			Name:  sampleCompanyName,
			Lines: []string{"PO Box 41220", "Austin, TX 78704"},
			Details: []KeyValue{
				{Label: "ACH", Value: "Routing 111000614 / Account 8842091"},
			},
		},
		Shipper: AddressBlock{
			Name:  sampleShipperName,
			Lines: []string{"1200 Airport Rd", "Pharr, TX 78577"},
			Details: []KeyValue{
				{Label: "Loaded", Value: "2026-07-11 06:40 CDT"},
			},
		},
		Consignee: AddressBlock{
			Name:  "Halstead DC 14",
			Lines: []string{"900 Distribution Pkwy", "Fort Worth, TX 76106"},
			Details: []KeyValue{
				{Label: "Delivered", Value: "2026-07-12 14:05 CDT"},
			},
		},
		//nolint:gosec // A compile-time constant data: URI; see the field's doc comment.
		LogoDataURI: template.URL(sampleLogoDataURI),
		HeaderRows: []KeyValue{
			{Label: "Pro Number", Value: sampleProNumber},
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
		Subtotal:   "USD 2,001.24",
		Other:      "USD 337.50",
		Total:      sampleTotalAmount,
		BalanceDue: sampleTotalAmount,
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
		InvoiceNumber:          sampleInvoiceNo,
		InvoiceDate:            "2026-07-15",
		DueDate:                "2026-08-14",
		PaymentTerm:            "Net30",
		Total:                  sampleTotalAmount,
		CurrencyCode:           sampleCurrency,
		Memo:                   "Reefer set to 36F continuous per customer instruction.",
		CustomerName:           sampleCustomerName,
		CustomerCode:           "HALSTEAD",
		CompanyName:            sampleCompanyName,
		ShipmentProNumber:      sampleProNumber,
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
func (r *Registry) registerBillingKinds() {
	_ = r.Register(&KindDefinition{
		Kind:           KindInvoicePDF,
		DisplayName:    "Customer Invoice",
		Description:    "The invoice document sent to a customer and attached to the delivery email.",
		Category:       "Billing",
		Channels:       []Channel{ChannelPDF},
		Paged:          true,
		CustomerScoped: true,
		sampleFactory:  newInvoiceSampleContext,
		Variables: []VariableDefinition{
			invoiceNumberVariable(
				true,
				"The invoice number. Customers reference it on payment, so an invoice without it cannot be matched to a remittance.",
			),
			invoiceDateVariable(),
			dueDateVariable(),
			paymentTermVariable(),
			currencyCodeVariable(),
			logoVariable(),
			{
				Path:        "Subtotal",
				Type:        VariableMoney,
				Description: "Charges before other amounts, with currency.",
			},
			{
				Path:        "Other",
				Type:        VariableMoney,
				Description: "Accessorial and adjustment amounts, with currency.",
			},
			{
				Path:        sampleTotalsLabel,
				Type:        VariableMoney,
				Required:    true,
				Description: "The invoice total, with currency. An invoice that does not state its total is one a customer can refuse to pay.",
			},
			{
				Path:        "BalanceDue",
				Type:        VariableMoney,
				Description: "Amount still owed. Blank when the billing settings hide it.",
			},
			{
				Path:        "InvoiceFooter",
				Type:        VariableString,
				Description: "The footer text from the organization's billing settings.",
			},
			{
				Path:        "Organization",
				Type:        VariableObject,
				Description: "The carrier issuing the invoice.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        "BillTo",
				Type:        VariableObject,
				Required:    true,
				Description: "The party being billed. Required: an invoice with no addressee cannot be collected on.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        "RemitTo",
				Type:        VariableObject,
				Required:    true,
				Description: "Where payment should be sent. Required: without it a customer has nowhere to pay.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        "Shipper",
				Type:        VariableObject,
				Description: "Origin party and pickup detail.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        "Consignee",
				Type:        VariableObject,
				Description: "Destination party and delivery detail.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        "HeaderRows",
				Type:        VariableCollection,
				Description: "Reference numbers for the shipment: pro number, BOL, customer PO.",
				Fields:      keyValueFields(),
			},
			{
				Path:        "CommodityRows",
				Type:        VariableCollection,
				Description: "The freight billed, one row per commodity.",
				Fields: []VariableDefinition{
					{Path: "Quantity", Type: VariableString, Description: "Piece count."},
					{
						Path:        "Type",
						Type:        VariableString,
						Description: "Packaging, such as Pallets.",
					},
					{
						Path:        "DescriptionLines",
						Type:        VariableStringList,
						Description: "Commodity description, one entry per line.",
					},
					{Path: "Weight", Type: VariableString, Description: "Billed weight."},
					{
						Path:        "NMFC",
						Type:        VariableString,
						Description: "NMFC freight classification code.",
					},
					{Path: "Class", Type: VariableString, Description: "Freight class."},
				},
			},
			{
				Path:        "ChargeRows",
				Type:        VariableCollection,
				Required:    true,
				Description: "The billed lines. Required: a customer cannot verify a total with no lines behind it.",
				Fields: []VariableDefinition{
					{Path: "Line", Type: VariableString, Description: "Line number."},
					{
						Path:        "Description",
						Type:        VariableString,
						Description: "What is being charged.",
					},
					{
						Path:        "Quantity",
						Type:        VariableString,
						Description: "Billed quantity and unit.",
					},
					{Path: "UnitPrice", Type: VariableString, Description: "Rate per unit."},
					{
						Path:        "Amount",
						Type:        VariableString,
						Description: "Extended amount for the line.",
					},
				},
			},
			{
				Path:        "Terms",
				Type:        VariableStringList,
				Description: "Terms and conditions, one entry per paragraph.",
			},
			{
				Path:        "InvoiceTerms",
				Type:        VariableStringList,
				Description: "Additional terms from the organization's billing settings.",
			},
			{
				Path:        "Notes",
				Type:        VariableStringList,
				Description: "The invoice memo, one entry per paragraph.",
			},
			{
				Path:        "Attachments",
				Type:        VariableStringList,
				Description: "Filenames of the documents accompanying this invoice.",
			},
		},
	})

	_ = r.Register(&KindDefinition{
		Kind:           KindInvoiceEmail,
		DisplayName:    "Invoice Delivery Email",
		Description:    "The message a customer receives with their invoice attached.",
		Category:       "Billing",
		Channels:       []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		CustomerScoped: true,
		sampleFactory:  newInvoiceEmailSampleContext,
		Variables: []VariableDefinition{
			invoiceNumberVariable(
				true,
				"The invoice number. A delivery email that does not name the invoice forces the recipient to open the attachment to find out what arrived.",
			),
			invoiceDateVariable(),
			dueDateVariable(),
			paymentTermVariable(),
			{Path: "Total", Type: VariableMoney, Description: "The invoice total, with currency."},
			currencyCodeVariable(),
			{
				Path:        "Memo",
				Type:        VariableString,
				Description: "The invoice memo, if one was entered.",
			},
			customerNameVariable(false, "The billed customer's name."),
			{
				Path:        "CustomerCode",
				Type:        VariableString,
				Description: "The billed customer's code.",
			},
			companyNameVariable(),
			proNumberVariable(false, "Pro number of the shipment billed."),
			{Path: "ShipmentBOL", Type: VariableString, Description: "Bill of lading number."},
			{Path: "ShipmentOrigin", Type: VariableString, Description: "Origin city and state."},
			{
				Path:        "ShipmentDestination",
				Type:        VariableString,
				Description: "Destination city and state.",
			},
			{Path: "ShipmentServiceDate", Type: VariableDate, Description: "Delivery date."},
			{
				Path:        "RemittanceInstructions",
				Type:        VariableString,
				Description: "How to pay, from the invoice.",
			},
			{
				Path:        "PartLabel",
				Type:        VariableString,
				Description: "Set to \"2 of 3\" when an invoice is too large for one message and had to be split. Blank otherwise.",
			},
			{
				Path:        "AttachmentName",
				Type:        VariableString,
				Description: "Filename of the attached invoice PDF.",
			},
		},
	})
}

// The sample fixtures below describe one shipment consistently across every kind,
// so an administrator previewing an invoice, the email that delivers it, and the
// detention notice that preceded it sees the same load rather than three unrelated
// inventions.
const (
	sampleCompanyName  = "Trenova Logistics, Inc."
	sampleCustomerName = "Halstead Grocery Group"
	sampleProNumber    = "TRV-884120"
	sampleBOL          = "BOL-77401"
	sampleFacilityName = "Halstead DC 14"
	sampleCurrency     = "USD"
	sampleInvoiceNo    = "INV-2026-1042"
	sampleShipperName  = "Rio Grande Produce"
	sampleTotalAmount  = "USD 2,338.74"
	sampleTotalsLabel  = "Total"
)

// sampleLogoDataURI is a 2x2 grey PNG. It exists so a preview shows an image in
// the logo slot without reaching storage, and so a template that sizes its logo can
// be verified at publish time.
const sampleLogoDataURI = "data:image/png;base64," +
	"iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAEUlEQVR4nGJiYmJgYGBgYAAAAAsAAyQ" +
	"YbFAAAAAASUVORK5CYII="
