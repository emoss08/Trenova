package kernelports

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
)

type AnalyticsPage string

const ShipmentAnalyticsPage = AnalyticsPage("shipment-management")

const APIKeyAnalyticsPage = AnalyticsPage("api-key-management")

const AnalyticsIncludeMetrics = "metrics"

type DateRange struct {
	StartDate int64 `json:"startDate"` // Unix timestamp
	EndDate   int64 `json:"endDate"`   // Unix timestamp
}

type AnalyticsRequestOptions struct {
	OrgID      pulid.ID      `json:"organizationId"`
	BuID       pulid.ID      `json:"businessUnitId"`
	UserID     pulid.ID      `json:"userId"`
	Principal  PrincipalInfo `json:"-"`
	Page       AnalyticsPage `json:"page"`
	DateRange  *DateRange    `json:"dateRange,omitempty"`
	Timezone   string        `json:"timezone"`
	Limit      int           `json:"limit,omitempty"`
	Offset     int           `json:"offset,omitempty"`
	WindowDays int           `json:"windowDays,omitempty"`
	Include    string        `json:"include,omitempty"`
}

type AnalyticsData map[string]any

type AnalyticsPageProvider interface {
	GetPage() AnalyticsPage
	GetAnalyticsData(ctx context.Context, opts *AnalyticsRequestOptions) (AnalyticsData, error)
}

type AnalyticsRegistry interface {
	RegisterProvider(provider AnalyticsPageProvider)
	GetProvider(page AnalyticsPage) (AnalyticsPageProvider, bool)
}

type AnalyticsService interface {
	GetRegistry() AnalyticsRegistry
	GetAnalytics(ctx context.Context, opts *AnalyticsRequestOptions) (AnalyticsData, error)
}
