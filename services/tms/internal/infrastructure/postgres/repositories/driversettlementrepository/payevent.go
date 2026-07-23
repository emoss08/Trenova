package driversettlementrepository

import (
	"context"
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

type payEventRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPayEvent(p Params) repositories.PayEventRepository {
	return &payEventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pay-event-repository"),
	}
}

func (r *payEventRepository) List(
	ctx context.Context,
	req *repositories.ListPayEventsRequest,
) (*pagination.ListResult[*driversettlement.PayEvent], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*driversettlement.PayEvent, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dpe.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("dpe.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("dpe.event_date DESC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where("dpe.pro_number ILIKE ?", "%"+req.Filter.Query+"%")
	}
	if !req.WorkerID.IsNil() {
		query = query.Where("dpe.worker_id = ?", req.WorkerID)
	}
	if !req.ShipmentID.IsNil() {
		query = query.Where("dpe.shipment_id = ?", req.ShipmentID)
	}
	if req.Status != "" {
		query = query.Where("dpe.status = ?", req.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pay events: %w", err)
	}

	return &pagination.ListResult[*driversettlement.PayEvent]{Items: items, Total: total}, nil
}

func (r *payEventRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListPayEventConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.PayEvent], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driversettlement.PayEvent)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"dpe",
				req.Filter,
				(*driversettlement.PayEvent)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count pay events", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*driversettlement.PayEvent]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*driversettlement.PayEvent) *bun.SelectQuery {
			return dba.NewSelect().
				Model(entities).
				ColumnExpr(buncolgen.PayEventTable.All()).
				Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q })
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				"dpe",
				req.Filter,
				req.Cursor,
				(*driversettlement.PayEvent)(nil),
			)
		},
	})
	if err != nil {
		log.Error("failed to scan pay events", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *payEventRepository) GetByID(
	ctx context.Context,
	req repositories.GetPayEventByIDRequest,
) (*driversettlement.PayEvent, error) {
	entity := new(driversettlement.PayEvent)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dpe.id = ?", req.ID).
		Where("dpe.organization_id = ?", req.TenantInfo.OrgID).
		Where("dpe.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "PayEvent")
	}
	return entity, nil
}

func (r *payEventRepository) GetByIdempotencyKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key string,
) (*driversettlement.PayEvent, error) {
	entity := new(driversettlement.PayEvent)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dpe.organization_id = ?", tenantInfo.OrgID).
		Where("dpe.business_unit_id = ?", tenantInfo.BuID).
		Where("dpe.idempotency_key = ?", key).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "PayEvent")
	}
	return entity, nil
}

func (r *payEventRepository) ListAccruedForWorker(
	ctx context.Context,
	req *repositories.ListAccruedPayEventsRequest,
) ([]*driversettlement.PayEvent, error) {
	items := make([]*driversettlement.PayEvent, 0)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dpe.organization_id = ?", req.TenantInfo.OrgID).
		Where("dpe.business_unit_id = ?", req.TenantInfo.BuID).
		Where("dpe.worker_id = ?", req.WorkerID).
		Where("dpe.status = ?", driversettlement.PayEventStatusAccrued).
		Where("dpe.on_hold = false")
	if len(req.EventIDs) > 0 {
		q = q.Where("dpe.id IN (?)", bun.List(req.EventIDs))
	} else if req.PeriodEnd > 0 {
		q = q.Where("dpe.event_date < ?", req.PeriodEnd)
	}
	err := q.
		Order("dpe.event_date ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accrued pay events: %w", err)
	}
	return items, nil
}

func (r *payEventRepository) ListWorkerIDsWithAccruedEvents(
	ctx context.Context,
	req repositories.ListWorkersWithAccruedEventsRequest,
) ([]pulid.ID, error) {
	ids := make([]pulid.ID, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driversettlement.PayEvent)(nil)).
		ColumnExpr("DISTINCT worker_id").
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Where("status = ?", driversettlement.PayEventStatusAccrued).
		Where("on_hold = false").
		Where("event_date < ?", req.PeriodEnd).
		Scan(ctx, &ids)
	if err != nil {
		return nil, fmt.Errorf("list workers with accrued pay events: %w", err)
	}
	return ids, nil
}

func (r *payEventRepository) GetAccruedTotalsForWorker(
	ctx context.Context,
	req repositories.GetAccruedTotalsForWorkerRequest,
) (*repositories.AccruedPayTotals, error) {
	totals := new(repositories.AccruedPayTotals)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driversettlement.PayEvent)(nil)).
		ColumnExpr("COUNT(*) AS event_count").
		ColumnExpr("COALESCE(SUM(gross_amount_minor), 0) AS gross_amount_minor").
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Where("worker_id = ?", req.WorkerID).
		Where("status = ?", driversettlement.PayEventStatusAccrued).
		Scan(ctx, totals)
	if err != nil {
		return nil, fmt.Errorf("get accrued totals for worker: %w", err)
	}
	return totals, nil
}

func (r *payEventRepository) ListByShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*driversettlement.PayEvent, error) {
	items := make([]*driversettlement.PayEvent, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dpe.organization_id = ?", tenantInfo.OrgID).
		Where("dpe.business_unit_id = ?", tenantInfo.BuID).
		Where("dpe.shipment_id = ?", shipmentID).
		Order("dpe.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pay events by shipment: %w", err)
	}
	return items, nil
}

func (r *payEventRepository) ListByMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) ([]*driversettlement.PayEvent, error) {
	items := make([]*driversettlement.PayEvent, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dpe.organization_id = ?", tenantInfo.OrgID).
		Where("dpe.business_unit_id = ?", tenantInfo.BuID).
		Where("dpe.move_id = ?", moveID).
		Order("dpe.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pay events by move: %w", err)
	}
	return items, nil
}

func (r *payEventRepository) ListByMovesForWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	moveIDs []pulid.ID,
) ([]*driversettlement.PayEvent, error) {
	if len(moveIDs) == 0 {
		return []*driversettlement.PayEvent{}, nil
	}
	cols := buncolgen.PayEventColumns
	items := make([]*driversettlement.PayEvent, 0, len(moveIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PayEventScopeTenant(sq, tenantInfo).
				Where(cols.WorkerID.Eq(), workerID).
				Where(cols.MoveID.In(), bun.List(moveIDs)).
				Where(cols.Status.NotEq(), driversettlement.PayEventStatusVoided)
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pay events by moves for worker: %w", err)
	}
	return items, nil
}

func (r *payEventRepository) ListUnsettledWorkerSummaries(
	ctx context.Context,
	req *repositories.ListUnsettledWorkerSummariesRequest,
) ([]*repositories.UnsettledWorkerSummary, error) {
	summaries := make([]*repositories.UnsettledWorkerSummary, 0)
	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT
			dpe.worker_id,
			TRIM(w.first_name || ' ' || w.last_name) AS worker_name,
			COUNT(*) FILTER (WHERE NOT dpe.on_hold) AS event_count,
			COALESCE(SUM(dpe.gross_amount_minor) FILTER (WHERE NOT dpe.on_hold), 0) AS gross_amount_minor,
			COUNT(*) FILTER (WHERE dpe.on_hold) AS held_count,
			COALESCE(SUM(dpe.gross_amount_minor) FILTER (WHERE dpe.on_hold), 0) AS held_gross_minor,
			EXISTS (
				SELECT 1 FROM driver_settlements dstl
				WHERE dstl.worker_id = dpe.worker_id
					AND dstl.organization_id = dpe.organization_id
					AND dstl.business_unit_id = dpe.business_unit_id
					AND dstl.period_start = ?
					AND dstl.period_end = ?
					AND dstl.status != 'Voided'
			) AS has_settlement
		FROM driver_pay_events dpe
		JOIN workers w
			ON w.id = dpe.worker_id
			AND w.organization_id = dpe.organization_id
			AND w.business_unit_id = dpe.business_unit_id
		WHERE dpe.organization_id = ?
			AND dpe.business_unit_id = ?
			AND dpe.status = 'Accrued'
			AND dpe.event_date <= ?
		GROUP BY dpe.worker_id, dpe.organization_id, dpe.business_unit_id, w.first_name, w.last_name
		ORDER BY gross_amount_minor DESC
	`,
		req.PeriodStart, req.PeriodEnd,
		req.TenantInfo.OrgID, req.TenantInfo.BuID, req.PeriodEnd,
	).Scan(ctx, &summaries)
	if err != nil {
		return nil, fmt.Errorf("list unsettled worker summaries: %w", err)
	}
	return summaries, nil
}

func (r *payEventRepository) GetUnsettledSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.UnsettledPayEventSummary, error) {
	summary := new(repositories.UnsettledPayEventSummary)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driversettlement.PayEvent)(nil)).
		ColumnExpr("COUNT(*) FILTER (WHERE on_hold = false) AS accrued_count").
		ColumnExpr(
			"COALESCE(SUM(gross_amount_minor) FILTER (WHERE on_hold = false), 0) AS accrued_gross_minor",
		).
		ColumnExpr("COUNT(*) FILTER (WHERE on_hold = true) AS held_count").
		ColumnExpr(
			"COALESCE(SUM(gross_amount_minor) FILTER (WHERE on_hold = true), 0) AS held_gross_minor",
		).
		ColumnExpr("COUNT(DISTINCT worker_id) AS worker_count").
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Where("status = ?", driversettlement.PayEventStatusAccrued).
		Scan(ctx, summary)
	if err != nil {
		return nil, fmt.Errorf("get unsettled pay event summary: %w", err)
	}
	return summary, nil
}

func (r *payEventRepository) Create(
	ctx context.Context,
	entity *driversettlement.PayEvent,
) (*driversettlement.PayEvent, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("dpe_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create pay event: %w", err)
	}
	return entity, nil
}

func (r *payEventRepository) Update(
	ctx context.Context,
	entity *driversettlement.PayEvent,
) (*driversettlement.PayEvent, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("gross_amount_minor = ?", entity.GrossAmountMinor).
		Set("total_miles = ?", entity.TotalMiles).
		Set("components = ?", entity.Components).
		Set("settlement_id = ?", entity.SettlementID).
		Set("settlement_line_id = ?", entity.SettlementLineID).
		Set("voided_at = ?", entity.VoidedAt).
		Set("void_reason = ?", entity.VoidReason).
		Set("on_hold = ?", entity.OnHold).
		Set("hold_reason = ?", entity.HoldReason).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update pay event: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "PayEvent", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *payEventRepository) MarkSettled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventIDs []pulid.ID,
	settlementID pulid.ID,
) error {
	if len(eventIDs) == 0 {
		return nil
	}
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*driversettlement.PayEvent)(nil)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Where("id IN (?)", bun.List(eventIDs)).
		Where("status = ?", driversettlement.PayEventStatusAccrued).
		Set("status = ?", driversettlement.PayEventStatusSettled).
		Set("settlement_id = ?", settlementID).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mark pay events settled: %w", err)
	}
	return nil
}

func (r *payEventRepository) ReleaseSettled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) error {
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*driversettlement.PayEvent)(nil)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Where("settlement_id = ?", settlementID).
		Where("status = ?", driversettlement.PayEventStatusSettled).
		Set("status = ?", driversettlement.PayEventStatusAccrued).
		Set("settlement_id = NULL").
		Set("settlement_line_id = NULL").
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("release settled pay events: %w", err)
	}
	return nil
}

func (r *payEventRepository) ReleaseEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventIDs []pulid.ID,
) error {
	if len(eventIDs) == 0 {
		return nil
	}
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*driversettlement.PayEvent)(nil)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Where("id IN (?)", bun.List(eventIDs)).
		Where("status = ?", driversettlement.PayEventStatusSettled).
		Set("status = ?", driversettlement.PayEventStatusAccrued).
		Set("settlement_id = NULL").
		Set("settlement_line_id = NULL").
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("release pay events: %w", err)
	}
	return nil
}
