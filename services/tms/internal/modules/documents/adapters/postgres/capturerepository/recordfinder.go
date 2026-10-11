package capturerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen/conversationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tractorcol"
	"github.com/emoss08/trenova/pkg/buncolgen/trailercol"
	"github.com/emoss08/trenova/pkg/buncolgen/workercol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

// recordKind is how one fileable kind is found and named. The title is what
// a person recognises the record by; the subtitle is the second number they
// would quote if the first were ambiguous.
type recordKind struct {
	model    func() any
	tenant   func(ti pagination.TenantInfo) func(*bun.SelectQuery) *bun.SelectQuery
	id       buncolgen.Column
	title    string
	subtitle string
}

var recordKinds = map[string]recordKind{
	permission.ResourceShipment.String(): {
		model:    func() any { return (*shipment.Shipment)(nil) },
		tenant:   shipmentcol.ShipmentApplyTenant,
		id:       shipmentcol.ShipmentColumns.ID,
		title:    shipmentcol.ShipmentColumns.ProNumber.Expr("COALESCE({}, '')"),
		subtitle: shipmentcol.ShipmentColumns.BOL.Expr("COALESCE({}, '')"),
	},
	permission.ResourceWorker.String(): {
		model:  func() any { return (*worker.Worker)(nil) },
		tenant: workercol.WorkerApplyTenant,
		id:     workercol.WorkerColumns.ID,
		title: buncolgen.Expr(
			"CONCAT_WS(' ', {0}, {1})",
			workercol.WorkerColumns.FirstName,
			workercol.WorkerColumns.LastName,
		),
		subtitle: "''",
	},
	permission.ResourceTractor.String(): {
		model:    func() any { return (*tractor.Tractor)(nil) },
		tenant:   tractorcol.TractorApplyTenant,
		id:       tractorcol.TractorColumns.ID,
		title:    tractorcol.TractorColumns.Code.Expr("COALESCE({}, '')"),
		subtitle: tractorcol.TractorColumns.LicensePlateNumber.Expr("COALESCE({}, '')"),
	},
	permission.ResourceTrailer.String(): {
		model:    func() any { return (*trailer.Trailer)(nil) },
		tenant:   trailercol.TrailerApplyTenant,
		id:       trailercol.TrailerColumns.ID,
		title:    trailercol.TrailerColumns.Code.Expr("COALESCE({}, '')"),
		subtitle: trailercol.TrailerColumns.LicensePlateNumber.Expr("COALESCE({}, '')"),
	},
	permission.ResourceCustomer.String(): {
		model:    func() any { return (*customer.Customer)(nil) },
		tenant:   customercol.CustomerApplyTenant,
		id:       customercol.CustomerColumns.ID,
		title:    customercol.CustomerColumns.Name.Expr("COALESCE({}, '')"),
		subtitle: customercol.CustomerColumns.Code.Expr("COALESCE({}, '')"),
	},
	capture.ResourceAssistantThread: {
		model:    func() any { return (*conversation.Thread)(nil) },
		tenant:   ownThread,
		id:       conversationcol.ThreadColumns.ID,
		title:    conversationcol.ThreadColumns.Title.Expr("COALESCE({}, '')"),
		subtitle: "''",
	},
	permission.ResourceCarrier.String(): {
		model:    func() any { return (*carrier.Carrier)(nil) },
		tenant:   carriercol.CarrierApplyTenant,
		id:       carriercol.CarrierColumns.ID,
		title:    carriercol.CarrierColumns.Name.Expr("COALESCE({}, '')"),
		subtitle: carriercol.CarrierColumns.Code.Expr("COALESCE({}, '')"),
	},
}

func ownThread(ti pagination.TenantInfo) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Apply(conversationcol.ThreadApplyTenant(ti)).
			Where(conversationcol.ThreadColumns.UserID.Eq(), ti.UserID)
	}
}

func lookupKind(resourceType string) (recordKind, error) {
	kind, ok := recordKinds[resourceType]
	if !ok {
		return recordKind{}, fmt.Errorf("captures cannot be filed onto %q records", resourceType)
	}

	return kind, nil
}

type recordFinder struct {
	db *postgres.Connection
}

func NewRecordFinder(p Params) captureports.CaptureRecordFinder {
	return &recordFinder{db: p.DB}
}

func (f *recordFinder) Exists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	id pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, f.db, func(ctx context.Context) (bool, error) {
		kind, err := lookupKind(resourceType)
		if err != nil {
			return false, err
		}

		return f.db.DBForContext(ctx).
			NewSelect().
			Model(kind.model()).
			Apply(kind.tenant(tenantInfo)).
			Where(kind.id.Eq(), id).
			Exists(ctx)
	})
}

func (f *recordFinder) Labels(
	ctx context.Context,
	req *captureports.ListCaptureRecordLabelsRequest,
) ([]*captureports.CaptureRecordLabel, error) {
	return dbtx.Read(
		ctx,
		f.db,
		func(ctx context.Context) ([]*captureports.CaptureRecordLabel, error) {
			kind, err := lookupKind(req.ResourceType)
			if err != nil {
				return nil, err
			}
			if len(req.IDs) == 0 {
				return []*captureports.CaptureRecordLabel{}, nil
			}

			labels := make([]*captureports.CaptureRecordLabel, 0, len(req.IDs))
			if err = f.db.DBForContext(ctx).
				NewSelect().
				Model(kind.model()).
				ColumnExpr(kind.id.As("id")).
				ColumnExpr(kind.title+" AS title").
				ColumnExpr(kind.subtitle+" AS subtitle").
				Apply(kind.tenant(req.TenantInfo)).
				Where(kind.id.In(), bun.List(req.IDs)).
				Scan(ctx, &labels); err != nil {
				return nil, err
			}
			for _, label := range labels {
				label.ResourceType = req.ResourceType
			}

			return labels, nil
		},
	)
}
