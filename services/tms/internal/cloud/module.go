package cloud

import (
	"github.com/emoss08/trenova/internal/cloud/adapters/objectstore/archivebucket"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/cloudcommandrepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/cloudtenantrepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantcopyrepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantpeoplerepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantpurgerepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/redis/tenantkeys"
	"github.com/emoss08/trenova/internal/cloud/adapters/search/tenantsearch"
	"github.com/emoss08/trenova/internal/cloud/api/tenanthandler"
	"github.com/emoss08/trenova/internal/cloud/app/archiveengine"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/jobs/archivejobs"
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
			cloudcommandrepository.New,
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

func ArchiveOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			tenantcopyrepository.New,
			archivebucket.New,
			archiveengine.New,
		),
	)
}

func WorkerOptions() fx.Option {
	return fx.Options(
		ArchiveOptions(),
		tenantjobs.Module,
		archivejobs.Module,
	)
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
