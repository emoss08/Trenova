package assistantartifacts

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Conversations conversationports.ConversationRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Proposals     agentproposalports.AgentProposalRepository
	Plans         agentplanports.AgentPlanRepository
	Artifacts     assistantartifactports.AssistantArtifactRepository
	Activity      agentactivityports.AgentActivityPublisher
	Permissions   permissionports.PermissionEngine
	Runs          agentruntimeports.AgentRunRepository
	Users         tenantports.UserRepository
	Completion    llmports.StructuredCompleter
	Reports       *reporting.Service
	Queries       agenttoolports.AgentQueryToolRegistry
	PDFs          documenttemplateports.PDFRenderer
	Boards        assistantports.BoardFiller
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		logger:        p.Logger.Named("service.assistant"),
		conversations: p.Conversations,
		definitions:   p.Definitions,
		artifacts:     p.Artifacts,
		permissions:   p.Permissions,
		activity:      p.Activity,
		proposals:     p.Proposals,
		plans:         proposalstores.PlansOrNil(p.Plans),
		queries:       p.Queries,
		reports:       p.Reports,
		reportRuns:    reportRerunnerOf(p.Reports),
		boards:        p.Boards,
		pdfs:          p.PDFs,
		completion:    p.Completion,
		runs:          p.Runs,
		users:         p.Users,
	}
}
