package aianalyticsjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
)

const RefreshAIAnalyticsWorkflowName = "RefreshAIAnalyticsWorkflow"

const (
	// activeWindow is how recent an organization's AI activity must be for
	// the refresh to visit it: twelve runs of the schedule, so a refresh
	// that failed is retried for an hour before the tenant waits on its next
	// activity.
	activeWindow = time.Hour
	// maxActiveTenants bounds each source read of the active set.
	maxActiveTenants = 10_000
)

type RefreshAIAnalyticsInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
}

type ListAIAnalyticsOrganizationsInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Limit int                          `json:"limit"`
}

type OrganizationAIAnalyticsInput struct {
	temporaljobs.TenantWorkItem
}

type OrganizationAIAnalyticsResult struct {
	Units    int  `json:"units"`
	Hours    int  `json:"hours"`
	CaughtUp bool `json:"caughtUp"`
}

type RefreshAIAnalyticsResult struct {
	OrganizationsProcessed int      `json:"organizationsProcessed"`
	Units                  int      `json:"units"`
	FailedOrganizations    []string `json:"failedOrganizations"`
}

func newRefreshAIAnalyticsResult() *RefreshAIAnalyticsResult {
	return &RefreshAIAnalyticsResult{FailedOrganizations: make([]string, 0)}
}
