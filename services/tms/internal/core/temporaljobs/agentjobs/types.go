package agentjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	BillingExceptionAgentWorkflowName = "BillingExceptionAgentWorkflow"
	AgentDecisionSignalName           = "agent_decision"

	billingExceptionPromptVersionFallback = "billing-exception-v1"
)

type AgentRunPayload struct {
	temporaltype.BasePayload

	RunID                  pulid.ID          `json:"runId"`
	SubjectType            agent.SubjectType `json:"subjectType"`
	SubjectID              pulid.ID          `json:"subjectId"`
	PromptVersion          string            `json:"promptVersion"`
	ShadowMode             bool              `json:"shadowMode"`
	DecisionTimeoutSeconds int               `json:"decisionTimeoutSeconds"`
}

func (p *AgentRunPayload) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.UserID,
	}
}

type DecisionSignal struct {
	ProposalID      pulid.ID           `json:"proposalId"`
	Decision        agent.DecisionType `json:"decision"`
	DecidedByUserID pulid.ID           `json:"decidedByUserId"`
	ReasonCode      string             `json:"reasonCode"`
}

type GatherContextResult struct {
	Context          serviceports.DelimitedContext `json:"context"`
	InputContextHash string                        `json:"inputContextHash"`
	SubjectID        pulid.ID                      `json:"subjectId"`
}

type DiagnoseActivityInput struct {
	RunID         pulid.ID                      `json:"runId"`
	PromptVersion string                        `json:"promptVersion"`
	TenantInfo    pagination.TenantInfo         `json:"tenantInfo"`
	Context       serviceports.DelimitedContext `json:"context"`
}

type DiagnoseActivityResult struct {
	Proposals       []serviceports.ProposedAction  `json:"proposals"`
	Exceptions      []serviceports.RaisedException `json:"exceptions"`
	ModelIdentifier string                         `json:"modelIdentifier"`
}

type PersistDiagnosisInput struct {
	RunID           pulid.ID                       `json:"runId"`
	SubjectType     agent.SubjectType              `json:"subjectType"`
	SubjectID       pulid.ID                       `json:"subjectId"`
	ModelIdentifier string                         `json:"modelIdentifier"`
	TenantInfo      pagination.TenantInfo          `json:"tenantInfo"`
	Proposals       []serviceports.ProposedAction  `json:"proposals"`
	Exceptions      []serviceports.RaisedException `json:"exceptions"`
}

type PersistDiagnosisResult struct {
	ProposalsPersisted  int `json:"proposalsPersisted"`
	ExceptionsPersisted int `json:"exceptionsPersisted"`
}

type CompleteRunInput struct {
	RunID      pulid.ID              `json:"runId"`
	Status     agent.RunStatus       `json:"status"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ExpireProposalsInput struct {
	RunID      pulid.ID              `json:"runId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}
