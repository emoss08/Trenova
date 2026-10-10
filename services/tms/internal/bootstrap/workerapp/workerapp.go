package workerapp

import (
	"context"
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/core/services/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/core/services/agenttoolservice"
	"github.com/emoss08/trenova/internal/core/services/bulkeditservice"
	"github.com/emoss08/trenova/internal/core/services/runstepledger"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/accountingsyncjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentqualityjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentwaitjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/aiauditjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/aicorrectionjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/aifeedbackjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/aituneupjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/auditjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingqueuejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingtransferjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/briefingjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/bulkeditjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/capturejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/carrierintelligencejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/carriersettlementjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/completionjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/compliancejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/conversationschedulejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/decisioncommitjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/detentionjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/dispatchjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/distancemileagejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentintelligencejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/edijobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/emailjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/exchangeratejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/extractionevaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/extractionshadowjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/fiscaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/formulatemplatejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/fuelcardjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/fuelpricejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/iftajobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/importassistantjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/inboundjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/insightjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/invoiceadjustmentjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/ptojobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/ratesimjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/recurringshipmentjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/reflectionjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/reportjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/retrievaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/samsarajobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/core/temporaljobs/settlementjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/shipmentjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/smsjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/telematicsjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/tenderjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/thumbnailjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/watchtowerjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/weatheralertjobs"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionrouter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/turnstream"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

//nolint:funlen // the module list is the worker-only application graph
func Modules() fx.Option {
	return fx.Options(
		schedule.Module,
		auditjobs.Module,
		aiauditjobs.Module,
		billingjobs.Module,
		billingtransferjobs.Module,
		bulkeditjobs.Module,
		fx.Provide(func(svc *bulkeditservice.Service) bulkeditjobs.Runner { return svc }),
		billingqueuejobs.Module,
		detentionjobs.Module,
		agentwaitjobs.Module,
		ptojobs.Module,
		distancemileagejobs.Module,
		documentintelligencejobs.Module,
		documentuploadjobs.Module,
		edijobs.Module,
		emailjobs.Module,
		exchangeratejobs.Module,
		fuelcardjobs.Module,
		carrierintelligencejobs.Module,
		accountingsyncjobs.Module,
		fuelpricejobs.Module,
		formulatemplatejobs.Module,
		ratesimjobs.Module,
		thumbnailjobs.Module,
		smsjobs.Module,
		samsarajobs.Module,
		telematicsjobs.Module,
		shipmentjobs.Module,
		agentjobs.Module,
		reflectionjobs.Module,
		agentqualityjobs.Module,
		agentflow.Module,
		assistantjobs.Module,
		completionjobs.Module,
		importassistantjobs.Module,
		agenttoolservice.Module,
		agentquerytoolservice.Module,
		agenttoolcatalog.Module,
		agenttoolpolicy.Module,
		agentruntime.Module,
		runstepledger.Module,
		turnstream.Module,
		conversationschedulejobs.Module,
		completionrouter.Module,
		recurringshipmentjobs.Module,
		settlementjobs.Module,
		carriersettlementjobs.Module,
		tenderjobs.Module,
		compliancejobs.Module,
		insightjobs.Module,
		watchtowerjobs.Module,
		inboundjobs.Module,
		capturejobs.Module,
		briefingjobs.Module,
		aicorrectionjobs.Module,
		extractionevaljobs.Module,
		extractionshadowjobs.Module,
		aifeedbackjobs.Module,
		aituneupjobs.Module,
		decisioncommitjobs.Module,
		retrievaljobs.Module,
		iftajobs.Module,
		dispatchjobs.Module,
		weatheralertjobs.Module,
		fiscaljobs.Module,
		invoiceadjustmentjobs.Module,
		reportjobs.Module,
	)
}

func Options() fx.Option {
	return OptionsFor(edition.Current())
}

func OptionsFor(e *edition.Edition) fx.Option {
	return fx.Options(
		temporaljobs.WorkerModule,
		agentjobs.WorkerModule,
		conversationschedulejobs.WorkerModule,
		agentqualityjobs.WorkerModule,
		e.WorkerOption(),
	)
}

func UnscheduledOptions() fx.Option {
	return temporaljobs.UnscheduledWorkerModule
}

type RunParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

func Run(base fx.Option, queues []string) {
	opts := []fx.Option{base, Options(), fx.Invoke(logLifecycle)}
	if len(queues) > 0 {
		opts = append(opts, fx.Supply(&registry.QueueFilter{Queues: queues}))
	}

	fx.New(opts...).Run()
}

func logLifecycle(lc fx.Lifecycle, params RunParams) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			params.Logger.Info("Starting worker service",
				zap.String("environment", params.Config.App.Env),
				zap.String("version", params.Config.App.Version),
			)

			params.Logger.Info("Temporal workers initialized and running")

			return nil
		},
		OnStop: func(_ context.Context) error {
			params.Logger.Info("Stopping worker service")
			params.Logger.Info("All workers stopped gracefully")

			return nil
		},
	})
}
