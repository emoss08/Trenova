package core

import (
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/bootstrap/infrastructure"
	"github.com/emoss08/trenova/internal/bootstrap/modules"
	modulesinfra "github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
	"github.com/emoss08/trenova/internal/core/services/agentextensionservice"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agentruneventservice"
	"github.com/emoss08/trenova/internal/core/services/agenttoolruleservice"
	"github.com/emoss08/trenova/internal/core/services/agentwaitservice"
	"github.com/emoss08/trenova/internal/core/services/aiauditservice"
	"github.com/emoss08/trenova/internal/core/services/aidocumentservice"
	"github.com/emoss08/trenova/internal/core/services/aiproviderspendservice"
	"github.com/emoss08/trenova/internal/core/services/aitraininghistoryservice"
	"github.com/emoss08/trenova/internal/core/services/analyticsservice"
	"github.com/emoss08/trenova/internal/core/services/assistantcaseservice"
	"github.com/emoss08/trenova/internal/core/services/assistantfollowupservice"
	"github.com/emoss08/trenova/internal/core/services/assistantqueueservice"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/services/casechecklistservice"
	"github.com/emoss08/trenova/internal/core/services/conversationscheduleservice"
	"github.com/emoss08/trenova/internal/core/services/editransport"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formulaassistantservice"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/internal/core/services/productguideservice"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/internal/core/services/quotaservice"
	"github.com/emoss08/trenova/internal/core/services/rateengine"
	"github.com/emoss08/trenova/internal/core/services/retrievalquery"
	"github.com/emoss08/trenova/internal/core/services/samsarasyncservice"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/samsarajobs/samsaracontract"
	accountinginfra "github.com/emoss08/trenova/internal/infrastructure/accounting"
	agentextensioninfra "github.com/emoss08/trenova/internal/infrastructure/agentextension"
	carrierintelinfra "github.com/emoss08/trenova/internal/infrastructure/carrierintel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/fuelcard"
	reportingexecutor "github.com/emoss08/trenova/internal/infrastructure/reporting/executor"
	reportingrender "github.com/emoss08/trenova/internal/infrastructure/reporting/render"
	reportingresultcache "github.com/emoss08/trenova/internal/infrastructure/reporting/resultcache"
	telematicsinfra "github.com/emoss08/trenova/internal/infrastructure/telematics"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"go.uber.org/fx"
)

func Options() fx.Option {
	return OptionsFor(edition.Current())
}

//nolint:funlen // the module list is the shared application graph
func OptionsFor(e *edition.Edition) fx.Option {
	return fx.Options(
		config.SectionsOption(e.ConfigSections...),
		config.Module,
		config.Hooks(),
		infrastructure.ObservabilityModule,
		infrastructure.RedisModule,
		infrastructure.RedisRepositoriesModule,
		infrastructure.RateLimitModule,
		infrastructure.DatabaseModule,
		modules.ValidatorModule,
		modules.PostgresRepositoryModule,
		planservice.Module,
		quotaservice.Module,
		modules.QueryCacheModule,
		fx.Provide(encryptionservice.New),
		fx.Provide(integrationservice.New),
		fx.Provide(fx.Annotate(
			func(svc *integrationservice.Service) fuelpurchaseports.FuelCardFeedResolver { return svc },
		)),
		fuelcard.Module,
		carrierintelinfra.Module,
		accountinginfra.Module,
		agentextensioninfra.Module,
		agentextensionservice.Module,
		fx.Provide(telematicsinfra.NewFactory),
		formula.Module,
		formulatemplateservice.Module,
		formulaassistantservice.Module,
		rateengine.Module,
		editransport.Module,
		temporaljobs.Module,
		productguideservice.Module,
		retrievalquery.Module,
		agentguard.Module,
		agentruneventservice.Module,
		aiauditservice.Module,
		watchtowersources.Module,
		proposalrecorder.Module,
		assistantservice.Module,
		assistantturnservice.Module,
		assistantfollowupservice.Module,
		assistantqueueservice.Module,
		agentwaitservice.Module,
		assistantcaseservice.Module,
		casechecklistservice.Module,
		conversationscheduleservice.Module,
		aiproviderspendservice.Module,
		agenttoolruleservice.LoaderModule,
		fx.Provide(aitraininghistoryservice.NewEmpty),
		fx.Provide(aidocumentservice.NewContract),
		reportingrender.Module,
		reportingresultcache.Module,
		fx.Provide(reportingexecutor.New),
		analyticsservice.Module,
		ServiceModule,
		fx.Provide(samsarasyncservice.New),
		fx.Provide(samsaracontract.NewManager),
		modulesinfra.StorageModule,
		modulesinfra.SMSModule,
		modulesinfra.PDFRenderModule,
		modulesinfra.TemplatingModule,
		modulesinfra.RealtimePublisherModule,
		modulesinfra.MeilisearchClientModule,
		e.Option(),
	)
}
