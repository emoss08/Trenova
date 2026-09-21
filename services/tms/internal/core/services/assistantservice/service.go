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
	}
}
