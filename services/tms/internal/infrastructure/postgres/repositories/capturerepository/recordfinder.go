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
		tenant:   buncolgen.ShipmentApplyTenant,
		id:       buncolgen.ShipmentColumns.ID,
		title:    buncolgen.ShipmentColumns.ProNumber.Expr("COALESCE({}, '')"),
		subtitle: buncolgen.ShipmentColumns.BOL.Expr("COALESCE({}, '')"),
	},
	permission.ResourceWorker.String(): {
		model:  func() any { return (*worker.Worker)(nil) },
		tenant: buncolgen.WorkerApplyTenant,
		id:     buncolgen.WorkerColumns.ID,
		title: buncolgen.Expr(
			"CONCAT_WS(' ', {0}, {1})",
			buncolgen.WorkerColumns.FirstName,
			buncolgen.WorkerColumns.LastName,
		),
		subtitle: "''",
	},
	permission.ResourceTractor.String(): {
		model:    func() any { return (*tractor.Tractor)(nil) },
		tenant:   buncolgen.TractorApplyTenant,
		id:       buncolgen.TractorColumns.ID,
		title:    buncolgen.TractorColumns.Code.Expr("COALESCE({}, '')"),
		subtitle: buncolgen.TractorColumns.LicensePlateNumber.Expr("COALESCE({}, '')"),
	},
	permission.ResourceTrailer.String(): {
		model:    func() any { return (*trailer.Trailer)(nil) },
		tenant:   buncolgen.TrailerApplyTenant,
		id:       buncolgen.TrailerColumns.ID,
		title:    buncolgen.TrailerColumns.Code.Expr("COALESCE({}, '')"),
		subtitle: buncolgen.TrailerColumns.LicensePlateNumber.Expr("COALESCE({}, '')"),
	},
	permission.ResourceCustomer.String(): {
		model:    func() any { return (*customer.Customer)(nil) },
		tenant:   buncolgen.CustomerApplyTenant,
		id:       buncolgen.CustomerColumns.ID,
		title:    buncolgen.CustomerColumns.Name.Expr("COALESCE({}, '')"),
		subtitle: buncolgen.CustomerColumns.Code.Expr("COALESCE({}, '')"),
	},
	permission.ResourceCarrier.String(): {
		model:    func() any { return (*carrier.Carrier)(nil) },
		tenant:   buncolgen.CarrierApplyTenant,
		id:       buncolgen.CarrierColumns.ID,
		title:    buncolgen.CarrierColumns.Name.Expr("COALESCE({}, '')"),
		subtitle: buncolgen.CarrierColumns.Code.Expr("COALESCE({}, '')"),
	},
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

func NewRecordFinder(p Params) repositories.CaptureRecordFinder {
	return &recordFinder{db: p.DB}
}

func (f *recordFinder) Exists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	id pulid.ID,
) (bool, error) {
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
}

func (f *recordFinder) Labels(
	ctx context.Context,
	req *repositories.ListCaptureRecordLabelsRequest,
) ([]*repositories.CaptureRecordLabel, error) {
	kind, err := lookupKind(req.ResourceType)
	if err != nil {
		return nil, err
	}
	if len(req.IDs) == 0 {
		return []*repositories.CaptureRecordLabel{}, nil
	}

	labels := make([]*repositories.CaptureRecordLabel, 0, len(req.IDs))
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
}
