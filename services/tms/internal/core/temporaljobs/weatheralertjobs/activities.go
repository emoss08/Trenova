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
	FeedUnchanged bool `json:"feedUnchanged"`
	AlertsInFeed  int  `json:"alertsInFeed"`
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

func (a *Activities) PollNWSAlertsActivity(ctx context.Context) (*PollNWSAlertsResult, error) {
	recordActivityHeartbeat(ctx, "polling-nws-alerts")

	polled, err := a.service.PollNWSAlerts(ctx, func(details ...any) {
		recordActivityHeartbeat(ctx, details...)
	})
	if err != nil {
		a.logger.Error("Weather alert poll activity failed", zap.Error(err))
		return nil, err
	}

	result := &PollNWSAlertsResult{
		FeedUnchanged: polled.FeedUnchanged,
		AlertsInFeed:  polled.AlertsInFeed,
	}
	result.TenantsScanned = polled.TenantsScanned
	for _, synced := range polled.Tenants {
		if synced.Err != nil {
			a.logger.Error("Weather alert tenant poll failed",
				zap.String("orgID", synced.TenantInfo.OrgID.String()),
				zap.String("buID", synced.TenantInfo.BuID.String()),
				zap.Error(synced.Err))
			result.AddFailure(temporaljobs.NewTenantWorkItem(synced.TenantInfo, 1), synced.Err)
			continue
		}

		result.AddTenantResult(synced.Written, synced.Unchanged)
	}

	a.logger.Info("Weather alert poll activity completed",
		zap.Bool("feedUnchanged", result.FeedUnchanged),
		zap.Int("alertsInFeed", result.AlertsInFeed),
		zap.Int("tenantsProcessed", result.TenantsProcessed),
		zap.Int("alertsWritten", result.RecordsProcessed),
		zap.Int("failures", result.FailureCount))

	return result, nil
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
