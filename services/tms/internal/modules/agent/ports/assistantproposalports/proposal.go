package assistantproposalports

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

// AssistantProposal is a write the agent proposed during a conversation. It is a
// persisted agent proposal, so it can be approved or rejected through the same
// decision endpoint as any other agent's proposal.
type AssistantProposal struct {
	ID              pulid.ID             `json:"id"`
	RunID           pulid.ID             `json:"runId"`
	ToolName        string               `json:"toolName"`
	Arguments       map[string]any       `json:"arguments"`
	Rationale       string               `json:"rationale"`
	AutonomyTier    agent.AutonomyTier   `json:"autonomyTier"`
	Status          agent.ProposalStatus `json:"status"`
	SourceMessageID pulid.ID             `json:"sourceMessageId"`
	// Confidence is the model's own estimate, 0 to 1, shown so an approver can
	// weigh the rationale.
	Confidence float64 `json:"confidence"`
	// ExecutedAt and ExecutionError report what happened after approval. An
	// accepted proposal with neither set was approved but has not run yet.
	ExecutedAt     *int64 `json:"executedAt"`
	ExecutionError string `json:"executionError"`
	// ExecutionResult is what the run made, and for a write over many
	// records each one that did not go through.
	ExecutionResult *agentrecord.ToolExecutionResult `json:"executionResult,omitempty"`
	// ExpiresAt is when a pending proposal stops being decidable. Zero means
	// it was made before expiry existed.
	ExpiresAt int64 `json:"expiresAt"`
	// Hold is set while a shadow switch keeps the proposal from being decided,
	// so the client can say so instead of offering an approval the server
	// will refuse. Nil means it can be decided.
	Hold *ProposalHold `json:"hold"`
	// PlanID and PlanStep are set when the proposal is one step of a plan
	// decided as a whole; the client groups such proposals under the plan.
	PlanID   pulid.ID `json:"planId"`
	PlanStep int      `json:"planStep"`
	// SimulatedAt and Simulation report a write previewed instead of made,
	// because the agent was in simulation when it was cleared.
	SimulatedAt *int64                `json:"simulatedAt"`
	Simulation  *agent.ToolSimulation `json:"simulation"`
	// Fields are the tool's parameters as a person may edit them before
	// approving, derived from the tool's schema. Set only while the proposal
	// is pending, since a decided one can no longer be changed.
	Fields []toolschema.Field `json:"fields"`
	// Modifications are the values the approver changed before approving,
	// keyed by parameter. Nil when it was approved as proposed.
	Modifications map[string]any `json:"modifications"`
	// PendingModifications are the values a person changed and has not yet
	// approved with, keyed by parameter, kept so the edit survives a reload.
	// Nil once the proposal is decided, or when nothing was changed.
	PendingModifications map[string]any `json:"pendingModifications"`
	// AgentID and AgentName are the agent that proposed it: the
	// conversation's own, or another agent it handed a task to, whose
	// proposal it is and whose trust a decision on it teaches.
	AgentID   pulid.ID `json:"agentId,omitempty"`
	AgentName string   `json:"agentName,omitempty"`

	CreatedAt       int64    `json:"createdAt"`
	DecidedAt       *int64   `json:"decidedAt"`
	DecidedByUserID pulid.ID `json:"decidedByUserId"`
	DecisionNote    string   `json:"decisionNote"`
	ReviewRequired  bool     `json:"reviewRequired"`
}

// ProposalEdits is what a pending proposal holds as changed once a person
// saved their edits: only the values that differ from what the agent proposed.
type ProposalEdits struct {
	ProposalID           pulid.ID       `json:"proposalId"`
	PendingModifications map[string]any `json:"pendingModifications"`
}

// AssistantPlan is several of a turn's proposals as one decision, as the
// client shows it: approve or reject all of them, in order.
type AssistantPlan struct {
	ID              pulid.ID              `json:"id"`
	RunID           pulid.ID              `json:"runId"`
	Title           string                `json:"title"`
	Summary         string                `json:"summary"`
	Status          agentplan.PlanStatus  `json:"status"`
	StepCount       int                   `json:"stepCount"`
	CompletedSteps  int                   `json:"completedSteps"`
	FailedStep      *int                  `json:"failedStep"`
	FailureError    string                `json:"failureError"`
	DecidedAt       *int64                `json:"decidedAt"`
	DecidedByUserID pulid.ID              `json:"decidedByUserId"`
	DecidedByName   string                `json:"decidedByName"`
	DecisionNote    string                `json:"decisionNote"`
	PreviewDigest   string                `json:"previewDigest"`
	ExpiresAt       int64                 `json:"expiresAt"`
	CommitsAt       *int64                `json:"commitsAt"`
	UndoneAt        *int64                `json:"undoneAt"`
	RunMode         agentplan.PlanRunMode `json:"runMode"`
	Hold            *ProposalHold         `json:"hold"`
	CreatedAt       int64                 `json:"createdAt"`
	// AgentID and AgentName are the agent whose proposals the plan groups.
	AgentID   pulid.ID `json:"agentId,omitempty"`
	AgentName string   `json:"agentName,omitempty"`
	// Script is the script whose prepared changes are the plan's steps;
	// absent for a plan of the model's own calls.
	Script *agentplan.ScriptOrigin `json:"script,omitempty"`
}

// ProposalHold names the switch holding a proposal and, when it is an agent's
// own, which agent.
type ProposalHold struct {
	Reason    string `json:"reason"`
	AgentName string `json:"agentName"`
}
