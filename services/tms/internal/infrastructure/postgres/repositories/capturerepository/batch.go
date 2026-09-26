//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const maxMaintenanceBatch = 500

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
				return narrowBatches(querybuilder.ApplyFiltersWithoutSort(
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

			return narrowBatches(sq, req), nil
		},
	})
}

// narrowBatches applies the queue's own filters on top of the generic ones.
func narrowBatches(
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
	if req.TargetType != "" && req.TargetID.IsNotNil() {
		q = q.Where(cols.TargetType.Eq(), req.TargetType).Where(cols.TargetID.Eq(), req.TargetID)
	}

	return q
}

func (r *batchRepository) ListStale(
	ctx context.Context,
	req repositories.ListStaleCaptureBatchesRequest,
) ([]*capture.CaptureBatch, error) {
	limit := boundedLimit(req.Limit)
	entities := make([]*capture.CaptureBatch, 0, limit)
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
	entities := make([]*capture.CaptureBatch, 0, limit)
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
