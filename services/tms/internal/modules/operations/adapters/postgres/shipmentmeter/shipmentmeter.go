package shipmentmeter

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
)

const hour = 3600

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
	ctx = dbscope.WithSystem(ctx, "count the shipments created each hour on this managed instance")
	return dbtx.Read(ctx, c.db, func(ctx context.Context) ([]managedusageports.HourCount, error) {
		s := shipmentdb.Shipments
		bucket := keel.Int64Of(s.CreatedAt).Div(keel.Int64(hour)).Mul(keel.Int64(hour))
		counts, err := keel.Into[managedusageports.HourCount](
			ctx,
			c.db.Keel(ctx),
			s.AllTenants().
				Group(bucket).
				Pick(bucket, keel.CastInt64(keel.Count())).
				Where(s.CreatedAt.Gte(req.Since), s.CreatedAt.Lt(req.Until)),
		)
		if err != nil {
			return nil, fmt.Errorf("count shipments by hour: %w", err)
		}
		return counts, nil
	})
}
