package aituneupjobs

import "github.com/emoss08/trenova/internal/core/temporaljobs"

const ComputeAITuneUpsWorkflowName = "ComputeAITuneUpsWorkflow"

type ComputeAITuneUpsInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
}

type ListAITuneUpOrganizationsInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Limit int                          `json:"limit"`
}

type OrganizationAITuneUpsInput struct {
	temporaljobs.TenantWorkItem
}

type OrganizationAITuneUpsResult struct {
	Suggested int `json:"suggested"`
}

type ComputeAITuneUpsResult struct {
	OrganizationsProcessed int      `json:"organizationsProcessed"`
	Suggested              int      `json:"suggested"`
	FailedOrganizations    []string `json:"failedOrganizations"`
}

func newComputeAITuneUpsResult() *ComputeAITuneUpsResult {
	return &ComputeAITuneUpsResult{FailedOrganizations: make([]string, 0)}
}
