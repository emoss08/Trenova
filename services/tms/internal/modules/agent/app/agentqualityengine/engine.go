package agentqualityengine

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualitysettings"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentfingerprint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentevaluationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type fingerprinter interface {
	Fingerprint(
		definition *agentdefinition.Definition,
		providerID pulid.ID,
		model string,
	) *agentfingerprint.Fingerprint
}

type regressionNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

type engineSettings interface {
	Control(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*agentqualitycontrol.Control, error)
	Timezone(
		ctx context.Context,
		control *agentqualitycontrol.Control,
		tenant pagination.TenantInfo,
	) string
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            kernelports.Transactor
	SuiteRuns     agentqualityports.AgentSuiteRunRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Cases         agentqualityports.AgentEvalCaseRepository
	Evaluations   agentevaluationports.AgentEvaluationRepository
	Usage         aiusageports.AIUsageRepository
	Providers     aiproviderports.AIProviderRepository
	Settings      *agentqualitysettings.Reader
	Runtime       *agentruntime.Service
	Completion    llmports.CompletionService          `optional:"true"`
	Batches       llmports.BatchCompletionService     `optional:"true"`
	Watchtower    watchtowerports.WatchtowerProjector `optional:"true"`
	Notifications *notificationservice.Service        `optional:"true"`
}

type Engine struct {
	l           *zap.Logger
	db          kernelports.Transactor
	suiteRuns   agentqualityports.AgentSuiteRunRepository
	definitions agentdefinitionports.AgentDefinitionRepository
	cases       agentqualityports.AgentEvalCaseRepository
	evaluations agentevaluationports.AgentEvaluationRepository
	usage       aiusageports.AIUsageRepository
	providers   aiproviderports.AIProviderRepository
	settings    engineSettings
	runtime     fingerprinter
	completion  llmports.CompletionService
	batches     llmports.BatchCompletionService
	watchtower  watchtowerports.WatchtowerProjector
	notifier    regressionNotifier
	now         func() int64
}

func New(p Params) *Engine {
	var notifier regressionNotifier
	if p.Notifications != nil {
		notifier = p.Notifications
	}
	var runtime fingerprinter
	if p.Runtime != nil {
		runtime = p.Runtime
	}

	return &Engine{
		l:           p.Logger.Named("service.agentquality"),
		db:          p.DB,
		suiteRuns:   p.SuiteRuns,
		definitions: p.Definitions,
		cases:       p.Cases,
		evaluations: p.Evaluations,
		usage:       p.Usage,
		providers:   p.Providers,
		settings:    p.Settings,
		runtime:     runtime,
		completion:  p.Completion,
		batches:     p.Batches,
		watchtower:  p.Watchtower,
		notifier:    notifier,
		now:         timeutils.NowUnix,
	}
}

func AsEvaluationBudget(e *Engine) agentqualityports.EvaluationBudget { return e }
