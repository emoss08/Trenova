package invoicesendservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/invoicetemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// invoiceWordingParams is what deciding an invoice email's wording needs.
type invoiceWordingParams struct {
	TenantInfo pagination.TenantInfo
	Entity     *invoice.Invoice
	Profile    *invoicedocument.DeliveryProfile
	Context    map[string]string
}

type invoiceWording struct {
	Subject invoicedocument.TemplateResult
	Body    invoicedocument.TemplateResult
	Message string
}

func (s *Service) resolveInvoiceWording(
	ctx context.Context,
	p *invoiceWordingParams,
) (*invoiceWording, error) {
	emailProfile := p.Profile.Email
	subject, subjectFound := resolveSubject(p.Entity, emailProfile, p.Context)
	body, bodyFound := resolveBody(p.Entity, emailProfile, p.Context)

	out := &invoiceWording{Subject: subject, Body: body}
	if bodyFound {
		out.Message = body.Value
	}
	override := ""
	if subjectFound {
		override = subject.Value
	}

	rendered, err := s.renderInvoiceEmail(ctx, &invoiceEmailRenderParams{
		TenantInfo: p.TenantInfo,
		Entity:     p.Entity,
		Profile:    p.Profile,
		Message:    out.Message,
		Subject:    override,
	})
	if err != nil {
		return nil, err
	}

	out.Subject.Value = rendered.Subject
	out.Body.Value = rendered.Text

	return out, nil
}

type invoiceEmailWording struct {
	Subject string
	Text    string
	HTML    string
}

type invoiceEmailRenderParams struct {
	TenantInfo pagination.TenantInfo
	Entity     *invoice.Invoice
	Profile    *invoicedocument.DeliveryProfile

	PartLabel      string
	AttachmentName string
	Message        string
	Subject        string
	Links          []invoicetemplate.InvoiceEmailLink
}

func (s *Service) renderInvoiceEmail(
	ctx context.Context,
	p *invoiceEmailRenderParams,
) (*invoiceEmailWording, error) {
	var customerID *pulid.ID
	if p.Profile != nil && p.Profile.Customer != nil && !p.Profile.Customer.ID.IsNil() {
		customerID = &p.Profile.Customer.ID
	}

	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: p.TenantInfo,
		Kind:       documenttemplate.KindInvoiceEmail,
		CustomerID: customerID,
		Data: invoicedocument.EmailContext(&invoicedocument.EmailContextParams{
			Entity:         p.Entity,
			Profile:        p.Profile,
			PartLabel:      p.PartLabel,
			AttachmentName: p.AttachmentName,
			Message:        p.Message,
			Links:          p.Links,
		}),
		ReferenceID: p.Entity.ID,
		Subject:     p.Subject,
	})
	if err != nil {
		return nil, err
	}

	return &invoiceEmailWording{
		Subject: rendered.Subject,
		Text:    rendered.Text,
		HTML:    rendered.HTML,
	}, nil
}

// resolveSubject applies the two free-text tiers: the operator's draft, then the
// customer's email profile. Both are free text with {number}-style placeholders.
// found reports whether either had anything to say; what they say is handed to the
// organization's template, which keeps the email on the standard either way.
func resolveSubject(
	entity *invoice.Invoice,
	emailProfile *customer.CustomerEmailProfile,
	values map[string]string,
) (result invoicedocument.TemplateResult, found bool) {
	if strings.TrimSpace(entity.EmailSubjectSnapshot) != "" {
		return renderInvoiceTemplate(entity.EmailSubjectSnapshot, values), true
	}
	if emailProfile != nil && strings.TrimSpace(emailProfile.Subject) != "" {
		return renderInvoiceTemplate(emailProfile.Subject, values), true
	}

	return invoicedocument.TemplateResult{}, false
}

// resolveBody is resolveSubject for the message body. The template prints it in
// place of its own greeting.
func resolveBody(
	entity *invoice.Invoice,
	emailProfile *customer.CustomerEmailProfile,
	values map[string]string,
) (result invoicedocument.TemplateResult, found bool) {
	if strings.TrimSpace(entity.EmailBodySnapshot) != "" {
		return renderInvoiceTemplate(entity.EmailBodySnapshot, values), true
	}
	if emailProfile != nil && strings.TrimSpace(emailProfile.Comment) != "" {
		return renderInvoiceTemplate(emailProfile.Comment, values), true
	}

	return invoicedocument.TemplateResult{}, false
}

func partSubject(subject string, part, total int) string {
	if total <= 1 {
		return subject
	}
	return fmt.Sprintf("%s (%d of %d)", subject, part, total)
}

func partLabel(part, total int) string {
	if total <= 1 {
		return ""
	}
	return fmt.Sprintf("%d of %d", part, total)
}
