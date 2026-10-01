package invoiceservice

import (
	"context"
	"fmt"
	"html"
	"net/mail"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

func invoiceTemplateContext(
	entity *invoice.Invoice,
	deliveryProfile *invoiceDeliveryProfile,
) map[string]string {
	values := map[string]string{
		"number":              entity.Number,
		"invoice.number":      entity.Number,
		"invoiceNumber":       entity.Number,
		"invoice.date":        unixDate(entity.InvoiceDate),
		"invoiceDate":         unixDate(entity.InvoiceDate),
		"invoice.dueDate":     unixDatePtr(entity.DueDate),
		"dueDate":             unixDatePtr(entity.DueDate),
		"invoice.paymentTerm": string(entity.PaymentTerm),
		"paymentTerm":         string(entity.PaymentTerm),
		"invoice.total": moneyString(
			entity.CurrencyCode,
			entity.TotalAmount.StringFixed(2),
		),
		"invoiceTotal": moneyString(
			entity.CurrencyCode,
			entity.TotalAmount.StringFixed(2),
		),
		"invoice.currency":        entity.CurrencyCode,
		"currency":                entity.CurrencyCode,
		"customer":                entity.BillToName,
		"customer.name":           entity.BillToName,
		"customerName":            entity.BillToName,
		"customer.code":           entity.BillToCode,
		"customerCode":            entity.BillToCode,
		"company":                 "",
		"organization.name":       "",
		"organizationName":        "",
		"shipment.pro":            entity.ShipmentProNumber,
		"shipmentPro":             entity.ShipmentProNumber,
		"shipment.bol":            entity.ShipmentBOL,
		"shipmentBol":             entity.ShipmentBOL,
		"shipment.serviceDate":    unixDatePtr(entity.ServiceDate),
		"serviceDate":             unixDatePtr(entity.ServiceDate),
		"remittance.instructions": entity.RemittanceInstructions,
		"remittanceInstructions":  entity.RemittanceInstructions,
	}
	if deliveryProfile == nil {
		return values
	}
	if deliveryProfile.Organization != nil {
		values["company"] = deliveryProfile.Organization.Name
		values["organization.name"] = deliveryProfile.Organization.Name
		values["organizationName"] = deliveryProfile.Organization.Name
	}
	if deliveryProfile.Customer != nil {
		values["customer.name"] = stringutils.FirstNonEmpty(
			deliveryProfile.Customer.Name,
			values["customer.name"],
		)
		values["customer.code"] = stringutils.FirstNonEmpty(
			deliveryProfile.Customer.Code,
			values["customer.code"],
		)
		values["customer"] = values["customer.name"]
		values["customerName"] = values["customer.name"]
		values["customerCode"] = values["customer.code"]
		if deliveryProfile.Organization == nil && deliveryProfile.Customer.Organization != nil {
			values["company"] = deliveryProfile.Customer.Organization.Name
			values["organization.name"] = deliveryProfile.Customer.Organization.Name
			values["organizationName"] = deliveryProfile.Customer.Organization.Name
		}
	}
	if deliveryProfile.Shipment != nil {
		shp := deliveryProfile.Shipment
		values["shipment.pro"] = stringutils.FirstNonEmpty(shp.ProNumber, values["shipment.pro"])
		values["shipment.bol"] = stringutils.FirstNonEmpty(shp.BOL, values["shipment.bol"])
		values["shipment.pickupDate"] = unixDatePtr(shp.ActualShipDate)
		values["shipment.deliveryDate"] = unixDatePtr(shp.ActualDeliveryDate)
		values["shipment.origin"] = shipmentOrigin(shp)
		values["shipment.destination"] = shipmentDestination(shp)
		values["shipmentPro"] = values["shipment.pro"]
		values["shipmentBol"] = values["shipment.bol"]
		values["pickupDate"] = values["shipment.pickupDate"]
		values["deliveryDate"] = values["shipment.deliveryDate"]
		values["origin"] = values["shipment.origin"]
		values["destination"] = values["shipment.destination"]
	}
	return values
}

func renderInvoiceTemplate(template string, values map[string]string) invoiceTemplateResult {
	unknown := make([]string, 0)
	rendered := invoiceTemplateVariablePattern.ReplaceAllStringFunc(
		template,
		func(match string) string {
			parts := invoiceTemplateVariablePattern.FindStringSubmatch(match)
			if len(parts) != 3 {
				return match
			}
			key := parts[1]
			if key == "" {
				key = parts[2]
			}
			value, ok := values[key]
			if !ok {
				unknown = append(unknown, key)
				return match
			}
			return value
		},
	)
	return invoiceTemplateResult{
		Value:   strings.TrimSpace(rendered),
		Unknown: sliceutils.DedupeSorted(unknown),
	}
}

func templateWarnings(field string, unknown []string) []string {
	if len(unknown) == 0 {
		return nil
	}
	warnings := make([]string, 0, len(unknown))
	for _, variable := range unknown {
		warnings = append(warnings, "Unknown "+field+" template variable: "+variable)
	}
	return warnings
}

func resolveFromEmail(
	profile *email.Profile,
	emailProfile *customer.CustomerEmailProfile,
) (string, error) {
	if profile == nil {
		return "", nil
	}
	fromEmail := strings.TrimSpace(profile.SenderEmail)
	if emailProfile == nil || strings.TrimSpace(emailProfile.FromEmail) == "" {
		return fromEmail, nil
	}
	override := strings.TrimSpace(emailProfile.FromEmail)
	parsed, err := mail.ParseAddress(override)
	if err != nil || parsed.Address != override {
		return fromEmail, errortypes.NewValidationError(
			"fromEmail",
			errortypes.ErrInvalid,
			"Customer invoice sender email is invalid",
		)
	}
	return override, nil
}

type invoiceSenderNotice struct {
	Origin  string
	Warning string
}

func describeInvoiceSender(
	profile *email.Profile,
	cus *customer.Customer,
	fromEmail string,
) invoiceSenderNotice {
	fromEmail = strings.TrimSpace(fromEmail)
	if profile == nil || fromEmail == "" {
		return invoiceSenderNotice{}
	}
	profileSender := strings.TrimSpace(profile.SenderEmail)
	if strings.EqualFold(fromEmail, profileSender) {
		return invoiceSenderNotice{}
	}
	customerName := invoiceCustomerLabel(cus)
	notice := invoiceSenderNotice{
		Origin: "the From address on the Email profile tab of customer " + customerName,
	}
	fromDomain := stringutils.EmailDomain(fromEmail)
	if fromDomain == "" || fromDomain == stringutils.EmailDomain(profileSender) {
		return notice
	}
	notice.Warning = fmt.Sprintf(
		"Invoices to %s are sent from %s, the From address on the customer's Email profile tab, "+
			"instead of the Billing email profile's sender %s. The email provider will refuse "+
			"the send unless %s is verified with it.",
		customerName,
		fromEmail,
		profileSender,
		fromDomain,
	)
	return notice
}

func invoiceCustomerLabel(cus *customer.Customer) string {
	if cus == nil {
		return "on this invoice"
	}
	if name := strings.TrimSpace(cus.Name); name != "" {
		return name
	}
	if code := strings.TrimSpace(cus.Code); code != "" {
		return code
	}
	return "on this invoice"
}

func resolveDeliveryHeaders(
	fromEmail string,
	emailProfile *customer.CustomerEmailProfile,
) map[string]string {
	if emailProfile == nil || !emailProfile.ReadReceipt || strings.TrimSpace(fromEmail) == "" {
		return nil
	}
	return map[string]string{"Disposition-Notification-To": strings.TrimSpace(fromEmail)}
}

func invoicePDFAttachmentName(rendered string, entity *invoice.Invoice) string {
	name := fileutils.SafeFilename(rendered)
	if strings.TrimSpace(name) == "" || name == "." {
		name = invoicePDFName(entity)
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	base = strings.TrimSpace(strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '-'
		default:
			return r
		}
	}, base))
	if base == "" {
		base = strings.TrimSuffix(invoicePDFName(entity), ".pdf")
	}
	return base + ".pdf"
}

func appendShipmentDetail(body string, entity *invoice.Invoice, shp *shipment.Shipment) string {
	detail := shipmentDetailBlock(entity, shp)
	if detail == "" {
		return body
	}
	if strings.TrimSpace(body) == "" {
		return detail
	}
	return strings.TrimSpace(body) + "\n\n" + detail
}

func shipmentDetailBlock(entity *invoice.Invoice, shp *shipment.Shipment) string {
	lines := []string{"Shipment Detail"}
	appendDetailLine := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			lines = append(lines, label+": "+strings.TrimSpace(value))
		}
	}
	appendDetailLine("PRO", stringutils.FirstNonEmpty(entity.ShipmentProNumber, shipmentPro(shp)))
	appendDetailLine("BOL", stringutils.FirstNonEmpty(entity.ShipmentBOL, shipmentBOL(shp)))
	appendDetailLine("Route", shipmentRoute(shp))
	appendDetailLine("Service Date", unixDatePtr(entity.ServiceDate))
	if shp != nil {
		appendDetailLine("Pickup", unixDatePtr(shp.ActualShipDate))
		appendDetailLine("Delivery", unixDatePtr(shp.ActualDeliveryDate))
		appendDetailLine("Commodities", commoditySummary(shp))
		appendDetailLine("Pieces", int64PtrString(shp.Pieces))
		appendDetailLine("Weight", int64PtrString(shp.Weight))
	}
	if len(entity.Lines) > 0 {
		lines = append(lines, "Charges:")
		for _, line := range entity.Lines {
			if line == nil {
				continue
			}
			lines = append(lines, fmt.Sprintf(
				"- %s: %s",
				line.Description,
				moneyString(entity.CurrencyCode, line.Amount.StringFixed(2)),
			))
		}
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

type partBodyHTMLParams struct {
	Request  *servicesports.InvoiceSendRequest
	Entity   *invoice.Invoice
	Profile  *invoiceDeliveryProfile
	Plan     *servicesports.InvoiceSendPlan
	Part     *servicesports.InvoiceSendPlanPart
	PartBody string
}

func (s *Service) partBodyHTML(ctx context.Context, p *partBodyHTMLParams) string {
	if !p.Plan.FromTemplate {
		return bodyHTML(p.PartBody)
	}

	rendered := p.Plan.BodyHTML
	if len(p.Plan.Parts) > 1 && p.Profile != nil {
		wording, err := s.renderInvoiceEmail(ctx, &invoiceEmailRenderParams{
			TenantInfo: p.Request.TenantInfo,
			Entity:     p.Entity,
			Profile:    p.Profile,
			PartLabel:  fmt.Sprintf("%d of %d", p.Part.PartNumber, len(p.Plan.Parts)),
		})
		if err != nil {
			s.l.Warn("could not re-render the invoice email for this part",
				zap.String("invoiceId", p.Entity.ID.String()),
				zap.Error(err))
		} else {
			rendered = wording.HTML
		}
	}

	extra := strings.TrimSpace(strings.TrimPrefix(p.PartBody, strings.TrimSpace(p.Plan.Body)))
	if extra != "" {
		rendered += bodyHTML(extra)
	}

	return rendered
}

func bodyHTML(body string) string {
	escaped := html.EscapeString(body)
	return "<p>" + strings.ReplaceAll(escaped, "\n", "<br>") + "</p>"
}
