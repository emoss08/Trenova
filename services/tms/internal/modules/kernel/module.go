package kernel

import (
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/settingversionrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/usstaterepository"
	"github.com/emoss08/trenova/internal/modules/kernel/app/entitlementservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/platformbillingservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usstateservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			usstaterepository.New,
			settingversionrepository.New,
			fx.Annotate(
				entitlementservice.NewLocalEntitlementProvider,
				fx.As(new(platformcatalogports.EntitlementProvider)),
			),
			fx.Annotate(
				platformbillingservice.NewLocalBillingProvider,
				fx.As(new(platformcatalogports.BillingProvider)),
			),
			fx.Annotate(
				usageservice.NewNoopUsageProvider,
				fx.As(new(platformcatalogports.UsageProvider)),
			),
			usstateservice.New,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
