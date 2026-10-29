package assistantservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordanchorports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Guard         *agentguard.Service
	Runtime       *agentruntime.Service
	Contexts      agentruntimeports.RuntimeContextBuilder
	Conversations conversationports.ConversationRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Recorder      *proposalrecorder.Service
	Proposals     agentproposalports.AgentProposalRepository
	Plans         agentplanports.AgentPlanRepository `optional:"true"`
	AIProviders   aiproviderports.AIProviderRepository
	Shadow        *agentshadow.Resolver
	Budgets       agentbudgetports.AgentBudgetService        `optional:"true"`
	Decisions     agentdecisionports.AgentDecisionRepository `optional:"true"`
	// Artifacts keeps what a turn produced besides words; Subjects describes
	// the record a conversation was opened from.
	Artifacts assistantartifactports.AssistantArtifactRepository `optional:"true"`
	Subjects  agentruntimeports.AgentSubjectDescriber            `optional:"true"`
	Anchors   recordanchorports.RecordAnchorReader               `optional:"true"`
	Activity  agentactivityports.AgentActivityPublisher          `optional:"true"`
	// Documents checks that an attached file is the person's own; Contents
	// reads what document intelligence made of it.
	Documents documentports.DocumentRepository     `optional:"true"`
	Contents  documentports.DocumentContentService `optional:"true"`
	// Permissions decides whether the person may read the record a
	// conversation is about before any of it reaches the model.
	Permissions permissionports.PermissionEngine
	// Runs says which agent raised each of a conversation's proposals: its
	// own, or one it handed a task to.
	Runs agentruntimeports.AgentRunRepository `optional:"true"`
	// AgentControls holds the per-person allowance; without it nobody is
	// limited.
	AgentControls tenantports.AgentControlRepository    `optional:"true"`
	Users         tenantports.UserRepository            `optional:"true"`
	Queries       agenttoolports.AgentQueryToolRegistry `optional:"true"`
	Boards        assistantports.BoardFiller            `optional:"true"`
	// Memories describes the memories a reply used or saved.
	Memories agentmemoryports.AgentMemoryService `optional:"true"`
	Quota    platformplanports.QuotaGuard        `optional:"true"`
	// Waits reads a wait whose ending a turn picks up.
	Waits agentwaitports.AgentWaitRepository `optional:"true"`
	// ReadWaits parks a conversation on a file still being read.
	ReadWaits agentwaitports.DocumentReadWaiter `optional:"true"`
}

// Module provides the turn service for the worker that runs turns; the
// services everything else reads the assistant through have modules of their own.
var Module = fx.Module("assistant",
	fx.Provide(New),
	fx.Provide(fx.Annotate(
		func(s *Service) agenteventports.AttachmentExtractionObserver { return s },
		fx.ResultTags(`group:"attachment_extraction_observers"`),
	)),
)

type Service struct {
	logger        *zap.Logger
	guard         *agentguard.Service
	runtime       *agentruntime.Service
	contexts      agentruntimeports.RuntimeContextBuilder
	conversations conversationports.ConversationRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	recorder      *proposalrecorder.Service
	proposals     proposalstores.ProposalStore
	plans         proposalstores.PlanStore
	providers     aiproviderports.AIProviderRepository
	shadow        *agentshadow.Resolver
	budgets       agentbudgetports.AgentBudgetService
	decisions     agentdecisionports.AgentDecisionRepository
	artifacts     assistantartifactports.AssistantArtifactRepository
	subjects      agentruntimeports.AgentSubjectDescriber
	anchors       recordanchorports.RecordAnchorReader
	activity      agentactivityports.AgentActivityPublisher
	documents     documentports.DocumentRepository
	contents      documentports.DocumentContentService
	permissions   permissionports.PermissionEngine
	runs          agentruntimeports.AgentRunRepository
	agentControls tenantports.AgentControlRepository
	users         tenantports.UserRepository
	queries       agenttoolports.AgentQueryToolRegistry
	boards        assistantports.BoardFiller
	memories      agentmemoryports.AgentMemoryService
	quota         platformplanports.QuotaGuard
	waits         agentwaitports.AgentWaitRepository
	readWaits     agentwaitports.DocumentReadWaiter
}

func New(p Params) *Service {
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
		plans:         proposalstores.PlansOrNil(p.Plans),
		shadow:        p.Shadow,
		budgets:       p.Budgets,
		decisions:     p.Decisions,
		artifacts:     p.Artifacts,
		subjects:      p.Subjects,
		anchors:       p.Anchors,
		activity:      p.Activity,
		documents:     p.Documents,
		contents:      p.Contents,
		permissions:   p.Permissions,
		runs:          p.Runs,
		waits:         p.Waits,
		readWaits:     p.ReadWaits,
		agentControls: p.AgentControls,
		users:         p.Users,
		queries:       p.Queries,
		boards:        p.Boards,
		memories:      p.Memories,
		quota:         p.Quota,
	}
}
