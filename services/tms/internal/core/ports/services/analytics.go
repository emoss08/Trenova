/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package services

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
)

// AnalyticsPage represents a specific analytics page
type AnalyticsPage string

const (
	ShipmentAnalyticsPage        AnalyticsPage = "shipment-management"
	BillingClientAnalyticsPage   AnalyticsPage = "billing-client"
	DedicatedLaneSuggestionsPage AnalyticsPage = "dedicated-lane-suggestions"
)

// DateRange represents a time range for analytics queries
type DateRange struct {
	StartDate int64 `json:"startDate"` // Unix timestamp
	EndDate   int64 `json:"endDate"`   // Unix timestamp
}

// AnalyticsRequestOptions provides filtering parameters for analytics queries
type AnalyticsRequestOptions struct {
	OrgID     pulid.ID      `json:"organizationId"`
	BuID      pulid.ID      `json:"businessUnitId"`
	UserID    pulid.ID      `json:"userId"`
	Page      AnalyticsPage `json:"page"`
	DateRange *DateRange    `json:"dateRange,omitempty"`
	Limit     int           `json:"limit,omitempty"`
}

// AnalyticsData is a generic container for analytics data
type AnalyticsData map[string]any

// AnalyticsPageProvider is an interface that each analytics page provider must implement
type AnalyticsPageProvider interface {
	// GetPage returns the page identifier this provider handles
	GetPage() AnalyticsPage

	// GetAnalyticsData returns the analytics data for this page
	GetAnalyticsData(ctx context.Context, opts *AnalyticsRequestOptions) (AnalyticsData, error)
}

// AnalyticsRegistry defines a registry for analytics page providers
type AnalyticsRegistry interface {
	// RegisterProvider registers an analytics page provider
	RegisterProvider(provider AnalyticsPageProvider)

	// GetProvider returns the provider for a specific page
	GetProvider(page AnalyticsPage) (AnalyticsPageProvider, bool)
}

// AnalyticsService defines the interface for retrieving analytics data
type AnalyticsService interface {
	// GetRegistry returns the analytics registry
	GetRegistry() AnalyticsRegistry

	// GetAnalytics returns analytics data for a specific page
	GetAnalytics(ctx context.Context, opts *AnalyticsRequestOptions) (AnalyticsData, error)
}
