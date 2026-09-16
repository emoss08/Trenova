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

// GetInsightDetailRequest reads one finding in full.
type GetInsightDetailRequest struct {
	ID         pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// InsightDetail is one finding with the context that only matters once somebody
// has stopped to look at it.
type InsightDetail struct {
	Insight *insight.Insight `json:"insight"`
	// History is earlier runs of the same finding, newest first. It is what turns
	// a number into a trend: whether this is getting worse is the first question
	// anyone asks and the card alone cannot answer it.
	History []*insight.Insight `json:"history"`
	// Explanation is what the rule looks for and what it declines to report, so a
	// reader can audit why this reached them.
	Explanation InsightExplanation `json:"explanation"`
}

// InsightExplanation mirrors a detector's own description of its rule. It is a
// port-level type rather than the detector's so the HTTP layer does not reach
// into the detector package for a wire shape.
type InsightExplanation struct {
	Measures  string `json:"measures"`
	Threshold string `json:"threshold"`
	Excludes  string `json:"excludes"`
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
