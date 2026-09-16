package services

import (
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ListInsightsRequest reads the findings a particular person may see.
//
// UserID is required rather than optional because the answer depends on it: an
// insight naming a customer and a revenue figure is filtered out for a reader
// who cannot read customers, so there is no such thing as an unattributed list.
type ListInsightsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Categories []insight.Category
	Limit      int
}

// BrowseInsightsRequest reads the history rather than the home screen's slice:
// any status, any severity, a page at a time.
type BrowseInsightsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Categories []insight.Category
	Severities []insight.Severity
	Statuses   []insight.Status
	Limit      int
	Offset     int
}

type RestoreInsightRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type DismissInsightRequest struct {
	ID         pulid.ID
	UserID     pulid.ID
	Reason     string
	TenantInfo pagination.TenantInfo
}
