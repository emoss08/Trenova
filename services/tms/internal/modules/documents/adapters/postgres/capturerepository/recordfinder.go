package capturerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type recordLabel struct {
	ID       pulid.ID
	Title    string
	Subtitle string
}

// recordKind is how one fileable kind is found and named. The title is what
// a person recognises the record by; the subtitle is the second number they
// would quote if the first were ambiguous.
type recordKind struct {
	exists func(ctx context.Context, db *keel.DB, ti pagination.TenantInfo, id pulid.ID) (bool, error)
	labels func(
		ctx context.Context,
		db *keel.DB,
		ti pagination.TenantInfo,
		ids []pulid.ID,
	) ([]recordLabel, error)
}

type recordTable[T any] interface {
	Where(ps ...keel.Pred) *keel.Query[T]
}

type recordShape struct {
	id       keelkinds.PulidIDCol
	title    keel.Expr
	subtitle keel.Expr
	scope    func(ti pagination.TenantInfo) keel.Pred
}

func kindOf[T any](table recordTable[T], shape recordShape) recordKind {
	scope := func(ti pagination.TenantInfo) keel.Pred {
		if shape.scope == nil {
			return nil
		}

		return shape.scope(ti)
	}

	return recordKind{
		exists: func(
			ctx context.Context,
			db *keel.DB,
			ti pagination.TenantInfo,
			id pulid.ID,
		) (bool, error) {
			return table.Where(shape.id.Eq(id), scope(ti)).Exists(ctx, db)
		},
		labels: func(
			ctx context.Context,
			db *keel.DB,
			ti pagination.TenantInfo,
			ids []pulid.ID,
		) ([]recordLabel, error) {
			return keel.Into[recordLabel](ctx, db, table.
				Where(shape.id.In(ids...), scope(ti)).
				Pick(shape.id, shape.title, shape.subtitle).
				Limit(len(ids)))
		},
	}
}

func orBlank(e keel.Expr) keel.Expr {
	return keel.DynOf(e).Coalesce("")
}

var noSubtitle = keel.DynLit("").Cast(keel.TypeString)

var recordKinds = map[string]recordKind{
	permission.ResourceShipment.String(): kindOf(shipmentdb.Shipments, recordShape{
		id:       shipmentdb.Shipments.ID,
		title:    orBlank(shipmentdb.Shipments.ProNumber),
		subtitle: orBlank(shipmentdb.Shipments.BOL),
	}),
	permission.ResourceWorker.String(): kindOf(workerdb.Workers, recordShape{
		id:       workerdb.Workers.ID,
		title:    keel.DynFunc("concat_ws", " ", workerdb.Workers.FirstName, workerdb.Workers.LastName),
		subtitle: noSubtitle,
	}),
	permission.ResourceTractor.String(): kindOf(tractordb.Tractors, recordShape{
		id:       tractordb.Tractors.ID,
		title:    orBlank(tractordb.Tractors.Code),
		subtitle: orBlank(tractordb.Tractors.LicensePlateNumber),
	}),
	permission.ResourceTrailer.String(): kindOf(trailerdb.Trailers, recordShape{
		id:       trailerdb.Trailers.ID,
		title:    orBlank(trailerdb.Trailers.Code),
		subtitle: orBlank(trailerdb.Trailers.LicensePlateNumber),
	}),
	permission.ResourceCustomer.String(): kindOf(customerdb.Customers, recordShape{
		id:       customerdb.Customers.ID,
		title:    orBlank(customerdb.Customers.Name),
		subtitle: orBlank(customerdb.Customers.Code),
	}),
	capture.ResourceAssistantThread: kindOf(conversationdb.Threads, recordShape{
		id:       conversationdb.Threads.ID,
		title:    orBlank(conversationdb.Threads.Title),
		subtitle: noSubtitle,
		scope: func(ti pagination.TenantInfo) keel.Pred {
			return conversationdb.Threads.UserID.Eq(ti.UserID)
		},
	}),
	permission.ResourceCarrier.String(): kindOf(carrierdb.Carriers, recordShape{
		id:       carrierdb.Carriers.ID,
		title:    orBlank(carrierdb.Carriers.Name),
		subtitle: orBlank(carrierdb.Carriers.Code),
	}),
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

		return kind.exists(ctx, f.db.Keel(ctx), tenantInfo, id)
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

			rows, err := kind.labels(ctx, f.db.Keel(ctx), req.TenantInfo, req.IDs)
			if err != nil {
				return nil, err
			}

			labels := make([]*captureports.CaptureRecordLabel, 0, len(rows))
			for _, row := range rows {
				labels = append(labels, &captureports.CaptureRecordLabel{
					ResourceType: req.ResourceType,
					ID:           row.ID,
					Title:        row.Title,
					Subtitle:     row.Subtitle,
				})
			}

			return labels, nil
		},
	)
}
