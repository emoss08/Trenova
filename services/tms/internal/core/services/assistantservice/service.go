package assistantservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Guard         *agentguard.Service
	Runtime       serviceports.AgentRuntime
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
}

type Service struct {
	logger        *zap.Logger
	guard         *agentguard.Service
	runtime       serviceports.AgentRuntime
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
}

func New(p Params) serviceports.AssistantService {
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
	}
}
