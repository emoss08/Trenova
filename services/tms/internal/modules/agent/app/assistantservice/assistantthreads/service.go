package assistantthreads

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/agentaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistantproviders"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/attribution"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/memorynotes"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/turnlimits"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/zap"
)

type Service struct {
	logger        *zap.Logger
	conversations conversationports.ConversationRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	schedules     conversationscheduleports.ConversationThreadSchedules
	runtime       *agentruntime.Service
	shadow        *agentshadow.Resolver
	cases         deskcaseports.AssistantCaseStates
	plans         proposalstores.PlanStore
	proposals     proposalstores.ProposalStore
	budgets       agentports.AgentBudgetService
	notifier      raiseNotifier
	pageThreads   conversationports.PageThreadRepository
	systemAgents  agentdefinitionports.SystemAgentProvisioner
	permissions   permissionports.PermissionEngine
	providers     aiproviderports.AIProviderRepository
	queries       agenttoolports.AgentQueryToolRegistry
	runs          agentruntimeports.AgentRunRepository
	users         tenantports.UserRepository
	agentControls tenantports.AgentControlRepository
	quota         platformplanports.QuotaGuard
	memories      agentmemoryports.AgentMemoryService
}

func (s *Service) attribution() *attribution.Attributor {
	return &attribution.Attributor{
		Logger:      s.logger,
		Definitions: s.definitions,
		Runs:        s.runs,
		Users:       s.users,
	}
}

func (s *Service) access() *agentaccess.Access {
	return &agentaccess.Access{
		Permissions: s.permissions,
		Definitions: s.definitions,
		Queries:     s.queries,
	}
}

func (s *Service) limits() *turnlimits.Limits {
	return &turnlimits.Limits{
		Budgets:       s.budgets,
		Conversations: s.conversations,
		AgentControls: s.agentControls,
		Quota:         s.quota,
	}
}

func (s *Service) memoryNotes() *memorynotes.Describer {
	return &memorynotes.Describer{Logger: s.logger, Memories: s.memories}
}

func (s *Service) providerChoice() *assistantproviders.Options {
	return &assistantproviders.Options{Providers: s.providers}
}
