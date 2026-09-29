package extractionshadowjobs

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ExtractionShadowWorkflowName = "ExtractionShadowWorkflow"
	runAttempts                  = 3
)

type ShadowPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ResultID       pulid.ID `json:"resultId"`
}

func (p *ShadowPayload) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type FailInput struct {
	ShadowPayload
	Message string `json:"message"`
}
