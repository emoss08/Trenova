package weatheralertjobs

import (
	"context"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ListWeatherAlertTenantsPayload struct {
	Limit int `json:"limit"`
}

type ListWeatherAlertTenantsResult struct {
	Tenants []temporaljobs.TenantWorkItem `json:"tenants"`
}

type PollNWSAlertsTenantPayload struct {
	temporaljobs.TenantWorkItem
}

type PollNWSAlertsResult struct {
	temporaljobs.TenantRunResult
}

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

func (a *Activities) ListWeatherAlertTenantsActivity(
	ctx context.Context,
	payload *ListWeatherAlertTenantsPayload,
) (*ListWeatherAlertTenantsResult, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantScanLimit)
	tenants, err := a.service.ListWeatherAlertTenants(ctx, limit)
	if err != nil {
		return nil, err
	}

	return &ListWeatherAlertTenantsResult{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
	}, nil
}

func (a *Activities) PollNWSAlertsForTenantActivity(
	ctx context.Context,
	payload *PollNWSAlertsTenantPayload,
) error {
	tenantInfo := payload.TenantInfo()
	recordActivityHeartbeat(ctx, "polling-nws-alerts", tenantInfo.OrgID.String())
	if err := a.service.PollNWSAlertsForTenant(ctx, tenantInfo); err != nil {
		a.logger.Error("Weather alert tenant poll activity failed",
			zap.String("orgID", tenantInfo.OrgID.String()),
			zap.String("buID", tenantInfo.BuID.String()),
			zap.Error(err))
		return err
	}

	return nil
}

func (a *Activities) ExpireStaleWeatherAlertsActivity(ctx context.Context) error {
	recordActivityHeartbeat(ctx, "expiring-stale-weather-alerts")
	return a.service.ExpireStaleWeatherAlerts(ctx)
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
