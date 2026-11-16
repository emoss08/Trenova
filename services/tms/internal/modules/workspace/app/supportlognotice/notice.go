package supportlognotice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportlogports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/shared/i18n"
	"go.uber.org/fx"
)

const (
	source         = "supportlognotice"
	EventSent      = "support_logs.sent"
	EventFailed    = "support_logs.failed"
	SupportLogPath = "/admin/support-access"
)

type NotificationCreator interface {
	Create(ctx context.Context, entity *notification.Notification) (*notification.Notification, error)
}

type Params struct {
	fx.In

	Notifications *notificationservice.Service
}

type Notifier struct {
	notifications NotificationCreator
}

var _ supportlogports.Notifier = (*Notifier)(nil)

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Notifier {
	return &Notifier{notifications: p.Notifications}
}

func (n *Notifier) LogRequestFinished(
	ctx context.Context,
	notice *supportlogports.FinishedNotice,
) error {
	req := notice.Request
	if req == nil || req.RequestedByID.IsNil() || !req.Status.Finished() {
		return nil
	}

	buID, userID, correlation := req.BusinessUnitID, req.RequestedByID, req.ID.String()
	entity := &notification.Notification{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: &buID,
		TargetUserID:   &userID,
		Channel:        notification.ChannelUser,
		Source:         source,
		CorrelationID:  &correlation,
		Data: map[string]any{
			"link":   SupportLogPath,
			"period": req.Period.String(),
			"status": req.Status.String(),
		},
	}
	if req.Status == supportlog.StatusSent {
		entity.EventType = EventSent
		entity.Priority = notification.PriorityMedium
		entity.Title = i18n.T(ctx, "Your logs reached Trenova support")
		entity.Message = sentMessage(ctx, req.Period)
	} else {
		entity.EventType = EventFailed
		entity.Priority = notification.PriorityHigh
		entity.Title = i18n.T(ctx, "Your logs could not be sent to Trenova support")
		entity.Message = req.FailureMessage
	}

	if _, err := n.notifications.Create(ctx, entity); err != nil {
		return fmt.Errorf("notify who asked for the logs: %w", err)
	}
	return nil
}

func sentMessage(ctx context.Context, period supportlog.Period) string {
	switch period {
	case supportlog.PeriodHour:
		return i18n.T(ctx, "The logs from the last hour were sent. Trenova support keeps them for 30 days.")
	case supportlog.PeriodWeek:
		return i18n.T(ctx, "The logs from the last 7 days were sent. Trenova support keeps them for 30 days.")
	default:
		return i18n.T(ctx, "The logs from the last 24 hours were sent. Trenova support keeps them for 30 days.")
	}
}
