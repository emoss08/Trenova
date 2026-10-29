package assistantcompaction

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Conversations conversationports.ConversationRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Proposals     agentports.AgentProposalRepository
	Permissions   permissionports.PermissionEngine
	Budgets       agentports.AgentBudgetService           `optional:"true"`
	Decisions     agentports.AgentDecisionRepository      `optional:"true"`
	AgentControls tenantports.AgentControlRepository      `optional:"true"`
	Queries       agenttoolports.AgentQueryToolRegistry   `optional:"true"`
	Quota         platformplanports.QuotaGuard            `optional:"true"`
	Chapters      assistantchapterports.ChapterRepository `optional:"true"`
}

func New(p Params) *Service {
	return &Service{
		logger:        p.Logger.Named("service.assistant"),
		conversations: p.Conversations,
		definitions:   p.Definitions,
		permissions:   p.Permissions,
		queries:       p.Queries,
		budgets:       p.Budgets,
		agentControls: p.AgentControls,
		quota:         p.Quota,
		chapters:      p.Chapters,
		proposals:     p.Proposals,
		decisions:     p.Decisions,
	}
}
