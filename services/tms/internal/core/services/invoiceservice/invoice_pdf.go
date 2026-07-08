package invoiceservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/jung-kurt/gofpdf"
)

type invoicePDFData struct {
	InvoiceNumber string
	InvoiceDate   string
	DueDate       string
	PaymentTerm   string
	CurrencyCode  string
	Organization  invoicePDFAddressBlock
	Logo          *invoicePDFLogo
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

type invoicePDFLogo struct {
	Data      []byte
	ImageType string
	Width     int
	Height    int
}

type invoicePDFBox struct {
	X     float64
	Y     float64
	Width float64
	Title string
}

const (
	invoicePDFMaxLogoBytes       = 1 << 20
	invoicePDFContentX           = 10.0
	invoicePDFContentWidth       = 196.0
	invoicePDFLogoX              = 10.0
	invoicePDFLogoY              = 8.0
	invoicePDFLogoMaxWidth       = 72.0
	invoicePDFLogoMaxHeight      = 11.0
	invoicePDFOrgAddressX        = 11.0
	invoicePDFOrgAddressGap      = 2.5
	invoicePDFOrgAddressMinY     = 18.5
	invoicePDFHeaderDetailY      = 40.8
	invoicePDFHeaderBottomY      = 48.5
	invoicePDFHeaderDetailHeight = 4.2
	invoicePDFAddressBoxHeight   = 29.0
	invoicePDFShipmentBoxHeight  = 31.0
	invoicePDFShipmentLabelWidth = 7.5
	invoicePDFSectionBarHeight   = 4.6
)

func renderInvoicePDF(
	ctx context.Context,
	entity *invoice.Invoice,
	deliveryProfile *invoiceDeliveryProfile,
	storageClient storage.Client,
) ([]byte, error) {
	data := buildInvoicePDFDataWithLogo(ctx, entity, deliveryProfile, storageClient)
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetTitle("Invoice "+data.InvoiceNumber, false)
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()

	drawInvoicePDFHeader(pdf, data)
	drawInvoicePDFAddressSections(pdf, data)
	drawInvoicePDFShipmentSections(pdf, data)
	drawInvoicePDFCharges(pdf, data)
	drawInvoicePDFFooter(pdf, data)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildInvoicePDFData(entity *invoice.Invoice, deliveryProfile *invoiceDeliveryProfile) invoicePDFData {
	return buildInvoicePDFDataWithLogo(context.Background(), entity, deliveryProfile, nil)
}

func buildInvoicePDFDataWithLogo(
	ctx context.Context,
	entity *invoice.Invoice,
	deliveryProfile *invoiceDeliveryProfile,
	storageClient storage.Client,
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
		Logo:          resolveInvoicePDFLogo(ctx, org, storageClient),
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

func resolveInvoicePDFLogo(
	ctx context.Context,
	org *tenant.Organization,
	storageClient storage.Client,
) *invoicePDFLogo {
	if org == nil || strings.TrimSpace(org.LogoURL) == "" {
		return nil
	}

	body, contentType, err := loadInvoicePDFLogo(ctx, org.LogoURL, storageClient)
	if err != nil {
		return nil
	}
	logo, err := normalizeInvoicePDFLogo(body, contentType)
	if err != nil {
		return nil
	}
	return logo
}

func loadInvoicePDFLogo(
	ctx context.Context,
	logoURL string,
	storageClient storage.Client,
) ([]byte, string, error) {
	trimmed := strings.TrimSpace(logoURL)
	if isExternalInvoicePDFLogoURL(trimmed) {
		return loadExternalInvoicePDFLogo(ctx, trimmed)
	}
	if storageClient == nil {
		return nil, "", errors.New("invoice logo storage client is unavailable")
	}

	result, err := storageClient.Download(ctx, trimmed)
	if err != nil {
		return nil, "", err
	}
	defer result.Body.Close()

	body, err := io.ReadAll(io.LimitReader(result.Body, invoicePDFMaxLogoBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(body) > invoicePDFMaxLogoBytes {
		return nil, "", errors.New("invoice logo exceeds max size")
	}
	return body, result.ContentType, nil
}

func loadExternalInvoicePDFLogo(ctx context.Context, logoURL string) ([]byte, string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, logoURL, http.NoBody)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("invoice logo returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, invoicePDFMaxLogoBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(body) > invoicePDFMaxLogoBytes {
		return nil, "", errors.New("invoice logo exceeds max size")
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func normalizeInvoicePDFLogo(body []byte, contentType string) (*invoicePDFLogo, error) {
	if len(body) == 0 {
		return nil, errors.New("invoice logo is empty")
	}

	format := logoImageFormat(body, contentType)
	switch format {
	case "PNG":
		config, err := png.DecodeConfig(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		return &invoicePDFLogo{
			Data:      body,
			ImageType: "PNG",
			Width:     config.Width,
			Height:    config.Height,
		}, nil
	case "JPG":
		config, err := jpeg.DecodeConfig(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		return &invoicePDFLogo{
			Data:      body,
			ImageType: "JPG",
			Width:     config.Width,
			Height:    config.Height,
		}, nil
	case "WEBP":
		img, err := webp.Decode(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err = png.Encode(&buf, img); err != nil {
			return nil, err
		}
		bounds := img.Bounds()
		return &invoicePDFLogo{
			Data:      buf.Bytes(),
			ImageType: "PNG",
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
		}, nil
	default:
		if _, _, err := image.DecodeConfig(bytes.NewReader(body)); err == nil {
			return nil, errors.New("invoice logo image format is unsupported")
		}
		return nil, errors.New("invoice logo content type is unsupported")
	}
}

func logoImageFormat(body []byte, contentType string) string {
	detected := strings.ToLower(strings.TrimSpace(contentType))
	if detected == "" || detected == "application/octet-stream" {
		detected = http.DetectContentType(body)
	}
	switch {
	case strings.Contains(detected, "png"):
		return "PNG"
	case strings.Contains(detected, "jpeg"), strings.Contains(detected, "jpg"):
		return "JPG"
	case strings.Contains(detected, "webp"):
		return "WEBP"
	case bytes.HasPrefix(body, []byte{0x89, 'P', 'N', 'G'}):
		return "PNG"
	case bytes.HasPrefix(body, []byte{0xff, 0xd8, 0xff}):
		return "JPG"
	case len(body) >= 12 && string(body[:4]) == "RIFF" && string(body[8:12]) == "WEBP":
		return "WEBP"
	default:
		return ""
	}
}

func isExternalInvoicePDFLogoURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func drawInvoicePDFHeader(pdf *gofpdf.Fpdf, data invoicePDFData) {
	drawInvoicePDFLogo(pdf, data)

	pdf.SetXY(82, 10)
	pdf.SetFont("Helvetica", "B", 22)
	pdf.CellFormat(52, 9, "INVOICE", "", 0, "C", false, 0, "")

	drawInvoiceMetadataBox(pdf, invoicePDFBox{X: 144, Y: 10, Width: 62}, data)

	pdf.SetXY(invoicePDFContentX, invoicePDFHeaderDetailY)
	drawHeaderDetailLine(pdf, data.HeaderRows)
	pdf.SetY(invoicePDFHeaderBottomY)
}

func drawInvoicePDFAddressSections(pdf *gofpdf.Fpdf, data invoicePDFData) {
	y := pdf.GetY()
	drawAddressBox(pdf, invoicePDFBox{X: 10, Y: y, Width: 96, Title: "BILL TO"}, data.BillTo)
	drawAddressBox(pdf, invoicePDFBox{X: 110, Y: y, Width: 96, Title: "REMIT PAYMENT TO"}, data.RemitTo)
	pdf.SetY(y + 32)
}

func drawInvoicePDFShipmentSections(pdf *gofpdf.Fpdf, data invoicePDFData) {
	y := pdf.GetY()
	drawLabeledAddressBox(pdf, invoicePDFBox{X: 10, Y: y, Width: 96, Title: "SHIPPER"}, data.Shipper)
	drawLabeledAddressBox(pdf, invoicePDFBox{X: 110, Y: y, Width: 96, Title: "CONSIGNEE"}, data.Consignee)
	pdf.SetY(y + 33)
	drawShipmentCommodityTable(pdf, data.CommodityRows)
}

func drawInvoicePDFCharges(pdf *gofpdf.Fpdf, data invoicePDFData) {
	pdf.Ln(1.6)
	drawSectionBar(pdf, "CHARGES")
	pdf.SetFont("Helvetica", "B", 7.8)
	pdf.CellFormat(16, 5.0, "LINE", "1", 0, "", false, 0, "")
	pdf.CellFormat(88, 5.0, "DESCRIPTION", "1", 0, "", false, 0, "")
	pdf.CellFormat(24, 5.0, "QTY", "1", 0, "R", false, 0, "")
	pdf.CellFormat(34, 5.0, "UNIT", "1", 0, "R", false, 0, "")
	pdf.CellFormat(34, 5.0, "AMOUNT", "1", 1, "R", false, 0, "")
	pdf.SetFont("Helvetica", "", 7.8)
	for _, row := range data.ChargeRows {
		pdf.CellFormat(16, 5.1, row.Line, "1", 0, "", false, 0, "")
		pdf.CellFormat(88, 5.1, row.Description, "1", 0, "", false, 0, "")
		pdf.CellFormat(24, 5.1, row.Quantity, "1", 0, "R", false, 0, "")
		pdf.CellFormat(34, 5.1, row.UnitPrice, "1", 0, "R", false, 0, "")
		pdf.CellFormat(34, 5.1, row.Amount, "1", 1, "R", false, 0, "")
	}
	pdf.Ln(1.0)
	drawTotalLine(pdf, "Subtotal", data.Subtotal, false)
	drawTotalLine(pdf, "Accessorial/Other", data.Other, false)
	if data.DueDate != "" && len(data.InvoiceTerms) > 0 {
		drawTotalLine(pdf, "Due Date", invoicePDFMetadataDate(data.DueDate), false)
	}
	showBalanceDue := data.BalanceDue != ""
	drawTotalLine(pdf, "Total", data.Total, !showBalanceDue)
	if showBalanceDue {
		drawTotalLine(pdf, "Balance Due", data.BalanceDue, true)
	}
}

func drawInvoicePDFFooter(pdf *gofpdf.Fpdf, data invoicePDFData) {
	y := pdf.GetY() + 2
	if len(data.InvoiceTerms) > 0 {
		y = drawInvoicePDFTermsAndConditions(pdf, y, data.InvoiceTerms)
		drawInvoicePDFFooterText(pdf, data.InvoiceFooter, y)
		return
	}

	drawTextBox(pdf, invoicePDFBox{X: 10, Y: y, Width: 62, Title: "TERMS"}, data.Terms)
	drawTextBox(
		pdf,
		invoicePDFBox{X: 76, Y: y, Width: 62, Title: "SIGNATURE"},
		[]string{"Received in good order.", "Signature:", "Date:"},
	)
	lines := append([]string{}, data.Notes...)
	if len(data.Attachments) > 0 {
		lines = append(lines, "Supporting Documents:")
		lines = append(lines, data.Attachments...)
	}
	drawTextBox(pdf, invoicePDFBox{X: 144, Y: y, Width: 62, Title: "INTERNAL USE"}, append(
		stringutils.FilterEmpty([]string{
			"Currency: " + data.CurrencyCode,
			"Invoice: " + data.InvoiceNumber,
		}),
		lines...,
	))
	drawInvoicePDFFooterText(pdf, data.InvoiceFooter, y+44)
}

func drawInvoicePDFLogo(pdf *gofpdf.Fpdf, data invoicePDFData) {
	addressY := invoicePDFOrgAddressMinY
	drawnLogo := false
	if data.Logo != nil && len(data.Logo.Data) > 0 {
		width, height := fittedImageSize(
			float64(data.Logo.Width),
			float64(data.Logo.Height),
			invoicePDFLogoMaxWidth,
			invoicePDFLogoMaxHeight,
		)
		info := pdf.RegisterImageOptionsReader(
			"organization-logo",
			gofpdf.ImageOptions{ImageType: data.Logo.ImageType},
			bytes.NewReader(data.Logo.Data),
		)
		if info != nil {
			pdf.ImageOptions("organization-logo", invoicePDFLogoX, invoicePDFLogoY, width, height, false, gofpdf.ImageOptions{
				ImageType: data.Logo.ImageType,
			}, 0, "")
			addressY = invoicePDFLogoY + height + invoicePDFOrgAddressGap
			drawnLogo = true
		}
	}

	if !drawnLogo && data.Organization.Name != "" {
		pdf.SetXY(invoicePDFLogoX, invoicePDFLogoY+2)
		pdf.SetFont("Helvetica", "B", 15)
		pdf.CellFormat(58, 7, data.Organization.Name, "", 1, "", false, 0, "")
		addressY = invoicePDFLogoY + 9 + invoicePDFOrgAddressGap
	}

	if addressY < invoicePDFOrgAddressMinY {
		addressY = invoicePDFOrgAddressMinY
	}

	drawOrganizationHeaderLines(pdf, data.Organization, addressY, drawnLogo)
}

func fittedImageSize(
	sourceWidth float64,
	sourceHeight float64,
	maxWidth float64,
	maxHeight float64,
) (float64, float64) {
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return maxWidth, maxHeight
	}

	scale := maxWidth / sourceWidth
	if sourceHeight*scale > maxHeight {
		scale = maxHeight / sourceHeight
	}
	return sourceWidth * scale, sourceHeight * scale
}

func drawOrganizationHeaderLines(
	pdf *gofpdf.Fpdf,
	block invoicePDFAddressBlock,
	y float64,
	includeName bool,
) {
	lines := make([]string, 0, len(block.Lines)+1)
	if includeName {
		lines = append(lines, block.Name)
	}
	lines = append(lines, block.Lines...)
	lines = stringutils.FilterEmpty(lines)

	pdf.SetXY(invoicePDFOrgAddressX, y)
	for index, line := range lines {
		pdf.SetX(invoicePDFOrgAddressX)
		if includeName && index == 0 {
			pdf.SetFont("Helvetica", "B", 8.1)
		} else {
			pdf.SetFont("Helvetica", "B", 7.8)
		}
		pdf.CellFormat(74, 4, line, "", 1, "", false, 0, "")
	}
}

func drawInvoiceMetadataBox(pdf *gofpdf.Fpdf, box invoicePDFBox, data invoicePDFData) {
	const (
		topRowHeight    = 6.8
		dateLabelHeight = 4.8
		dateValueHeight = 5.7
		proRowHeight    = 7.0
	)

	totalHeight := topRowHeight + dateLabelHeight + dateValueHeight + proRowHeight
	splitWidth := box.Width / 2
	splitX := box.X + splitWidth
	dateLabelY := box.Y + topRowHeight
	dateValueY := dateLabelY + dateLabelHeight
	proY := dateValueY + dateValueHeight

	pdf.Rect(box.X, box.Y, box.Width, totalHeight, "D")
	pdf.Line(splitX, box.Y, splitX, box.Y+totalHeight)
	pdf.Line(box.X, dateLabelY, box.X+box.Width, dateLabelY)
	pdf.Line(box.X, dateValueY, box.X+box.Width, dateValueY)
	pdf.Line(box.X, proY, box.X+box.Width, proY)

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "B", 6.8)
	pdf.SetXY(box.X+2, box.Y+1.4)
	pdf.CellFormat(splitWidth-4, 3.8, "INVOICE NO.", "", 0, "", false, 0, "")
	pdf.SetFont("Helvetica", "B", 8.2)
	pdf.SetXY(splitX, box.Y+1)
	pdf.CellFormat(splitWidth, 4.4, data.InvoiceNumber, "", 0, "C", false, 0, "")

	pdf.SetFont("Helvetica", "B", 6.8)
	pdf.SetXY(box.X, dateLabelY+0.4)
	pdf.CellFormat(splitWidth, 3.8, "DATE", "", 0, "C", false, 0, "")
	pdf.CellFormat(splitWidth, 3.8, "PAGE", "", 0, "C", false, 0, "")

	pdf.SetFont("Helvetica", "B", 8.0)
	pdf.SetXY(box.X, dateValueY+0.6)
	pdf.CellFormat(splitWidth, 4.3, invoicePDFMetadataDate(data.InvoiceDate), "", 0, "C", false, 0, "")
	pdf.CellFormat(splitWidth, 4.3, "1 of 1", "", 0, "C", false, 0, "")

	pdf.SetFont("Helvetica", "B", 6.8)
	pdf.SetXY(box.X+2, proY+1.6)
	pdf.CellFormat(splitWidth-4, 3.8, "PRO NO.", "", 0, "", false, 0, "")
	pdf.SetFont("Helvetica", "B", 8.0)
	pdf.SetXY(splitX, proY+1.4)
	pdf.CellFormat(splitWidth, 4, proHeaderValue(data.HeaderRows), "", 0, "C", false, 0, "")
}

func drawHeaderDetailLine(pdf *gofpdf.Fpdf, rows []invoicePDFKeyValue) {
	y := pdf.GetY()
	index := 0
	for _, row := range rows {
		if strings.TrimSpace(row.Value) == "" || row.Label == "PRO" {
			continue
		}
		drawHeaderDetailItem(pdf, row, headerDetailX(row.Label, index), y)
		index++
	}
	pdf.SetY(y + invoicePDFHeaderDetailHeight + 1)
}

func drawHeaderDetailItem(pdf *gofpdf.Fpdf, row invoicePDFKeyValue, x float64, y float64) {
	label := row.Label + ":"
	pdf.SetXY(x, y)
	pdf.SetFont("Helvetica", "B", 7.8)
	labelWidth := pdf.GetStringWidth(label) + 1.2
	pdf.CellFormat(labelWidth, invoicePDFHeaderDetailHeight, label, "", 0, "", false, 0, "")

	pdf.SetFont("Helvetica", "", 7.8)
	pdf.CellFormat(42-labelWidth, invoicePDFHeaderDetailHeight, row.Value, "", 0, "", false, 0, "")
}

func headerDetailX(label string, fallbackIndex int) float64 {
	switch label {
	case "DOT":
		return 16
	case "SCAC":
		return 58
	case "Payment Terms":
		return 112
	case "PRO":
		return 142
	default:
		return 16 + float64(fallbackIndex)*42
	}
}

func drawLabeledAddressBox(pdf *gofpdf.Fpdf, box invoicePDFBox, block invoicePDFAddressBlock) {
	pdf.SetXY(box.X, box.Y)
	pdf.SetFillColor(0, 0, 0)
	labelWidth := invoicePDFShipmentLabelWidth
	boxHeight := invoicePDFShipmentBoxHeight
	pdf.Rect(box.X, box.Y, labelWidth, boxHeight, "F")
	pdf.Rect(box.X+labelWidth, box.Y, box.Width-labelWidth, boxHeight, "D")

	pdf.SetFont("Helvetica", "B", 8.2)
	pdf.SetTextColor(255, 255, 255)
	drawRotatedLabel(pdf, box.X, box.Y, labelWidth, boxHeight, box.Title)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetXY(box.X+labelWidth+3.2, box.Y+3.1)
	lineCount := 5
	if len(block.Details) > 0 {
		lineCount = 4
	}
	drawPlainAddressLines(pdf, box.Width-labelWidth-6.4, block, lineCount)
	drawAddressDetails(pdf, box, labelWidth, block.Details)
}

func drawAddressBox(pdf *gofpdf.Fpdf, box invoicePDFBox, block invoicePDFAddressBlock) {
	boxHeight := invoicePDFAddressBoxHeight
	title := box.Title + ":"
	pdf.Rect(box.X, box.Y, box.Width, boxHeight, "D")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "B", 8.4)
	pdf.SetXY(box.X+3.2, box.Y+2.6)
	pdf.CellFormat(box.Width-6.4, 4.2, title, "", 1, "", false, 0, "")
	titleWidth := pdf.GetStringWidth(title)
	pdf.Line(box.X+3.2, box.Y+7.2, box.X+3.2+titleWidth, box.Y+7.2)

	pdf.SetXY(box.X+3.2, box.Y+8.7)
	drawPlainAddressLines(pdf, box.Width-6.4, block, 4)
}

func drawRotatedLabel(
	pdf *gofpdf.Fpdf,
	x float64,
	y float64,
	width float64,
	height float64,
	label string,
) {
	centerX := x + width/2
	centerY := y + height/2
	labelWidth := pdf.GetStringWidth(label)
	labelHeight := 4.2
	pdf.TransformBegin()
	pdf.TransformRotate(90, centerX, centerY)
	pdf.SetXY(centerX-labelWidth/2, centerY-labelHeight/2)
	pdf.CellFormat(labelWidth, labelHeight, label, "", 0, "C", false, 0, "")
	pdf.TransformEnd()
}

func drawPlainAddressLines(pdf *gofpdf.Fpdf, width float64, block invoicePDFAddressBlock, lineCount int) {
	lines := append([]string{}, block.Name)
	lines = append(lines, block.Lines...)
	lines = stringutils.FilterEmpty(lines)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 8.2)
	for i := range lineCount {
		if i >= len(lines) {
			break
		}
		x := pdf.GetX()
		pdf.CellFormat(width, 4.45, lines[i], "", 1, "", false, 0, "")
		pdf.SetX(x)
	}
}

func drawAddressDetails(
	pdf *gofpdf.Fpdf,
	box invoicePDFBox,
	labelWidth float64,
	rows []invoicePDFKeyValue,
) {
	if len(rows) == 0 {
		return
	}

	detailOffset := 42.0
	detailX := box.X + labelWidth + detailOffset
	detailY := box.Y + 21.0
	for _, row := range rows {
		if strings.TrimSpace(row.Value) == "" {
			continue
		}
		label := row.Label + ":"
		pdf.SetXY(detailX, detailY)
		pdf.SetFont("Helvetica", "B", 8.2)
		labelCellWidth := pdf.GetStringWidth(label) + 1.5
		pdf.CellFormat(labelCellWidth, 4.4, label, "", 0, "", false, 0, "")
		pdf.SetFont("Helvetica", "", 8.2)
		pdf.CellFormat(box.Width-labelWidth-detailOffset-labelCellWidth-3, 4.4, row.Value, "", 1, "", false, 0, "")
		detailY += 4.6
	}
}

func drawSectionBar(pdf *gofpdf.Fpdf, title string) {
	pdf.SetFillColor(0, 0, 0)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 8.2)
	pdf.CellFormat(invoicePDFContentWidth, invoicePDFSectionBarHeight, title, "1", 1, "", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

func drawShipmentCommodityTable(pdf *gofpdf.Fpdf, rows []invoicePDFCommodityRow) {
	if len(rows) == 0 {
		return
	}

	drawSectionBar(pdf, "SHIPMENT INFORMATION")
	pdf.SetFont("Helvetica", "B", 7.8)
	pdf.CellFormat(22, 5.4, "QUANTITY", "1", 0, "C", false, 0, "")
	pdf.CellFormat(14, 5.4, "TYPE", "1", 0, "C", false, 0, "")
	pdf.CellFormat(98, 5.4, "DESCRIPTION OF ARTICLES, SPECIAL MARKS AND EXCEPTIONS", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 5.4, "WEIGHT (LBS)", "1", 0, "C", false, 0, "")
	pdf.CellFormat(18, 5.4, "NMFC #", "1", 0, "C", false, 0, "")
	pdf.CellFormat(14, 5.4, "CLASS", "1", 1, "C", false, 0, "")

	for _, row := range rows {
		drawShipmentCommodityRow(pdf, row)
	}
	drawShipmentCommodityTotals(pdf, rows)
}

func drawShipmentCommodityRow(pdf *gofpdf.Fpdf, row invoicePDFCommodityRow) {
	const rowHeight = 15.4

	startX := pdf.GetX()
	y := pdf.GetY()

	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(22, rowHeight, row.Quantity, "1", 0, "C", false, 0, "")
	pdf.CellFormat(14, rowHeight, row.Type, "1", 0, "C", false, 0, "")

	descriptionX := startX + 36
	pdf.Rect(descriptionX, y, 98, rowHeight, "D")
	pdf.SetXY(descriptionX+3, y+2.4)
	pdf.SetFont("Helvetica", "", 8)
	for index, line := range wrappedPDFLines(pdf, row.DescriptionLines, 92) {
		if index >= 3 {
			break
		}
		pdf.CellFormat(92, 4.1, line, "", 1, "", false, 0, "")
		pdf.SetX(descriptionX + 3)
	}

	pdf.SetXY(startX+134, y)
	pdf.CellFormat(30, rowHeight, row.Weight, "1", 0, "C", false, 0, "")
	pdf.CellFormat(18, rowHeight, row.NMFC, "1", 0, "C", false, 0, "")
	pdf.CellFormat(14, rowHeight, row.Class, "1", 1, "C", false, 0, "")
}

func drawShipmentCommodityTotals(pdf *gofpdf.Fpdf, rows []invoicePDFCommodityRow) {
	var totalPieces int64
	var totalWeight int64
	for _, row := range rows {
		if row.PiecesValue > 0 {
			totalPieces += row.PiecesValue
		}
		if row.WeightValue > 0 {
			totalWeight += row.WeightValue
		}
	}

	pdf.SetFont("Helvetica", "B", 8.4)
	pdf.CellFormat(22, 6.2, "TOTALS", "1", 0, "C", false, 0, "")
	pdf.CellFormat(14, 6.2, positiveInt64PDFString(totalPieces), "1", 0, "C", false, 0, "")
	pdf.CellFormat(98, 6.2, "", "1", 0, "", false, 0, "")
	pdf.CellFormat(30, 6.2, positiveInt64PDFString(totalWeight), "1", 0, "C", false, 0, "")
	pdf.CellFormat(18, 6.2, "", "1", 0, "", false, 0, "")
	pdf.CellFormat(14, 6.2, "", "1", 1, "", false, 0, "")
}

func drawTextBox(pdf *gofpdf.Fpdf, box invoicePDFBox, lines []string) {
	pdf.SetXY(box.X, box.Y)
	pdf.SetFont("Helvetica", "B", 8.2)
	pdf.CellFormat(box.Width, 4.7, box.Title, "1", 1, "", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	for i := range 8 {
		value := ""
		if i < len(lines) {
			value = lines[i]
		}
		pdf.SetX(box.X)
		pdf.CellFormat(box.Width, 4.8, value, "LR", 1, "", false, 0, "")
	}
	pdf.SetX(box.X)
	pdf.CellFormat(box.Width, 1, "", "T", 1, "", false, 0, "")
}

func drawInvoicePDFTermsAndConditions(pdf *gofpdf.Fpdf, y float64, lines []string) float64 {
	const (
		bodyHeight      = 35.5
		bodyPaddingX    = 6.0
		bodyPaddingY    = 2.8
		lineHeight      = 3.9
		signatureHeight = 9.2
	)

	pdf.SetXY(invoicePDFContentX, y)
	drawSectionBar(pdf, "TERMS AND CONDITIONS")

	bodyY := pdf.GetY()
	pdf.Rect(invoicePDFContentX, bodyY, invoicePDFContentWidth, bodyHeight, "D")

	textX := invoicePDFContentX + bodyPaddingX
	textY := bodyY + bodyPaddingY
	textWidth := invoicePDFContentWidth - bodyPaddingX*2
	signatureY := bodyY + bodyHeight - signatureHeight
	maxTextLines := int((signatureY - textY - 1) / lineHeight)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetXY(textX, textY)
	for index, line := range wrappedPDFLines(pdf, lines, textWidth) {
		if index >= maxTextLines {
			break
		}
		pdf.SetX(textX)
		pdf.CellFormat(textWidth, lineHeight, line, "", 1, "", false, 0, "")
	}

	leftLineX := textX
	rightLineX := invoicePDFContentX + 132
	lineY := signatureY + 3.0
	pdf.Line(leftLineX, lineY, leftLineX+122, lineY)
	pdf.Line(rightLineX, lineY, invoicePDFContentX+invoicePDFContentWidth-9, lineY)

	pdf.SetFont("Helvetica", "", 6.8)
	pdf.SetXY(leftLineX, lineY+2.6)
	pdf.CellFormat(122, 3.5, "SIGNATURE / NAME (PRINT)", "", 0, "", false, 0, "")
	pdf.SetXY(rightLineX, lineY+2.6)
	pdf.CellFormat(invoicePDFContentX+invoicePDFContentWidth-rightLineX-9, 3.5, "DATE", "", 0, "", false, 0, "")

	return bodyY + bodyHeight
}

func drawInvoicePDFFooterText(pdf *gofpdf.Fpdf, footer string, minY float64) {
	trimmed := strings.TrimSpace(footer)
	if trimmed == "" {
		return
	}

	pdf.SetAutoPageBreak(false, 0)

	_, pageHeight := pdf.GetPageSize()
	lineHeight := 4.0
	lines := wrappedPDFLines(pdf, []string{trimmed}, invoicePDFContentWidth)
	if len(lines) > 2 {
		lines = lines[:2]
	}

	y := pageHeight - 13
	contentHeight := float64(len(lines)) * lineHeight
	if minY+1.5 > y {
		y = minY + 1.5
	}
	if y+contentHeight > pageHeight-5.5 {
		y = pageHeight - 5.5 - contentHeight
	}

	pdf.SetFont("Helvetica", "I", 8.2)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetXY(invoicePDFContentX, y)
	for _, line := range lines {
		pdf.CellFormat(invoicePDFContentWidth, lineHeight, line, "", 1, "C", false, 0, "")
		pdf.SetX(invoicePDFContentX)
	}
}

func drawTotalLine(pdf *gofpdf.Fpdf, label string, amount string, bold bool) {
	if bold {
		pdf.SetFont("Helvetica", "B", 8.5)
	} else {
		pdf.SetFont("Helvetica", "", 7.8)
	}
	pdf.CellFormat(162, 5.2, label, "", 0, "R", false, 0, "")
	pdf.CellFormat(34, 5.2, amount, "B", 1, "R", false, 0, "")
}

func billToPDFAddressBlock(entity *invoice.Invoice, cus *customer.Customer) invoicePDFAddressBlock {
	name := strings.TrimSpace(entity.BillToName)
	if name == "" && cus != nil {
		name = cus.Name
	}
	lines := []string{
		stringutils.FirstNonEmpty(entity.BillToAddressLine1, customerString(cus, func(c *customer.Customer) string {
			return c.AddressLine1
		})),
		stringutils.FirstNonEmpty(entity.BillToAddressLine2, customerString(cus, func(c *customer.Customer) string {
			return c.AddressLine2
		})),
		cityStatePostal(
			stringutils.FirstNonEmpty(entity.BillToCity, customerString(cus, func(c *customer.Customer) string {
				return c.City
			})),
			stringutils.FirstNonEmpty(entity.BillToState, customerState(cus)),
			stringutils.FirstNonEmpty(entity.BillToPostalCode, customerString(cus, func(c *customer.Customer) string {
				return c.PostalCode
			})),
		),
		stringutils.FirstNonEmpty(entity.BillToCountry, customerCountry(cus)),
	}
	return invoicePDFAddressBlock{Name: name, Lines: stringutils.FilterEmpty(lines)}
}

func remitPDFAddressBlock(org *tenant.Organization, remittanceInstructions string) invoicePDFAddressBlock {
	block := organizationPDFAddressBlock(org)
	block.Lines = append(block.Lines, stringutils.FilterEmpty(strings.Split(remittanceInstructions, "\n"))...)
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

func proHeaderValue(rows []invoicePDFKeyValue) string {
	for _, row := range rows {
		if row.Label == "PRO" {
			return row.Value
		}
	}
	return ""
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
			Line:        fmt.Sprintf("%d", line.LineNumber),
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

func wrappedPDFLines(pdf *gofpdf.Fpdf, lines []string, width float64) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		for _, splitLine := range pdf.SplitLines([]byte(line), width) {
			result = append(result, string(splitLine))
		}
	}
	return result
}

func cityStatePostal(city string, state string, postalCode string) string {
	left := strings.TrimSpace(city)
	statePostal := strings.TrimSpace(strings.Join(stringutils.FilterEmpty([]string{state, postalCode}), " "))
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

func invoicePDFMetadataDate(value string) string {
	trimmed := strings.TrimSpace(value)
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return trimmed
	}
	return parsed.Format("01/02/2006")
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
