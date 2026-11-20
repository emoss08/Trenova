package cloud

import (
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/cloudtenantrepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantpeoplerepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantpurgerepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/redis/tenantkeys"
	"github.com/emoss08/trenova/internal/cloud/adapters/search/tenantsearch"
	"github.com/emoss08/trenova/internal/cloud/api/tenanthandler"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			cloudtenantrepository.New,
			tenantpeoplerepository.New,
			tenantpurgerepository.New,
			tenantkeys.New,
			tenantsearch.New,
			tenantlifecycle.New,
		),
		fx.Decorate(instanceTags),
	)
}

func APIOptions() fx.Option {
	return fx.Options(
		fx.Provide(tenanthandler.New),
	)
}

func WorkerOptions() fx.Option {
	return tenantjobs.Module
}

func instanceTags(
	cfg *managedconfig.Config,
	configured emailports.InstanceTags,
	tenants cloudtenantports.Repository,
) emailports.InstanceTags {
	if cfg.SharedTenancy() {
		return tenantlifecycle.NewInstanceTags(tenants)
	}
	return configured
}
