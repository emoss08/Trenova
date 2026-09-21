package agentjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	AgentRunWorkflowName              = "AgentRunWorkflow"
	AgentEvaluationWorkflowName       = "AgentEvaluationWorkflow"
	AgentSweepWorkflowName            = "AgentSweepWorkflow"
	ExpireStaleProposalsWorkflowName  = "ExpireStaleAgentProposalsWorkflow"
	ExpireStaleProposalsScheduleID    = "agent-proposal-expiry"
	DeleteStaleAskThreadsWorkflowName = "DeleteStaleAskThreadsWorkflow"
	DeleteStaleAskThreadsScheduleID   = "assistant-ask-thread-retention"
	AgentDecisionSignalName           = "agent-decision"
	SweepScheduleID                   = "agent-definition-sweep"
)

type AgentRunPayload struct {
	temporaltype.BasePayload

	RunID        pulid.ID          `json:"runId"`
	DefinitionID pulid.ID          `json:"definitionId"`
	Trigger      agent.RunTrigger  `json:"trigger"`
	SubjectType  agent.SubjectType `json:"subjectType"`
	SubjectID    pulid.ID          `json:"subjectId"`
	EventKind    agent.EventKind   `json:"eventKind,omitempty"`
}

func (p *AgentRunPayload) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: p.OrganizationID,
		BuID:  p.BusinessUnitID,
	}
}

type DecisionSignal struct {
	ProposalID      pulid.ID           `json:"proposalId"`
	Decision        agent.DecisionType `json:"decision"`
	DecidedByUserID pulid.ID           `json:"decidedByUserId"`
	ReasonCode      string             `json:"reasonCode"`
}

type PrepareRunResult struct {
	Definition             *agentdefinition.Definition     `json:"definition"`
	Subject                *agentdefinition.RuntimeSubject `json:"subject,omitempty"`
	ShadowMode             bool                            `json:"shadowMode"`
	DecisionTimeoutSeconds int                             `json:"decisionTimeoutSeconds"`
	RunTimeoutSeconds      int                             `json:"runTimeoutSeconds"`
}

type RunAgentInput struct {
	Payload    *AgentRunPayload                `json:"payload"`
	Definition *agentdefinition.Definition     `json:"definition"`
	Subject    *agentdefinition.RuntimeSubject `json:"subject,omitempty"`
}

type RunAgentResult struct {
	Reply            string `json:"reply"`
	Model            string `json:"model"`
	ToolCallsUsed    int    `json:"toolCallsUsed"`
	Exhausted        bool   `json:"exhausted"`
	ProposalsRaised  int    `json:"proposalsRaised"`
	PendingProposals int    `json:"pendingProposals"`
}

type CompleteRunInput struct {
	RunID      pulid.ID              `json:"runId"`
	Status     agent.RunStatus       `json:"status"`
	Error      string                `json:"error,omitempty"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ExpireProposalsInput struct {
	RunID      pulid.ID              `json:"runId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DueDefinition struct {
	DefinitionID   pulid.ID `json:"definitionId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	NextRunAt      int64    `json:"nextRunAt"`
}

type ListDueDefinitionsInput struct {
	Now   int64 `json:"now"`
	Limit int   `json:"limit"`
}

type ListDueDefinitionsResult struct {
	Due []DueDefinition `json:"due"`
}

type StartDueRunResult struct {
	Started bool   `json:"started"`
	RunID   string `json:"runId,omitempty"`
	Skipped string `json:"skipped,omitempty"`
}

type SweepResult struct {
	Found   int `json:"found"`
	Started int `json:"started"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

// ExpireStaleProposalsResult is what one expiry sweep did.
type ExpireStaleProposalsResult struct {
	Expired  int `json:"expired"`
	Reminded int `json:"reminded"`
}

// ExpireStaleProposalsInput carries the workflow's clock, so the activity is
// deterministic with respect to the workflow that ran it.
type ExpireStaleProposalsInput struct {
	Now int64 `json:"now"`
}

// DeleteStaleAskThreadsInput names the cut-off: quick questions with no
// activity since Before, and never kept, are removed.
type DeleteStaleAskThreadsInput struct {
	Before int64 `json:"before"`
}

// DeleteStaleAskThreadsResult is what one retention sweep removed.
type DeleteStaleAskThreadsResult struct {
	Deleted int `json:"deleted"`
}

// RemindPendingProposalsInput says how long a proposal may wait before its
// deciders are told again.
type RemindPendingProposalsInput struct {
	Now              int64 `json:"now"`
	OlderThanSeconds int64 `json:"olderThanSeconds"`
}

type RemindPendingProposalsResult struct {
	Reminded int `json:"reminded"`
}

// AgentEvaluationPayload names the evaluation to carry out: a recorded run
// replayed against its agent as it is now.
type AgentEvaluationPayload struct {
	temporaltype.BasePayload

	EvaluationID pulid.ID `json:"evaluationId"`
}

func (p *AgentEvaluationPayload) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: p.OrganizationID,
		BuID:  p.BusinessUnitID,
	}
}

type ReplayRunResult struct {
	Model         string `json:"model"`
	ToolCallsUsed int    `json:"toolCallsUsed"`
	Actions       int    `json:"actions"`
}

type FailEvaluationInput struct {
	EvaluationID pulid.ID              `json:"evaluationId"`
	Error        string                `json:"error"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
}
