package aianalyticsjobs

import "github.com/emoss08/trenova/internal/infrastructure/temporaljobs"

const RefreshAIAnalyticsWorkflowName = "RefreshAIAnalyticsWorkflow"

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
