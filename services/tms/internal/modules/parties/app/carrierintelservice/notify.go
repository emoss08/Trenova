package carrierintelservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	EventCarrierIntelBlock        = "carrier_intel_block"
	EventCarrierIntelChange       = "carrier_intel_change"
	EventCarrierIntelDigest       = "carrier_intel_digest"
	EventCarrierIntelProviderDown = "carrier_intel_provider_paused"
	EventCarrierEquipmentMismatch = "carrier_equipment_mismatch"
	eventSpendSoftCap             = "carrier_intel_spend_soft_cap"
	eventSpendHardCap             = "carrier_intel_spend_cap"

	notificationSource     = "carrier_intelligence"
	MonitoringLink         = "/dispatch/carrier-monitoring"
	CarrierLinkPrefix      = "/dispatch/carriers?panelType=edit&panelEntityId="
	notificationDedupeDays = 7
)

type NotificationRequest struct {
	Tenant      pagination.TenantInfo
	EventType   string
	Correlation string
	Priority    notification.Priority
	Title       string
	Message     string
	Link        string
	Related     map[string]any
	Data        map[string]any
}

func (s *Service) SendNotification(ctx context.Context, req *NotificationRequest) bool {
	if s.notifications == nil {
		return false
	}
	ctx = context.WithoutCancel(ctx)

	if req.Correlation != "" {
		exists, err := s.notifications.ExistsRecent(
			ctx,
			notificationports.ExistsRecentNotificationRequest{
				OrganizationID: req.Tenant.OrgID,
				BusinessUnitID: req.Tenant.BuID,
				EventType:      req.EventType,
				CorrelationID:  req.Correlation,
				Since:          timeutils.NowUnix() - notificationDedupeDays*timeutils.SecondsPerDay,
			},
		)
		if err != nil {
			s.l.Warn("failed to check notification dedupe", zap.Error(err))
			return false
		}
		if exists {
			return false
		}
	}

	data := map[string]any{"link": req.Link}
	for key, value := range req.Data {
		data[key] = value
	}

	buID := req.Tenant.BuID
	entity := &notification.Notification{
		OrganizationID:  req.Tenant.OrgID,
		BusinessUnitID:  &buID,
		EventType:       req.EventType,
		Channel:         notification.ChannelGlobal,
		Priority:        req.Priority,
		Title:           req.Title,
		Message:         req.Message,
		Data:            data,
		RelatedEntities: req.Related,
		Source:          notificationSource,
	}
	if req.Correlation != "" {
		correlation := req.Correlation
		entity.CorrelationID = &correlation
	}

	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create carrier intelligence notification",
			zap.String("eventType", req.EventType), zap.Error(err))
		return false
	}
	return true
}
