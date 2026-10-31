package reviewprompt

import (
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuereview"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const commentRunes = 400

// FreightCode names the line haul where a charge code is asked for.
const FreightCode = "FREIGHT"

var sourceText = map[billingqueue.ChargeSource]string{
	billingqueue.ChargeSourceRating:    "rated by the tariff",
	billingqueue.ChargeSourceAgreement: "from the rate agreement",
	billingqueue.ChargeSourceFuel:      "from the fuel surcharge program",
	billingqueue.ChargeSourceDetention: "detention",
	billingqueue.ChargeSourceQuote:     "from the rate quote",
	billingqueue.ChargeSourceNone:      "no pricing source",
}

// Sections is the material a review reads, the checks' findings apart from
// the evidence they were drawn from.
func Sections(e *Evidence) []llmports.ContextSection {
	sections := []llmports.ContextSection{
		{Title: "The invoice", Trusted: true, Content: invoiceText(e)},
		{
			Title:   "What the five checks found (the biller already sees this)",
			Trusted: true,
			Content: checksText(e.Item),
		},
	}
	if shp := e.Item.Shipment; shp != nil {
		sections = append(sections,
			llmports.ContextSection{Title: "The shipment", Trusted: true, Content: shipmentText(shp)},
		)
		if stops := stopsText(shp); stops != "" {
			sections = append(sections,
				llmports.ContextSection{Title: "The stops (times in UTC)", Trusted: true, Content: stops},
			)
		}
	}
	if terms := termsText(e); terms != "" {
		sections = append(sections, llmports.ContextSection{
			Title:   PayerName(e) + "'s billing terms",
			Trusted: true,
			Content: terms,
		})
	}
	if profile := billingProfile(e); profile != nil && strings.TrimSpace(profile.BillingNotes) != "" {
		sections = append(sections, llmports.ContextSection{
			Title:   "Billing notes kept for " + PayerName(e),
			Content: strings.TrimSpace(profile.BillingNotes),
		})
	}
	if len(e.Comments) > 0 {
		sections = append(sections, llmports.ContextSection{
			Title:   "Shipment comments, newest first",
			Content: commentsText(e.Comments),
		})
	}
	if len(e.Recent) > 0 {
		sections = append(sections, llmports.ContextSection{
			Title:   "Recent invoices to " + PayerName(e) + ", newest first",
			Trusted: true,
			Content: recentText(e.Recent),
		})
	}
	if len(e.Catalog) > 0 {
		lines := make([]string, 0, len(e.Catalog))
		for _, charge := range e.Catalog {
			lines = append(lines, charge.ID.String()+" · "+charge.Code+" · "+charge.Description)
		}
		sections = append(sections, llmports.ContextSection{
			Title:   "Accessorial catalog",
			Trusted: true,
			Content: strings.Join(lines, "\n"),
		})
	}
	for _, doc := range e.Documents {
		sections = append(sections, llmports.ContextSection{
			Title:   documentHeader(doc),
			Content: documentText(e.Excerpts, doc.ID),
		})
	}

	return sections
}

func invoiceText(e *Evidence) string {
	item := e.Item
	lines := []string{
		"Bill to: " + PayerName(e),
		"Shipment: " + ShipmentRef(e),
		"Amount: " + billingqueuereview.FormatMoney(item.AllocatedTotalAmount),
	}
	if item.Shipment != nil && item.Shipment.BOL != "" {
		lines = append(lines, "BOL: "+item.Shipment.BOL)
	}
	if item.PONumber != "" {
		lines = append(lines, "Customer PO: "+item.PONumber)
	}
	charges := item.Review.Charges
	if charges == nil {
		return strings.Join(lines, "\n")
	}

	byID := additionalCharges(item.Shipment)
	for _, line := range charges.Lines {
		if line.Removed {
			continue
		}
		lines = append(lines, chargeText(line, byID[line.AdditionalChargeID])...)
	}
	if !charges.HasRateCon {
		lines = append(lines, "Nothing priced this load to compare against.")
	}

	return strings.Join(lines, "\n")
}

func chargeText(line *billingqueue.ChargeLine, charge *shipment.AdditionalCharge) []string {
	text := "- " + line.Label + ": billed " + billingqueuereview.FormatMoney(line.Billed)
	if line.Basis != "" {
		text += " (" + line.Basis + ")"
	}
	if charge != nil && charge.AccessorialCharge != nil && charge.AccessorialCharge.Code != "" {
		text += " · code " + charge.AccessorialCharge.Code
	}
	source := sourceText[line.Source]
	if line.Source == billingqueue.ChargeSourceNone && charge != nil && !charge.IsSystemGenerated {
		source = "entered by hand"
	}
	if source != "" {
		text += " · " + source
	}
	if line.Expected.Valid {
		text += " · its source priced it at " + billingqueuereview.FormatMoney(line.Expected.Decimal)
	}

	out := []string{text}
	if charge != nil && charge.FuelSurchargeDetail != nil {
		out = append(out, "  "+fuelText(charge.FuelSurchargeDetail))
	}

	return out
}

func fuelText(fuel *shipment.FuelSurchargeDetail) string {
	parts := []string{"fuel surcharge: program " + firstOf(fuel.ProgramName, fuel.ProgramCode)}
	if fuel.Method != "" {
		parts = append(parts, "method "+fuel.Method)
	}
	if fuel.IndexCode != "" {
		parts = append(parts, "index "+fuel.IndexCode+" at "+number(fuel.Price)+" on "+fuel.PriceDate)
	}
	if fuel.PegPrice != nil {
		parts = append(parts, "peg "+number(*fuel.PegPrice))
	}
	if fuel.Miles != nil {
		parts = append(parts, number(*fuel.Miles)+" miles")
	}
	if fuel.RatePerMile != nil {
		parts = append(parts, number(*fuel.RatePerMile)+" per mile")
	}
	if fuel.Percent != nil {
		parts = append(parts, number(*fuel.Percent)+"% of "+firstOf(fuel.PercentBasis, "the base"))
	}
	parts = append(parts, "computed "+number(fuel.Amount))
	if fuel.CapApplied {
		parts = append(parts, "capped")
	}
	if fuel.FloorApplied {
		parts = append(parts, "floored")
	}
	if fuel.Stale {
		parts = append(parts, "index price is stale")
	}
	if fuel.UsedFallback {
		parts = append(parts, "used a fallback index")
	}
	if fuel.TermsOverridden {
		parts = append(parts, "the contract overrides the program's terms")
	}

	return strings.Join(parts, " · ")
}

func checksText(item *billingqueue.BillingQueueItem) string {
	lines := make([]string, 0, len(item.Review.Checks)+len(item.Review.Issues))
	for _, check := range item.Review.Checks {
		lines = append(lines, string(check.Key)+": "+string(check.State)+" · "+check.Detail)
	}
	for _, issue := range item.Review.Issues {
		switch {
		case !issue.IsOpen():
		case issue.Code.IsSuggestion():
			lines = append(lines, "suggested, not required: "+issue.Summary)
		default:
			lines = append(lines, "open: "+issue.Summary)
		}
	}

	return strings.Join(lines, "\n")
}

func commentsText(comments []*shipment.ShipmentComment) string {
	lines := make([]string, 0, len(comments))
	for _, comment := range comments {
		author := "someone"
		if comment.User != nil && comment.User.Name != "" {
			author = comment.User.Name
		}
		text := timeutils.FormatInstantUTC(comment.CreatedAt) + " · " + author + " · " +
			string(comment.Type) + ", " + string(comment.Visibility)
		if comment.ResolvedAt != nil {
			text += ", resolved"
		}
		lines = append(lines, text+": "+stringutils.Ellipsize(strings.TrimSpace(comment.Comment), commentRunes))
	}

	return strings.Join(lines, "\n")
}

func recentText(recent []*billingqueueports.RecentInvoice) string {
	lines := make([]string, 0, len(recent))
	for _, invoice := range recent {
		lines = append(lines, InvoiceLine(invoice, nil))
	}

	return strings.Join(lines, "\n")
}

// InvoiceLine is one past invoice and what it billed; keep, when given,
// picks the charges shown.
func InvoiceLine(
	invoice *billingqueueports.RecentInvoice,
	keep func(*billingqueueports.RecentCharge) bool,
) string {
	charges := make([]string, 0, len(invoice.Charges))
	for _, charge := range invoice.Charges {
		if keep != nil && !keep(charge) {
			continue
		}
		charges = append(charges, ChargeLabel(charge)+" "+billingqueuereview.FormatMoney(charge.Amount))
	}
	text := invoice.Number + " · " + timeutils.FormatCalendarDate(invoice.InvoiceDate, nil) +
		" · total " + billingqueuereview.FormatMoney(invoice.Total)
	if len(charges) == 0 {
		return text
	}

	return text + ": " + strings.Join(charges, "; ")
}

// ChargeLabel names a past charge by its code and description.
func ChargeLabel(charge *billingqueueports.RecentCharge) string {
	label := strings.TrimSpace(charge.Code + " " + charge.Description)
	if label == "" {
		return string(charge.Type)
	}

	return label
}

func documentHeader(doc *document.Document) string {
	header := "Document " + doc.ID.String() + " · " + DocumentKind(doc) + " · signature " +
		string(doc.SignatureStatus)
	if recorded := factsText(doc.DeliveryFacts); recorded != "" {
		header += " · recorded: " + recorded
	}

	return header
}

func factsText(facts *document.DeliveryFacts) string {
	if facts == nil {
		return ""
	}
	parts := make([]string, 0, 8)
	if facts.SignedBy != "" {
		parts = append(parts, "signed by "+facts.SignedBy)
	}
	if facts.Pieces != "" {
		parts = append(parts, facts.Pieces+" pieces")
	}
	if facts.OSD != "" {
		osd := "over/short/damaged " + string(facts.OSD)
		if facts.OSDNote != "" {
			osd += " (" + facts.OSDNote + ")"
		}
		parts = append(parts, osd)
	}
	if facts.Seal != "" {
		parts = append(parts, "seal "+facts.Seal)
	}
	if facts.ArrivedAt != nil {
		parts = append(parts, "arrived "+timeutils.FormatInstantUTC(*facts.ArrivedAt))
	}
	if facts.DepartedAt != nil {
		parts = append(parts, "departed "+timeutils.FormatInstantUTC(*facts.DepartedAt))
	}
	if facts.Amount.Valid {
		receipt := "receipt " + billingqueuereview.FormatMoney(facts.Amount.Decimal)
		if facts.Reference != "" {
			receipt += " ref " + facts.Reference
		}
		if facts.ChargeLabel != "" {
			receipt += " for " + facts.ChargeLabel
		}
		parts = append(parts, receipt)
	}

	return strings.Join(parts, ", ")
}

func documentText(excerpts map[pulid.ID]Excerpt, id pulid.ID) string {
	excerpt, ok := excerpts[id]
	if !ok {
		return "(no text was read from this document)"
	}
	if excerpt.Parts <= 1 {
		return excerpt.Text
	}

	return excerpt.Text + "\n(part 1 of " + strconv.Itoa(excerpt.Parts) + "; read_document reads the rest)"
}

// PayerName is the payer the item bills.
func PayerName(e *Evidence) string {
	if e.Item.BillToCustomer != nil && e.Item.BillToCustomer.Name != "" {
		return e.Item.BillToCustomer.Name
	}

	return "the customer"
}

// ShipmentRef is the shipment's PRO number, or the item's own when it has none.
func ShipmentRef(e *Evidence) string {
	if e.Item.Shipment != nil && e.Item.Shipment.ProNumber != "" {
		return e.Item.Shipment.ProNumber
	}

	return e.Item.Number
}

// DocumentKind names a document by its type, or its file when it has none.
func DocumentKind(doc *document.Document) string {
	if doc.DocumentType != nil && doc.DocumentType.Name != "" {
		return doc.DocumentType.Name
	}

	return doc.OriginalName
}

// ChargeCodes are the codes of the charges on the invoice, in its order:
// FREIGHT for the line haul, then each coded charge once.
func ChargeCodes(e *Evidence) []string {
	if e.Item.Review == nil || e.Item.Review.Charges == nil {
		return nil
	}
	byID := additionalCharges(e.Item.Shipment)
	codes := make([]string, 0, len(e.Item.Review.Charges.Lines))
	for _, line := range e.Item.Review.Charges.Lines {
		if line.Removed {
			continue
		}
		code := FreightCode
		if line.AdditionalChargeID.IsNotNil() {
			charge := byID[line.AdditionalChargeID]
			if charge == nil || charge.AccessorialCharge == nil || charge.AccessorialCharge.Code == "" {
				continue
			}
			code = charge.AccessorialCharge.Code
		}
		if !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
	}

	return codes
}

func additionalCharges(shp *shipment.Shipment) map[pulid.ID]*shipment.AdditionalCharge {
	if shp == nil {
		return nil
	}
	byID := make(map[pulid.ID]*shipment.AdditionalCharge, len(shp.AdditionalCharges))
	for _, charge := range shp.AdditionalCharges {
		byID[charge.ID] = charge
	}

	return byID
}

func firstOf(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func number(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
