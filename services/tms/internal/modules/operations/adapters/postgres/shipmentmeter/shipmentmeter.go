package shipmentmeter

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
)

const (
	hour                     = 3600
	organizationsCountReason = "count the shipments each organization of a shared server created in each hour, in one statement, for the control plane's usage report; each organization is counted only for itself"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Result struct {
	fx.Out

	Counter managedusageports.HourlyCounter `group:"managed_hourly_counters"`
}

type counter struct {
	db *postgres.Connection
}

func New(p Params) Result {
	return Result{Counter: &counter{db: p.DB}}
}

func (c *counter) Meter() platformcatalog.MeterKey {
	return platformcatalog.MeterShipmentsTotal
}

func (c *counter) CountByHour(
	ctx context.Context,
	req *managedusageports.HourCountRequest,
) ([]managedusageports.HourCount, error) {
	if tenant, ok := req.Scope.Tenant(); ok {
		ctx = dbscope.WithTenant(ctx, tenant)
	} else {
		ctx = dbscope.WithSystem(
			ctx,
			"count the shipments created each hour on this managed instance",
		)
	}
	return dbtx.Read(ctx, c.db, func(ctx context.Context) ([]managedusageports.HourCount, error) {
		s := shipmentdb.Shipments
		bucket := keel.Int64Of(s.CreatedAt).Div(keel.Int64(hour)).Mul(keel.Int64(hour))
		counts, err := keel.Into[managedusageports.HourCount](
			ctx,
			c.db.Keel(ctx),
			tenantscope.Scoped(ctx, s.Where(s.CreatedAt.Gte(req.Since), s.CreatedAt.Lt(req.Until))).
				Group(bucket).
				Pick(bucket, keel.CastInt64(keel.Count())),
		)
		if err != nil {
			return nil, fmt.Errorf("count shipments by hour: %w", err)
		}
		return counts, nil
	})
}

func (c *counter) CountByHourForOrganizations(
	ctx context.Context,
	req *managedusageports.OrganizationsHourCountRequest,
) ([]managedusageports.OrganizationHourCount, error) {
	if len(req.OrganizationIDs) == 0 {
		return []managedusageports.OrganizationHourCount{}, nil
	}
	ctx = dbscope.WithSystem(ctx, organizationsCountReason)
	return dbtx.Read(
		ctx,
		c.db,
		func(ctx context.Context) ([]managedusageports.OrganizationHourCount, error) {
			s := shipmentdb.Shipments
			bucket := keel.Int64Of(s.CreatedAt).Div(keel.Int64(hour)).Mul(keel.Int64(hour))
			counts, err := keel.Into[managedusageports.OrganizationHourCount](
				ctx,
				c.db.Keel(ctx),
				s.Where(
					s.OrganizationID.In(req.OrganizationIDs...),
					s.CreatedAt.Gte(req.Since),
					s.CreatedAt.Lt(req.Until),
				).
					AllTenants().
					Group(s.OrganizationID, bucket).
					Pick(s.OrganizationID, bucket, keel.CastInt64(keel.Count())),
			)
			if err != nil {
				return nil, fmt.Errorf("count shipments by hour and organization: %w", err)
			}
			return counts, nil
		},
	)
}
