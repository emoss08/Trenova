package services

import (
	"context"
	"errors"
	"github.com/emoss08/trenova/pkg/pagination"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

type ToolExecuteParams struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Actor          *RequestActor
	IdempotencyKey string
	// RunID is the agent run the call belongs to, when there is one. Tools
	// that record something about the run itself, such as an exception, read
	// it from here rather than trusting a model-supplied id.
	RunID  pulid.ID
	Params map[string]any
	// Taint is the run's, handed only to a tool whose policy CarriesTaint.
	Taint *agent.RunTaint
}

// ToolSimulator is a tool that can say what it would change without
// changing it. The runtime and the executor use it when the agent is in
// simulation; a tool without it is described by its name and parameters.
type ToolSimulator interface {
	Simulate(ctx context.Context, params ToolExecuteParams) (*agent.ToolSimulation, error)
}

// ToolValidator is a tool that can check its arguments before anything is
// recorded. The runtime asks it before raising a proposal, so a call that
// would fail on execution is refused to the model now, while it can still
// fix the call, rather than after a person has approved it.
type ToolValidator interface {
	Validate(ctx context.Context, params ToolExecuteParams) error
}

// ToolResultReporter is a write whose caller needs to know what it made. A
// saved report's id is what the next call takes; told only that the write
// ran, a model reaches for the one id it holds, the proposal's, and passes
// that instead. ExecuteWithResult does what Execute does and names the record
// it made or changed.
type ToolResultReporter interface {
	ExecuteWithResult(
		ctx context.Context,
		params ToolExecuteParams,
	) (*agent.ToolExecutionResult, error)
}

// ExecuteTool runs a write and returns what it reports making, bounded to
// what a proposal keeps. A tool that reports nothing runs through Execute and
// returns a nil result.
func ExecuteTool(
	ctx context.Context,
	tool AgentTool,
	params ToolExecuteParams,
) (*agent.ToolExecutionResult, error) {
	reporter, reports := tool.(ToolResultReporter)
	if !reports {
		return nil, tool.Execute(ctx, params)
	}

	result, err := reporter.ExecuteWithResult(ctx, params)
	if err != nil {
		return nil, err
	}

	return result.Bounded(), nil
}

type AgentTool interface {
	Name() string
	Description() string
	ParamSchema() map[string]any
	Policy() ToolPolicy
	Execute(ctx context.Context, params ToolExecuteParams) error
}

type AgentToolDescriptor struct {
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Parameters   map[string]any     `json:"parameters"`
	AutonomyTier agent.AutonomyTier `json:"autonomyTier"`
	// Query is true for a read tool. Only a read is ever granted to an agent
	// because a tool it holds depends on it.
	Query         bool     `json:"query"`
	SearchTerms   []string `json:"searchTerms,omitempty"`
	Prerequisites []string `json:"prerequisites,omitempty"`
}

// SearchableTool is a tool with words a person uses for it that its name and
// description do not carry. "My dashboard" is how people name their home page;
// neither word is in get_my_home_layout's name.
type SearchableTool interface {
	SearchTerms() []string
}

// PrerequisiteTool is a tool whose arguments come from other tools. A
// dashboard tile needs a report id, and only list_reports hands one out; an
// agent given create_dashboard without it invents the id. Prerequisites are
// loaded alongside the tool, and a prerequisite read is held by any agent
// holding the tool.
type PrerequisiteTool interface {
	Prerequisites() []string
}

// SelfScopeOwnerParam is where the runtime records whose records a
// self-scoped call is about. The runtime writes it from the turn's actor and
// overwrites anything the model sent; the tool refuses to run for anyone else,
// so a proposal approved from someone else's queue cannot land on their own
// home screen instead.
const SelfScopeOwnerParam = "_owner"

// DescribeTool builds a tool's descriptor, reading the optional interfaces
// both registries share.
func DescribeTool(
	tool interface {
		Name() string
		Description() string
		ParamSchema() map[string]any
	},
	tier agent.AutonomyTier,
	query bool,
) AgentToolDescriptor {
	descriptor := AgentToolDescriptor{
		Name:         tool.Name(),
		Description:  tool.Description(),
		Parameters:   tool.ParamSchema(),
		AutonomyTier: tier,
		Query:        query,
	}
	if searchable, ok := tool.(SearchableTool); ok {
		descriptor.SearchTerms = searchable.SearchTerms()
	}
	if dependent, ok := tool.(PrerequisiteTool); ok {
		descriptor.Prerequisites = dependent.Prerequisites()
	}

	return descriptor
}

// ToolTarget is the one record a tool call acts on, when there is one.
//
// A proposal is a promise to change something later, and "later" is the
// problem: the shipment a hold was proposed on Tuesday may have been delivered
// by Thursday. Naming the target at proposal time is what lets its version be
// remembered, and compared, before the change is made.
type ToolTarget struct {
	Resource permission.Resource
	ID       pulid.ID
}

// TargetedTool is a tool that can say which record a call would change, from
// the arguments alone. It is pure: no lookup, no service — the tool already
// knows which argument names its subject.
type TargetedTool interface {
	Target(params map[string]any) (ToolTarget, bool)
}

// ErrRecordVersionUnsupported reports a resource the reader has no table for.
var ErrRecordVersionUnsupported = errors.New("record version is not tracked for this resource")

// RecordVersionReader reads the current version of a record. One reader
// serves every tool, which is why tools name their target rather than
// fetching it: each holds only the narrow service it acts through, and
// none of those hands back a version.
type RecordVersionReader interface {
	Version(ctx context.Context, tenant pagination.TenantInfo, target ToolTarget) (int64, error)
}

type AgentToolRegistry interface {
	Get(name string) (AgentTool, bool)
	All() []AgentTool
	Descriptors() []AgentToolDescriptor
}
