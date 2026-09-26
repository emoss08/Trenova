package invoiceshareservice

import (
	"context"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	invoiceSharedEventType = "invoice_shared"
	notificationSource     = "invoiceshareservice"
	invoicesPath           = "/billing/invoices"
)

type deliveryParams struct {
	invoice    *invoice.Invoice
	sharer     *tenant.User
	recipients []*tenant.User
	note       string
	tab        invoice.ShareTab
}

func invoiceSharePath(invoiceID string, tab invoice.ShareTab) string {
	query := url.Values{}
	query.Set("item", invoiceID)
	if tab != "" && tab != invoice.ShareTabOverview {
		query.Set("tab", string(tab))
	}
	return invoicesPath + "?" + query.Encode()
}

func (s *Service) deliver(
	ctx context.Context,
	p *deliveryParams,
) (int, servicesports.InvoiceShareEmailStatus) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  p.invoice.OrganizationID,
		BuID:   p.invoice.BusinessUnitID,
		UserID: p.sharer.ID,
	}
	path := invoiceSharePath(p.invoice.ID.String(), p.tab)
	baseURL := s.cfg.App.GetWebBaseURL()
	canEmail := baseURL != "" && s.emailService != nil && s.templates != nil

	companyName := ""
	if canEmail {
		companyName = s.companyName(ctx, tenantInfo)
	}

	sentAt := timeutils.NowUnix()
	queued := 0
	for _, recipient := range p.recipients {
		s.notify(ctx, tenantInfo, p, recipient, path)

		if canEmail && s.sendEmail(ctx, &emailParams{
			tenantInfo:  tenantInfo,
			delivery:    p,
			recipient:   recipient,
			invoiceURL:  baseURL + path,
			companyName: companyName,
			sentAt:      sentAt,
		}) {
			queued++
		}
	}

	return queued, emailStatus(canEmail, queued, len(p.recipients))
}

func emailStatus(canEmail bool, queued, recipients int) servicesports.InvoiceShareEmailStatus {
	switch {
	case !canEmail:
		return servicesports.InvoiceShareEmailNotConfigured
	case queued == recipients:
		return servicesports.InvoiceShareEmailQueued
	case queued == 0:
		return servicesports.InvoiceShareEmailFailed
	default:
		return servicesports.InvoiceShareEmailPartial
	}
}

func (s *Service) notify(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	p *deliveryParams,
	recipient *tenant.User,
	path string,
) {
	if s.notificationService == nil || s.templates == nil {
		return
	}

	rendered, err := s.templates.RenderMessage(ctx, &servicesports.RenderMessageRequest{
		TenantInfo: tenantInfo,
		Kind:       documenttemplate.KindNotificationInvoiceShared,
		Data: &documenttemplate.InvoiceShareNotificationContext{
			SharedByName:  p.sharer.Name,
			InvoiceNumber: p.invoice.Number,
			CustomerName:  p.invoice.BillToName,
		},
		ReferenceID:       p.invoice.ID,
		UserID:            recipient.ID,
		FallbackToBuiltIn: true,
		Locale:            recipientLocale(recipient),
	})
	if err != nil {
		s.l.Warn("failed to render invoice share notification",
			zap.String("invoiceId", p.invoice.ID.String()),
			zap.String("userId", recipient.ID.String()),
			zap.Error(err))
		return
	}

	targetUserID := recipient.ID
	if _, err = s.notificationService.Create(ctx, &notification.Notification{
		OrganizationID: p.invoice.OrganizationID,
		BusinessUnitID: &p.invoice.BusinessUnitID,
		TargetUserID:   &targetUserID,
		Channel:        notification.ChannelUser,
		EventType:      invoiceSharedEventType,
		Priority:       notification.PriorityMedium,
		Title:          rendered.Subject,
		Message:        rendered.Text,
		Data: map[string]any{
			"link":          path,
			"invoiceId":     p.invoice.ID.String(),
			"invoiceNumber": p.invoice.Number,
			"sharedById":    p.sharer.ID.String(),
			"sharedByName":  p.sharer.Name,
		},
		RelatedEntities: map[string]any{
			"invoiceId": p.invoice.ID.String(),
		},
		Source: notificationSource,
	}); err != nil {
		s.l.Warn("failed to create invoice share notification",
			zap.String("invoiceId", p.invoice.ID.String()),
			zap.String("userId", recipient.ID.String()),
			zap.Error(err))
	}
}

type emailParams struct {
	tenantInfo  pagination.TenantInfo
	delivery    *deliveryParams
	recipient   *tenant.User
	invoiceURL  string
	companyName string
	sentAt      int64
}

func (s *Service) sendEmail(ctx context.Context, p *emailParams) bool {
	address := strings.TrimSpace(p.recipient.EmailAddress)
	log := s.l.With(
		zap.String("invoiceId", p.delivery.invoice.ID.String()),
		zap.String("userId", p.recipient.ID.String()),
	)
	if address == "" {
		log.Warn("invoice share recipient has no email address")
		return false
	}

	rendered, err := s.renderShareEmail(ctx, p)
	if err != nil {
		log.Warn("failed to render invoice share email", zap.Error(err))
		return false
	}

	if _, err = s.emailService.Send(ctx, &servicesports.SendEmailRequest{
		TenantInfo: p.tenantInfo,
		Purpose:    email.PurposeNotifications,
		To:         []string{address},
		Subject:    rendered.Subject,
		HTML:       rendered.HTML,
		Text:       rendered.Text,
		IdempotencyKey: "invoice-share-" + p.delivery.invoice.ID.String() + "-" +
			p.recipient.ID.String() + "-" + strconv.FormatInt(p.sentAt, 10),
	}); err != nil {
		log.Warn("failed to queue invoice share email", zap.Error(err))
		return false
	}

	return true
}

func (s *Service) renderShareEmail(
	ctx context.Context,
	p *emailParams,
) (*servicesports.RenderedMessage, error) {
	return s.templates.RenderMessage(ctx, &servicesports.RenderMessageRequest{
		TenantInfo: p.tenantInfo,
		Kind:       documenttemplate.KindInvoiceShareEmail,
		Data: &documenttemplate.InvoiceShareEmailContext{
			RecipientFirstName: stringutils.FirstName(p.recipient.Name),
			SharedByName:       p.delivery.sharer.Name,
			InvoiceNumber:      p.delivery.invoice.Number,
			CustomerName:       p.delivery.invoice.BillToName,
			Note:               p.delivery.note,
			InvoiceURL: template.URL(
				p.invoiceURL,
			), //nolint:gosec // configured base URL + server-encoded path
			CompanyName: p.companyName,
		},
		ReferenceID:       p.delivery.invoice.ID,
		UserID:            p.recipient.ID,
		FallbackToBuiltIn: true,
		Locale:            recipientLocale(p.recipient),
	})
}

func recipientLocale(user *tenant.User) i18n.Locale {
	locale, _ := i18n.Parse(user.Locale)
	return locale
}

func (s *Service) companyName(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	if s.organizationRepo == nil {
		return ""
	}

	org, err := s.organizationRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return ""
	}

	return org.Name
}
