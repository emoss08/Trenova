package notificationjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	NotificationService services.NotificationService
}

type Activities struct {
	ns services.NotificationService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ns: p.NotificationService,
	}
}

func (a *Activities) SendNotificationActivity(
	ctx context.Context,
	payload *SendNotificationPayload,
) error {
	activity.RecordHeartbeat(ctx, "sending notification")

	if err := a.ns.SendJobCompletionNotification(ctx, payload); err != nil {
		return err
	}

	activity.RecordHeartbeat(ctx, "notification sent successfully")

	return nil
}

func (a *Activities) SendConfigurationCopiedNotificationActivity(
	ctx context.Context,
	payload *SendConfigurationCopiedNotificationPayload,
) error {
	activity.RecordHeartbeat(ctx, "sending configuration copied notification")

	if err := a.ns.SendConfigurationCopiedNotification(ctx, payload); err != nil {
		return err
	}

	activity.RecordHeartbeat(ctx, "configuration copied notification sent successfully")

	return nil
}
