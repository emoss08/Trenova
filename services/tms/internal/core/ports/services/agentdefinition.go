package services

import (
	"context"
	"github.com/shopspring/decimal"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type SaveAgentDefinitionRequest struct {
	ID                     pulid.ID
	Name                   string
	Description            string
	Template               agentdefinition.Template
	Instructions           string
	Guardrails             []string
	ToolNames              []string
	ToolTiers              map[string]agent.AutonomyTier
	AutonomyCeiling        agent.AutonomyTier
	Enabled                bool
	ShadowMode             bool
	DecisionTimeoutSeconds int
	TriggerMode            agentdefinition.TriggerMode
	CronExpression         string
	CronTimezone           string
	EventKinds             []agent.EventKind
	IntervalSeconds        int
	EndsAt                 *int64
	MaxConcurrentRuns      int
	RunTimeoutSeconds      int
	MaxToolCalls           int
	MonthlyBudgetUSD       *decimal.Decimal
	DailyRunLimit          int
	ToolDailyLimits        map[string]int
	SimulationMode         bool
	Icon                   string
	Accent                 string
	ContextProviders       []agentdefinition.ContextProvider
	OutputMode             agentdefinition.OutputMode
	PreferredProviderID    pulid.ID
	// DelegateIDs replaces the agents this one may hand a task to. Nil keeps
	// the ones it has, so a save from a form that does not show them cannot
	// clear them; an empty list clears them.
	DelegateIDs *[]pulid.ID
	Version     int64
	TenantInfo  pagination.TenantInfo
}

type AgentTemplateDescriptor struct {
	Template            agentdefinition.Template          `json:"template"`
	Label               string                            `json:"label"`
	Description         string                            `json:"description"`
	StarterInstructions string                            `json:"starterInstructions"`
	StarterTools        []string                          `json:"starterTools"`
	StarterTrigger      agentdefinition.TriggerMode       `json:"starterTrigger"`
	StarterEvents       []agent.EventKind                 `json:"starterEvents"`
	StarterCron         string                            `json:"starterCron"`
	StarterCeiling      agent.AutonomyTier                `json:"starterCeiling"`
	StarterOutput       agentdefinition.OutputMode        `json:"starterOutput"`
	SystemKey           string                            `json:"systemKey"`
	ContextProviders    []agentdefinition.ContextProvider `json:"contextProviders"`
}

type ToolCatalogKind string

const (
	ToolCatalogKindQuery  = ToolCatalogKind("query")
	ToolCatalogKindAction = ToolCatalogKind("action")
)

type ToolCatalogEntry struct {
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	Parameters          map[string]any       `json:"parameters"`
	Kind                ToolCatalogKind      `json:"kind"`
	Resource            permission.Resource  `json:"resource"`
	Operation           permission.Operation `json:"operation"`
	DefaultAutonomyTier agent.AutonomyTier   `json:"defaultAutonomyTier"`
	Reversible          bool                 `json:"reversible"`
	Core                bool                 `json:"core"`
	Effect              agent.ToolEffect     `json:"effect"`
	// Prerequisites are the tools this one takes its arguments from. The
	// reads among them are held by any agent holding this tool.
	Prerequisites []string `json:"prerequisites"`
}

type PreviewPromptRequest struct {
	Definition *SaveAgentDefinitionRequest
	Actor      *RequestActor
}

// AgentDefinitionScheduler keeps the schedule behind a scheduled or continuous
// agent in line with the agent. Neither call fails the save that made it: a
// schedule a failed call left behind is repaired by the next reconcile.
type AgentDefinitionScheduler interface {
	Sync(ctx context.Context, definition *agentdefinition.Definition)
	Remove(ctx context.Context, definitionID pulid.ID)
}

type AgentDefinitionService interface {
	List(
		ctx context.Context,
		req *repositories.ListAgentDefinitionRequest,
	) (*pagination.ListResult[*agentdefinition.Definition], error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentDefinitionConnectionRequest,
	) (*pagination.CursorListResult[*agentdefinition.Definition], error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
	GetBySystemKey(
		ctx context.Context,
		req repositories.GetAgentDefinitionBySystemKeyRequest,
	) (*agentdefinition.Definition, error)
	Create(
		ctx context.Context,
		req *SaveAgentDefinitionRequest,
		actor *RequestActor,
	) (*agentdefinition.Definition, error)
	Update(
		ctx context.Context,
		req *SaveAgentDefinitionRequest,
		actor *RequestActor,
	) (*agentdefinition.Definition, error)
	Delete(
		ctx context.Context,
		req repositories.DeleteAgentDefinitionRequest,
		actor *RequestActor,
	) error
	Templates() []AgentTemplateDescriptor
	ToolCatalog() []ToolCatalogEntry
	EventKinds() []agent.EventDescriptor
	PreviewPrompt(ctx context.Context, req *PreviewPromptRequest) (string, error)
}
