package extractionevaljobs

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ExtractionEvalRunWorkflowName = "ExtractionEvalRunWorkflow"
	casesPerExecution             = 100
	evaluateAttempts              = 3
)

type RunPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	RunID          pulid.ID `json:"runId"`
	AfterOrdinal   int      `json:"afterOrdinal,omitempty"`
	Evaluated      int      `json:"evaluated,omitempty"`
}

func (p *RunPayload) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type RunInput struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	RunID          pulid.ID `json:"runId"`
}

type ListPendingInput struct {
	RunInput
	AfterOrdinal int `json:"afterOrdinal"`
	Limit        int `json:"limit"`
}

type PendingCase struct {
	ResultID pulid.ID `json:"resultId"`
	Ordinal  int      `json:"ordinal"`
}

type ContinueDecision struct {
	Stop   bool   `json:"stop"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type EvaluateInput struct {
	RunInput
	ResultID pulid.ID `json:"resultId"`
}

type FinishInput struct {
	RunInput
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type FailInput struct {
	RunInput
	Message string `json:"message"`
}

type RunOutcome struct {
	RunID  pulid.ID `json:"runId"`
	Status string   `json:"status"`
}

func (in *RunInput) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: in.OrganizationID, BuID: in.BusinessUnitID}
}
