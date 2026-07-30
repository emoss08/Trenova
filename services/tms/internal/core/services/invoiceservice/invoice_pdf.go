package invoiceservice

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
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
	Subtotal      string
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
	if deliveryProfile != nil {
		cus = deliveryProfile.Customer
		org = deliveryProfile.Organization
		shp = deliveryProfile.Shipment
		control = deliveryProfile.BillingControl
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
		Shipper:       shipmentStopPDFAddressBlock(shp, true),
		Consignee:     shipmentStopPDFAddressBlock(shp, false),
		CommodityRows: shipmentCommodityPDFRows(shp),
		ChargeRows:    chargePDFRows(entity),
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
	return data
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
