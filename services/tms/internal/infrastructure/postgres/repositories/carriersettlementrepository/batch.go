package carriersettlementrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
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

func NewBatch(p Params) repositories.CarrierSettlementBatchRepository {
	return &batchRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-settlement-batch-repository"),
	}
}

func (r *batchRepository) List(
	ctx context.Context,
	req *repositories.ListCarrierSettlementBatchesRequest,
) (*pagination.ListResult[*carriersettlement.CarrierSettlementBatch], error) {
	cols := buncolgen.CarrierSettlementBatchColumns
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*carriersettlement.CarrierSettlementBatch, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Apply(buncolgen.CarrierSettlementBatchApplyTenant(req.Filter.TenantInfo)).
		Order(cols.PeriodEnd.OrderDesc()).
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where(cols.Name.ILike(), "%"+req.Filter.Query+"%")
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list carrier settlement batches: %w", err)
	}

	return &pagination.ListResult[*carriersettlement.CarrierSettlementBatch]{
		Items: items,
		Total: total,
	}, nil
}

func (r *batchRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListCarrierSettlementBatchConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CarrierSettlementBatch], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*carriersettlement.CarrierSettlementBatch)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.CarrierSettlementBatchTable.Alias,
				req.Filter,
				(*carriersettlement.CarrierSettlementBatch)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count carrier settlement batches", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*carriersettlement.CarrierSettlementBatch]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*carriersettlement.CarrierSettlementBatch) *bun.SelectQuery {
				return dba.NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.CarrierSettlementBatchTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.CarrierSettlementBatchTable.Alias,
					req.Filter,
					req.Cursor,
					(*carriersettlement.CarrierSettlementBatch)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan carrier settlement batches", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *batchRepository) GetByID(
	ctx context.Context,
	req repositories.GetCarrierSettlementBatchByIDRequest,
) (*carriersettlement.CarrierSettlementBatch, error) {
	cols := buncolgen.CarrierSettlementBatchColumns
	entity := new(carriersettlement.CarrierSettlementBatch)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierSettlementBatchScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		})
	if req.IncludeSettlements {
		query = query.Relation(
			buncolgen.CarrierSettlementBatchRelations.Settlements,
			func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Order(buncolgen.CarrierSettlementColumns.SettlementNumber.OrderAsc())
			},
		).Relation(buncolgen.Rel(
			buncolgen.CarrierSettlementBatchRelations.Settlements,
			buncolgen.CarrierSettlementRelations.Carrier,
		))
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "CarrierSettlementBatch")
	}
	return entity, nil
}

func (r *batchRepository) GetForPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	periodStart, periodEnd int64,
) (*carriersettlement.CarrierSettlementBatch, error) {
	cols := buncolgen.CarrierSettlementBatchColumns
	entity := new(carriersettlement.CarrierSettlementBatch)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierSettlementBatchScopeTenant(sq, tenantInfo).
				Where(cols.PeriodStart.Eq(), periodStart).
				Where(cols.PeriodEnd.Eq(), periodEnd).
				Where(cols.Status.NotEq(), carriersettlement.BatchStatusCanceled)
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // nil batch means no batch exists for the period
		}
		return nil, fmt.Errorf("get carrier settlement batch for period: %w", err)
	}
	return entity, nil
}

func (r *batchRepository) RecalculateAggregates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID pulid.ID,
) (*carriersettlement.CarrierSettlementBatch, error) {
	if _, err := r.db.DBForContext(ctx).NewRaw(`
		UPDATE carrier_settlement_batches AS b SET
			settlement_count = agg.settlement_count,
			total_gross_minor = agg.total_gross_minor,
			total_net_minor = agg.total_net_minor,
			updated_at = `+r.db.NowEpoch()+`
		FROM (
			SELECT
				COUNT(*) FILTER (WHERE s.status != 'Voided') AS settlement_count,
				COALESCE(SUM(s.gross_cost_minor) FILTER (WHERE s.status != 'Voided'), 0) AS total_gross_minor,
				COALESCE(SUM(s.net_payable_minor) FILTER (WHERE s.status != 'Voided'), 0) AS total_net_minor
			FROM carrier_settlements s
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
		return nil, fmt.Errorf("recalculate carrier settlement batch aggregates: %w", err)
	}

	return r.GetByID(ctx, repositories.GetCarrierSettlementBatchByIDRequest{
		ID:         batchID,
		TenantInfo: tenantInfo,
	})
}

func (r *batchRepository) Create(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlementBatch,
) (*carriersettlement.CarrierSettlementBatch, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("carstlb_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create carrier settlement batch: %w", err)
	}
	return entity, nil
}

func (r *batchRepository) Update(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlementBatch,
) (*carriersettlement.CarrierSettlementBatch, error) {
	cols := buncolgen.CarrierSettlementBatchColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierSettlementBatchScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), entity.Version)
		}).
		Set(cols.Status.Set(), entity.Status).
		Set(cols.Name.Set(), entity.Name).
		Set(cols.SettlementCount.Set(), entity.SettlementCount).
		Set(cols.TotalGrossMinor.Set(), entity.TotalGrossMinor).
		Set(cols.TotalNetMinor.Set(), entity.TotalNetMinor).
		Set(cols.Notes.Set(), entity.Notes).
		Set(cols.GeneratedByID.Set(), entity.GeneratedByID).
		Set(cols.GeneratedAt.Set(), entity.GeneratedAt).
		Set(cols.CompletedAt.Set(), entity.CompletedAt).
		Set(cols.CanceledByID.Set(), entity.CanceledByID).
		Set(cols.CanceledAt.Set(), entity.CanceledAt).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update carrier settlement batch: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		res,
		"CarrierSettlementBatch",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}
	return entity, nil
}
