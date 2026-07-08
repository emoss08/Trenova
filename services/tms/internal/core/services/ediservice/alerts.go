package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	EDIAlertEventMessageDeadLettered    = "edi.message.dead_lettered"
	EDIAlertEventInboundFileQuarantined = "edi.inbound_file.quarantined"

	ediAlertThrottleWindowSeconds = int64(15 * 60)
	ediAlertSource                = "ediservice"
)

type EDIOperationalAlert struct {
	OrganizationID  pulid.ID
	BusinessUnitID  pulid.ID
	EventType       string
	PartnerID       pulid.ID
	Title           string
	Message         string
	RelatedEntities map[string]any
	Data            map[string]any
}

func (s *Service) NotifyOperationalFailure(ctx context.Context, alert *EDIOperationalAlert) {
	if s.notifications == nil || alert == nil {
		return
	}
	correlationID := alert.EventType + ":" + alert.PartnerID.String()
	exists, err := s.notificationThrottled(ctx, alert, correlationID)
	if err != nil {
		s.l.Warn(
			"failed to check EDI alert throttle window",
			zap.String("eventType", alert.EventType),
			zap.Error(err),
		)
		return
	}
	if exists {
		return
	}
	businessUnitID := alert.BusinessUnitID
	entity := &notification.Notification{
		OrganizationID:  alert.OrganizationID,
		BusinessUnitID:  &businessUnitID,
		EventType:       alert.EventType,
		Priority:        notification.PriorityHigh,
		Channel:         notification.ChannelGlobal,
		Title:           alert.Title,
		Message:         alert.Message,
		Data:            alert.Data,
		RelatedEntities: alert.RelatedEntities,
		Source:          ediAlertSource,
		CorrelationID:   &correlationID,
	}
	if _, err = s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn(
			"failed to send EDI operational alert",
			zap.String("eventType", alert.EventType),
			zap.Error(err),
		)
	}
}

func (s *Service) notificationThrottled(
	ctx context.Context,
	alert *EDIOperationalAlert,
	correlationID string,
) (bool, error) {
	return s.notifications.ExistsRecent(ctx, repositories.ExistsRecentNotificationRequest{
		OrganizationID: alert.OrganizationID,
		BusinessUnitID: alert.BusinessUnitID,
		EventType:      alert.EventType,
		CorrelationID:  correlationID,
		Since:          timeutils.NowUnix() - ediAlertThrottleWindowSeconds,
	})
}
