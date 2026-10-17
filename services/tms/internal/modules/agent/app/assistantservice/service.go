package assistantservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Guard         *agentguard.Service
	Runtime       *agentruntime.Service
	Contexts      agentports.RuntimeContextBuilder
	Conversations agentports.ConversationRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Recorder      *proposalrecorder.Service
	Proposals     agentports.AgentProposalRepository
	Plans         agentports.AgentPlanRepository `optional:"true"`
	AIProviders   agentports.AIProviderRepository
	Shadow        *agentshadow.Resolver
	Budgets       agentports.AgentBudgetService `optional:"true"`
	// Tools names the parameters a proposal's tool takes, for the fields a
	// person may edit before approving; Decisions carries what they changed.
	Tools     agentports.AgentToolRegistry       `optional:"true"`
	Decisions agentports.AgentDecisionRepository `optional:"true"`
	// Labeler names the records a proposal offers for a person to untick.
	Labeler agentports.RecordLabeler `optional:"true"`
	// Artifacts keeps what a turn produced besides words; Subjects describes
	// the record a conversation was opened from.
	Artifacts assistantartifactports.AssistantArtifactRepository `optional:"true"`
	Subjects  agentports.AgentSubjectDescriber                   `optional:"true"`
	Anchors   agentports.RecordAnchorReader                      `optional:"true"`
	Activity  agentports.AgentActivityPublisher                  `optional:"true"`
	// Documents checks that an attached file is the person's own; Contents
	// reads what document intelligence made of it.
	Documents documentports.DocumentRepository     `optional:"true"`
	Contents  documentports.DocumentContentService `optional:"true"`
	// Permissions decides whether the person may read the record a
	// conversation is about before any of it reaches the model.
	Permissions permissionports.PermissionEngine
	// Cases says where each conversation that is a case stands.
	Cases deskcaseports.AssistantCaseStates `optional:"true"`
	// Runs says which agent raised each of a conversation's proposals: its
	// own, or one it handed a task to.
	Runs         agentports.AgentRunRepository               `optional:"true"`
	SystemAgents agentdefinitionports.SystemAgentProvisioner `optional:"true"`
	PageThreads  agentports.PageThreadRepository             `optional:"true"`
	// AgentControls holds the per-person allowance; without it nobody is
	// limited.
	AgentControls tenantports.AgentControlRepository `optional:"true"`
	// Users names who turned an agent off.
	Users tenantports.UserRepository `optional:"true"`
	// Notifier tells the people who run AI Control that someone asked for
	// more room: access to an agent, more allowance, more budget.
	Notifications *notificationservice.Service `optional:"true"`
	// Completion rewrites a passage of a document a person asked to change,
	// with the same models the conversation's agent uses.
	Completion agentports.StructuredCompleter `optional:"true"`
	// Reports re-runs a report preview whole for its download, and Queries
	// re-runs the list a table was read from; PDFs prints a document.
	Reports *reporting.Service                `optional:"true"`
	Queries agentports.AgentQueryToolRegistry `optional:"true"`
	PDFs    documenttemplateports.PDFRenderer `optional:"true"`
	// Memories describes the memories a reply used or saved, as the thread
	// is served.
	Memories agentports.AgentMemoryService `optional:"true"`
	Quota    platformplanports.QuotaGuard  `optional:"true"`
	// Waits reads a wait whose ending a turn picks up.
	Waits agentwaitports.AgentWaitRepository `optional:"true"`
	// ReadWaits parks a conversation on a file still being read.
	ReadWaits agentwaitports.DocumentReadWaiter `optional:"true"`
	// Schedules settles the requests scheduled in a conversation being
	// deleted, so none fires again and no open list keeps showing one.
	Schedules conversationscheduleports.ConversationThreadSchedules `optional:"true"`
}

// Module provides the assistant once, as itself for the worker that runs its
// turns and as the AssistantService port for everything else.
var Module = fx.Module("assistant",
	fx.Provide(fx.Annotate(
		New,
		fx.As(fx.Self()),
		fx.As(new(agentports.AssistantService)),
		fx.As(new(agentports.PageAssistant)),
		fx.As(new(agentports.AgentRunTranscriptService)),
	)),
	fx.Provide(fx.Annotate(
		func(s *Service) agentports.AttachmentExtractionObserver { return s },
		fx.ResultTags(`group:"attachment_extraction_observers"`),
	)),
)

type Service struct {
	logger        *zap.Logger
	guard         *agentguard.Service
	runtime       *agentruntime.Service
	contexts      agentports.RuntimeContextBuilder
	conversations agentports.ConversationRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	recorder      *proposalrecorder.Service
	proposals     chatProposalStore
	proposalEdits proposalEditStore
	plans         chatPlanStore
	providers     agentports.AIProviderRepository
	shadow        *agentshadow.Resolver
	budgets       agentports.AgentBudgetService
	tools         agentports.AgentToolRegistry
	decisions     agentports.AgentDecisionRepository
	labeler       agentports.RecordLabeler
	artifacts     assistantartifactports.AssistantArtifactRepository
	subjects      agentports.AgentSubjectDescriber
	anchors       agentports.RecordAnchorReader
	activity      agentports.AgentActivityPublisher
	documents     documentports.DocumentRepository
	contents      documentports.DocumentContentService
	permissions   permissionports.PermissionEngine
	cases         deskcaseports.AssistantCaseStates
	runs          agentports.AgentRunRepository
	systemAgents  agentdefinitionports.SystemAgentProvisioner
	pageThreads   agentports.PageThreadRepository
	agentControls tenantports.AgentControlRepository
	users         tenantports.UserRepository
	notifier      raiseNotifier
	completion    agentports.StructuredCompleter
	reports       *reporting.Service
	queries       agentports.AgentQueryToolRegistry
	pdfs          documenttemplateports.PDFRenderer
	memories      agentports.AgentMemoryService
	quota         platformplanports.QuotaGuard
	waits         agentwaitports.AgentWaitRepository
	readWaits     agentwaitports.DocumentReadWaiter
	schedules     conversationscheduleports.ConversationThreadSchedules
}

func New(p Params) *Service {
	s := &Service{
		logger:        p.Logger.Named("service.assistant"),
		guard:         p.Guard,
		runtime:       p.Runtime,
		contexts:      p.Contexts,
		conversations: p.Conversations,
		definitions:   p.Definitions,
		providers:     p.AIProviders,
		recorder:      p.Recorder,
		proposals:     p.Proposals,
		proposalEdits: p.Proposals,
		plans:         planStoreOrNil(p.Plans),
		shadow:        p.Shadow,
		budgets:       p.Budgets,
		tools:         p.Tools,
		decisions:     p.Decisions,
		labeler:       p.Labeler,
		artifacts:     p.Artifacts,
		subjects:      p.Subjects,
		anchors:       p.Anchors,
		activity:      p.Activity,
		documents:     p.Documents,
		contents:      p.Contents,
		permissions:   p.Permissions,
		cases:         p.Cases,
		runs:          p.Runs,
		waits:         p.Waits,
		readWaits:     p.ReadWaits,
		schedules:     p.Schedules,
		systemAgents:  p.SystemAgents,
		pageThreads:   p.PageThreads,
		agentControls: p.AgentControls,
		users:         p.Users,
		notifier:      raiseNotifierOf(p.Notifications),
		completion:    p.Completion,
		queries:       p.Queries,
		pdfs:          p.PDFs,
		reports:       p.Reports,
		memories:      p.Memories,
		quota:         p.Quota,
	}

	return s
}

// raiseNotifierOf keeps a missing notification service nil as an interface,
// so the service can tell that requests cannot be sent.
func raiseNotifierOf(notifications *notificationservice.Service) raiseNotifier {
	if notifications == nil {
		return nil
	}

	return notifications
}
