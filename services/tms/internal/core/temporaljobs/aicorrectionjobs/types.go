package aicorrectionjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs"
)

const (
	AICorrectionRetentionWorkflowName = "AICorrectionRetentionWorkflow"
)

type AICorrectionRetentionInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Now   int64                        `json:"now,omitempty"`
}

type ListAICorrectionOrganizationsInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Limit int                          `json:"limit"`
}

type OrganizationAICorrectionInput struct {
	temporaljobs.TenantWorkItem
	Now int64 `json:"now"`
}

type OrganizationAICorrectionResult struct {
	Purged int64 `json:"purged"`
}

type AICorrectionRetentionResult struct {
	OrganizationsProcessed int      `json:"organizationsProcessed"`
	Purged                 int64    `json:"purged"`
	FailedOrganizations    []string `json:"failedOrganizations"`
}

func newAICorrectionRetentionResult() *AICorrectionRetentionResult {
	return &AICorrectionRetentionResult{FailedOrganizations: make([]string, 0)}
}

func (r *AICorrectionRetentionResult) absorb(organization *OrganizationAICorrectionResult) {
	r.OrganizationsProcessed++
	r.Purged += organization.Purged
}
