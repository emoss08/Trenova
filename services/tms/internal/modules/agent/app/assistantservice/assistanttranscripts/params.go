package assistanttranscripts

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
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
	Runs          agentruntimeports.AgentRunRepository    `optional:"true"`
	Users         tenantports.UserRepository              `optional:"true"`
	Chapters      assistantchapterports.ChapterRepository `optional:"true"`
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		logger:        p.Logger.Named("service.assistant"),
		conversations: p.Conversations,
		definitions:   p.Definitions,
		proposals:     p.Proposals,
		runs:          p.Runs,
		users:         p.Users,
		chapters:      p.Chapters,
	}
}
