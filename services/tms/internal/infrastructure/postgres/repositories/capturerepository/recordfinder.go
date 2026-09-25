package capturerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

// recordLookup scopes an existence query to one record kind in one tenant.
type recordLookup func(q *bun.SelectQuery, ti pagination.TenantInfo, id pulid.ID) *bun.SelectQuery

var recordLookups = map[string]recordLookup{
	permission.ResourceShipment.String(): func(q *bun.SelectQuery, ti pagination.TenantInfo, id pulid.ID) *bun.SelectQuery {
		return q.Model((*shipment.Shipment)(nil)).
			Apply(buncolgen.ShipmentApplyTenant(ti)).
			Where(buncolgen.ShipmentColumns.ID.Eq(), id)
	},
	permission.ResourceWorker.String(): func(q *bun.SelectQuery, ti pagination.TenantInfo, id pulid.ID) *bun.SelectQuery {
		return q.Model((*worker.Worker)(nil)).
			Apply(buncolgen.WorkerApplyTenant(ti)).
			Where(buncolgen.WorkerColumns.ID.Eq(), id)
	},
	permission.ResourceTractor.String(): func(q *bun.SelectQuery, ti pagination.TenantInfo, id pulid.ID) *bun.SelectQuery {
		return q.Model((*tractor.Tractor)(nil)).
			Apply(buncolgen.TractorApplyTenant(ti)).
			Where(buncolgen.TractorColumns.ID.Eq(), id)
	},
	permission.ResourceTrailer.String(): func(q *bun.SelectQuery, ti pagination.TenantInfo, id pulid.ID) *bun.SelectQuery {
		return q.Model((*trailer.Trailer)(nil)).
			Apply(buncolgen.TrailerApplyTenant(ti)).
			Where(buncolgen.TrailerColumns.ID.Eq(), id)
	},
	permission.ResourceCustomer.String(): func(q *bun.SelectQuery, ti pagination.TenantInfo, id pulid.ID) *bun.SelectQuery {
		return q.Model((*customer.Customer)(nil)).
			Apply(buncolgen.CustomerApplyTenant(ti)).
			Where(buncolgen.CustomerColumns.ID.Eq(), id)
	},
	permission.ResourceCarrier.String(): func(q *bun.SelectQuery, ti pagination.TenantInfo, id pulid.ID) *bun.SelectQuery {
		return q.Model((*carrier.Carrier)(nil)).
			Apply(buncolgen.CarrierApplyTenant(ti)).
			Where(buncolgen.CarrierColumns.ID.Eq(), id)
	},
}

type recordFinder struct {
	db *postgres.Connection
}

func NewRecordFinder(p Params) repositories.CaptureRecordFinder {
	return &recordFinder{db: p.DB}
}

func (f *recordFinder) Exists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	id pulid.ID,
) (bool, error) {
	lookup, ok := recordLookups[resourceType]
	if !ok {
		return false, fmt.Errorf("captures cannot be filed onto %q records", resourceType)
	}

	return lookup(f.db.DBForContext(ctx).NewSelect(), tenantInfo, id).Exists(ctx)
}
