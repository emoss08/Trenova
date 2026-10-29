package templatecatalog

// Several kinds publish the same variable. Sharing the definition rather than
// repeating it keeps the description identical everywhere, which matters because the
// editor shows it verbatim: an administrator authoring an invoice and a detention
// notice should not be told two different things about the same field.

// sampleReportTitle is the report every reporting sample names, so an
// administrator comparing a report PDF, its delivery email, and its notification
// sees one report rather than three.
const SampleReportTitle = "Revenue by Customer"

const (
	ExpiresInDaysPath = "ExpiresInDays"
)

func LogoVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "LogoDataURI",
		Type:        VariableString,
		Description: "The organization logo, already embedded. Use it as an image source directly; a template never names a URL the renderer would fetch.",
	}
}

func CompanyNameVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "CompanyName",
		Type:        VariableString,
		Description: "Your organization's name, as the recipient knows it.",
	}
}

func CustomerNameVariable(required bool, description string) VariableDefinition {
	return VariableDefinition{
		Path:        "CustomerName",
		Type:        VariableString,
		Required:    required,
		Description: description,
	}
}

func ProNumberVariable(required bool, description string) VariableDefinition {
	return VariableDefinition{
		Path:        "ShipmentProNumber",
		Type:        VariableString,
		Required:    required,
		Description: description,
	}
}

func RowCountVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "RowCount",
		Type:        VariableInt,
		Description: "How many rows the run returned.",
	}
}

func TruncatedVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "Truncated",
		Type:        VariableBool,
		Description: "True when the row limit cut the results short. Say so when it is, or a reader will treat a partial answer as the whole one.",
	}
}

func InvoiceNumberVariable(required bool, description string) VariableDefinition {
	return VariableDefinition{
		Path:        "InvoiceNumber",
		Type:        VariableString,
		Required:    required,
		Description: description,
	}
}

func AddressBlockFields() []VariableDefinition {
	return []VariableDefinition{
		{Path: "Name", Type: VariableString, Description: "Party name."},
		{
			Path:        "Lines",
			Type:        VariableStringList,
			Description: "Address lines. Join them with the join function to skip the blanks and avoid a dangling separator.",
		},
		{
			Path:        "Details",
			Type:        VariableCollection,
			Description: "Extra labelled detail for this party.",
			Fields:      KeyValueFields(),
		},
	}
}

func KeyValueFields() []VariableDefinition {
	return []VariableDefinition{
		{Path: "Label", Type: VariableString, Description: "The label."},
		{Path: "Value", Type: VariableString, Description: "The value."},
	}
}

func NoticeOfAssignmentVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "NoticeOfAssignment",
		Type:        VariableString,
		Description: "The factor's notice of assignment when the invoice is assigned to a factoring company, blank otherwise. Print it whenever it is set: a factored invoice without its notice can be paid to the wrong party.",
	}
}

//nolint:gosec // G101: rendering fixtures of fake values, not credentials
const (
	SampleOriginSummary      = "Dallas, TX"
	SampleDestinationSummary = "Memphis, TN"
	SamplePickupWindow       = "2026-07-18 08:00 – 12:00 CDT"
	SampleDeliveryWindow     = "2026-07-19 09:00 – 15:00 CDT"
)

// routeVariables are where a load picks up and delivers, as city and state.
func RouteVariables() []VariableDefinition {
	return []VariableDefinition{
		{
			Path:        "OriginSummary",
			Type:        VariableString,
			Description: "Where the load picks up, as city and state.",
		},
		{
			Path:        "DestinationSummary",
			Type:        VariableString,
			Description: "Where the load delivers, as city and state.",
		},
	}
}

// routeScheduleVariables are the route plus its planned pickup and delivery
// windows.
func RouteScheduleVariables() []VariableDefinition {
	return append(RouteVariables(),
		VariableDefinition{
			Path:        "PickupWindow",
			Type:        VariableString,
			Description: "The planned pickup window.",
		},
		VariableDefinition{
			Path:        "DeliveryWindow",
			Type:        VariableString,
			Description: "The planned delivery window.",
		},
	)
}

const (
	InvoiceIssuedDescription   = "When the invoice was issued."
	SampleCompanyName          = "Trenova Logistics, Inc."
	SampleCustomerName         = "Halstead Grocery Group"
	SampleProNumber            = "TRV-884120"
	SampleFacilityName         = "Halstead DC 14"
	SampleCurrency             = "USD"
	SampleInvoiceNo            = "INV-2026-1042"
	SampleShipperName          = "Rio Grande Produce"
	SampleTotalAmount          = "USD 2,338.74"
	SampleTotalsLabel          = "Total"
	ExpiresAtPath              = "ExpiresAt"
	BillingCategory            = "Billing"
	customerCodePath           = "CustomerCode"
	NotePath                   = "Note"
	CurrencyCodePath           = "CurrencyCode"
	CustomerPath               = "Customer"
	InvoiceDatePath            = "InvoiceDate"
	DueDatePath                = "DueDate"
	RemittanceInstructionsPath = "RemittanceInstructions"
	SampleCustomerCode         = "HALSTEAD"
	SampleRemittanceText       = "Remit by ACH to routing 111000614, account 8842091."
	SampleAmountPartial        = "USD 1,763.62"
	SampleAmountOverdue        = "USD 4,102.36"
	SampleDateJul15            = "2026-07-15"
	SampleDateAug14            = "2026-08-14"
)

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

// sampleLogoDataURI is a 2x2 grey PNG. It exists so a preview shows an image in
// the logo slot without reaching storage, and so a template that sizes its logo can
// be verified at publish time.
const SampleLogoDataURI = "data:image/png;base64," +
	"iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAEUlEQVR4nGJiYmJgYGBgYAAAAAsAAyQ" +
	"YbFAAAAAASUVORK5CYII="

func StatementCustomerCodeVariable() VariableDefinition {
	return VariableDefinition{
		Path:        customerCodePath,
		Type:        VariableString,
		Description: "The customer's code.",
	}
}
