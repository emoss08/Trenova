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
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Runtime       *agentruntime.Service
	Conversations conversationports.ConversationRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Proposals     agentproposalports.AgentProposalRepository
	Plans         agentplanports.AgentPlanRepository `optional:"true"`
	AIProviders   aiproviderports.AIProviderRepository
	Shadow        *agentshadow.Resolver
	Budgets       agentbudgetports.AgentBudgetService `optional:"true"`
	Permissions   permissionports.PermissionEngine
	Cases         deskcaseports.AssistantCaseStates                     `optional:"true"`
	Runs          agentruntimeports.AgentRunRepository                  `optional:"true"`
	SystemAgents  agentdefinitionports.SystemAgentProvisioner           `optional:"true"`
	PageThreads   conversationports.PageThreadRepository                `optional:"true"`
	AgentControls tenantports.AgentControlRepository                    `optional:"true"`
	Users         tenantports.UserRepository                            `optional:"true"`
	Notifications *notificationservice.Service                          `optional:"true"`
	Queries       agenttoolports.AgentQueryToolRegistry                 `optional:"true"`
	Memories      agentmemoryports.AgentMemoryService                   `optional:"true"`
	Quota         platformplanports.QuotaGuard                          `optional:"true"`
	Schedules     conversationscheduleports.ConversationThreadSchedules `optional:"true"`
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		logger:        p.Logger.Named("service.assistant"),
		conversations: p.Conversations,
		definitions:   p.Definitions,
		schedules:     p.Schedules,
		runtime:       p.Runtime,
		shadow:        p.Shadow,
		cases:         p.Cases,
		plans:         proposalstores.PlansOrNil(p.Plans),
		proposals:     p.Proposals,
		budgets:       p.Budgets,
		notifier:      raiseNotifierOf(p.Notifications),
		pageThreads:   p.PageThreads,
		systemAgents:  p.SystemAgents,
		permissions:   p.Permissions,
		providers:     p.AIProviders,
		queries:       p.Queries,
		runs:          p.Runs,
		users:         p.Users,
		agentControls: p.AgentControls,
		quota:         p.Quota,
		memories:      p.Memories,
	}
}

func (s *Service) SelectableProviders(
	ctx context.Context,
	actor permissionports.RequestActor,
) ([]assistantports.AssistantProviderOption, error) {
	return s.providerChoice().SelectableProviders(ctx, actor)
}
