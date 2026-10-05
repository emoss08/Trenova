package cloud

import (
	"io/fs"

	"github.com/emoss08/trenova/internal/api/graphql"
	"github.com/emoss08/trenova/internal/api/handlers/publicconfighandler"
	"github.com/emoss08/trenova/internal/api/routegroup"
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/cloud/aitraining"
	"github.com/emoss08/trenova/internal/cloud/aitraining/aicli"
	"github.com/emoss08/trenova/internal/cloud/catalog"
	"github.com/emoss08/trenova/internal/cloud/catalog/platformcataloghandler"
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/cloud/cloudplan"
	"github.com/emoss08/trenova/internal/cloud/cloudquota"
	"github.com/emoss08/trenova/internal/cloud/controlplane"
	"github.com/emoss08/trenova/internal/cloud/controlplane/accessmiddleware"
	"github.com/emoss08/trenova/internal/cloud/controlplane/controlplaneprovisioninghandler"
	"github.com/emoss08/trenova/internal/cloud/controlplane/featureaccess"
	"github.com/emoss08/trenova/internal/cloud/controlplane/tenantprovisioningrepository"
	"github.com/emoss08/trenova/internal/cloud/controlplane/tenantprovisioningservice"
	"github.com/emoss08/trenova/internal/cloud/lifecycle/cloudlifecyclejobs"
	"github.com/emoss08/trenova/internal/cloud/lifecycle/tenantpurgerepository"
	"github.com/emoss08/trenova/internal/cloud/networkpulse/networkpulsehandler"
	"github.com/emoss08/trenova/internal/cloud/networkpulse/networkpulserepository"
	"github.com/emoss08/trenova/internal/cloud/networkpulse/networkpulseservice"
	"github.com/emoss08/trenova/internal/cloud/planbilling"
	"github.com/emoss08/trenova/internal/cloud/platformemailservice"
	"github.com/emoss08/trenova/internal/cloud/signup/cloudsignuphandler"
	"github.com/emoss08/trenova/internal/cloud/signup/cloudsignuprepository"
	"github.com/emoss08/trenova/internal/cloud/signup/cloudsignupservice"
	"github.com/emoss08/trenova/internal/cloud/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/migrations"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessgraphql"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccesshandler"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessmiddleware"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessrepository"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessservice"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportcli"
	"github.com/emoss08/trenova/internal/cloud/turnstile"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func Edition() edition.Edition {
	return edition.Edition{
		Name:               EditionName,
		Options:            []fx.Option{Options()},
		APIOptions:         []fx.Option{APIOptions()},
		Commands:           []*cobra.Command{aicli.AICmd, supportcli.CloudCmd},
		PostgresMigrations: []fs.FS{migrations.FS()},
		ConfigSections:     []config.Section{cloudconfig.Section()},
	}
}

func Options() fx.Option {
	return fx.Options(
		catalogOptions(),
		cloudplan.Module,
		cloudquota.Module,
		controlPlaneOptions(),
		signupOptions(),
		lifecycleOptions(),
		networkPulseOptions(),
		aitraining.Module,
		supportaccess.PermissionOptions(),
		fx.Provide(
			supportaccessrepository.New,
			platformemailservice.New,
			planbilling.NewLocalPlanBillingProvider,
			planbilling.NewLocalPlanUsageProvider,
		),
		fx.Decorate(
			SelectEntitlementProvider,
			SelectBillingProvider,
			SelectUsageProvider,
			SelectEditionInfo,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			cloudsignuphandler.New,
			networkpulsehandler.New,
			controlplaneprovisioninghandler.New,
			platformcataloghandler.New,
			accessmiddleware.NewControlPlaneAccessMiddleware,
			featureaccess.NewFeatureAccessExtension,
		),
		supportAccessAPIOptions(),
		apiRegistrations(),
	)
}

func supportAccessAPIOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				supportaccessservice.NewRedisStartLimiter,
				fx.As(new(supportaccessservice.StartLimiter)),
			),
			supportaccessservice.New,
			supportaccessmiddleware.New,
			supportaccesshandler.New,
			supportaccessgraphql.New,
		),
		fx.Decorate(supportaccessservice.DecorateAuthService),
	)
}

func apiRegistrations() fx.Option {
	return fx.Provide(
		routegroup.AsProtectedMiddleware(identity[*supportaccessmiddleware.Middleware]),
		routegroup.AsProtectedRoutes(identity[*supportaccesshandler.Handler]),
		graphql.AsExtension(identity[*supportaccessgraphql.Extension]),
		routegroup.AsPublicRoutes(identity[*cloudsignuphandler.Handler]),
		publicconfighandler.AsContributor(identity[*cloudsignuphandler.Handler]),
		routegroup.AsPublicRoutes(identity[*networkpulsehandler.Handler]),
		routegroup.AsPublicRoutes(identity[*controlplaneprovisioninghandler.Handler]),
		routegroup.AsProtectedRoutes(identity[*platformcataloghandler.Handler]),
		routegroup.AsProtectedMiddleware(
			identity[*accessmiddleware.ControlPlaneAccessMiddleware],
		),
		graphql.AsExtension(identity[*featureaccess.FeatureAccessExtension]),
	)
}

func catalogOptions() fx.Option {
	return fx.Provide(
		fx.Annotate(
			catalog.NewStaticProvider,
			fx.ResultTags(`group:"platform_catalog_providers"`),
			fx.As(new(platformcatalog.CatalogProvider)),
		),
		catalog.NewRegistry,
		fx.Annotate(
			identity[*catalog.Registry],
			fx.As(new(platformcatalog.FeatureCatalog)),
		),
	)
}

func controlPlaneOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				controlplane.NewHTTPControlPlaneClient,
				fx.As(new(controlplane.Client)),
			),
			controlplane.NewCloudEntitlementProvider,
			controlplane.NewCloudBillingProvider,
			controlplane.NewCloudUsageProvider,
			fx.Annotate(
				controlplane.NewCloudAccessAuthorizer,
				fx.As(new(services.AccessAuthorizer)),
			),
			controlplane.NewHeartbeatReporter,
			controlplane.NewTenantSyncer,
			tenantprovisioningrepository.New,
			tenantprovisioningservice.New,
		),
		fx.Invoke(
			func(*controlplane.HeartbeatReporter) {},
			func(*controlplane.TenantSyncer) {},
		),
	)
}

func signupOptions() fx.Option {
	return fx.Provide(
		cloudsignuprepository.New,
		turnstile.New,
		cloudsignupservice.New,
	)
}

func lifecycleOptions() fx.Option {
	return fx.Options(
		fx.Provide(tenantpurgerepository.New),
		cloudlifecyclejobs.Module,
	)
}

func networkPulseOptions() fx.Option {
	return fx.Provide(
		networkpulserepository.New,
		networkpulseservice.New,
	)
}

func identity[T any](value T) T {
	return value
}
