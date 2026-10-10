package aifeedbackjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs"
)

const (
	AIFeedbackMaintenanceWorkflowName = "AIFeedbackMaintenanceWorkflow"
)

type AIFeedbackMaintenanceInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Now   int64                        `json:"now,omitempty"`
}

type ListAIFeedbackOrganizationsInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Limit int                          `json:"limit"`
}

type OrganizationAIFeedbackInput struct {
	temporaljobs.TenantWorkItem
	Now int64 `json:"now"`
}

type OrganizationAIFeedbackResult struct {
	Purged     int64 `json:"purged"`
	Suggested  int   `json:"suggested"`
	Covered    int   `json:"covered"`
	BelowFloor int   `json:"belowFloor"`
}

type AIFeedbackMaintenanceResult struct {
	OrganizationsProcessed int      `json:"organizationsProcessed"`
	Purged                 int64    `json:"purged"`
	Suggested              int      `json:"suggested"`
	Covered                int      `json:"covered"`
	FailedOrganizations    []string `json:"failedOrganizations"`
}

func newAIFeedbackMaintenanceResult() *AIFeedbackMaintenanceResult {
	return &AIFeedbackMaintenanceResult{FailedOrganizations: make([]string, 0)}
}

func (r *AIFeedbackMaintenanceResult) absorb(organization *OrganizationAIFeedbackResult) {
	r.OrganizationsProcessed++
	r.Purged += organization.Purged
	r.Suggested += organization.Suggested
	r.Covered += organization.Covered
}
