package weatheralertjobs

import (
	"context"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Service serviceports.WeatherAlertService
	Logger  *zap.Logger
}

type Activities struct {
	service serviceports.WeatherAlertService
	logger  *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		service: p.Service,
		logger:  p.Logger.Named("temporal.weather-alert"),
	}
}

func (a *Activities) PollNWSAlertsActivity(ctx context.Context) error {
	a.logger.Info("Starting weather alert poll activity")
	recordActivityHeartbeat(ctx, "polling-nws-alerts")

	if err := a.service.PollNWSAlerts(ctx); err != nil {
		a.logger.Error("Weather alert poll activity failed", zap.Error(err))
		return err
	}

	a.logger.Info("Weather alert poll activity completed")
	return nil
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
