package kernel

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/importsconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/managedfeaturesrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/managedusagerepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/recordvisibilityrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/settingversionrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/usstaterepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/workqueuerepository"
	"github.com/emoss08/trenova/internal/modules/kernel/app/editioninfo"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/entitlementservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedplanservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedusageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/realtimeservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usstateservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/versionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			editioninfo.SelfHosted,
			realtimeservice.New,
			versionservice.New,
			encryptionservice.New,
			encryptionservice.NewSecretCipher,
			usstaterepository.New,
			settingversionrepository.New,
			recordvisibilityrepository.New,
			workqueuerepository.New,
			managedfeaturesrepository.New,
			managedfeaturesservice.New,
			managedfeaturesservice.NewReceiver,
			managedplanservice.New,
			featureGate,
			managedusagerepository.New,
			managedusageservice.New,
			managedusageservice.NewReceiver,
			func(s *managedusageservice.Service) managedusageports.Gate { return s },
			usageProvider,
			usstateservice.New,
			spreadsheetLimits,
			timeutils.SystemClock,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}

func usageProvider(
	cfg *managedconfig.Config,
	gate managedusageports.Gate,
) platformcatalogports.UsageProvider {
	if cfg.Enabled {
		return usageservice.NewManagedUsageProvider(gate)
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
