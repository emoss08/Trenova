package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/capturecol"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type pageRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPageRepository(p Params) captureports.CapturePageRepository {
	return &pageRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-page-repository"),
	}
}

// Insert relies on the (batch, sequence) unique index rather than a read
// first: two retries of the same page racing each other both reach the index,
// and exactly one of them wins.
func (r *pageRepository) Insert(
	ctx context.Context,
	entity *capture.CapturePage,
) (*capture.CapturePage, bool, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*capture.CapturePage, bool, error) {
		result, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On("CONFLICT (batch_id, business_unit_id, organization_id, sequence) DO NOTHING").
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, false, err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return nil, false, err
		}
		if affected > 0 {
			return entity, true, nil
		}

		existing, err := r.GetBySequence(ctx, captureports.GetCapturePageBySequenceRequest{
			BatchID: entity.BatchID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			Sequence: entity.Sequence,
		})
		if err != nil {
			return nil, false, err
		}

		return existing, false, nil
	})
}

func (r *pageRepository) Update(
	ctx context.Context,
	entity *capture.CapturePage,
) (*capture.CapturePage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CapturePage, error) {
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, err
		}
		if err = dberror.CheckRowsAffected(
			results,
			"Capture page",
			entity.ID.String(),
		); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *pageRepository) RecordInspection(
	ctx context.Context,
	entity *capture.CapturePage,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		cols := capturecol.CapturePageColumns
		ti := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
		markers, err := sonic.Marshal(entity.Markers)
		if err != nil {
			return false, err
		}

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*capture.CapturePage)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return capturecol.CapturePageScopeTenantUpdate(uq, ti).
					Where(cols.ID.Eq(), entity.ID).
					Where(cols.Status.Eq(), capture.PageReceived)
			}).
			Set(cols.Status.Set(), entity.Status).
			Set(cols.WidthPx.Set(), entity.WidthPx).
			Set(cols.HeightPx.Set(), entity.HeightPx).
			Set(cols.ThumbnailPath.Set(), bun.NullZero(entity.ThumbnailPath)).
			Set(cols.BlankScore.Set(), entity.BlankScore).
			Set(cols.Markers.String()+" = ?::jsonb", string(markers)).
			Set(cols.FailureMessage.Set(), bun.NullZero(entity.FailureMessage)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(ctx)
		if err != nil {
			return false, err
		}

		affected, err := results.RowsAffected()
		if err != nil {
			return false, err
		}

		return affected == 1, nil
	})
}

func (r *pageRepository) GetByID(
	ctx context.Context,
	req captureports.GetCapturePageByIDRequest,
) (*capture.CapturePage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CapturePage, error) {
		entity := new(capture.CapturePage)

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Apply(capturecol.CapturePageApplyTenant(req.TenantInfo)).
			Where(capturecol.CapturePageColumns.ID.Eq(), req.ID).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Capture page")
		}

		return entity, nil
	})
}

func (r *pageRepository) GetBySequence(
	ctx context.Context,
	req captureports.GetCapturePageBySequenceRequest,
) (*capture.CapturePage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CapturePage, error) {
		entity := new(capture.CapturePage)
		cols := capturecol.CapturePageColumns

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Apply(capturecol.CapturePageApplyTenant(req.TenantInfo)).
			Where(cols.BatchID.Eq(), req.BatchID).
			Where(cols.Sequence.Eq(), req.Sequence).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Capture page")
		}

		return entity, nil
	})
}

func (r *pageRepository) ListByBatch(
	ctx context.Context,
	req captureports.ListCapturePagesRequest,
) ([]*capture.CapturePage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*capture.CapturePage, error) {
		entities := make([]*capture.CapturePage, 0)
		cols := capturecol.CapturePageColumns

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Apply(capturecol.CapturePageApplyTenant(req.TenantInfo)).
			Where(cols.BatchID.Eq(), req.BatchID).
			Order(cols.Sequence.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, err
		}

		return entities, nil
	})
}
