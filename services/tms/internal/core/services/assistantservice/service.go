package assistantservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Guard         *agentguard.Service
	Runtime       *agentruntime.Service
	Contexts      serviceports.RuntimeContextBuilder
	Conversations repositories.ConversationRepository
	Definitions   repositories.AgentDefinitionRepository
	Recorder      *proposalrecorder.Service
	Proposals     repositories.AgentProposalRepository
	Plans         repositories.AgentPlanRepository `optional:"true"`
	AIProviders   repositories.AIProviderRepository
	Shadow        *agentshadow.Resolver
	Budgets       serviceports.AgentBudgetService `optional:"true"`
	// Tools names the parameters a proposal's tool takes, for the fields a
	// person may edit before approving; Decisions carries what they changed.
	Tools     serviceports.AgentToolRegistry       `optional:"true"`
	Decisions repositories.AgentDecisionRepository `optional:"true"`
	// Artifacts keeps what a turn produced besides words; Subjects describes
	// the record a conversation was opened from.
	Artifacts repositories.AssistantArtifactRepository `optional:"true"`
	Subjects  serviceports.AgentSubjectDescriber       `optional:"true"`
	Activity  serviceports.AgentActivityPublisher      `optional:"true"`
	// Documents checks that an attached file is the person's own; Contents
	// reads what document intelligence made of it.
	Documents repositories.DocumentRepository     `optional:"true"`
	Contents  serviceports.DocumentContentService `optional:"true"`
	// Permissions decides whether the person may read the record a
	// conversation is about before any of it reaches the model.
	Permissions serviceports.PermissionEngine
	// Runs says which agent raised each of a conversation's proposals: its
	// own, or one it handed a task to.
	Runs repositories.AgentRunRepository `optional:"true"`
}

// Module provides the assistant once, as itself for the worker that runs its
// turns and as the AssistantService port for everything else.
var Module = fx.Module("assistant",
	fx.Provide(fx.Annotate(New, fx.As(fx.Self()), fx.As(new(serviceports.AssistantService)))),
)

type Service struct {
	logger        *zap.Logger
	guard         *agentguard.Service
	runtime       *agentruntime.Service
	contexts      serviceports.RuntimeContextBuilder
	conversations repositories.ConversationRepository
	definitions   repositories.AgentDefinitionRepository
	recorder      *proposalrecorder.Service
	proposals     chatProposalStore
	plans         chatPlanStore
	providers     repositories.AIProviderRepository
	shadow        *agentshadow.Resolver
	budgets       serviceports.AgentBudgetService
	tools         serviceports.AgentToolRegistry
	decisions     repositories.AgentDecisionRepository
	artifacts     repositories.AssistantArtifactRepository
	subjects      serviceports.AgentSubjectDescriber
	activity      serviceports.AgentActivityPublisher
	documents     repositories.DocumentRepository
	contents      serviceports.DocumentContentService
	permissions   serviceports.PermissionEngine
	runs          repositories.AgentRunRepository
}

func New(p Params) *Service {
	return &Service{
		logger:        p.Logger.Named("service.assistant"),
		guard:         p.Guard,
		runtime:       p.Runtime,
		contexts:      p.Contexts,
		conversations: p.Conversations,
		definitions:   p.Definitions,
		providers:     p.AIProviders,
		recorder:      p.Recorder,
		proposals:     p.Proposals,
		plans:         planStoreOrNil(p.Plans),
		shadow:        p.Shadow,
		budgets:       p.Budgets,
		tools:         p.Tools,
		decisions:     p.Decisions,
		artifacts:     p.Artifacts,
		subjects:      p.Subjects,
		activity:      p.Activity,
		documents:     p.Documents,
		contents:      p.Contents,
		permissions:   p.Permissions,
		runs:          p.Runs,
	}
}
