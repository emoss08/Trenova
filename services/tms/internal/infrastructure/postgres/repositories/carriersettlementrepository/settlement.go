package carriersettlementrepository

import (
	"context"
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
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type settlementRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSettlement(p Params) repositories.CarrierSettlementRepository {
	return &settlementRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-settlement-repository"),
	}
}

func (r *settlementRepository) List(
	ctx context.Context,
	req *repositories.ListCarrierSettlementsRequest,
) (*pagination.ListResult[*carriersettlement.CarrierSettlement], error) {
	cols := buncolgen.CarrierSettlementColumns
	rel := buncolgen.CarrierSettlementRelations
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*carriersettlement.CarrierSettlement, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Apply(buncolgen.CarrierSettlementApplyTenant(req.Filter.TenantInfo)).
		Relation(rel.Carrier).
		Order(cols.PeriodEnd.OrderDesc()).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where(cols.SettlementNumber.ILike(), "%"+req.Filter.Query+"%")
	}
	if !req.CarrierID.IsNil() {
		query = query.Where(cols.CarrierID.Eq(), req.CarrierID)
	}
	if !req.BatchID.IsNil() {
		query = query.Where(cols.BatchID.Eq(), req.BatchID)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if len(req.Statuses) > 0 {
		query = query.Where(cols.Status.In(), bun.List(req.Statuses))
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list carrier settlements: %w", err)
	}

	return &pagination.ListResult[*carriersettlement.CarrierSettlement]{
		Items: items,
		Total: total,
	}, nil
}

func (r *settlementRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListCarrierSettlementConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CarrierSettlement], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*carriersettlement.CarrierSettlement)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.CarrierSettlementTable.Alias,
				req.Filter,
				(*carriersettlement.CarrierSettlement)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count carrier settlements", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*carriersettlement.CarrierSettlement]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*carriersettlement.CarrierSettlement) *bun.SelectQuery {
				return dba.NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.CarrierSettlementTable.All()).
					Relation(buncolgen.CarrierSettlementRelations.Carrier)
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.CarrierSettlementTable.Alias,
					req.Filter,
					req.Cursor,
					(*carriersettlement.CarrierSettlement)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan carrier settlements", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *settlementRepository) GetByID(
	ctx context.Context,
	req repositories.GetCarrierSettlementByIDRequest,
) (*carriersettlement.CarrierSettlement, error) {
	cols := buncolgen.CarrierSettlementColumns
	rel := buncolgen.CarrierSettlementRelations
	entity := new(carriersettlement.CarrierSettlement)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierSettlementScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Relation(rel.Carrier)
	if req.IncludeLines {
		query = query.Relation(rel.Lines, func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order(buncolgen.CarrierSettlementLineColumns.LineNumber.OrderAsc())
		})
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "CarrierSettlement")
	}
	return entity, nil
}

func (r *settlementRepository) GetWorkspaceCounts(
	ctx context.Context,
	req *repositories.GetCarrierSettlementWorkspaceCountsRequest,
) (*repositories.CarrierSettlementWorkspaceCounts, error) {
	counts := new(repositories.CarrierSettlementWorkspaceCounts)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carriersettlement.CarrierSettlement)(nil)).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS draft_count",
			carriersettlement.StatusDraft).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS pending_approval_count",
			carriersettlement.StatusPendingApproval).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS approved_count",
			carriersettlement.StatusApproved).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS posted_count",
			carriersettlement.StatusPosted).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS paid_count",
			carriersettlement.StatusPaid).
		ColumnExpr("COALESCE(SUM(net_payable_minor) FILTER (WHERE status != ?), 0) "+
			"AS total_net_minor", carriersettlement.StatusVoided).
		ColumnExpr("COALESCE(SUM(gross_cost_minor) FILTER (WHERE status != ?), 0) "+
			"AS total_gross_minor", carriersettlement.StatusVoided).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Where("period_start = ?", req.PeriodStart).
		Where("period_end = ?", req.PeriodEnd).
		Scan(ctx, counts)
	if err != nil {
		return nil, fmt.Errorf("get carrier settlement workspace counts: %w", err)
	}
	return counts, nil
}

func (r *settlementRepository) ExistsForCarrierPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
	periodStart, periodEnd int64,
) (bool, error) {
	cols := buncolgen.CarrierSettlementColumns
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carriersettlement.CarrierSettlement)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierSettlementScopeTenant(sq, tenantInfo).
				Where(cols.CarrierID.Eq(), carrierID).
				Where(cols.PeriodStart.Eq(), periodStart).
				Where(cols.PeriodEnd.Eq(), periodEnd).
				Where(cols.Status.NotEq(), carriersettlement.StatusVoided)
		}).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check carrier settlement exists for period: %w", err)
	}
	return exists, nil
}

func (r *settlementRepository) Create(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
) (*carriersettlement.CarrierSettlement, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("carstl_")
	}
	assignLineFields(entity)
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create carrier settlement: %w", err)
	}
	if len(entity.Lines) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Lines).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("create carrier settlement lines: %w", err)
		}
	}
	return r.GetByID(ctx, repositories.GetCarrierSettlementByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeLines: true,
	})
}

func (r *settlementRepository) Update(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
) (*carriersettlement.CarrierSettlement, error) {
	cols := buncolgen.CarrierSettlementColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierSettlementScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), entity.Version)
		}).
		Set(cols.Status.Set(), entity.Status).
		Set(cols.BatchID.Set(), entity.BatchID).
		Set(cols.PayDate.Set(), entity.PayDate).
		Set(cols.GrossCostMinor.Set(), entity.GrossCostMinor).
		Set(cols.AdjustmentsMinor.Set(), entity.AdjustmentsMinor).
		Set(cols.NetPayableMinor.Set(), entity.NetPayableMinor).
		Set(cols.ShipmentCount.Set(), entity.ShipmentCount).
		Set(cols.CurrencyCode.Set(), entity.CurrencyCode).
		Set(cols.Notes.Set(), entity.Notes).
		Set(cols.SubmittedByID.Set(), entity.SubmittedByID).
		Set(cols.SubmittedAt.Set(), entity.SubmittedAt).
		Set(cols.ApprovedByID.Set(), entity.ApprovedByID).
		Set(cols.ApprovedAt.Set(), entity.ApprovedAt).
		Set(cols.PostedByID.Set(), entity.PostedByID).
		Set(cols.PostedAt.Set(), entity.PostedAt).
		Set(cols.PostedJournalBatchID.Set(), entity.PostedJournalBatchID).
		Set(cols.PaidAt.Set(), entity.PaidAt).
		Set(cols.PaidByID.Set(), entity.PaidByID).
		Set(cols.PaymentMethod.Set(), entity.PaymentMethod).
		Set(cols.PaymentReference.Set(), entity.PaymentReference).
		Set(cols.PaidJournalBatchID.Set(), entity.PaidJournalBatchID).
		Set(cols.VoidedByID.Set(), entity.VoidedByID).
		Set(cols.VoidedAt.Set(), entity.VoidedAt).
		Set(cols.VoidReason.Set(), entity.VoidReason).
		Set(cols.VoidJournalBatchID.Set(), entity.VoidJournalBatchID).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update carrier settlement: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "CarrierSettlement", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetCarrierSettlementByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeLines: true,
	})
}

func (r *settlementRepository) ReplaceLines(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
) error {
	lineCols := buncolgen.CarrierSettlementLineColumns
	assignLineFields(entity)
	if _, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*carriersettlement.CarrierSettlementLine)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.CarrierSettlementLineScopeTenantDelete(dq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(lineCols.SettlementID.Eq(), entity.ID)
		}).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete carrier settlement lines: %w", err)
	}
	if len(entity.Lines) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Lines).
			Exec(ctx); err != nil {
			return fmt.Errorf("insert carrier settlement lines: %w", err)
		}
	}
	return nil
}

func assignLineFields(entity *carriersettlement.CarrierSettlement) {
	for idx, line := range entity.Lines {
		if line == nil {
			continue
		}
		line.ID = pulid.Nil
		line.OrganizationID = entity.OrganizationID
		line.BusinessUnitID = entity.BusinessUnitID
		line.SettlementID = entity.ID
		line.LineNumber = idx + 1
	}
}
