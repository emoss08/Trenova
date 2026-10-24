package driversettlementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const recalculateBatchAggregatesSQL = `UPDATE driver_settlement_batches AS b SET
	settlement_count = agg.settlement_count,
	exception_count = agg.exception_count,
	total_gross_minor = agg.total_gross_minor,
	total_net_minor = agg.total_net_minor,
	updated_at = extract(epoch from current_timestamp)::bigint
FROM (
	SELECT
		COUNT(*) FILTER (WHERE s.status != 'Voided') AS settlement_count,
		COUNT(*) FILTER (WHERE s.has_exceptions AND s.status != 'Voided') AS exception_count,
		COALESCE(SUM(s.gross_earnings_minor) FILTER (WHERE s.status != 'Voided'), 0) AS total_gross_minor,
		COALESCE(SUM(s.net_pay_minor) FILTER (WHERE s.status != 'Voided'), 0) AS total_net_minor
	FROM driver_settlements s
	WHERE s.batch_id = $1
		AND s.organization_id = $2
		AND s.business_unit_id = $3
) AS agg
WHERE b.id = $1
	AND b.organization_id = $2
	AND b.business_unit_id = $3`

var batchErrors = dberror.Mapping{Entity: "SettlementBatch"}

type batchRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewBatch(p Params) driversettlementports.SettlementBatchRepository {
	return &batchRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.settlement-batch-repository"),
	}
}

func (r *batchRepository) List(
	ctx context.Context,
	req *driversettlementports.ListSettlementBatchesRequest,
) (*pagination.ListResult[*driversettlement.SettlementBatch], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*driversettlement.SettlementBatch], error) {
			b := driversettlementdb.SettlementBatches
			q := b.Where(
				when(req.Filter.Query != "", b.Name.ContainsFold(req.Filter.Query)),
				when(req.Status != "", b.Status.Eq(req.Status)),
			).Order(b.PeriodEnd.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list settlement batches: %w", err)
			}

			return result, nil
		},
	)
}

func (r *batchRepository) ListConnection(
	ctx context.Context,
	req *driversettlementports.ListSettlementBatchConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.SettlementBatch], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driversettlement.SettlementBatch], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				driversettlementdb.SettlementBatches.Query(),
				&listquery.Request[driversettlement.SettlementBatch]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list settlement batches", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *batchRepository) GetByID(
	ctx context.Context,
	req driversettlementports.GetSettlementBatchByIDRequest,
) (*driversettlement.SettlementBatch, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*driversettlement.SettlementBatch, error) {
			b := driversettlementdb.SettlementBatches
			q := b.Where(b.ID.Eq(req.ID))
			if req.IncludeSettlements {
				s := b.Settlements()
				q = q.Load(s.Order(s.SettlementNumber.Asc()).Load(s.Worker()))
			}

			entity, err := q.One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, batchErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *batchRepository) GetForPeriod(
	ctx context.Context,
	_ pagination.TenantInfo,
	periodStart, periodEnd int64,
) (*driversettlement.SettlementBatch, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*driversettlement.SettlementBatch, error) {
			b := driversettlementdb.SettlementBatches
			entity, err := b.Where(
				b.PeriodStart.Eq(periodStart),
				b.PeriodEnd.Eq(periodEnd),
				b.Status.Ne(driversettlement.BatchStatusCanceled),
			).Order(b.CreatedAt.Desc()).First(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // nil batch means no batch exists for the period
			}
			if err != nil {
				return nil, fmt.Errorf("get settlement batch for period: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *batchRepository) RecalculateAggregates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID pulid.ID,
) (*driversettlement.SettlementBatch, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*driversettlement.SettlementBatch, error) {
			db := r.db.Keel(ctx)
			if _, err := keel.Exec(
				ctx,
				db,
				recalculateBatchAggregatesSQL,
				batchID.String(),
				tenantInfo.OrgID.String(),
				tenantInfo.BuID.String(),
			); err != nil {
				return nil, fmt.Errorf("recalculate settlement batch aggregates: %w", err)
			}
			keel.MarkChanged(ctx, db, driversettlementdb.SettlementBatchesSchema().Name)

			return r.GetByID(ctx, driversettlementports.GetSettlementBatchByIDRequest{
				ID:         batchID,
				TenantInfo: tenantInfo,
			})
		},
	)
}

func (r *batchRepository) Create(
	ctx context.Context,
	entity *driversettlement.SettlementBatch,
) (*driversettlement.SettlementBatch, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*driversettlement.SettlementBatch, error) {
			err := driversettlementdb.SettlementBatches.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, batchErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *batchRepository) Update(
	ctx context.Context,
	entity *driversettlement.SettlementBatch,
) (*driversettlement.SettlementBatch, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*driversettlement.SettlementBatch, error) {
			b := driversettlementdb.SettlementBatches
			if err := b.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
				b.Status,
				b.Name,
				b.SettlementCount,
				b.ExceptionCount,
				b.TotalGrossMinor,
				b.TotalNetMinor,
				b.Notes,
				b.GeneratedByID,
				b.GeneratedAt,
				b.CompletedAt,
				b.CanceledByID,
				b.CanceledAt,
			); err != nil {
				return nil, batchErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
