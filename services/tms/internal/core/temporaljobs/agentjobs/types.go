package agentjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

// Workflow names. A workflow a schedule starts is started by its function, so
// its name is the function's name: the SDK derives one from the other.
const (
	// agentflowChange marks where a run's loop moved into workflow code. A run
	// that started before it replays on the code it started on.
	agentflowChange = "agent-loop-in-workflow"

	AgentRunWorkflowName              = "AgentRunWorkflow"
	AgentEvaluationWorkflowName       = "AgentEvaluationWorkflow"
	AgentScheduledRunWorkflowName     = "AgentScheduledRunWorkflow"
	ReconcileSchedulesWorkflowName    = "ReconcileDefinitionSchedulesWorkflow"
	ReconcileSchedulesScheduleID      = "agent-definition-schedules"
	ExpireStaleProposalsWorkflowName  = "ExpireStaleProposalsWorkflow"
	ExpireStaleProposalsScheduleID    = "agent-proposal-expiry"
	DeleteStaleAskThreadsWorkflowName = "DeleteStaleAskThreadsWorkflow"
	DeleteStaleAskThreadsScheduleID   = "assistant-ask-thread-retention"
	AgentDecisionSignalName           = "agent-decision"
)

type AgentRunPayload struct {
	temporaltype.BasePayload

	RunID        pulid.ID          `json:"runId"`
	DefinitionID pulid.ID          `json:"definitionId"`
	Trigger      agent.RunTrigger  `json:"trigger"`
	SubjectType  agent.SubjectType `json:"subjectType"`
	SubjectID    pulid.ID          `json:"subjectId"`
	EventKind    agent.EventKind   `json:"eventKind,omitempty"`
	// Origin is the W3C traceparent of whatever started the run: the
	// request, the event or the schedule firing. Empty on a run started
	// before it was kept, or from outside any trace.
	Origin string `json:"traceOrigin,omitempty"`
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

// OpenRunInput is a prepared run, to be opened as a turn.
type OpenRunInput struct {
	Payload    *AgentRunPayload                `json:"payload"`
	Definition *agentdefinition.Definition     `json:"definition"`
	Subject    *agentdefinition.RuntimeSubject `json:"subject,omitempty"`
	Shadow     bool                            `json:"shadow"`
}

// OpenRunResult is the run as the workflow drives it.
type OpenRunResult struct {
	Run  agentflow.RunContext   `json:"run"`
	Turn agentruntime.TurnState `json:"turn"`
}

// FinishRunInput is what a run's loop came to.
type FinishRunInput struct {
	Payload    *AgentRunPayload                `json:"payload"`
	Definition *agentdefinition.Definition     `json:"definition"`
	Subject    *agentdefinition.RuntimeSubject `json:"subject,omitempty"`
	Run        *serviceports.RunResult         `json:"run,omitempty"`
	Failure    *modelcall.Failure              `json:"failure,omitempty"`
	Events     []temporaltype.StreamItem       `json:"events,omitempty"`
}

// FinishRunResult is what a finished run left for a person to decide.
type FinishRunResult struct {
	ProposalsRaised  int `json:"proposalsRaised"`
	PendingProposals int `json:"pendingProposals"`
}

type PendingProposalsInput struct {
	RunID      pulid.ID              `json:"runId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
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

// ScheduledRunPayload is what a definition's schedule fires with. Slot is the
// time the schedule fired for, filled in by the workflow it starts.
type ScheduledRunPayload struct {
	DefinitionID   pulid.ID `json:"definitionId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Slot           int64    `json:"slot,omitempty"`
}

type StartScheduledRunResult struct {
	Started bool   `json:"started"`
	RunID   string `json:"runId,omitempty"`
	Skipped string `json:"skipped,omitempty"`
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

// OpenReplayResult is an evaluation's replay as the workflow drives it. Done
// is set for an evaluation that had already finished.
type OpenReplayResult struct {
	Done      bool                     `json:"done"`
	Run       agentflow.RunContext     `json:"run"`
	Turn      agentruntime.TurnState   `json:"turn"`
	Originals []agent.OriginalProposal `json:"originals,omitempty"`
}

// FinishReplayInput is what a replay's loop came to.
type FinishReplayInput struct {
	Payload   *AgentEvaluationPayload  `json:"payload"`
	Originals []agent.OriginalProposal `json:"originals,omitempty"`
	Run       *serviceports.RunResult  `json:"run,omitempty"`
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
