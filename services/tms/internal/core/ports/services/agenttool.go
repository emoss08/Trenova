package services

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"

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
	Taint      *agent.RunTaint
	ProposalID pulid.ID
}

func (p ToolExecuteParams) ApprovedFromProposal() bool {
	return p.ProposalID.IsNotNil()
}

func (p ToolExecuteParams) CarriedTaint(at int64) *agent.RunTaint {
	if p.Taint != nil || p.ApprovedFromProposal() {
		return p.Taint
	}

	return agent.RunRecordTaint(p.RunID, at)
}

// ToolPreviewer is a tool that can say, record by record, what its write
// would do. Preview is read-only and deterministic given the database and
// the parameters, and it decides what changes with the same code Execute
// does. The preview service runs it inside a read-only snapshot, so a write
// it attempted would fail, and filters what it returns for each reader.
type ToolPreviewer interface {
	Preview(ctx context.Context, params ToolExecuteParams) (*agent.ToolPreview, error)
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

//nolint:gosec // G101: record kinds; "credit" matches the credential pattern, not a secret
const (
	RecordInvoiceAdjustment     permission.Resource = "invoice_adjustment"
	RecordCreditMemoApplication permission.Resource = "credit_memo_application"
)

// TargetedTool is a tool that can say which record a call would change, from
// the arguments alone. It is pure: no lookup, no service — the tool already
// knows which argument names its subject.
type TargetedTool interface {
	Target(params map[string]any) (ToolTarget, bool)
}

// TargetParameters names the parameters that hold the id of the record a
// call acts on: those whose value is the target's id. They are what a
// person may not change when approving, since changing them would point the
// write at a record nobody proposed a change to.
func TargetParameters(tool any, params map[string]any) []string {
	targeted, ok := tool.(TargetedTool)
	if !ok || len(params) == 0 {
		return nil
	}

	target, ok := targeted.Target(params)
	if !ok || target.ID.IsNil() {
		return nil
	}

	names := make([]string, 0, 1)
	for name, value := range params {
		if text, isText := value.(string); isText && text == target.ID.String() {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	return names
}

// ProposalFields is a pending proposal's parameters as a person may edit
// them, from the tool's schema, with the parameters that name its target
// read-only and each record-subset parameter listing the records proposed,
// named by their ids until LabelSubsetChoices names them.
func ProposalFields(tool AgentTool, params map[string]any) []toolschema.Field {
	schema := tool.ParamSchema()
	fields := toolschema.Fields(schema)
	targets := TargetParameters(tool, params)
	choices := toolschema.SubsetChoices(schema, params)
	for i := range fields {
		fields[i].ReadOnly = slices.Contains(targets, fields[i].Name)
		if fields[i].Kind == toolschema.KindRecordSubset {
			fields[i].Choices = choices[fields[i].Name]
		}
	}

	return fields
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
