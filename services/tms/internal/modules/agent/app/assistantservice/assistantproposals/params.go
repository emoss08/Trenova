package assistantproposals

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Conversations conversationports.ConversationRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Proposals     agentproposalports.AgentProposalRepository
	Plans         agentplanports.AgentPlanRepository `optional:"true"`
	Shadow        *agentshadow.Resolver
	Tools         agenttoolports.AgentToolRegistry           `optional:"true"`
	Decisions     agentdecisionports.AgentDecisionRepository `optional:"true"`
	Labeler       agentruntimeports.RecordLabeler            `optional:"true"`
	Permissions   permissionports.PermissionEngine
	Runs          agentruntimeports.AgentRunRepository `optional:"true"`
	Users         tenantports.UserRepository           `optional:"true"`
}

func New(p Params) *Service {
	return &Service{
		logger:        p.Logger.Named("service.assistant"),
		conversations: p.Conversations,
		definitions:   p.Definitions,
		proposals:     p.Proposals,
		proposalEdits: p.Proposals,
		plans:         proposalstores.PlansOrNil(p.Plans),
		shadow:        p.Shadow,
		decisions:     p.Decisions,
		labeler:       p.Labeler,
		permissions:   p.Permissions,
		tools:         p.Tools,
		runs:          p.Runs,
		users:         p.Users,
	}
}
