package invoicesendservice

import (
	"context"
	"fmt"
	"html"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

func invoiceTemplateContext(
	entity *invoice.Invoice,
	deliveryProfile *invoicedocument.DeliveryProfile,
) map[string]string {
	values := map[string]string{
		"number":              entity.Number,
		"invoice.number":      entity.Number,
		"invoiceNumber":       entity.Number,
		"invoice.date":        invoicedocument.UnixDate(entity.InvoiceDate),
		"invoiceDate":         invoicedocument.UnixDate(entity.InvoiceDate),
		"invoice.dueDate":     invoicedocument.UnixDatePtr(entity.DueDate),
		"dueDate":             invoicedocument.UnixDatePtr(entity.DueDate),
		"invoice.paymentTerm": string(entity.PaymentTerm),
		"paymentTerm":         string(entity.PaymentTerm),
		"invoice.total": invoicedocument.MoneyString(
			entity.CurrencyCode,
			entity.TotalAmount.StringFixed(2),
		),
		"invoiceTotal": invoicedocument.MoneyString(
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
		"shipment.serviceDate":    invoicedocument.UnixDatePtr(entity.ServiceDate),
		"serviceDate":             invoicedocument.UnixDatePtr(entity.ServiceDate),
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
		values["shipment.pickupDate"] = invoicedocument.UnixDatePtr(shp.ActualShipDate)
		values["shipment.deliveryDate"] = invoicedocument.UnixDatePtr(shp.ActualDeliveryDate)
		values["shipment.origin"] = invoicedocument.ShipmentOrigin(shp)
		values["shipment.destination"] = invoicedocument.ShipmentDestination(shp)
		values["shipmentPro"] = values["shipment.pro"]
		values["shipmentBol"] = values["shipment.bol"]
		values["pickupDate"] = values["shipment.pickupDate"]
		values["deliveryDate"] = values["shipment.deliveryDate"]
		values["origin"] = values["shipment.origin"]
		values["destination"] = values["shipment.destination"]
	}
	return values
}

func renderInvoiceTemplate(
	template string,
	values map[string]string,
) invoicedocument.TemplateResult {
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
	return invoicedocument.TemplateResult{
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

func billingSenderEmail(profile *email.Profile) string {
	if profile == nil {
		return ""
	}
	return strings.TrimSpace(profile.SenderEmail)
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
		name = invoicedocument.PDFName(entity)
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
		base = strings.TrimSuffix(invoicedocument.PDFName(entity), ".pdf")
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
	appendDetailLine(
		"PRO",
		stringutils.FirstNonEmpty(entity.ShipmentProNumber, invoicedocument.ShipmentPro(shp)),
	)
	appendDetailLine(
		"BOL",
		stringutils.FirstNonEmpty(entity.ShipmentBOL, invoicedocument.ShipmentBOL(shp)),
	)
	appendDetailLine("Route", invoicedocument.ShipmentRoute(shp))
	appendDetailLine("Service Date", invoicedocument.UnixDatePtr(entity.ServiceDate))
	if shp != nil {
		appendDetailLine("Pickup", invoicedocument.UnixDatePtr(shp.ActualShipDate))
		appendDetailLine("Delivery", invoicedocument.UnixDatePtr(shp.ActualDeliveryDate))
		appendDetailLine("Commodities", invoicedocument.CommoditySummary(shp))
		appendDetailLine("Pieces", invoicedocument.Int64PtrString(shp.Pieces))
		appendDetailLine("Weight", invoicedocument.Int64PtrString(shp.Weight))
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
				invoicedocument.MoneyString(entity.CurrencyCode, line.Amount.StringFixed(2)),
			))
		}
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

type partBodyHTMLParams struct {
	Request  *invoiceports.InvoiceSendRequest
	Entity   *invoice.Invoice
	Profile  *invoicedocument.DeliveryProfile
	Plan     *invoiceports.InvoiceSendPlan
	Part     *invoiceports.InvoiceSendPlanPart
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
			rendered = withShipmentDetailHTML(
				wording.HTML,
				includedShipmentDetail(p.Entity, p.Profile),
			)
		}
	}

	extra := strings.TrimSpace(strings.TrimPrefix(p.PartBody, strings.TrimSpace(p.Plan.Body)))
	if extra != "" {
		rendered += bodyHTML(extra)
	}

	return rendered
}

func includedShipmentDetail(
	entity *invoice.Invoice,
	profile *invoicedocument.DeliveryProfile,
) string {
	if profile == nil || profile.Email == nil || !profile.Email.IncludeShipmentDetail {
		return ""
	}
	return shipmentDetailBlock(entity, profile.Shipment)
}

func withShipmentDetailHTML(rendered, detail string) string {
	if rendered == "" || detail == "" {
		return rendered
	}
	return rendered + bodyHTML(detail)
}

func bodyHTML(body string) string {
	escaped := html.EscapeString(body)
	return "<p>" + strings.ReplaceAll(escaped, "\n", "<br>") + "</p>"
}
