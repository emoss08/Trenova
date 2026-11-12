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
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
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
	Plans         agentplanports.AgentPlanRepository
	AIProviders   aiproviderports.AIProviderRepository
	Shadow        *agentshadow.Resolver
	Budgets       agentbudgetports.AgentBudgetService
	Decisions     agentdecisionports.AgentDecisionRepository
	// Artifacts keeps what a turn produced besides words; Subjects describes
	// the record a conversation was opened from.
	Artifacts assistantartifactports.AssistantArtifactRepository
	Subjects  agentruntimeports.AgentSubjectDescriber
	Anchors   recordanchorports.RecordAnchorReader
	Activity  agentactivityports.AgentActivityPublisher
	// Documents checks that an attached file is the person's own; Contents
	// reads what document intelligence made of it.
	Documents documentports.DocumentRepository
	Contents  documentports.DocumentContentService
	// Permissions decides whether the person may read the record a
	// conversation is about before any of it reaches the model.
	Permissions permissionports.PermissionEngine
	// Runs says which agent raised each of a conversation's proposals: its
	// own, or one it handed a task to.
	Runs agentruntimeports.AgentRunRepository
	// AgentControls holds the per-person allowance.
	AgentControls tenantports.AgentControlRepository
	// PaletteQuestions are counted against the same allowance.
	PaletteQuestions paletteports.QuestionRepository
	Users            tenantports.UserRepository
	Boards           assistantports.BoardFiller
	// Labels names the records a table was narrowed by.
	Labels agentruntimeports.RecordLabeler
	// Memories describes the memories a reply used or saved.
	Memories agentmemoryports.AgentMemoryService
	Quota    platformplanports.QuotaGuard
	// Waits reads a wait whose ending a turn picks up.
	Waits agentwaitports.AgentWaitRepository
	// ReadWaits parks a conversation on a file still being read.
	ReadWaits agentwaitports.DocumentReadWaiter
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
	logger           *zap.Logger
	guard            *agentguard.Service
	runtime          *agentruntime.Service
	contexts         agentruntimeports.RuntimeContextBuilder
	conversations    conversationports.ConversationRepository
	definitions      agentdefinitionports.AgentDefinitionRepository
	recorder         *proposalrecorder.Service
	proposals        proposalstores.ProposalStore
	plans            proposalstores.PlanStore
	providers        aiproviderports.AIProviderRepository
	shadow           *agentshadow.Resolver
	budgets          agentbudgetports.AgentBudgetService
	decisions        agentdecisionports.AgentDecisionRepository
	artifacts        assistantartifactports.AssistantArtifactRepository
	subjects         agentruntimeports.AgentSubjectDescriber
	anchors          recordanchorports.RecordAnchorReader
	activity         agentactivityports.AgentActivityPublisher
	documents        documentports.DocumentRepository
	contents         documentports.DocumentContentService
	permissions      permissionports.PermissionEngine
	runs             agentruntimeports.AgentRunRepository
	agentControls    tenantports.AgentControlRepository
	paletteQuestions paletteports.QuestionRepository
	users            tenantports.UserRepository
	boards           assistantports.BoardFiller
	labels           agentruntimeports.RecordLabeler
	memories         agentmemoryports.AgentMemoryService
	quota            platformplanports.QuotaGuard
	waits            agentwaitports.AgentWaitRepository
	readWaits        agentwaitports.DocumentReadWaiter
}

func New(p Params) *Service {
	return &Service{
		logger:           p.Logger.Named("service.assistant"),
		guard:            p.Guard,
		runtime:          p.Runtime,
		contexts:         p.Contexts,
		conversations:    p.Conversations,
		definitions:      p.Definitions,
		providers:        p.AIProviders,
		recorder:         p.Recorder,
		proposals:        p.Proposals,
		plans:            proposalstores.PlansOrNil(p.Plans),
		shadow:           p.Shadow,
		budgets:          p.Budgets,
		decisions:        p.Decisions,
		artifacts:        p.Artifacts,
		subjects:         p.Subjects,
		anchors:          p.Anchors,
		activity:         p.Activity,
		documents:        p.Documents,
		contents:         p.Contents,
		permissions:      p.Permissions,
		runs:             p.Runs,
		waits:            p.Waits,
		readWaits:        p.ReadWaits,
		agentControls:    p.AgentControls,
		paletteQuestions: p.PaletteQuestions,
		users:            p.Users,
		boards:           p.Boards,
		labels:           p.Labels,
		memories:         p.Memories,
		quota:            p.Quota,
	}
}
