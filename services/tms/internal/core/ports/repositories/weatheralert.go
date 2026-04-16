package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/weatheralert"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetWeatherAlertByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type UpsertWeatherAlertResult struct {
	Alert        *weatheralert.WeatherAlert
	Activity     *weatheralert.Activity
	Created      bool
	Changed      bool
	ActivityType weatheralert.ActivityType
}

type ExpireWeatherAlertsResult struct {
	ExpiredCount int
}

type WeatherAlertRepository interface {
	ListTenants(ctx context.Context) ([]pagination.TenantInfo, error)
	GetActiveAlerts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*weatheralert.WeatherAlert, error)
	GetByID(ctx context.Context, req GetWeatherAlertByIDRequest) (*weatheralert.WeatherAlert, error)
	GetActivities(
		ctx context.Context,
		req GetWeatherAlertByIDRequest,
	) ([]*weatheralert.Activity, error)
	UpsertAlert(
		ctx context.Context,
		alert *weatheralert.WeatherAlert,
	) (*UpsertWeatherAlertResult, error)
	ExpireStaleAlerts(ctx context.Context) (*ExpireWeatherAlertsResult, error)
}
