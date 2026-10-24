//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package capturerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const maxMaintenanceBatch = 500

// pageCodeMatch is true when any code on a page, the scanner's or the
// server's, matches the pattern.
const pageCodeMatch = "EXISTS (SELECT 1 FROM jsonb_array_elements_text(" +
	"COALESCE(?->'deviceBarcodes', '[]'::jsonb) || COALESCE(?->'readCodes', '[]'::jsonb)" +
	") AS code WHERE code ILIKE ?)"

var batchErrors = dberror.Mapping{Entity: "Capture batch"}

type batchRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewBatchRepository(p Params) captureports.CaptureBatchRepository {
	return &batchRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-batch-repository"),
	}
}

// inFlightBatchStatuses are never swept by retention: their pages are still
// arriving or being read, and the sweep that notices a lost run handles them.
var inFlightBatchStatuses = []capture.BatchStatus{
	capture.BatchReceiving,
	capture.BatchSealed,
	capture.BatchProcessing,
}

// awaitingPersonStatuses are the batches a person still has something to file
// in, the only ones worth a reminder. The reminder index names the same two.
var awaitingPersonStatuses = []capture.BatchStatus{
	capture.BatchReady,
	capture.BatchPartiallyFiled,
}

func (r *batchRepository) Create(
	ctx context.Context,
	entity *capture.CaptureBatch,
) (*capture.CaptureBatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CaptureBatch, error) {
		if err := capturedb.CaptureBatches.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

// Update writes everything but the received page count, which the upload path
// increments on its own; the stored count is read back with the new version.
func (r *batchRepository) Update(
	ctx context.Context,
	entity *capture.CaptureBatch,
) (*capture.CaptureBatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CaptureBatch, error) {
		b := capturedb.CaptureBatches
		db := r.db.Keel(ctx)
		if err := b.SaveExcept(ctx, db, entity, b.ReceivedPageCount); err != nil {
			return nil, batchErrors.Map(err, entity.ID.String())
		}

		stored, err := b.Pick(b.ReceivedPageCount).Where(b.ID.Eq(entity.ID)).One(ctx, db)
		if err != nil {
			return nil, batchErrors.Map(err, entity.ID.String())
		}
		entity.ReceivedPageCount = stored.ReceivedPageCount

		return entity, nil
	})
}

func (r *batchRepository) GetByID(
	ctx context.Context,
	req *captureports.GetCaptureBatchByIDRequest,
) (*capture.CaptureBatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CaptureBatch, error) {
		b := capturedb.CaptureBatches
		pages := b.Pages()
		var loads []keel.Loader
		if req.IncludePages {
			loads = append(loads, pages.Order(pages.Sequence.Asc()))
		}
		if req.IncludeItems {
			loads = append(loads, orderedItems(b))
		}
		if req.IncludeDevice {
			loads = append(loads, b.CaptureDevice())
		}

		entity, err := b.Where(b.ID.Eq(req.ID)).Load(loads...).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, batchErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func orderedItems(b *capturedb.CaptureBatchesTable) keel.Loader {
	items := b.Items()

	return items.Order(items.Position.Asc())
}

func (r *batchRepository) GetByClientKey(
	ctx context.Context,
	req captureports.GetCaptureBatchByClientKeyRequest,
) (*capture.CaptureBatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CaptureBatch, error) {
		b := capturedb.CaptureBatches
		entity, err := b.Where(b.DeviceID.Eq(req.DeviceID), b.ClientKey.Eq(req.ClientKey)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, batchErrors.Map(err, req.ClientKey)
		}

		return entity, nil
	})
}

// ListCursor is the intake queue, newest first by default. The count runs
// only when the caller asked for it.
func (r *batchRepository) ListCursor(
	ctx context.Context,
	req *captureports.ListCaptureBatchesRequest,
) (*pagination.CursorListResult[*capture.CaptureBatch], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*capture.CaptureBatch], error) {
			b := capturedb.CaptureBatches

			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				b.Where(narrowBatches(req)...),
				&listquery.Request[capture.CaptureBatch]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[capture.CaptureBatch]) *keel.Query[capture.CaptureBatch] {
						q = q.Load(b.CaptureDevice())
						if req.IncludeItems {
							q = q.Load(orderedItems(b))
						}

						return q
					},
				},
			)
		},
	)
}

// narrowBatches is the queue's own filters on top of the generic ones.
func narrowBatches(req *captureports.ListCaptureBatchesRequest) []keel.Pred {
	b := capturedb.CaptureBatches

	return []keel.Pred{
		keel.When(len(req.Statuses) > 0, b.Status.In(req.Statuses...)),
		keel.When(req.Source != "", b.Source.Eq(req.Source)),
		keel.When(req.UserID.IsNotNil(), b.UserID.Eq(req.UserID)),
		keel.When(
			req.TargetType != "" && req.TargetID.IsNotNil(),
			forRecord(req.TargetType, req.TargetID),
		),
		keel.When(req.CreatedFrom > 0, b.CreatedAt.Gte(req.CreatedFrom)),
		keel.When(req.CreatedTo > 0, b.CreatedAt.Lte(req.CreatedTo)),
		keel.When(req.Search != "", searchBatches(req.Search)),
	}
}

// forRecord keeps the stacks for one record: scanned into it, or with a
// document filed or suggested onto it.
func forRecord(resourceType string, resourceID pulid.ID) keel.Pred {
	b := capturedb.CaptureBatches
	item := capturedb.CaptureItems

	return keel.Or(
		keel.And(b.TargetType.Eq(resourceType), b.TargetID.Eq(resourceID)),
		keel.Exists(item.Pick(item.ID).Where(
			item.BatchID.EqCol(b.ID),
			item.Status.Ne(capture.ItemDiscarded),
			keel.Or(
				keel.And(item.FiledType.Eq(resourceType), item.FiledID.Eq(resourceID)),
				keel.And(item.SuggestedType.Eq(resourceType), item.SuggestedID.Eq(resourceID)),
			),
		)),
	)
}

// searchBatches keeps the stacks that match a term anywhere a person would
// look for one: its scanner or print job, the computer that sent it, whose it
// is, a code on one of its pages, or a shipment it is for or went onto.
func searchBatches(term string) keel.Pred {
	pattern := "%" + stringutils.EscapeLikePattern(term) + "%"
	b := capturedb.CaptureBatches
	device := capturedb.CaptureDevices
	user := tenantdb.Users
	page := capturedb.CapturePages
	item := capturedb.CaptureItems
	sp := shipmentdb.Shipments
	shipmentType := permission.ResourceShipment.String()

	routed := keel.UnionAll(
		item.Pick(item.FiledID).Where(item.BatchID.EqCol(b.ID), item.FiledType.Eq(shipmentType)),
		item.Pick(item.SuggestedID).
			Where(item.BatchID.EqCol(b.ID), item.SuggestedType.Eq(shipmentType)),
	)

	return keel.Or(
		b.JobName.ContainsFold(term),
		b.SourceName.ContainsFold(term),
		keel.Exists(device.Pick(device.ID).Where(
			device.ID.EqCol(b.DeviceID),
			keel.Or(device.Name.ContainsFold(term), device.MachineName.ContainsFold(term)),
		)),
		keel.Exists(user.Pick(user.ID).Where(
			user.ID.EqCol(b.UserID),
			user.Name.ContainsFold(term),
		)),
		keel.Exists(page.Pick(page.ID).Where(
			page.BatchID.EqCol(b.ID),
			keel.Raw(pageCodeMatch, page.Markers, page.Markers, pattern),
		)),
		keel.Exists(sp.Pick(sp.ID).Where(
			keel.Or(sp.ProNumber.ContainsFold(term), sp.BOL.ContainsFold(term)),
			keel.Or(
				keel.And(b.TargetType.Eq(shipmentType), sp.ID.EqCol(b.TargetID)),
				sp.ID.InQuery(routed),
			),
		)),
	)
}

func (r *batchRepository) ListStale(
	ctx context.Context,
	req captureports.ListStaleCaptureBatchesRequest,
) ([]*capture.CaptureBatch, error) {
	ctx = dbscope.WithSystem(ctx, "list stale capture batches across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*capture.CaptureBatch, error) {
		b := capturedb.CaptureBatches
		return orEmptyBatches(b.AllTenants().
			Where(b.Status.In(req.Statuses...), b.UpdatedAt.Lt(req.UpdatedBefore)).
			Order(b.UpdatedAt.Asc()).
			Limit(boundedLimit(req.Limit)).
			List(ctx, r.db.Keel(ctx)))
	})
}

func (r *batchRepository) ListRetentionDue(
	ctx context.Context,
	req captureports.ListRetentionDueCaptureBatchesRequest,
) ([]*capture.CaptureBatch, error) {
	ctx = dbscope.WithSystem(ctx, "list capture batches past retention across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*capture.CaptureBatch, error) {
		b := capturedb.CaptureBatches
		return orEmptyBatches(b.AllTenants().
			Where(b.Status.NotIn(inFlightBatchStatuses...), b.RetainUntil.Lte(req.Now)).
			Order(b.RetainUntil.Asc()).
			Limit(boundedLimit(req.Limit)).
			List(ctx, r.db.Keel(ctx)))
	})
}

func (r *batchRepository) ListRetentionReminders(
	ctx context.Context,
	req captureports.ListRetentionReminderCaptureBatchesRequest,
) ([]*capture.CaptureBatch, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list capture batches due a retention reminder across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*capture.CaptureBatch, error) {
		b := capturedb.CaptureBatches
		return orEmptyBatches(b.AllTenants().
			Where(
				b.Status.In(awaitingPersonStatuses...),
				b.RetentionRemindedAt.IsNull(),
				b.RetainUntil.Gt(req.From),
				b.RetainUntil.Lte(req.Until),
			).
			Order(b.RetainUntil.Asc()).
			Limit(boundedLimit(req.Limit)).
			List(ctx, r.db.Keel(ctx)))
	})
}

func (r *batchRepository) ClaimRetentionReminder(
	ctx context.Context,
	req captureports.ClaimRetentionReminderRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		b := capturedb.CaptureBatches
		n, err := b.Where(b.ID.Eq(req.ID), b.RetentionRemindedAt.IsNull()).Update(
			ctx,
			r.db.Keel(ctx),
			b.RetentionRemindedAt.To(req.At),
			b.Version.ToExpr(b.Version),
			b.UpdatedAt.ToExpr(b.UpdatedAt),
		)
		if err != nil {
			return false, err
		}

		return n == 1, nil
	})
}

func (r *batchRepository) IncrementReceived(
	ctx context.Context,
	req captureports.IncrementCaptureBatchPagesRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		b := capturedb.CaptureBatches
		_, err := b.Where(b.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			b.ReceivedPageCount.Add(1),
			b.Version.ToExpr(b.Version),
			b.UpdatedAt.ToExpr(b.UpdatedAt),
		)

		return err
	})
}

func (r *batchRepository) Delete(
	ctx context.Context,
	req captureports.DeleteCaptureBatchRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		b := capturedb.CaptureBatches
		_, err := b.Where(b.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))

		return err
	})
}

func boundedLimit(limit int) int {
	if limit <= 0 || limit > maxMaintenanceBatch {
		return maxMaintenanceBatch
	}

	return limit
}

func orEmptyBatches(
	entities []*capture.CaptureBatch,
	err error,
) ([]*capture.CaptureBatch, error) {
	if err != nil {
		return nil, err
	}
	if entities == nil {
		return []*capture.CaptureBatch{}, nil
	}

	return entities, nil
}
