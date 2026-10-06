package assistantservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Guard         *agentguard.Service
	Runtime       *agentruntime.Service
	Contexts      serviceports.RuntimeContextBuilder
	Conversations repositories.ConversationRepository
	Definitions   repositories.AgentDefinitionRepository
	Recorder      *proposalrecorder.Service
	Proposals     repositories.AgentProposalRepository
	Plans         repositories.AgentPlanRepository `optional:"true"`
	AIProviders   repositories.AIProviderRepository
	Shadow        *agentshadow.Resolver
	Budgets       serviceports.AgentBudgetService `optional:"true"`
	// Tools names the parameters a proposal's tool takes, for the fields a
	// person may edit before approving; Decisions carries what they changed.
	Tools     serviceports.AgentToolRegistry       `optional:"true"`
	Decisions repositories.AgentDecisionRepository `optional:"true"`
	// Labeler names the records a proposal offers for a person to untick.
	Labeler serviceports.RecordLabeler `optional:"true"`
	// Artifacts keeps what a turn produced besides words; Subjects describes
	// the record a conversation was opened from.
	Artifacts repositories.AssistantArtifactRepository `optional:"true"`
	Subjects  serviceports.AgentSubjectDescriber       `optional:"true"`
	Activity  serviceports.AgentActivityPublisher      `optional:"true"`
	// Documents checks that an attached file is the person's own; Contents
	// reads what document intelligence made of it.
	Documents repositories.DocumentRepository     `optional:"true"`
	Contents  serviceports.DocumentContentService `optional:"true"`
	// Permissions decides whether the person may read the record a
	// conversation is about before any of it reaches the model.
	Permissions serviceports.PermissionEngine
	// Runs says which agent raised each of a conversation's proposals: its
	// own, or one it handed a task to.
	Runs         repositories.AgentRunRepository     `optional:"true"`
	SystemAgents serviceports.SystemAgentProvisioner `optional:"true"`
	PageThreads  repositories.PageThreadRepository   `optional:"true"`
	// AgentControls holds the per-person allowance; without it nobody is
	// limited.
	AgentControls repositories.AgentControlRepository `optional:"true"`
	// Users names who turned an agent off.
	Users repositories.UserRepository `optional:"true"`
	// Notifier tells the people who run AI Control that someone asked for
	// more room: access to an agent, more allowance, more budget.
	Notifications *notificationservice.Service `optional:"true"`
	// Completion rewrites a passage of a document a person asked to change,
	// with the same models the conversation's agent uses.
	Completion serviceports.StructuredCompleter `optional:"true"`
	// Reports re-runs a report preview whole for its download, and Queries
	// re-runs the list a table was read from; PDFs prints a document.
	Reports *reporting.Service                  `optional:"true"`
	Queries serviceports.AgentQueryToolRegistry `optional:"true"`
	PDFs    serviceports.PDFRenderer            `optional:"true"`
	// Memories describes the memories a reply used or saved, as the thread
	// is served.
	Memories serviceports.AgentMemoryService `optional:"true"`
	Quota    serviceports.QuotaGuard         `optional:"true"`
}

// Module provides the assistant once, as itself for the worker that runs its
// turns and as the AssistantService port for everything else.
var Module = fx.Module("assistant",
	fx.Provide(fx.Annotate(
		New,
		fx.As(fx.Self()),
		fx.As(new(serviceports.AssistantService)),
		fx.As(new(serviceports.PageAssistant)),
		fx.As(new(serviceports.AgentRunTranscriptService)),
	)),
)

type Service struct {
	logger        *zap.Logger
	guard         *agentguard.Service
	runtime       *agentruntime.Service
	contexts      serviceports.RuntimeContextBuilder
	conversations repositories.ConversationRepository
	definitions   repositories.AgentDefinitionRepository
	recorder      *proposalrecorder.Service
	proposals     chatProposalStore
	proposalEdits proposalEditStore
	plans         chatPlanStore
	providers     repositories.AIProviderRepository
	shadow        *agentshadow.Resolver
	budgets       serviceports.AgentBudgetService
	tools         serviceports.AgentToolRegistry
	decisions     repositories.AgentDecisionRepository
	labeler       serviceports.RecordLabeler
	artifacts     repositories.AssistantArtifactRepository
	subjects      serviceports.AgentSubjectDescriber
	activity      serviceports.AgentActivityPublisher
	documents     repositories.DocumentRepository
	contents      serviceports.DocumentContentService
	permissions   serviceports.PermissionEngine
	runs          repositories.AgentRunRepository
	systemAgents  serviceports.SystemAgentProvisioner
	pageThreads   repositories.PageThreadRepository
	agentControls repositories.AgentControlRepository
	users         repositories.UserRepository
	notifier      raiseNotifier
	completion    serviceports.StructuredCompleter
	reports       *reporting.Service
	queries       serviceports.AgentQueryToolRegistry
	pdfs          serviceports.PDFRenderer
	memories      serviceports.AgentMemoryService
	quota         serviceports.QuotaGuard
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
		activity:      p.Activity,
		documents:     p.Documents,
		contents:      p.Contents,
		permissions:   p.Permissions,
		runs:          p.Runs,
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
