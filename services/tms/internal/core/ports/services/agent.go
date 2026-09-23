package services

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ErrAgentRunAlreadyOpen is a start refused because the agent already has a
// run open for that subject. The open run will see what the new one would
// have; it is a skip, not a failure.
var ErrAgentRunAlreadyOpen = errors.New("this agent already has an open run for that subject")

// StartAgentRunForDefinitionRequest starts a background run of one agent.
// The definition is named by id or by its system key; the subject is optional
// and defaults to the organization itself.
type StartAgentRunForDefinitionRequest struct {
	DefinitionID pulid.ID
	SystemKey    string
	SubjectType  agent.SubjectType
	SubjectID    pulid.ID
	Trigger      agent.RunTrigger
	EventKind    agent.EventKind
	// Slot is the schedule slot the run fills. It keys the workflow id, so a
	// slot starts one run however often its start is retried.
	Slot       int64
	TenantInfo pagination.TenantInfo
}

// StartInlineAgentRunRequest opens a run for an agent whose reasoning happens in-process
// rather than in a Temporal workflow.
type StartInlineAgentRunRequest struct {
	AgentType         agent.Type
	AgentDefinitionID pulid.ID
	SubjectType       agent.SubjectType
	SubjectID         pulid.ID
	PromptVersion     string
	Summary           string
	Trigger           agent.RunTrigger
	TenantInfo        pagination.TenantInfo
}

type AgentRunService interface {
	StartForDefinition(
		ctx context.Context,
		req *StartAgentRunForDefinitionRequest,
		actor *RequestActor,
	) (*agent.AgentRun, error)
	// StartInline records a run for a deterministic agent that reasons in-process. Such
	// agents still need the run row so their proposals, decisions, and audit trail are
	// indistinguishable from an LLM-backed agent's.
	StartInline(
		ctx context.Context,
		req *StartInlineAgentRunRequest,
		actor *RequestActor,
	) (*agent.AgentRun, error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentRunConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentRun], error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentRunByIDRequest,
	) (*agent.AgentRun, error)
}

type AgentProposalService interface {
	List(
		ctx context.Context,
		req *repositories.ListAgentProposalRequest,
	) (*pagination.ListResult[*agent.AgentProposal], error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentProposalConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentProposal], error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentProposalByIDRequest,
	) (*agent.AgentProposal, error)
}

type FlagAgentExceptionRequest struct {
	RunID          pulid.ID
	Category       agent.ExceptionCategory
	Severity       agent.Severity
	SubjectType    agent.SubjectType
	SubjectID      pulid.ID
	AttemptSummary string
	Evidence       []agent.EvidenceRef
	BlastRadius    int
	TenantInfo     pagination.TenantInfo
}

type ResolveAgentExceptionRequest struct {
	ID              pulid.ID
	ResolutionState agent.ResolutionState
	ResolutionNotes string
	TenantInfo      pagination.TenantInfo
}

type AgentExceptionService interface {
	Flag(
		ctx context.Context,
		req *FlagAgentExceptionRequest,
		actor *RequestActor,
	) (*agent.AgentException, error)
	List(
		ctx context.Context,
		req *repositories.ListAgentExceptionRequest,
	) (*pagination.ListResult[*agent.AgentException], error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentExceptionConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentException], error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentExceptionByIDRequest,
	) (*agent.AgentException, error)
	Resolve(
		ctx context.Context,
		req *ResolveAgentExceptionRequest,
		actor *RequestActor,
	) (*agent.AgentException, error)
}

type DecideAgentProposalRequest struct {
	ProposalID    pulid.ID
	Decision      agent.DecisionType
	Modifications map[string]any
	ReasonCode    string
	TenantInfo    pagination.TenantInfo
	// WithinPlan says the decision is one step of a plan being decided as a
	// whole. The run's workflow is signalled once by the plan, not once per
	// step, and a step's execution failure is reported to the caller so the
	// plan can stop rather than logged and swallowed.
	WithinPlan bool
}

// DecideAgentPlanRequest decides every pending step of a plan at once.
type DecideAgentPlanRequest struct {
	PlanID     pulid.ID
	Decision   agent.DecisionType
	ReasonCode string
	TenantInfo pagination.TenantInfo
}

type AgentPlanService interface {
	Decide(
		ctx context.Context,
		req *DecideAgentPlanRequest,
		actor *RequestActor,
	) (*agent.AgentPlan, error)
	GetByID(ctx context.Context, req repositories.GetAgentPlanByIDRequest) (*agent.AgentPlan, error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentPlanConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentPlan], error)
}

// PendingProposalsNotice is what the recorder hands the notifier once a run's
// proposals are stored: the agent, the run and the proposals themselves.
type PendingProposalsNotice struct {
	Definition *agentdefinition.Definition
	Run        *agent.AgentRun
	Proposals  []*agent.AgentProposal
}

// RemindPendingProposalsRequest asks for every proposal from a background run
// that has waited longer than OlderThan to be brought to its deciders again.
type RemindPendingProposalsRequest struct {
	Now       int64
	OlderThan time.Duration
	Limit     int
}

// AgentProposalNotifier tells the people who can decide a proposal that one
// is waiting. A proposal a person cannot see from where they are is a decision
// that never gets made; the panel is not where a dispatcher lives.
type AgentProposalNotifier interface {
	NotifyPending(ctx context.Context, notice PendingProposalsNotice) error
	RemindPending(ctx context.Context, req RemindPendingProposalsRequest) (int, error)
}

// AgentTrustService keeps the earned-autonomy ledger. Every decision on a
// proposal and every failed execution is an outcome for the agent and tool
// concerned; a long enough streak of clean approvals moves the tool up one
// tier on the agent when the organization allows it, and a setback takes an
// earned tier back.
type AgentTrustService interface {
	RecordDecision(
		ctx context.Context,
		proposal *agent.AgentProposal,
		decision *agent.AgentDecision,
	) error
	RecordExecutionFailure(ctx context.Context, proposal *agent.AgentProposal) error
	ListForDefinition(
		ctx context.Context,
		req repositories.ListToolTrustRequest,
	) ([]*agent.ToolTrust, error)
}

// DecisionOutcome is a recorded decision plus what happened when it ran.
// The decision is the durable fact and is recorded whatever the tool did;
// ExecutionError is set when an approval's write did not go through.
type DecisionOutcome struct {
	Decision       *agent.AgentDecision
	ExecutionError error
}

type AgentDecisionService interface {
	Decide(
		ctx context.Context,
		req *DecideAgentProposalRequest,
		actor *RequestActor,
	) (*agent.AgentDecision, error)
	// DecideWithOutcome is Decide for a caller that must know whether the
	// approved write went through, such as a plan deciding where to stop.
	DecideWithOutcome(
		ctx context.Context,
		req *DecideAgentProposalRequest,
		actor *RequestActor,
	) (*DecisionOutcome, error)
}

type UpdateAgentControlRequest struct {
	ShadowMode bool
	// EarnedAutonomy and PromotionThreshold are optional so a client that
	// only knows the pause switch leaves them as they are.
	EarnedAutonomy         *bool
	PromotionThreshold     *int
	BillingAgentEnabled    *bool
	DecisionTimeoutSeconds *int
	TenantInfo             pagination.TenantInfo
}

type AgentControlService interface {
	Get(ctx context.Context, tenantInfo pagination.TenantInfo) (*tenant.AgentControl, error)
	Update(
		ctx context.Context,
		req *UpdateAgentControlRequest,
		actor *RequestActor,
	) (*tenant.AgentControl, error)
}

// BudgetRefusal says which cap a run or a write ran into. A zero value is
// no refusal.
type BudgetRefusal struct {
	// Cap names the limit: monthly_budget, daily_runs or tool_daily_limit.
	Cap string `json:"cap"`
	// Tool is set for a tool cap.
	Tool  string `json:"tool,omitempty"`
	Spent string `json:"spent"`
	Limit string `json:"limit"`
	// ResetsAt is when the window rolls over and the cap clears.
	ResetsAt int64 `json:"resetsAt"`
}

func (r BudgetRefusal) Refused() bool { return r.Cap != "" }

// Message says the refusal in words a person or a model can act on.
func (r BudgetRefusal) Message(agentName string) string {
	switch r.Cap {
	case BudgetCapMonthly:
		return "Agent " + agentName + " has spent its monthly budget (" + r.Spent + " of " +
			r.Limit + " USD). It runs again when the month rolls over, or when the budget is raised in AI Control."
	case BudgetCapDailyRuns:
		return "Agent " + agentName + " has started its " + r.Limit +
			" runs for today. It runs again tomorrow, or when the daily limit is raised in AI Control."
	case BudgetCapTool:
		return "Tool " + r.Tool + " has reached its daily limit of " + r.Limit +
			" for agent " + agentName + ". It can run again tomorrow, or when the limit is raised in AI Control."
	default:
		return ""
	}
}

const (
	BudgetCapMonthly   = "monthly_budget"
	BudgetCapDailyRuns = "daily_runs"
	BudgetCapTool      = "tool_daily_limit"
)

// ToolBudgetUse is one tool's executions today against its cap.
type ToolBudgetUse struct {
	Tool  string `json:"tool"`
	Used  int    `json:"used"`
	Limit int    `json:"limit"`
}

// AgentBudgetStatus is where an agent stands against its caps, for the page
// that sets them.
type AgentBudgetStatus struct {
	MonthStart int64 `json:"monthStart"`
	DayStart   int64 `json:"dayStart"`
	// SpentUSD is the priced cost of the month's calls; UnpricedCalls says how
	// many carried no price, so the figure can be labelled partial.
	SpentUSD       string          `json:"spentUsd"`
	MonthlyBudget  *string         `json:"monthlyBudgetUsd"`
	MonthCalls     int             `json:"monthCalls"`
	UnpricedCalls  int             `json:"unpricedCalls"`
	RunsToday      int             `json:"runsToday"`
	DailyRunLimit  int             `json:"dailyRunLimit"`
	Tools          []ToolBudgetUse `json:"tools"`
	SimulationMode bool            `json:"simulationMode"`
}

// CheckToolBudgetRequest asks whether one more write by a tool fits its cap.
type CheckToolBudgetRequest struct {
	Definition *agentdefinition.Definition
	ToolName   string
	// Unrecorded is how many times the tool already ran in the caller's run
	// or turn without a recorded proposal yet. A turn records its writes when
	// it ends, so without them a single turn could run past the cap.
	Unrecorded int
}

// AgentBudgetService enforces an agent's caps and reports where it stands.
type AgentBudgetService interface {
	// CheckRun says whether the agent may start another run or turn now.
	CheckRun(ctx context.Context, definition *agentdefinition.Definition) (BudgetRefusal, error)
	// CheckTool says whether the agent may execute the tool once more today.
	CheckTool(ctx context.Context, req CheckToolBudgetRequest) (BudgetRefusal, error)
	Status(ctx context.Context, definition *agentdefinition.Definition) (*AgentBudgetStatus, error)
}

// AgentActivityPublisher tells connected clients that an agent record
// changed, so a queue or a pane refreshes without polling. Every method is
// best effort: a lost invalidation costs a refresh, never the write.
type AgentActivityPublisher interface {
	ProposalChanged(
		ctx context.Context,
		proposal *agent.AgentProposal,
		actor AuditActor,
		action string,
	)
	PlanChanged(ctx context.Context, plan *agent.AgentPlan, actor AuditActor, action string)
	RunChanged(ctx context.Context, run *agent.AgentRun, actor AuditActor, action string)
	ArtifactChanged(
		ctx context.Context,
		artifact *assistantartifact.Artifact,
		actor AuditActor,
		action string,
	)
}

// The actions an activity publisher announces.
const (
	ActivityCreated = "created"
	ActivityUpdated = "updated"
)

// PendingDecision is one thing waiting on a person: a proposal that stands
// on its own, or a plan decided as a whole. Exactly one of the two is set.
type PendingDecision struct {
	Proposal  *agent.AgentProposal
	Plan      *agent.AgentPlan
	CreatedAt int64
	// Cursor is where a page continues from after this entry.
	Cursor string
}

type ListPendingDecisionsRequest struct {
	TenantInfo pagination.TenantInfo
	First      int
	After      string
	// AgentDefinitionID narrows the queue to one agent's work; ToolName to
	// one kind of write (a plan matches when any step uses it).
	AgentDefinitionID pulid.ID
	ToolName          string
	IncludeTotalCount bool
}

type PendingDecisionsPage struct {
	Items       []PendingDecision
	HasNextPage bool
	TotalCount  *int
}

// DecideAgentProposalsRequest decides several proposals of the same tool
// at once, each as proposed. A change to one proposal is a decision on
// that proposal alone.
type DecideAgentProposalsRequest struct {
	ProposalIDs []pulid.ID
	Decision    agent.DecisionType
	ReasonCode  string
	TenantInfo  pagination.TenantInfo
}

// AgentProposalDecisionResult is what became of one proposal in a batch.
// Error is written for the person; Executed says the approved write went
// through.
type AgentProposalDecisionResult struct {
	ProposalID pulid.ID
	Decision   *agent.AgentDecision
	Error      string
	Executed   bool
}

type AgentDecisionQueueService interface {
	ListPending(ctx context.Context, req ListPendingDecisionsRequest) (*PendingDecisionsPage, error)
	// Count is the size of the queue, for a badge.
	Count(ctx context.Context, tenant pagination.TenantInfo) (int, error)
	Summary(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*repositories.PendingDecisionSummary, error)
	DecideMany(
		ctx context.Context,
		req *DecideAgentProposalsRequest,
		actor *RequestActor,
	) ([]AgentProposalDecisionResult, error)
}
