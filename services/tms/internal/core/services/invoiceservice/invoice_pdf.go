package invoiceservice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

type invoicePDFData struct {
	InvoiceNumber string
	InvoiceDate   string
	DueDate       string
	PaymentTerm   string
	CurrencyCode  string
	Organization  invoicePDFAddressBlock
	HeaderRows    []invoicePDFKeyValue
	BillTo        invoicePDFAddressBlock
	RemitTo       invoicePDFAddressBlock
	Shipper       invoicePDFAddressBlock
	Consignee     invoicePDFAddressBlock
	CommodityRows []invoicePDFCommodityRow
	ChargeRows    []invoicePDFChargeRow

	// ShipmentRows lists the freight on an invoice covering more than one
	// shipment, and is empty on a single-shipment invoice.
	//
	// A consolidated invoice deliberately shows the group of shipments rather
	// than each one's shipper, consignee and commodities: a month of LTL is forty
	// freight blocks nobody reads, and the detail that matters at statement level
	// is which shipments are on the bill and what each one cost.
	ShipmentRows []invoicePDFShipmentRow
	// ShipmentCount is how many distinct shipments this invoice bills, blank
	// unless there is more than one.
	ShipmentCount string
	// Period is the billing window a consolidated invoice covers, blank otherwise.
	Period string

	Subtotal string
	Other         string
	Total         string
	BalanceDue    string
	Terms         []string
	InvoiceTerms  []string
	InvoiceFooter string
	Notes         []string
	Attachments   []string
}

type invoicePDFAddressBlock struct {
	Name    string
	Lines   []string
	Details []invoicePDFKeyValue
}

type invoicePDFKeyValue struct {
	Label string
	Value string
}

type invoicePDFChargeRow struct {
	Line        string
	Description string
	Quantity    string
	UnitPrice   string
	Amount      string
	// ProNumber attributes the charge to its shipment, and is blank on an invoice
	// that bills only one.
	ProNumber string
}

// invoicePDFShipmentRow is one shipment as a consolidated invoice lists it.
type invoicePDFShipmentRow struct {
	ProNumber   string
	BOL         string
	PONumber    string
	ServiceDate string
	Origin      string
	Destination string
	Amount      string
}

type invoicePDFCommodityRow struct {
	Quantity         string
	Type             string
	DescriptionLines []string
	Weight           string
	NMFC             string
	Class            string
	PiecesValue      int64
	WeightValue      int64
}

// buildInvoicePDFData assembles every value an invoice document names.
//
// It survived the move to templates unchanged: the imperative renderer that used
// to consume it is gone, but the mapping from an invoice to the words on the page
// is the part that encodes billing rules, and the template context is built from
// this exact struct.
func buildInvoicePDFData(
	entity *invoice.Invoice,
	deliveryProfile *invoiceDeliveryProfile,
) invoicePDFData {
	var cus *customer.Customer
	var org *tenant.Organization
	var shp *shipment.Shipment
	var control *tenant.BillingControl
	var summaries []*repositories.ShipmentSummary
	if deliveryProfile != nil {
		cus = deliveryProfile.Customer
		org = deliveryProfile.Organization
		shp = deliveryProfile.Shipment
		control = deliveryProfile.BillingControl
		summaries = deliveryProfile.Shipments
	}
	if shp == nil {
		shp = entity.Shipment
	}
	if cus == nil {
		cus = entity.Customer
	}

	data := invoicePDFData{
		InvoiceNumber: entity.Number,
		InvoiceDate:   unixDate(entity.InvoiceDate),
		DueDate:       invoicePDFDueDate(entity, control),
		PaymentTerm:   string(entity.PaymentTerm),
		CurrencyCode:  entity.CurrencyCode,
		Organization:  organizationPDFAddressBlock(org),
		HeaderRows:    headerPDFRows(entity, org),
		BillTo:        billToPDFAddressBlock(entity, cus),
		RemitTo:       remitPDFAddressBlock(org, entity.RemittanceInstructions),
		ChargeRows:    chargePDFRows(entity),
		Period:        invoicePDFPeriod(entity),
		Subtotal:      moneyString(entity.CurrencyCode, entity.SubtotalAmount.StringFixed(2)),
		Other:         moneyString(entity.CurrencyCode, entity.OtherAmount.StringFixed(2)),
		Total:         moneyString(entity.CurrencyCode, entity.TotalAmount.StringFixed(2)),
		BalanceDue:    invoicePDFBalanceDue(entity, control),
		Terms:         invoicePDFTerms(entity, control),
		InvoiceTerms:  billingControlPDFInvoiceTerms(control),
		InvoiceFooter: billingControlPDFInvoiceFooter(control),
		Notes:         stringutils.FilterEmpty([]string{entity.Memo}),
		Attachments:   attachmentPDFNames(entity),
	}

	applyInvoicePDFFreight(&data, entity, shp, summaries)

	return data
}

// applyInvoicePDFFreight decides whether this invoice describes one shipment's
// freight or lists a group of shipments.
//
// The two are mutually exclusive on purpose. A single shipment gets the Shipper,
// Consignee and commodity table it has always had — byte for byte, so nothing
// about an ordinary invoice changes. An invoice covering several gets a manifest
// instead, and no shipper or consignee at all, because there is no single answer
// and printing the first shipment's addresses at the top of a forty-shipment
// invoice states something untrue.
func applyInvoicePDFFreight(
	data *invoicePDFData,
	entity *invoice.Invoice,
	shp *shipment.Shipment,
	summaries []*repositories.ShipmentSummary,
) {
	if len(summaries) > 1 {
		data.ShipmentRows = shipmentPDFRows(entity, summaries)
		data.ShipmentCount = strconv.Itoa(len(summaries))
		return
	}

	data.Shipper = shipmentStopPDFAddressBlock(shp, true)
	data.Consignee = shipmentStopPDFAddressBlock(shp, false)
	data.CommodityRows = shipmentCommodityPDFRows(shp)
}

// invoicePDFPeriod is the window a consolidated invoice bills.
//
// The stored end is exclusive, and it is printed as the boundary the freight
// stops at rather than decremented by a day: a customer reading "Mar 1 - Mar 31"
// would reasonably expect a 31 March delivery to be on it, and it is not.
func invoicePDFPeriod(entity *invoice.Invoice) string {
	if entity.PeriodStart == nil || entity.PeriodEnd == nil {
		return ""
	}

	return unixDate(*entity.PeriodStart) + " - " + unixDate(*entity.PeriodEnd)
}

func shipmentPDFRows(
	entity *invoice.Invoice,
	summaries []*repositories.ShipmentSummary,
) []invoicePDFShipmentRow {
	rows := make([]invoicePDFShipmentRow, 0, len(summaries))
	for _, summary := range summaries {
		if summary == nil {
			continue
		}
		rows = append(rows, invoicePDFShipmentRow{
			ProNumber:   summary.ProNumber,
			BOL:         summary.BOL,
			PONumber:    summary.PONumber,
			ServiceDate: unixDatePtr(summary.ServiceDate),
			Origin:      cityState(summary.OriginCity, summary.OriginState),
			Destination: cityState(summary.DestinationCity, summary.DestinationState),
			Amount: moneyString(
				entity.CurrencyCode,
				summary.TotalCharge.Decimal.StringFixed(2),
			),
		})
	}

	return rows
}

func cityState(city string, state string) string {
	city = strings.TrimSpace(city)
	state = strings.TrimSpace(state)
	switch {
	case city == "":
		return state
	case state == "":
		return city
	default:
		return city + ", " + state
	}
}

func billToPDFAddressBlock(entity *invoice.Invoice, cus *customer.Customer) invoicePDFAddressBlock {
	name := strings.TrimSpace(entity.BillToName)
	if name == "" && cus != nil {
		name = cus.Name
	}
	lines := []string{
		stringutils.FirstNonEmpty(
			entity.BillToAddressLine1,
			customerString(cus, func(c *customer.Customer) string {
				return c.AddressLine1
			}),
		),
		stringutils.FirstNonEmpty(
			entity.BillToAddressLine2,
			customerString(cus, func(c *customer.Customer) string {
				return c.AddressLine2
			}),
		),
		cityStatePostal(
			stringutils.FirstNonEmpty(
				entity.BillToCity,
				customerString(cus, func(c *customer.Customer) string {
					return c.City
				}),
			),
			stringutils.FirstNonEmpty(entity.BillToState, customerState(cus)),
			stringutils.FirstNonEmpty(
				entity.BillToPostalCode,
				customerString(cus, func(c *customer.Customer) string {
					return c.PostalCode
				}),
			),
		),
		stringutils.FirstNonEmpty(entity.BillToCountry, customerCountry(cus)),
	}
	return invoicePDFAddressBlock{Name: name, Lines: stringutils.FilterEmpty(lines)}
}

func remitPDFAddressBlock(
	org *tenant.Organization,
	remittanceInstructions string,
) invoicePDFAddressBlock {
	block := organizationPDFAddressBlock(org)
	block.Lines = append(
		block.Lines,
		stringutils.FilterEmpty(strings.Split(remittanceInstructions, "\n"))...)
	block.Lines = stringutils.FilterEmpty(block.Lines)
	return block
}

func organizationPDFAddressBlock(org *tenant.Organization) invoicePDFAddressBlock {
	if org == nil {
		return invoicePDFAddressBlock{}
	}
	lines := []string{
		org.AddressLine1,
		org.AddressLine2,
		cityStatePostal(org.City, organizationState(org), org.PostalCode),
		organizationCountry(org),
	}
	return invoicePDFAddressBlock{Name: org.Name, Lines: stringutils.FilterEmpty(lines)}
}

func stopPDFAddressBlock(stop *shipment.Stop) invoicePDFAddressBlock {
	if stop == nil {
		return invoicePDFAddressBlock{}
	}
	if stop.Location == nil {
		return invoicePDFAddressBlock{Lines: stringutils.FilterEmpty([]string{stop.AddressLine})}
	}
	loc := stop.Location
	lines := []string{
		loc.AddressLine1,
		loc.AddressLine2,
		cityStatePostal(loc.City, locationState(loc), loc.PostalCode),
		locationCountry(loc),
	}
	return invoicePDFAddressBlock{Name: loc.Name, Lines: stringutils.FilterEmpty(lines)}
}

func shipmentStopPDFAddressBlock(shp *shipment.Shipment, pickup bool) invoicePDFAddressBlock {
	selected := firstDeliveryStop(shp)
	if pickup {
		selected = firstPickupStop(shp)
	}
	block := stopPDFAddressBlock(selected)
	if shp == nil {
		return block
	}
	if pickup {
		block.Details = append(block.Details, invoicePDFKeyValue{
			Label: "Pickup Date",
			Value: unixDatePtr(shp.ActualShipDate),
		})
		return block
	}
	block.Details = append(block.Details, invoicePDFKeyValue{
		Label: "Delivery Date",
		Value: unixDatePtr(shp.ActualDeliveryDate),
	})
	return block
}

func headerPDFRows(entity *invoice.Invoice, org *tenant.Organization) []invoicePDFKeyValue {
	rows := []invoicePDFKeyValue{
		{Label: "DOT", Value: organizationDOT(org)},
		{Label: "SCAC", Value: organizationSCAC(org)},
		{Label: "Payment Terms", Value: string(entity.PaymentTerm)},
		{Label: "PRO", Value: entity.ShipmentProNumber},
	}
	result := make([]invoicePDFKeyValue, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Value) != "" {
			result = append(result, row)
		}
	}
	return result
}

func invoicePDFTerms(entity *invoice.Invoice, control *tenant.BillingControl) []string {
	rows := []string{
		labeledPDFLine("Payment Terms", string(entity.PaymentTerm)),
	}
	if invoicePDFShowDueDate(control) {
		rows = append(rows, labeledPDFLine("Due Date", unixDatePtr(entity.DueDate)))
	}
	return stringutils.FilterEmpty(rows)
}

func invoicePDFDueDate(entity *invoice.Invoice, control *tenant.BillingControl) string {
	if !invoicePDFShowDueDate(control) {
		return ""
	}
	return unixDatePtr(entity.DueDate)
}

func invoicePDFBalanceDue(entity *invoice.Invoice, control *tenant.BillingControl) string {
	if !invoicePDFShowBalanceDue(control) {
		return ""
	}
	return moneyString(entity.CurrencyCode, entity.OpenBalanceAmount().StringFixed(2))
}

func invoicePDFShowDueDate(control *tenant.BillingControl) bool {
	return control == nil || control.ShowDueDateOnInvoice
}

func invoicePDFShowBalanceDue(control *tenant.BillingControl) bool {
	return control == nil || control.ShowBalanceDueOnInvoice
}

func billingControlPDFInvoiceTerms(control *tenant.BillingControl) []string {
	if control == nil {
		return []string{}
	}
	return stringutils.FilterEmpty(strings.Split(control.DefaultInvoiceTerms, "\n"))
}

func billingControlPDFInvoiceFooter(control *tenant.BillingControl) string {
	if control == nil {
		return ""
	}
	return strings.TrimSpace(control.DefaultInvoiceFooter)
}

func shipmentCommodityPDFRows(shp *shipment.Shipment) []invoicePDFCommodityRow {
	if shp == nil || len(shp.Commodities) == 0 {
		return []invoicePDFCommodityRow{}
	}

	rows := make([]invoicePDFCommodityRow, 0, len(shp.Commodities))
	for _, item := range shp.Commodities {
		if item == nil {
			continue
		}
		rows = append(rows, invoicePDFCommodityRow{
			Quantity:         positiveInt64PDFString(item.Pieces),
			DescriptionLines: shipmentCommodityDescriptionLines(item),
			Weight:           positiveInt64PDFString(item.Weight),
			Class:            shipmentCommodityClass(item),
			PiecesValue:      item.Pieces,
			WeightValue:      item.Weight,
		})
	}
	return rows
}

func shipmentCommodityDescriptionLines(item *shipment.ShipmentCommodity) []string {
	name := "Commodity"
	var description string
	if item != nil && item.Commodity != nil {
		if strings.TrimSpace(item.Commodity.Name) != "" {
			name = item.Commodity.Name
		}
		description = item.Commodity.Description
	}
	if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(description)) {
		description = ""
	}
	return stringutils.FilterEmpty([]string{name, description})
}

func shipmentCommodityClass(item *shipment.ShipmentCommodity) string {
	if item == nil || item.Commodity == nil {
		return ""
	}

	freightClass := strings.TrimSpace(string(item.Commodity.FreightClass))
	freightClass = strings.TrimPrefix(freightClass, "Class")
	return strings.ReplaceAll(freightClass, "_", ".")
}

func chargePDFRows(entity *invoice.Invoice) []invoicePDFChargeRow {
	if entity.Detail == customer.InvoiceDetailSummary {
		return summaryChargePDFRows(entity)
	}

	rows := make([]invoicePDFChargeRow, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		rows = append(rows, invoicePDFChargeRow{
			Line:        strconv.Itoa(line.LineNumber),
			Description: line.Description,
			Quantity:    line.Quantity.StringFixed(2),
			UnitPrice:   moneyString(entity.CurrencyCode, line.UnitPrice.StringFixed(2)),
			Amount:      moneyString(entity.CurrencyCode, line.Amount.StringFixed(2)),
			// Attribution rides on the row itself so a template that prints no
			// manifest still says which shipment a charge came from. Blank on a
			// single-shipment invoice, where the header already answers it.
			ProNumber: shipmentProNumberForLine(entity, line),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, invoicePDFChargeRow{
			Description: "Invoice Total",
			Quantity:    "1.00",
			UnitPrice:   moneyString(entity.CurrencyCode, entity.TotalAmount.StringFixed(2)),
			Amount:      moneyString(entity.CurrencyCode, entity.TotalAmount.StringFixed(2)),
		})
	}
	return rows
}

// summaryChargePDFRows collapses an invoice to one line per shipment.
//
// This is what a customer on Summary asked for: the freight, not the accessorial
// breakdown behind it. The per-shipment amounts still sum to the invoice total,
// because they are the same line amounts added up rather than a separate figure.
//
// Order-level charges carry no shipment and keep their own rows, so a customs
// brokerage fee on a consolidated invoice stays visible instead of being folded
// into whichever shipment happened to be first.
func summaryChargePDFRows(entity *invoice.Invoice) []invoicePDFChargeRow {
	order := make([]string, 0, len(entity.Lines))
	totals := make(map[string]decimal.Decimal, len(entity.Lines))
	counts := make(map[string]int, len(entity.Lines))
	rows := make([]invoicePDFChargeRow, 0, len(entity.Lines))

	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		if line.ShipmentID.IsNil() {
			rows = append(rows, invoicePDFChargeRow{
				Description: line.Description,
				Quantity:    line.Quantity.StringFixed(2),
				UnitPrice:   moneyString(entity.CurrencyCode, line.UnitPrice.StringFixed(2)),
				Amount:      moneyString(entity.CurrencyCode, line.Amount.StringFixed(2)),
			})
			continue
		}

		key := line.ShipmentID.String()
		if _, seen := totals[key]; !seen {
			order = append(order, key)
		}
		totals[key] = totals[key].Add(line.Amount)
		counts[key]++
	}

	grouped := make([]invoicePDFChargeRow, 0, len(order))
	for _, key := range order {
		amount := totals[key]
		grouped = append(grouped, invoicePDFChargeRow{
			Description: summaryLineDescription(entity, key, counts[key]),
			Quantity:    "1.00",
			UnitPrice:   moneyString(entity.CurrencyCode, amount.StringFixed(2)),
			Amount:      moneyString(entity.CurrencyCode, amount.StringFixed(2)),
			ProNumber:   shipmentProNumber(entity, key),
		})
	}

	// Shipments first, order-level charges after, so the document reads freight
	// then extras rather than interleaving them by line number.
	out := append(grouped, rows...)
	for i := range out {
		out[i].Line = strconv.Itoa(i + 1)
	}

	return out
}

func summaryLineDescription(entity *invoice.Invoice, shipmentID string, charges int) string {
	pro := shipmentProNumber(entity, shipmentID)
	if pro == "" {
		pro = "Shipment"
	}
	if charges <= 1 {
		return pro
	}

	return fmt.Sprintf("%s (%d charges)", pro, charges)
}

// shipmentProNumber reads the PRO off whichever line carries it, because only the
// lines know which shipment they belong to.
func shipmentProNumber(entity *invoice.Invoice, shipmentID string) string {
	for _, line := range entity.Lines {
		if line == nil || line.ShipmentID.String() != shipmentID {
			continue
		}
		if line.ShipmentProNumber != "" {
			return line.ShipmentProNumber
		}
	}

	return ""
}

// shipmentProNumberForLine attributes one line, and says nothing on an invoice
// that bills a single shipment — the header already names it there, and repeating
// it on every row is noise.
func shipmentProNumberForLine(entity *invoice.Invoice, line *invoice.InvoiceLine) string {
	if entity.ShipmentCount <= 1 {
		return ""
	}

	return line.ShipmentProNumber
}

func attachmentPDFNames(entity *invoice.Invoice) []string {
	names := make([]string, 0, len(entity.Attachments))
	for _, attachment := range entity.Attachments {
		if attachment == nil || attachment.Document == nil {
			continue
		}
		names = append(names, attachment.Document.OriginalName)
	}
	return names
}

func cityStatePostal(city string, state string, postalCode string) string {
	left := strings.TrimSpace(city)
	statePostal := strings.TrimSpace(
		strings.Join(stringutils.FilterEmpty([]string{state, postalCode}), " "),
	)
	if left == "" {
		return statePostal
	}
	if statePostal == "" {
		return left
	}
	return left + ", " + statePostal
}

func labeledPDFLine(label string, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return label + ": " + strings.TrimSpace(value)
}

func customerString(cus *customer.Customer, read func(*customer.Customer) string) string {
	if cus == nil {
		return ""
	}
	return read(cus)
}

func customerState(cus *customer.Customer) string {
	if cus == nil || cus.State == nil {
		return ""
	}
	return cus.State.Abbreviation
}

func customerCountry(cus *customer.Customer) string {
	if cus == nil || cus.State == nil {
		return ""
	}
	return cus.State.CountryName
}

func organizationState(org *tenant.Organization) string {
	if org == nil || org.State == nil {
		return ""
	}
	return org.State.Abbreviation
}

func organizationCountry(org *tenant.Organization) string {
	if org == nil || org.State == nil {
		return ""
	}
	return org.State.CountryName
}

func organizationDOT(org *tenant.Organization) string {
	if org == nil {
		return ""
	}
	return org.DOTNumber
}

func organizationSCAC(org *tenant.Organization) string {
	if org == nil {
		return ""
	}
	return org.ScacCode
}

func locationState(loc *location.Location) string {
	if loc == nil || loc.State == nil {
		return ""
	}
	return loc.State.Abbreviation
}

func locationCountry(loc *location.Location) string {
	if loc == nil || loc.State == nil {
		return ""
	}
	return loc.State.CountryName
}

func positiveInt64PDFString(value int64) string {
	if value <= 0 {
		return ""
	}
	return intutils.FormatWithCommas(value)
}
