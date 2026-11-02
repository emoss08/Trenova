package assistantthreads

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	Runtime          *agentruntime.Service
	Conversations    conversationports.ConversationRepository
	Definitions      agentdefinitionports.AgentDefinitionRepository
	Proposals        agentproposalports.AgentProposalRepository
	Plans            agentplanports.AgentPlanRepository
	AIProviders      aiproviderports.AIProviderRepository
	Shadow           *agentshadow.Resolver
	Budgets          agentbudgetports.AgentBudgetService
	Permissions      permissionports.PermissionEngine
	Cases            deskcaseports.AssistantCaseStates
	Runs             agentruntimeports.AgentRunRepository
	SystemAgents     agentdefinitionports.SystemAgentProvisioner
	PageThreads      conversationports.PageThreadRepository
	AgentControls    tenantports.AgentControlRepository
	PaletteQuestions paletteports.QuestionRepository
	Users            tenantports.UserRepository
	Notifications    *notificationservice.Service
	Memories         agentmemoryports.AgentMemoryService
	Quota            platformplanports.QuotaGuard
	Schedules        conversationscheduleports.ConversationThreadSchedules
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		logger:           p.Logger.Named("service.assistant"),
		conversations:    p.Conversations,
		definitions:      p.Definitions,
		schedules:        p.Schedules,
		runtime:          p.Runtime,
		shadow:           p.Shadow,
		cases:            p.Cases,
		plans:            proposalstores.PlansOrNil(p.Plans),
		proposals:        p.Proposals,
		budgets:          p.Budgets,
		notifier:         raiseNotifierOf(p.Notifications),
		pageThreads:      p.PageThreads,
		systemAgents:     p.SystemAgents,
		permissions:      p.Permissions,
		providers:        p.AIProviders,
		runs:             p.Runs,
		users:            p.Users,
		agentControls:    p.AgentControls,
		paletteQuestions: p.PaletteQuestions,
		quota:            p.Quota,
		memories:         p.Memories,
	}
}

func (s *Service) SelectableProviders(
	ctx context.Context,
	actor *permissionports.RequestActor,
) ([]assistantports.AssistantProviderOption, error) {
	return s.providerChoice().SelectableProviders(ctx, actor)
}
