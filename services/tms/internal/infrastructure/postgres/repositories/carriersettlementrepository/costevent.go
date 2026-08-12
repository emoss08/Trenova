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
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type costEventRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewCostEvent(p Params) repositories.CarrierCostEventRepository {
	return &costEventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-cost-event-repository"),
	}
}

func (r *costEventRepository) List(
	ctx context.Context,
	req *repositories.ListCarrierCostEventsRequest,
) (*pagination.ListResult[*carriersettlement.CostEvent], error) {
	cols := buncolgen.CostEventColumns
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*carriersettlement.CostEvent, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Apply(buncolgen.CostEventApplyTenant(req.Filter.TenantInfo)).
		Relation(buncolgen.CostEventRelations.Carrier).
		Order(cols.EventDate.OrderDesc()).
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where(cols.ProNumber.ILike(), "%"+req.Filter.Query+"%")
	}
	if !req.CarrierID.IsNil() {
		query = query.Where(cols.CarrierID.Eq(), req.CarrierID)
	}
	if !req.ShipmentID.IsNil() {
		query = query.Where(cols.ShipmentID.Eq(), req.ShipmentID)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list carrier cost events: %w", err)
	}

	return &pagination.ListResult[*carriersettlement.CostEvent]{Items: items, Total: total}, nil
}

func (r *costEventRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListCarrierCostEventConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CostEvent], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*carriersettlement.CostEvent)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.CostEventTable.Alias,
				req.Filter,
				(*carriersettlement.CostEvent)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count carrier cost events", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*carriersettlement.CostEvent]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*carriersettlement.CostEvent) *bun.SelectQuery {
				return dba.NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.CostEventTable.All()).
					Relation(buncolgen.CostEventRelations.Carrier)
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.CostEventTable.Alias,
					req.Filter,
					req.Cursor,
					(*carriersettlement.CostEvent)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan carrier cost events", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *costEventRepository) GetByID(
	ctx context.Context,
	req repositories.GetCarrierCostEventByIDRequest,
) (*carriersettlement.CostEvent, error) {
	cols := buncolgen.CostEventColumns
	entity := new(carriersettlement.CostEvent)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CostEventScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Relation(buncolgen.CostEventRelations.Carrier).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "CarrierCostEvent")
	}
	return entity, nil
}

func (r *costEventRepository) GetByIdempotencyKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key string,
) (*carriersettlement.CostEvent, error) {
	cols := buncolgen.CostEventColumns
	entity := new(carriersettlement.CostEvent)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CostEventScopeTenant(sq, tenantInfo).
				Where(cols.IdempotencyKey.Eq(), key)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "CarrierCostEvent")
	}
	return entity, nil
}

// ListPendingByCarrier deliberately applies no lower period bound: pending
// events accrued before the current period (late accruals, re-opened moves)
// are swept into the next settlement instead of being stranded. This mirrors
// the driver settlement repository.
func (r *costEventRepository) ListPendingByCarrier(
	ctx context.Context,
	req *repositories.ListPendingCostEventsRequest,
) ([]*carriersettlement.CostEvent, error) {
	cols := buncolgen.CostEventColumns
	items := make([]*carriersettlement.CostEvent, 0)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.CostEventScopeTenant(sq, req.TenantInfo).
				Where(cols.CarrierID.Eq(), req.CarrierID).
				Where(cols.Status.Eq(), carriersettlement.CostEventStatusPending)
			if req.PeriodEnd > 0 {
				sq = sq.Where(cols.EventDate.Lt(), req.PeriodEnd)
			}
			return sq
		}).
		Order(cols.EventDate.OrderAsc())
	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list pending carrier cost events: %w", err)
	}
	return items, nil
}

func (r *costEventRepository) ListCarrierIDsWithPendingEvents(
	ctx context.Context,
	req repositories.ListCarriersWithPendingEventsRequest,
) ([]pulid.ID, error) {
	cols := buncolgen.CostEventColumns
	ids := make([]pulid.ID, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carriersettlement.CostEvent)(nil)).
		ColumnExpr("DISTINCT "+cols.CarrierID.Qualified()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CostEventScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), carriersettlement.CostEventStatusPending).
				Where(cols.EventDate.Lt(), req.PeriodEnd)
		}).
		Scan(ctx, &ids)
	if err != nil {
		return nil, fmt.Errorf("list carriers with pending cost events: %w", err)
	}
	return ids, nil
}

func (r *costEventRepository) ListByShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*carriersettlement.CostEvent, error) {
	cols := buncolgen.CostEventColumns
	items := make([]*carriersettlement.CostEvent, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CostEventScopeTenant(sq, tenantInfo).
				Where(cols.ShipmentID.Eq(), shipmentID)
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list carrier cost events by shipment: %w", err)
	}
	return items, nil
}

func (r *costEventRepository) ListByMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) ([]*carriersettlement.CostEvent, error) {
	cols := buncolgen.CostEventColumns
	items := make([]*carriersettlement.CostEvent, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CostEventScopeTenant(sq, tenantInfo).
				Where(cols.MoveID.Eq(), moveID)
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list carrier cost events by move: %w", err)
	}
	return items, nil
}

func (r *costEventRepository) GetPendingSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.CarrierCostEventSummary, error) {
	summary := new(repositories.CarrierCostEventSummary)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carriersettlement.CostEvent)(nil)).
		ColumnExpr("COUNT(*) AS pending_count").
		ColumnExpr("COALESCE(SUM(amount_minor), 0) AS pending_amount_minor").
		ColumnExpr("COUNT(DISTINCT carrier_id) AS pending_carrier_count").
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Where("status = ?", carriersettlement.CostEventStatusPending).
		Scan(ctx, summary)
	if err != nil {
		return nil, fmt.Errorf("get pending carrier cost event summary: %w", err)
	}
	return summary, nil
}

func (r *costEventRepository) Create(
	ctx context.Context,
	entity *carriersettlement.CostEvent,
) (*carriersettlement.CostEvent, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("cacc_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create carrier cost event: %w", err)
	}
	return entity, nil
}

func (r *costEventRepository) Update(
	ctx context.Context,
	entity *carriersettlement.CostEvent,
) (*carriersettlement.CostEvent, error) {
	cols := buncolgen.CostEventColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CostEventScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), entity.Version)
		}).
		Set(cols.Status.Set(), entity.Status).
		Set(cols.AmountMinor.Set(), entity.AmountMinor).
		Set(cols.Description.Set(), entity.Description).
		Set(cols.ProNumber.Set(), entity.ProNumber).
		Set(cols.AssignmentVersion.Set(), entity.AssignmentVersion).
		Set(cols.SettlementID.Set(), entity.SettlementID).
		Set(cols.VoidedAt.Set(), entity.VoidedAt).
		Set(cols.VoidReason.Set(), entity.VoidReason).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update carrier cost event: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "CarrierCostEvent", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *costEventRepository) MarkAttached(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventIDs []pulid.ID,
	settlementID pulid.ID,
) error {
	if len(eventIDs) == 0 {
		return nil
	}
	cols := buncolgen.CostEventColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*carriersettlement.CostEvent)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CostEventScopeTenantUpdate(uq, tenantInfo).
				Where(cols.ID.In(), bun.List(eventIDs)).
				Where(cols.Status.Eq(), carriersettlement.CostEventStatusPending)
		}).
		Set(cols.Status.Set(), carriersettlement.CostEventStatusAttached).
		Set(cols.SettlementID.Set(), settlementID).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mark carrier cost events attached: %w", err)
	}
	// Zero rows means every selected event was attached or voided by a
	// concurrent generator; the caller must not keep a settlement built from
	// events it does not own.
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark carrier cost events attached: %w", err)
	}
	if rows == 0 {
		return errortypes.NewValidationError(
			"costEventIds",
			errortypes.ErrInvalidOperation,
			"The selected carrier cost events are no longer pending; they were attached or voided by a concurrent request",
		)
	}
	return nil
}

func (r *costEventRepository) MarkSettled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) error {
	cols := buncolgen.CostEventColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*carriersettlement.CostEvent)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CostEventScopeTenantUpdate(uq, tenantInfo).
				Where(cols.SettlementID.Eq(), settlementID).
				Where(cols.Status.Eq(), carriersettlement.CostEventStatusAttached)
		}).
		Set(cols.Status.Set(), carriersettlement.CostEventStatusSettled).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mark carrier cost events settled: %w", err)
	}
	return nil
}

func (r *costEventRepository) ReleaseSettled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) error {
	cols := buncolgen.CostEventColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*carriersettlement.CostEvent)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CostEventScopeTenantUpdate(uq, tenantInfo).
				Where(cols.SettlementID.Eq(), settlementID).
				Where(cols.Status.In(), bun.List([]carriersettlement.CostEventStatus{
					carriersettlement.CostEventStatusAttached,
					carriersettlement.CostEventStatusSettled,
				}))
		}).
		Set(cols.Status.Set(), carriersettlement.CostEventStatusPending).
		Set(cols.SettlementID.SetNull()).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("release settled carrier cost events: %w", err)
	}
	return nil
}
