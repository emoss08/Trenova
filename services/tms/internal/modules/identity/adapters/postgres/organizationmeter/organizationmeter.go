package organizationmeter

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Result struct {
	fx.Out

	Reader managedusageports.LevelReader `group:"managed_level_readers"`
}

type organizations struct {
	db *postgres.Connection
}

func New(p Params) Result {
	return Result{Reader: &organizations{db: p.DB}}
}

func (o *organizations) Meter() platformcatalog.MeterKey {
	return platformcatalog.MeterOrganizationsTotal
}

func (o *organizations) Level(ctx context.Context, scope managedscope.Scope) (int64, error) {
	if tenant, ok := scope.Tenant(); ok {
		ctx = dbscope.WithTenant(ctx, tenant)
	} else {
		ctx = dbscope.WithSystem(ctx, "count the organizations on this managed instance")
	}
	return dbtx.Read(ctx, o.db, func(ctx context.Context) (int64, error) {
		org := tenantdb.Organizations
		n, err := tenantscope.Scoped(ctx, org.Query()).Count(ctx, o.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count organizations: %w", err)
		}
		return n, nil
	})
}
