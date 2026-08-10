package tenderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	notificationSource       = "tender_service"
	notificationPriorityHigh = notification.PriorityHigh
	notificationPriorityMed  = notification.PriorityMedium

	dispatchConsoleLink = "/dispatch-console"
)

type dispatchNotification struct {
	EventType     string
	Priority      notification.Priority
	Title         string
	Message       string
	CorrelationID string
	Link          string
}

// notifyDispatch raises a global in-app notification for dispatchers. Failures
// are logged only: a notification must never fail the workflow transition it
// decorates.
func (s *Service) notifyDispatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	n *dispatchNotification,
) {
	if s.notifications == nil {
		return
	}

	link := n.Link
	if link == "" {
		link = dispatchConsoleLink
	}
	buID := tenantInfo.BuID
	entity := &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		EventType:      n.EventType,
		Priority:       n.Priority,
		Channel:        notification.ChannelGlobal,
		Title:          n.Title,
		Message:        n.Message,
		Source:         notificationSource,
		Data:           map[string]any{"link": link},
	}
	if n.CorrelationID != "" {
		correlationID := n.CorrelationID
		entity.CorrelationID = &correlationID
	}

	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create tender notification",
			zap.Error(err), zap.String("eventType", n.EventType))
	}
}
