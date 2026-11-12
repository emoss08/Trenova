package recordshareservice

import (
	"context"
	"html/template"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/recordshareports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	// RecordSharedEventType is the notification a share raises. The client's
	// notification registry reads the link and the record from its data.
	RecordSharedEventType = "record_shared"
	notificationSource    = "recordshareservice"
)

// deliver tells each recipient in the app and, where the app knows its own
// address, by email. A notification or an email that could not be sent is
// logged and counted; the share itself is already recorded.
func (s *Service) deliver(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	plan *sharePlan,
) (int, recordshareports.EmailStatus) {
	tenantInfo.UserID = plan.sharer.ID
	baseURL := s.cfg.GetWebBaseURL()
	canEmail := baseURL != ""
	companyName := ""
	if canEmail {
		companyName = s.companyName(ctx, tenantInfo)
	}

	sentAt := timeutils.NowUnix()
	queued := 0
	for _, recipient := range plan.recipients {
		s.notify(ctx, tenantInfo, plan, recipient)
		if canEmail && s.sendEmail(ctx, &emailParams{
			tenantInfo:  tenantInfo,
			plan:        plan,
			recipient:   recipient,
			recordURL:   baseURL + plan.input.path,
			companyName: companyName,
			sentAt:      sentAt,
		}) {
			queued++
		}
	}

	return queued, emailStatus(canEmail, queued, len(plan.recipients))
}

func emailStatus(canEmail bool, queued, recipients int) recordshareports.EmailStatus {
	switch {
	case !canEmail:
		return recordshareports.EmailNotConfigured
	case queued == recipients:
		return recordshareports.EmailQueued
	case queued == 0:
		return recordshareports.EmailFailed
	default:
		return recordshareports.EmailPartial
	}
}

func (s *Service) notify(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	plan *sharePlan,
	recipient *tenant.User,
) {
	record := plan.record
	log := s.l.With(
		zap.String("recordId", record.ID.String()),
		zap.String("userId", recipient.ID.String()),
	)
	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: tenantInfo,
		Kind:       documenttemplate.KindNotificationRecordShared,
		Data: &notificationtemplate.RecordShareNotificationContext{
			SharedByName:   plan.sharer.Name,
			RecordTitle:    record.Title(),
			RecordSubtitle: record.Subtitle,
		},
		ReferenceID:       record.ID,
		UserID:            recipient.ID,
		FallbackToBuiltIn: true,
		Locale:            recipientLocale(recipient),
	})
	if err != nil {
		log.Warn("failed to render record share notification", zap.Error(err))

		return
	}

	targetUserID := recipient.ID
	buID := tenantInfo.BuID
	if _, err = s.notificationService.Create(ctx, &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		TargetUserID:   &targetUserID,
		Channel:        notification.ChannelUser,
		EventType:      RecordSharedEventType,
		Priority:       notification.PriorityMedium,
		Title:          rendered.Subject,
		Message:        rendered.Text,
		Data: map[string]any{
			"link":         plan.input.path,
			"recordKind":   string(record.Kind),
			"recordId":     record.ID.String(),
			"recordLabel":  record.Label,
			"sharedById":   plan.sharer.ID.String(),
			"sharedByName": plan.sharer.Name,
		},
		RelatedEntities: map[string]any{
			"recordKind": string(record.Kind),
			"recordId":   record.ID.String(),
		},
		Source: notificationSource,
	}); err != nil {
		log.Warn("failed to create record share notification", zap.Error(err))
	}
}

type emailParams struct {
	tenantInfo  pagination.TenantInfo
	plan        *sharePlan
	recipient   *tenant.User
	recordURL   string
	companyName string
	sentAt      int64
}

func (s *Service) sendEmail(ctx context.Context, p *emailParams) bool {
	address := strings.TrimSpace(p.recipient.EmailAddress)
	log := s.l.With(
		zap.String("recordId", p.plan.record.ID.String()),
		zap.String("userId", p.recipient.ID.String()),
	)
	if address == "" {
		log.Warn("record share recipient has no email address")

		return false
	}

	rendered, err := s.renderShareEmail(ctx, p)
	if err != nil {
		log.Warn("failed to render record share email", zap.Error(err))

		return false
	}

	if _, err = s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo: p.tenantInfo,
		Purpose:    email.PurposeNotifications,
		To:         []string{address},
		Subject:    rendered.Subject,
		HTML:       rendered.HTML,
		Text:       rendered.Text,
		IdempotencyKey: "record-share-" + p.plan.record.ID.String() + "-" +
			p.recipient.ID.String() + "-" + strconv.FormatInt(p.sentAt, 10),
	}); err != nil {
		log.Warn("failed to queue record share email", zap.Error(err))

		return false
	}

	return true
}

func (s *Service) renderShareEmail(
	ctx context.Context,
	p *emailParams,
) (*documenttemplateports.RenderedMessage, error) {
	record := p.plan.record
	//nolint:gosec // G203: a configured base URL joined to a path validated as inside the app
	recordURL := template.URL(p.recordURL)

	return s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: p.tenantInfo,
		Kind:       documenttemplate.KindRecordShareEmail,
		Data: &notificationtemplate.RecordShareEmailContext{
			RecipientFirstName: stringutils.FirstName(p.recipient.Name),
			SharedByName:       p.plan.sharer.Name,
			RecordTitle:        record.Title(),
			RecordSubtitle:     record.Subtitle,
			RecordNoun:         record.Kind.Noun(),
			Note:               p.plan.input.note,
			RecordURL:          recordURL,
			CompanyName:        p.companyName,
		},
		ReferenceID:       record.ID,
		UserID:            p.recipient.ID,
		FallbackToBuiltIn: true,
		Locale:            recipientLocale(p.recipient),
	})
}

// PreviewShare plans the share and renders what each recipient would be
// emailed, without recording or sending anything.
func (s *Service) PreviewShare(
	ctx context.Context,
	req *recordshareports.ShareRequest,
	actor *permissionports.RequestActor,
) (*recordshareports.Preview, error) {
	plan, err := s.planShare(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	tenantInfo := req.TenantInfo
	tenantInfo.UserID = plan.sharer.ID
	baseURL := s.cfg.GetWebBaseURL()
	canEmail := baseURL != ""
	companyName := ""
	if canEmail {
		companyName = s.companyName(ctx, tenantInfo)
	}

	sentAt := timeutils.NowUnix()
	preview := &recordshareports.Preview{
		Record:          plan.record,
		SharedByName:    plan.sharer.Name,
		Note:            plan.input.note,
		Path:            plan.input.path,
		EmailConfigured: canEmail,
		Recipients:      make([]recordshareports.RecipientPreview, 0, len(plan.recipients)),
	}
	for _, recipient := range plan.recipients {
		entry := recordshareports.RecipientPreview{
			UserID:       recipient.ID,
			Name:         recipient.Name,
			EmailAddress: strings.TrimSpace(recipient.EmailAddress),
		}
		if canEmail && entry.EmailAddress != "" {
			rendered, renderErr := s.renderShareEmail(ctx, &emailParams{
				tenantInfo:  tenantInfo,
				plan:        plan,
				recipient:   recipient,
				recordURL:   baseURL + plan.input.path,
				companyName: companyName,
				sentAt:      sentAt,
			})
			if renderErr != nil {
				return nil, renderErr
			}
			entry.Emailed = true
			entry.Subject = rendered.Subject
			entry.Body = rendered.Text
		}
		preview.Recipients = append(preview.Recipients, entry)
	}

	return preview, nil
}

func recipientLocale(user *tenant.User) i18n.Locale {
	locale, _ := i18n.Parse(user.Locale)

	return locale
}

func (s *Service) companyName(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	org, err := s.organizationRepo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return ""
	}

	return org.Name
}
