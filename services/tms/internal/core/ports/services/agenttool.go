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

type AgentTool interface {
	Name() string
	Description() string
	ParamSchema() map[string]any
	Reversible() bool
	PermissionResource() permission.Resource
	PermissionOperation() permission.Operation
	RequiresIdempotencyKey() bool
	DefaultAutonomyTier() agent.AutonomyTier
	Execute(ctx context.Context, params ToolExecuteParams) error
}

type AgentToolDescriptor struct {
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Parameters   map[string]any     `json:"parameters"`
	AutonomyTier agent.AutonomyTier `json:"autonomyTier"`
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
