package organizationmeter

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const organizationsLevelReason = "count the organizations of each shared-server organization's business unit, in two statements, for the control plane's usage report; each organization is counted only for itself"

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

type organizationUnit struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type unitCount struct {
	BusinessUnitID pulid.ID
	Organizations  int64
}

func (o *organizations) LevelsForOrganizations(
	ctx context.Context,
	organizationIDs []pulid.ID,
) ([]managedusageports.OrganizationLevel, error) {
	if len(organizationIDs) == 0 {
		return []managedusageports.OrganizationLevel{}, nil
	}
	ctx = dbscope.WithSystem(ctx, organizationsLevelReason)
	return dbtx.Read(
		ctx,
		o.db,
		func(ctx context.Context) ([]managedusageports.OrganizationLevel, error) {
			org := tenantdb.Organizations
			db := o.db.Keel(ctx)
			units, err := keel.Into[organizationUnit](
				ctx,
				db,
				org.Where(org.ID.In(organizationIDs...)).
					AllTenants().
					Pick(org.ID, org.BusinessUnitID),
			)
			if err != nil {
				return nil, fmt.Errorf("read the business units of organizations: %w", err)
			}
			unitIDs := make([]pulid.ID, 0, len(units))
			for _, u := range units {
				unitIDs = append(unitIDs, u.BusinessUnitID)
			}
			counts, err := keel.Into[unitCount](
				ctx,
				db,
				org.Where(org.BusinessUnitID.In(unitIDs...)).
					AllTenants().
					Group(org.BusinessUnitID).
					Pick(org.BusinessUnitID, keel.CastInt64(keel.Count())),
			)
			if err != nil {
				return nil, fmt.Errorf("count organizations by business unit: %w", err)
			}
			byUnit := make(map[pulid.ID]int64, len(counts))
			for _, c := range counts {
				byUnit[c.BusinessUnitID] = c.Organizations
			}
			levels := make([]managedusageports.OrganizationLevel, 0, len(units))
			for _, u := range units {
				levels = append(levels, managedusageports.OrganizationLevel{
					OrganizationID: u.OrganizationID,
					Quantity:       byUnit[u.BusinessUnitID],
				})
			}
			return levels, nil
		},
	)
}
