package agentcontract

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const AgentRunWorkflowName = "AgentRunWorkflow"

const AgentEvaluationWorkflowName = "AgentEvaluationWorkflow"

const AgentDecisionSignalName = "agent-decision"

type AgentRunPayload struct {
	temporaltype.BasePayload

	RunID        pulid.ID          `json:"runId"`
	DefinitionID pulid.ID          `json:"definitionId"`
	Trigger      agent.RunTrigger  `json:"trigger"`
	SubjectType  agent.SubjectType `json:"subjectType"`
	SubjectID    pulid.ID          `json:"subjectId"`
	EventKind    agent.EventKind   `json:"eventKind,omitempty"`
	// WaitID is the wait whose ending started the run, on a run that picks
	// up work an earlier run parked.
	WaitID pulid.ID `json:"waitId,omitempty"`
	// Origin is the W3C traceparent of whatever started the run: the
	// request, the event or the schedule firing. Empty on a run started
	// before it was kept, or from outside any trace.
	Origin string `json:"traceOrigin,omitempty"`
}

func (p *AgentRunPayload) TenantInfo() pagination.TenantInfo {
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

// AgentEvaluationPayload names the evaluation to carry out: a recorded run
// replayed against its agent as it is now.
type AgentEvaluationPayload struct {
	temporaltype.BasePayload

	EvaluationID pulid.ID `json:"evaluationId"`
}

func (p *AgentEvaluationPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: p.OrganizationID,
		BuID:  p.BusinessUnitID,
	}
}
