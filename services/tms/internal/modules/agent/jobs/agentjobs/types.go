package agentjobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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

	ReconcileSchedulesWorkflowName    = "ReconcileDefinitionSchedulesWorkflow"
	ReconcileSchedulesScheduleID      = "agent-definition-schedules"
	ExpireStaleProposalsWorkflowName  = "ExpireStaleProposalsWorkflow"
	ExpireStaleProposalsScheduleID    = "agent-proposal-expiry"
	DeleteStaleAskThreadsWorkflowName = "DeleteStaleAskThreadsWorkflow"
	DeleteStaleAskThreadsScheduleID   = "assistant-ask-thread-retention"
)

type PrepareRunResult struct {
	Definition             *agentdefinition.Definition     `json:"definition"`
	Subject                *agentdefinition.RuntimeSubject `json:"subject,omitempty"`
	ShadowMode             bool                            `json:"shadowMode"`
	DecisionTimeoutSeconds int                             `json:"decisionTimeoutSeconds"`
	RunTimeoutSeconds      int                             `json:"runTimeoutSeconds"`
}

type RunAgentInput struct {
	Payload    *agentcontract.AgentRunPayload  `json:"payload"`
	Definition *agentdefinition.Definition     `json:"definition"`
	Subject    *agentdefinition.RuntimeSubject `json:"subject,omitempty"`
}

// OpenRunInput is a prepared run, to be opened as a turn.
type OpenRunInput struct {
	Payload    *agentcontract.AgentRunPayload  `json:"payload"`
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
	Payload    *agentcontract.AgentRunPayload  `json:"payload"`
	Definition *agentdefinition.Definition     `json:"definition"`
	Subject    *agentdefinition.RuntimeSubject `json:"subject,omitempty"`
	Run        *agentports.RunResult           `json:"run,omitempty"`
	Failure    *modelcallcontract.Failure      `json:"failure,omitempty"`
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
	Payload   *agentcontract.AgentEvaluationPayload `json:"payload"`
	Originals []agent.OriginalProposal              `json:"originals,omitempty"`
	Run       *agentports.RunResult                 `json:"run,omitempty"`
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
