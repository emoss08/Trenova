package kernel

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/importsconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/databasesize"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/entitledrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/managedfeaturesrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/managedusagerepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/recordvisibilityrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/settingversionrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/usstaterepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/workqueuerepository"
	"github.com/emoss08/trenova/internal/modules/kernel/app/datausageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/editioninfo"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/entitlementservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedplanservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedusageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/photoservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/realtimeservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usstateservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/versionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			editioninfo.New,
			scopeResolver,
			realtimeservice.New,
			versionservice.New,
			encryptionservice.New,
			encryptionservice.NewSecretCipher,
			encryptionservice.NewBlobCipher,
			usstaterepository.New,
			settingversionrepository.New,
			recordvisibilityrepository.New,
			workqueuerepository.New,
			managedfeaturesrepository.New,
			managedfeaturesservice.New,
			managedfeaturesservice.NewReceiver,
			func(r *managedfeaturesservice.Receiver) managedfeaturesports.PlanEndReminder { return r },
			func(r *managedfeaturesservice.Receiver) managedfeaturesports.Writer { return r },
			managedplanservice.New,
			featureGate,
			entitledrepository.New,
			standingReader,
			managedusagerepository.New,
			managedusageservice.New,
			managedusageservice.NewReceiver,
			func(r *managedusageservice.Receiver) managedusageports.Writer { return r },
			func(r *managedusageservice.Receiver) managedusageports.OrganizationReadings { return r },
			func(s *managedusageservice.Service) managedusageports.Gate { return s },
			func(s *managedusageservice.Service) managedusageports.UsageReader { return s },
			usageProvider,
			usstateservice.New,
			photoservice.New,
			func(s *photoservice.Service) kernelports.Photos { return s },
			spreadsheetLimits,
			timeutils.SystemClock,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			databasesize.New,
			datausageservice.New,
		),
	)
}

func usageProvider(
	cfg *managedconfig.Config,
	gate managedusageports.Gate,
	scopes *managedscope.Resolver,
) platformcatalogports.UsageProvider {
	if cfg.Enabled {
		return usageservice.NewManagedUsageProvider(gate, scopes)
	}
	return usageservice.NewNoopUsageProvider()
}

func featureGate(
	cfg *managedconfig.Config,
	managed *managedfeaturesservice.Service,
) platformcatalogports.FeatureGate {
	if cfg.Enabled {
		return managed
	}
	return entitlementservice.NewAllowAll()
}

func standingReader(
	cfg *managedconfig.Config,
	managed *managedfeaturesservice.Service,
) managedfeaturesports.StandingReader {
	if cfg.SharedTenancy() {
		return managed
	}
	return managedfeaturesports.AlwaysActive{}
}

func scopeResolver(cfg *managedconfig.Config) *managedscope.Resolver {
	return managedscope.NewResolver(cfg.SharedTenancy())
}

func spreadsheetLimits(cfg *importsconfig.Config) rateimport.Limits {
	limits := cfg.Spreadsheet
	return rateimport.Limits{
		MaxRows:          limits.MaxRows,
		MaxColumns:       limits.MaxColumns,
		MaxCellBytes:     limits.MaxCellBytes,
		MaxSheetBytes:    limits.MaxSheetBytes,
		MaxUnzippedBytes: limits.MaxUnzippedBytes,
		MaxPartBytes:     limits.MaxPartBytes,
	}
}
