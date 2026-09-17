package carrierintelservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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
	monitoringLink         = "/dispatch/carrier-monitoring"
	carrierLinkPrefix      = "/dispatch/carriers?panelType=edit&panelEntityId="
	notificationDedupeDays = 7
)

type notificationRequest struct {
	tenant      pagination.TenantInfo
	eventType   string
	correlation string
	priority    notification.Priority
	title       string
	message     string
	link        string
	related     map[string]any
	data        map[string]any
}

func (s *Service) sendNotification(ctx context.Context, req *notificationRequest) bool {
	if s.notifications == nil {
		return false
	}
	ctx = context.WithoutCancel(ctx)

	if req.correlation != "" {
		exists, err := s.notifications.ExistsRecent(
			ctx,
			repositories.ExistsRecentNotificationRequest{
				OrganizationID: req.tenant.OrgID,
				BusinessUnitID: req.tenant.BuID,
				EventType:      req.eventType,
				CorrelationID:  req.correlation,
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

	data := map[string]any{"link": req.link}
	for key, value := range req.data {
		data[key] = value
	}

	buID := req.tenant.BuID
	entity := &notification.Notification{
		OrganizationID:  req.tenant.OrgID,
		BusinessUnitID:  &buID,
		EventType:       req.eventType,
		Channel:         notification.ChannelGlobal,
		Priority:        req.priority,
		Title:           req.title,
		Message:         req.message,
		Data:            data,
		RelatedEntities: req.related,
		Source:          notificationSource,
	}
	if req.correlation != "" {
		correlation := req.correlation
		entity.CorrelationID = &correlation
	}

	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create carrier intelligence notification",
			zap.String("eventType", req.eventType), zap.Error(err))
		return false
	}
	return true
}
