package driversettlementrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type batchRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewBatch(p Params) repositories.SettlementBatchRepository {
	return &batchRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.settlement-batch-repository"),
	}
}

func (r *batchRepository) List(
	ctx context.Context,
	req *repositories.ListSettlementBatchesRequest,
) (*pagination.ListResult[*driversettlement.SettlementBatch], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*driversettlement.SettlementBatch, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dstlb.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("dstlb.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Order("dstlb.period_end DESC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where("dstlb.name ILIKE ?", "%"+req.Filter.Query+"%")
	}
	if req.Status != "" {
		query = query.Where("dstlb.status = ?", req.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list settlement batches: %w", err)
	}

	return &pagination.ListResult[*driversettlement.SettlementBatch]{
		Items: items,
		Total: total,
	}, nil
}

func (r *batchRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListSettlementBatchConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.SettlementBatch], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driversettlement.SettlementBatch)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"dstlb",
				req.Filter,
				(*driversettlement.SettlementBatch)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count settlement batches", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*driversettlement.SettlementBatch]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*driversettlement.SettlementBatch) *bun.SelectQuery {
				return dba.NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.SettlementBatchTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					"dstlb",
					req.Filter,
					req.Cursor,
					(*driversettlement.SettlementBatch)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan settlement batches", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *batchRepository) GetByID(
	ctx context.Context,
	req repositories.GetSettlementBatchByIDRequest,
) (*driversettlement.SettlementBatch, error) {
	entity := new(driversettlement.SettlementBatch)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dstlb.id = ?", req.ID).
		Where("dstlb.organization_id = ?", req.TenantInfo.OrgID).
		Where("dstlb.business_unit_id = ?", req.TenantInfo.BuID)
	if req.IncludeSettlements {
		query = query.Relation("Settlements", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("dstl.settlement_number ASC")
		}).Relation("Settlements.Worker", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q
		})
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "SettlementBatch")
	}
	return entity, nil
}

func (r *batchRepository) GetForPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	periodStart, periodEnd int64,
) (*driversettlement.SettlementBatch, error) {
	entity := new(driversettlement.SettlementBatch)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dstlb.organization_id = ?", tenantInfo.OrgID).
		Where("dstlb.business_unit_id = ?", tenantInfo.BuID).
		Where("dstlb.period_start = ?", periodStart).
		Where("dstlb.period_end = ?", periodEnd).
		Where("dstlb.status != ?", driversettlement.BatchStatusCanceled).
		Order("dstlb.created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // nil batch means no batch exists for the period
		}
		return nil, fmt.Errorf("get settlement batch for period: %w", err)
	}
	return entity, nil
}

func (r *batchRepository) RecalculateAggregates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID pulid.ID,
) (*driversettlement.SettlementBatch, error) {
	if _, err := r.db.DBForContext(ctx).NewRaw(`
		UPDATE driver_settlement_batches AS b SET
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
			WHERE s.batch_id = ?
				AND s.organization_id = ?
				AND s.business_unit_id = ?
		) AS agg
		WHERE b.id = ?
			AND b.organization_id = ?
			AND b.business_unit_id = ?
	`,
		batchID, tenantInfo.OrgID, tenantInfo.BuID,
		batchID, tenantInfo.OrgID, tenantInfo.BuID,
	).Exec(ctx); err != nil {
		return nil, fmt.Errorf("recalculate settlement batch aggregates: %w", err)
	}

	return r.GetByID(ctx, repositories.GetSettlementBatchByIDRequest{
		ID:         batchID,
		TenantInfo: tenantInfo,
	})
}

func (r *batchRepository) Create(
	ctx context.Context,
	entity *driversettlement.SettlementBatch,
) (*driversettlement.SettlementBatch, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("dstlb_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create settlement batch: %w", err)
	}
	return entity, nil
}

func (r *batchRepository) Update(
	ctx context.Context,
	entity *driversettlement.SettlementBatch,
) (*driversettlement.SettlementBatch, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("name = ?", entity.Name).
		Set("settlement_count = ?", entity.SettlementCount).
		Set("exception_count = ?", entity.ExceptionCount).
		Set("total_gross_minor = ?", entity.TotalGrossMinor).
		Set("total_net_minor = ?", entity.TotalNetMinor).
		Set("notes = ?", entity.Notes).
		Set("generated_by_id = ?", entity.GeneratedByID).
		Set("generated_at = ?", entity.GeneratedAt).
		Set("completed_at = ?", entity.CompletedAt).
		Set("canceled_by_id = ?", entity.CanceledByID).
		Set("canceled_at = ?", entity.CanceledAt).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update settlement batch: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "SettlementBatch", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}
