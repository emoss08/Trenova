//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const maxMaintenanceBatch = 500

// pageCodeMatch is true when any code on a page, the scanner's or the
// server's, matches the pattern.
const pageCodeMatch = "EXISTS (SELECT 1 FROM jsonb_array_elements_text(" +
	"COALESCE({}->'deviceBarcodes', '[]'::jsonb) || COALESCE({}->'readCodes', '[]'::jsonb)" +
	") AS code WHERE code ILIKE ?)"

type batchRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewBatchRepository(p Params) repositories.CaptureBatchRepository {
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
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *batchRepository) Update(
	ctx context.Context,
	entity *capture.CaptureBatch,
) (*capture.CaptureBatch, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		ExcludeColumn(buncolgen.CaptureBatchColumns.ReceivedPageCount.String()).
		WherePK().
		Where(buncolgen.CaptureBatchColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov

		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "Capture batch", entity.ID.String()); err != nil {
		entity.Version = ov

		return nil, err
	}

	return entity, nil
}

func (r *batchRepository) GetByID(
	ctx context.Context,
	req *repositories.GetCaptureBatchByIDRequest,
) (*capture.CaptureBatch, error) {
	entity := new(capture.CaptureBatch)
	rel := buncolgen.CaptureBatchRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(buncolgen.CaptureBatchColumns.ID.Eq(), req.ID).
		Apply(buncolgen.CaptureBatchApplyTenant(req.TenantInfo))
	if req.IncludePages {
		query = query.Relation(rel.Pages, func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order(buncolgen.CapturePageColumns.Sequence.OrderAsc())
		})
	}
	if req.IncludeItems {
		query = query.Relation(rel.Items, func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order(buncolgen.CaptureItemColumns.Position.OrderAsc())
		})
	}

	if req.IncludeDevice {
		query = query.Relation(rel.CaptureDevice)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Capture batch")
	}

	return entity, nil
}

func (r *batchRepository) GetByClientKey(
	ctx context.Context,
	req repositories.GetCaptureBatchByClientKeyRequest,
) (*capture.CaptureBatch, error) {
	entity := new(capture.CaptureBatch)
	cols := buncolgen.CaptureBatchColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.CaptureBatchApplyTenant(req.TenantInfo)).
		Where(cols.DeviceID.Eq(), req.DeviceID).
		Where(cols.ClientKey.Eq(), req.ClientKey).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Capture batch")
	}

	return entity, nil
}

// ListCursor is the intake queue, newest first by default. The count runs
// only when the caller asked for it.
func (r *batchRepository) ListCursor(
	ctx context.Context,
	req *repositories.ListCaptureBatchesRequest,
) (*pagination.CursorListResult[*capture.CaptureBatch], error) {
	dba := r.db.DBForContext(ctx)
	alias := buncolgen.CaptureBatchTable.Alias

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*capture.CaptureBatch)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return r.narrowBatches(querybuilder.ApplyFiltersWithoutSort(
					sq, alias, req.Filter, (*capture.CaptureBatch)(nil),
				), req)
			}).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		totalCount = &total
	}

	return dbhelper.CursorList(ctx, dbhelper.CursorListParams[*capture.CaptureBatch]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]*capture.CaptureBatch) *bun.SelectQuery {
			return dba.NewSelect().
				Model(entities).
				Relation(buncolgen.CaptureBatchRelations.CaptureDevice)
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, err := querybuilder.ApplyCursorFilters(
				sq, alias, req.Filter, req.Cursor, (*capture.CaptureBatch)(nil),
			)
			if err != nil {
				return sq, err
			}

			return r.narrowBatches(sq, req), nil
		},
	})
}

// narrowBatches applies the queue's own filters on top of the generic ones.
func (r *batchRepository) narrowBatches(
	q *bun.SelectQuery,
	req *repositories.ListCaptureBatchesRequest,
) *bun.SelectQuery {
	cols := buncolgen.CaptureBatchColumns
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	if req.Source != "" {
		q = q.Where(cols.Source.Eq(), req.Source)
	}
	if req.UserID.IsNotNil() {
		q = q.Where(cols.UserID.Eq(), req.UserID)
	}
	if req.TargetType != "" && req.TargetID.IsNotNil() && req.Filter != nil {
		q = r.forRecord(q, req.Filter.TenantInfo, req.TargetType, req.TargetID)
	}
	if req.CreatedFrom > 0 {
		q = q.Where(cols.CreatedAt.Gte(), req.CreatedFrom)
	}
	if req.CreatedTo > 0 {
		q = q.Where(cols.CreatedAt.Lte(), req.CreatedTo)
	}
	if req.Search != "" && req.Filter != nil {
		q = r.searchBatches(q, req.Filter.TenantInfo, req.Search)
	}

	return q
}

// forRecord keeps the stacks for one record: scanned into it, or with a
// document filed or suggested onto it.
func (r *batchRepository) forRecord(
	q *bun.SelectQuery,
	ti pagination.TenantInfo,
	resourceType string,
	resourceID pulid.ID,
) *bun.SelectQuery {
	batch := buncolgen.CaptureBatchColumns
	item := buncolgen.CaptureItemColumns
	items := buncolgen.CaptureItemScopeTenant(r.db.DB().NewSelect().
		Model((*capture.CaptureItem)(nil)).
		ColumnExpr("1").
		Where(item.BatchID.EqColumn(batch.ID)).
		Where(item.Status.NotEq(), capture.ItemDiscarded).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereGroup(" OR ", func(filed *bun.SelectQuery) *bun.SelectQuery {
				return filed.Where(item.FiledType.Eq(), resourceType).Where(item.FiledID.Eq(), resourceID)
			}).WhereGroup(" OR ", func(suggested *bun.SelectQuery) *bun.SelectQuery {
				return suggested.Where(item.SuggestedType.Eq(), resourceType).
					Where(item.SuggestedID.Eq(), resourceID)
			})
		}), ti)

	return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereGroup(" OR ", func(target *bun.SelectQuery) *bun.SelectQuery {
			return target.Where(batch.TargetType.Eq(), resourceType).Where(batch.TargetID.Eq(), resourceID)
		}).WhereOr("EXISTS (?)", items)
	})
}

// searchBatches keeps the stacks that match a term anywhere a person would
// look for one: its scanner or print job, the computer that sent it, whose it
// is, a code on one of its pages, or a shipment it is for or went onto.
func (r *batchRepository) searchBatches(
	q *bun.SelectQuery,
	ti pagination.TenantInfo,
	term string,
) *bun.SelectQuery {
	pattern := "%" + stringutils.EscapeLikePattern(term) + "%"
	batch := buncolgen.CaptureBatchColumns
	device := buncolgen.CaptureDeviceColumns
	user := buncolgen.UserColumns
	page := buncolgen.CapturePageColumns
	item := buncolgen.CaptureItemColumns
	shipment := buncolgen.ShipmentColumns
	db := r.db.DB()

	devices := buncolgen.CaptureDeviceScopeTenant(db.NewSelect().
		Model((*capture.CaptureDevice)(nil)).
		ColumnExpr("1").
		Where(device.ID.EqColumn(batch.DeviceID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(device.Name.ILike(), pattern).
				WhereOr(device.MachineName.ILike(), pattern)
		}), ti)

	people := db.NewSelect().
		Model((*tenant.User)(nil)).
		ColumnExpr("1").
		Where(user.ID.EqColumn(batch.UserID)).
		Where(user.Name.ILike(), pattern)

	codes := buncolgen.CapturePageScopeTenant(db.NewSelect().
		Model((*capture.CapturePage)(nil)).
		ColumnExpr("1").
		Where(page.BatchID.EqColumn(batch.ID)).
		Where(page.Markers.Expr(pageCodeMatch), pattern), ti)

	routed := buncolgen.CaptureItemScopeTenant(db.NewSelect().
		Model((*capture.CaptureItem)(nil)).
		Column(item.FiledID.String()).
		Where(item.BatchID.EqColumn(batch.ID)).
		Where(item.FiledType.Eq(), permission.ResourceShipment), ti).
		UnionAll(buncolgen.CaptureItemScopeTenant(db.NewSelect().
			Model((*capture.CaptureItem)(nil)).
			Column(item.SuggestedID.String()).
			Where(item.BatchID.EqColumn(batch.ID)).
			Where(item.SuggestedType.Eq(), permission.ResourceShipment), ti))

	shipments := buncolgen.ShipmentScopeTenant(db.NewSelect().
		Model((*shipmentdomain.Shipment)(nil)).
		ColumnExpr("1").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(shipment.ProNumber.ILike(), pattern).
				WhereOr(shipment.BOL.ILike(), pattern)
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereGroup(" OR ", func(target *bun.SelectQuery) *bun.SelectQuery {
				return target.Where(batch.TargetType.Eq(), permission.ResourceShipment).
					Where(shipment.ID.EqColumn(batch.TargetID))
			}).WhereOr(shipment.ID.In(), routed)
		}), ti)

	return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(batch.JobName.ILike(), pattern).
			WhereOr(batch.SourceName.ILike(), pattern).
			WhereOr("EXISTS (?)", devices).
			WhereOr("EXISTS (?)", people).
			WhereOr("EXISTS (?)", codes).
			WhereOr("EXISTS (?)", shipments)
	})
}

func (r *batchRepository) ListStale(
	ctx context.Context,
	req repositories.ListStaleCaptureBatchesRequest,
) ([]*capture.CaptureBatch, error) {
	limit := boundedLimit(req.Limit)
	entities := make([]*capture.CaptureBatch, 0)
	cols := buncolgen.CaptureBatchColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.Status.In(), bun.List(req.Statuses)).
		Where(cols.UpdatedAt.Lt(), req.UpdatedBefore).
		Order(cols.UpdatedAt.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *batchRepository) ListRetentionDue(
	ctx context.Context,
	req repositories.ListRetentionDueCaptureBatchesRequest,
) ([]*capture.CaptureBatch, error) {
	limit := boundedLimit(req.Limit)
	entities := make([]*capture.CaptureBatch, 0)
	cols := buncolgen.CaptureBatchColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.Status.NotIn(), bun.List(inFlightBatchStatuses)).
		Where(cols.RetainUntil.Lte(), req.Now).
		Order(cols.RetainUntil.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *batchRepository) ListRetentionReminders(
	ctx context.Context,
	req repositories.ListRetentionReminderCaptureBatchesRequest,
) ([]*capture.CaptureBatch, error) {
	limit := boundedLimit(req.Limit)
	entities := make([]*capture.CaptureBatch, 0)
	cols := buncolgen.CaptureBatchColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.Status.In(), bun.List(awaitingPersonStatuses)).
		Where(cols.RetentionRemindedAt.IsNull()).
		Where(cols.RetainUntil.Gt(), req.From).
		Where(cols.RetainUntil.Lte(), req.Until).
		Order(cols.RetainUntil.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *batchRepository) ClaimRetentionReminder(
	ctx context.Context,
	req repositories.ClaimRetentionReminderRequest,
) (bool, error) {
	cols := buncolgen.CaptureBatchColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*capture.CaptureBatch)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CaptureBatchScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID).
				Where(cols.RetentionRemindedAt.IsNull())
		}).
		Set(cols.RetentionRemindedAt.Set(), req.At).
		Exec(ctx)
	if err != nil {
		return false, err
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected == 1, nil
}

func (r *batchRepository) IncrementReceived(
	ctx context.Context,
	req repositories.IncrementCaptureBatchPagesRequest,
) error {
	cols := buncolgen.CaptureBatchColumns

	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*capture.CaptureBatch)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CaptureBatchScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.ReceivedPageCount.Inc(1)).
		Exec(ctx)

	return err
}

func (r *batchRepository) Delete(
	ctx context.Context,
	req repositories.DeleteCaptureBatchRequest,
) error {
	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*capture.CaptureBatch)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.CaptureBatchScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.CaptureBatchColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)

	return err
}

func boundedLimit(limit int) int {
	if limit <= 0 || limit > maxMaintenanceBatch {
		return maxMaintenanceBatch
	}

	return limit
}
