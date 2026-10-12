package notificationservice

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	deliveryDataKey      = "delivery"
	onBehalfOfIDKey      = "onBehalfOfUserId"
	onBehalfOfNameKey    = "onBehalfOfName"
	emailIdempotencyBase = "notification-email-"
)

type recipient struct {
	user       *tenant.User
	preference *notification.Preference
	location   *time.Location
}

func handOffEventTypes() []string {
	return []string{
		notification.MentionEventType,
		"tender_needs_review",
		"agent.proposals_pending",
		"agent.proposals_reminder",
	}
}

func (s *Service) loadRecipient(
	ctx context.Context,
	entity *notification.Notification,
	now time.Time,
) *recipient {
	if s.users == nil || entity.TargetUserID == nil || entity.TargetUserID.IsNil() ||
		entity.BusinessUnitID == nil {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   *entity.BusinessUnitID,
		UserID: *entity.TargetUserID,
	}
	user, err := s.users.GetTenantMember(ctx, tenantports.GetTenantMemberRequest{
		UserID:     *entity.TargetUserID,
		TenantInfo: tenantInfo,
		Now:        now.Unix(),
	})
	if err != nil {
		s.l.Warn("failed to load notification recipient",
			zap.String("userID", entity.TargetUserID.String()),
			zap.Error(err),
		)
		return nil
	}

	pref := notification.DefaultPreference(tenantInfo.OrgID, tenantInfo.BuID, tenantInfo.UserID)
	if s.preferences != nil {
		stored, found, prefErr := s.preferences.Get(ctx, notificationports.GetPreferenceRequest{
			TenantInfo: tenantInfo,
		})
		if prefErr != nil {
			s.l.Warn("failed to load notification preferences; using defaults",
				zap.String("userID", tenantInfo.UserID.String()),
				zap.Error(prefErr),
			)
		} else if found {
			pref = stored
		}
	}

	location, locErr := time.LoadLocation(user.Timezone)
	if locErr != nil || user.Timezone == "" {
		location = time.UTC
	}

	return &recipient{user: user, preference: pref, location: location}
}

func (r *recipient) decide(entity *notification.Notification, now time.Time) notification.DeliveryDecision {
	return r.preference.Decide(notification.DeliveryInput{
		EventType:    entity.EventType,
		Priority:     entity.Priority,
		Availability: notification.Availability(r.user.EffectiveAvailability(now.Unix())),
		Now:          now,
		Location:     r.location,
	})
}

func stampDecision(entity *notification.Notification, decision notification.DeliveryDecision) {
	if entity.Data == nil {
		entity.Data = make(map[string]any, 1)
	}
	entity.Data[deliveryDataKey] = map[string]any{
		"urgent":  decision.Urgent,
		"desktop": decision.Desktop,
		"sound":   decision.Sound,
		"email":   string(decision.Email),
	}
}

func (s *Service) deliverOutside(
	ctx context.Context,
	created *notification.Notification,
	to *recipient,
	decision notification.DeliveryDecision,
) {
	if decision.Desktop {
		s.sendWebPush(ctx, created)
	}
	if decision.Email == notification.EmailDeliveryNow {
		s.sendEmail(ctx, created, to)
	}
}

func (s *Service) sendEmail(
	ctx context.Context,
	created *notification.Notification,
	to *recipient,
) {
	address := strings.TrimSpace(to.user.EmailAddress)
	if s.email == nil || address == "" || created.BusinessUnitID == nil {
		return
	}

	rendered, err := renderNotificationEmail(&notificationEmail{
		RecipientName: to.user.Name,
		Title:         created.Title,
		Message:       created.Message,
		Link:          s.absoluteLink(created),
		SettingsURL:   s.settingsURL(),
	})
	if err != nil {
		s.l.Warn("failed to render notification email",
			zap.String("notificationID", created.ID.String()),
			zap.Error(err),
		)
		return
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  created.OrganizationID,
		BuID:   *created.BusinessUnitID,
		UserID: to.user.ID,
	}
	background := context.WithoutCancel(ctx)
	go func() {
		if _, sendErr := s.email.Send(background, &emailports.SendEmailRequest{
			TenantInfo:     tenantInfo,
			Purpose:        email.PurposeNotifications,
			To:             []string{address},
			Subject:        rendered.Subject,
			HTML:           rendered.HTML,
			Text:           rendered.Text,
			IdempotencyKey: emailIdempotencyBase + created.ID.String(),
		}); sendErr != nil {
			s.l.Warn("failed to send notification email",
				zap.String("notificationID", created.ID.String()),
				zap.String("userID", to.user.ID.String()),
				zap.Error(sendErr),
			)
		}
	}()
}

func (s *Service) handOff(
	ctx context.Context,
	created *notification.Notification,
	to *recipient,
	now time.Time,
) {
	if !slices.Contains(handOffEventTypes(), created.EventType) {
		return
	}
	if _, already := created.Data[onBehalfOfIDKey]; already {
		return
	}
	delegateID := to.user.ActiveDelegate(now.Unix())
	if delegateID == nil || created.BusinessUnitID == nil {
		return
	}

	if _, err := s.users.GetTenantMember(ctx, tenantports.GetTenantMemberRequest{
		UserID: *delegateID,
		TenantInfo: pagination.TenantInfo{
			OrgID: created.OrganizationID,
			BuID:  *created.BusinessUnitID,
		},
		Now: now.Unix(),
	}); err != nil {
		s.l.Info("skipping hand-off to someone outside the organization",
			zap.String("delegateID", delegateID.String()),
		)
		return
	}

	data := make(map[string]any, len(created.Data)+2)
	for key, value := range created.Data {
		if key == deliveryDataKey {
			continue
		}
		data[key] = value
	}
	data[onBehalfOfIDKey] = to.user.ID.String()
	data[onBehalfOfNameKey] = to.user.Name

	target := *delegateID
	copyOf := &notification.Notification{
		OrganizationID:  created.OrganizationID,
		BusinessUnitID:  created.BusinessUnitID,
		TargetUserID:    &target,
		EventType:       created.EventType,
		Priority:        created.Priority,
		Channel:         notification.ChannelUser,
		Title:           created.Title,
		Message:         created.Message,
		Data:            data,
		RelatedEntities: created.RelatedEntities,
		Actions:         created.Actions,
		ExpiresAt:       created.ExpiresAt,
		Source:          created.Source,
		CorrelationID:   created.CorrelationID,
		Tags:            created.Tags,
	}
	if _, err := s.Create(ctx, copyOf); err != nil {
		s.l.Warn("failed to hand off notification",
			zap.String("notificationID", created.ID.String()),
			zap.String("delegateID", delegateID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) absoluteLink(created *notification.Notification) string {
	link, _ := created.Data["link"].(string)
	if link == "" || !strings.HasPrefix(link, "/") || strings.HasPrefix(link, "//") ||
		s.webBaseURL == "" {
		return ""
	}
	return s.webBaseURL + link
}

func (s *Service) settingsURL() string {
	if s.webBaseURL == "" {
		return ""
	}
	return s.webBaseURL + "/?settings=notifications"
}
