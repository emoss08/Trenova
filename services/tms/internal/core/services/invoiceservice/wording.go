package invoiceservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// invoiceWordingParams is what deciding an invoice email's wording needs.
type invoiceWordingParams struct {
	TenantInfo pagination.TenantInfo
	Entity     *invoice.Invoice
	Profile    *invoiceDeliveryProfile
	Context    map[string]string
}

// invoiceWording is the settled subject and body plus where they came from.
type invoiceWording struct {
	Subject invoiceTemplateResult
	Body    invoiceTemplateResult

	// HTML is set only when the organization's template produced the body.
	HTML         string
	FromTemplate bool
}

// resolveInvoiceWording walks the three tiers in order: the operator's draft, the
// customer's email profile, then the organization's template.
//
// The template is asked for only when neither free-text tier has anything to say,
// which is exactly where the hardcoded "Invoice N" / "Please find invoice N
// attached" used to answer. The precedence a customer already relies on is
// untouched; what changed is that the last tier is now editable.
func (s *Service) resolveInvoiceWording(
	ctx context.Context,
	p *invoiceWordingParams,
) (*invoiceWording, error) {
	emailProfile := p.Profile.Email
	subject, subjectFound := resolveSubject(p.Entity, emailProfile, p.Context)
	body, bodyFound := resolveBody(p.Entity, emailProfile, p.Context)

	out := &invoiceWording{Subject: subject, Body: body}
	if subjectFound && bodyFound {
		return out, nil
	}

	rendered, err := s.renderInvoiceEmail(ctx, &invoiceEmailRenderParams{
		TenantInfo: p.TenantInfo,
		Entity:     p.Entity,
		Profile:    p.Profile,
	})
	if err != nil {
		return nil, err
	}

	if !subjectFound {
		out.Subject.Value = rendered.Subject
	}
	if !bodyFound {
		out.Body.Value = rendered.Text
		out.HTML = rendered.HTML
		out.FromTemplate = true
	}

	return out, nil
}

// invoiceEmailWording is one render of the organization's invoice email template.
type invoiceEmailWording struct {
	Subject string
	Text    string
	HTML    string
}

type invoiceEmailRenderParams struct {
	TenantInfo pagination.TenantInfo
	Entity     *invoice.Invoice
	Profile    *invoiceDeliveryProfile

	// PartLabel is set when a split invoice re-renders per message, so the
	// wording can say "2 of 3" instead of three identical emails arriving.
	PartLabel      string
	AttachmentName string
}

// renderInvoiceEmail renders the organization's invoice email.
//
// It resolves per customer, so a customer with an assigned invoice email template
// receives theirs. There is deliberately no fallback to the built-in: a customer
// must never receive a message that silently differs from the one their carrier
// authored, so a broken template surfaces to the sender instead of being papered
// over. That is the opposite of the notification and notice policy, and it is the
// reason RenderMessageRequest carries the choice at all.
func (s *Service) renderInvoiceEmail(
	ctx context.Context,
	p *invoiceEmailRenderParams,
) (*invoiceEmailWording, error) {
	if s.templates == nil {
		return nil, errortypes.NewBusinessError(
			"Invoice email rendering is not configured on this deployment",
		)
	}

	var customerID *pulid.ID
	if p.Profile != nil && p.Profile.Customer != nil && !p.Profile.Customer.ID.IsNil() {
		customerID = &p.Profile.Customer.ID
	}

	rendered, err := s.templates.RenderMessage(ctx, &servicesports.RenderMessageRequest{
		TenantInfo: p.TenantInfo,
		Kind:       documenttemplate.KindInvoiceEmail,
		CustomerID: customerID,
		Data: emailContext(&EmailContextParams{
			Entity:         p.Entity,
			Profile:        p.Profile,
			PartLabel:      p.PartLabel,
			AttachmentName: p.AttachmentName,
		}),
		ReferenceID: p.Entity.ID,
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
// customer's email profile.
//
// Both are unbounded free text with {number}-style placeholders, which is why
// they keep the ad-hoc substitution engine rather than moving into the template
// system — there is no variable catalog that could describe them. found reports
// whether either had anything to say; when neither does, the caller renders the
// organization's template, which is the tier the hardcoded "Invoice N" used to
// occupy.
func resolveSubject(
	entity *invoice.Invoice,
	emailProfile *customer.CustomerEmailProfile,
	context map[string]string,
) (result invoiceTemplateResult, found bool) {
	if strings.TrimSpace(entity.EmailSubjectSnapshot) != "" {
		return renderInvoiceTemplate(entity.EmailSubjectSnapshot, context), true
	}
	if emailProfile != nil && strings.TrimSpace(emailProfile.Subject) != "" {
		return renderInvoiceTemplate(emailProfile.Subject, context), true
	}

	return invoiceTemplateResult{}, false
}

// resolveBody is resolveSubject for the message body. The prose the hardcoded
// tier used to assemble — "Please find invoice N attached", plus the memo — now
// lives in the built-in starter, so an organization can reword it.
func resolveBody(
	entity *invoice.Invoice,
	emailProfile *customer.CustomerEmailProfile,
	context map[string]string,
) (result invoiceTemplateResult, found bool) {
	if strings.TrimSpace(entity.EmailBodySnapshot) != "" {
		return renderInvoiceTemplate(entity.EmailBodySnapshot, context), true
	}
	if emailProfile != nil && strings.TrimSpace(emailProfile.Comment) != "" {
		return renderInvoiceTemplate(emailProfile.Comment, context), true
	}

	return invoiceTemplateResult{}, false
}

func partSubject(subject string, part, total int) string {
	if total <= 1 {
		return subject
	}
	return fmt.Sprintf("%s (%d of %d)", subject, part, total)
}

func partBodyBase(part *servicesports.InvoiceSendPlanPart) string {
	if len(part.Links) == 0 {
		return ""
	}
	return "Some supporting documents are available through secure download links below."
}
