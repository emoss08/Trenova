package notificationjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SendDueDigestsPayload struct {
	At int64 `json:"at"`
}

type ActivitiesParams struct {
	fx.In

	Notifications *notificationservice.Service
	Logger        *zap.Logger
}

type Activities struct {
	notifications *notificationservice.Service
	logger        *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		notifications: p.Notifications,
		logger:        p.Logger.Named("temporal.notification-digest"),
	}
}

func (a *Activities) SendDueDigestsActivity(
	ctx context.Context,
	payload *SendDueDigestsPayload,
) (*notificationservice.DigestRunResult, error) {
	at := time.Now()
	if payload != nil && payload.At > 0 {
		at = time.Unix(payload.At, 0)
	}

	result, err := a.notifications.SendDueDigests(ctx, at)
	if err != nil {
		a.logger.Error("failed to send notification digests", zap.Error(err))
		return nil, err
	}

	return result, nil
}
