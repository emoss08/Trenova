package assistantcompaction

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	Conversations    conversationports.ConversationRepository
	Definitions      agentdefinitionports.AgentDefinitionRepository
	Proposals        agentproposalports.AgentProposalRepository
	Permissions      permissionports.PermissionEngine
	Budgets          agentbudgetports.AgentBudgetService
	Decisions        agentdecisionports.AgentDecisionRepository
	AgentControls    tenantports.AgentControlRepository
	PaletteQuestions paletteports.QuestionRepository
	Quota            platformplanports.QuotaGuard
	Chapters         assistantchapterports.ChapterRepository
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		logger:           p.Logger.Named("service.assistant"),
		conversations:    p.Conversations,
		definitions:      p.Definitions,
		permissions:      p.Permissions,
		budgets:          p.Budgets,
		agentControls:    p.AgentControls,
		paletteQuestions: p.PaletteQuestions,
		quota:            p.Quota,
		chapters:         p.Chapters,
		proposals:        p.Proposals,
		decisions:        p.Decisions,
	}
}
