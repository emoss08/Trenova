package core

import (
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/bootstrap/infrastructure"
	"github.com/emoss08/trenova/internal/bootstrap/modules"
	modulesinfra "github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
	"github.com/emoss08/trenova/internal/core/services/analyticsservice"
	"github.com/emoss08/trenova/internal/core/services/casechecklistservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
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
	"github.com/emoss08/trenova/internal/modules/accounting"
	"github.com/emoss08/trenova/internal/modules/agent"
	"github.com/emoss08/trenova/internal/modules/billing"
	"github.com/emoss08/trenova/internal/modules/documents"
	"github.com/emoss08/trenova/internal/modules/edi"
	"github.com/emoss08/trenova/internal/modules/equipment"
	"github.com/emoss08/trenova/internal/modules/identity"
	"github.com/emoss08/trenova/internal/modules/kernel"
	"github.com/emoss08/trenova/internal/modules/operations"
	"github.com/emoss08/trenova/internal/modules/parties"
	"github.com/emoss08/trenova/internal/modules/places"
	"github.com/emoss08/trenova/internal/modules/rating"
	"github.com/emoss08/trenova/internal/modules/reference"
	"github.com/emoss08/trenova/internal/modules/settlement"
	"github.com/emoss08/trenova/internal/modules/workforce"
	"github.com/emoss08/trenova/internal/modules/workspace"
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
		modules.QueryCacheModule,
		fx.Provide(encryptionservice.New),
		fuelcard.Module,
		carrierintelinfra.Module,
		accountinginfra.Module,
		agentextensioninfra.Module,
		fx.Provide(telematicsinfra.NewFactory),
		temporaljobs.Module,
		casechecklistservice.Module,
		reportingrender.Module,
		reportingresultcache.Module,
		fx.Provide(reportingexecutor.New),
		analyticsservice.Module,
		ServiceModule,
		fx.Provide(samsaracontract.NewManager),
		modulesinfra.StorageModule,
		modulesinfra.SMSModule,
		modulesinfra.PDFRenderModule,
		modulesinfra.TemplatingModule,
		modulesinfra.RealtimePublisherModule,
		modulesinfra.MeilisearchClientModule,
		kernel.SharedOptions(),
		identity.SharedOptions(),
		workspace.SharedOptions(),
		reference.SharedOptions(),
		places.SharedOptions(),
		equipment.SharedOptions(),
		parties.SharedOptions(),
		workforce.SharedOptions(),
		documents.SharedOptions(),
		rating.SharedOptions(),
		operations.SharedOptions(),
		edi.SharedOptions(),
		accounting.SharedOptions(),
		billing.SharedOptions(),
		settlement.SharedOptions(),
		agent.SharedOptions(),
		e.Option(),
	)
}
