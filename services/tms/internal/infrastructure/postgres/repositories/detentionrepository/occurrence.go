package detentionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type OccurrenceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type occurrenceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewOccurrenceRepository(
	p OccurrenceParams,
) repositories.DetentionOccurrenceRepository {
	return &occurrenceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-occurrence-repository"),
	}
}

func withOccurrenceNames(q *bun.SelectQuery) *bun.SelectQuery {
	occCols := buncolgen.DetentionOccurrenceColumns
	locCols := buncolgen.LocationColumns
	cusCols := buncolgen.CustomerColumns
	shipCols := buncolgen.ShipmentColumns

	return q.
		ColumnExpr("dto.*").
		ColumnExpr("COALESCE(?, '') AS location_name", bun.Safe(locCols.Name.Qualified())).
		ColumnExpr("COALESCE(?, '') AS customer_name", bun.Safe(cusCols.Name.Qualified())).
		ColumnExpr(
			"COALESCE(?, '') AS shipment_pro_number",
			bun.Safe(shipCols.ProNumber.Qualified()),
		).
		Join("LEFT JOIN locations AS loc ON ?", bun.Safe(
			locCols.ID.EqColumn(occCols.LocationID)+
				" AND "+locCols.OrganizationID.EqColumn(occCols.OrganizationID),
		)).
		Join("LEFT JOIN customers AS cus ON ?", bun.Safe(
			cusCols.ID.EqColumn(occCols.CustomerID)+
				" AND "+cusCols.OrganizationID.EqColumn(occCols.OrganizationID),
		)).
		Join("LEFT JOIN shipments AS sp ON ?", bun.Safe(
			shipCols.ID.EqColumn(occCols.ShipmentID)+
				" AND "+shipCols.OrganizationID.EqColumn(occCols.OrganizationID),
		))
}

func (r *occurrenceRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDetentionOccurrencesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.DetentionOccurrenceTable.Alias,
		req.Filter,
		(*detention.DetentionOccurrence)(nil),
	)

	cols := buncolgen.DetentionOccurrenceColumns

	if req.ShipmentID != nil && !req.ShipmentID.IsNil() {
		q = q.Where(cols.ShipmentID.Eq(), *req.ShipmentID)
	}

	if req.CustomerID != nil && !req.CustomerID.IsNil() {
		q = q.Where(cols.CustomerID.Eq(), *req.CustomerID)
	}

	if req.LocationID != nil && !req.LocationID.IsNil() {
		q = q.Where(cols.LocationID.Eq(), *req.LocationID)
	}

	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}

	if req.OpenOnly {
		q = q.Where(cols.IsOpen.IsTrue())
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *occurrenceRepository) List(
	ctx context.Context,
	req *repositories.ListDetentionOccurrencesRequest,
) (*pagination.ListResult[*detention.DetentionOccurrence], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*detention.DetentionOccurrence, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(withOccurrenceNames).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count detention occurrences", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*detention.DetentionOccurrence]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *occurrenceRepository) GetByID(
	ctx context.Context,
	req *repositories.GetDetentionOccurrenceByIDRequest,
) (*detention.DetentionOccurrence, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.OccurrenceID.String()),
	)

	cols := buncolgen.DetentionOccurrenceColumns
	entity := new(detention.DetentionOccurrence)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(withOccurrenceNames).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.OccurrenceID)
		})

	if req.IncludeEvidence {
		q = q.Relation(buncolgen.DetentionOccurrenceRelations.Evidence,
			func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(buncolgen.DetentionEvidenceColumns.Sequence.OrderAsc())
			})
	}

	if req.IncludeNotices {
		q = q.Relation(buncolgen.DetentionOccurrenceRelations.Notices,
			func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(buncolgen.DetentionNoticeColumns.CreatedAt.OrderAsc())
			})
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get detention occurrence", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DetentionOccurrence")
	}

	return entity, nil
}

func (r *occurrenceRepository) GetByStop(
	ctx context.Context,
	req *repositories.GetOccurrenceByStopRequest,
) (*detention.DetentionOccurrence, error) {
	cols := buncolgen.DetentionOccurrenceColumns
	entity := new(detention.DetentionOccurrence)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
				Where(cols.StopID.Eq(), req.StopID)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		r.l.Error("failed to get detention occurrence by stop", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *occurrenceRepository) GetByShipment(
	ctx context.Context,
	req *repositories.GetOccurrencesByShipmentRequest,
) ([]*detention.DetentionOccurrence, error) {
	cols := buncolgen.DetentionOccurrenceColumns
	entities := make([]*detention.DetentionOccurrence, 0, 4)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(withOccurrenceNames).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentID.Eq(), req.ShipmentID)
		}).
		Order(cols.ClockStartAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list detention occurrences for shipment", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *occurrenceRepository) ListOpen(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*detention.DetentionOccurrence, error) {
	cols := buncolgen.DetentionOccurrenceColumns
	entities := make([]*detention.DetentionOccurrence, 0, 32)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(withOccurrenceNames).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionOccurrenceScopeTenant(sq, tenantInfo).
				Where(cols.IsOpen.IsTrue())
		}).
		Order(cols.FreeTimeExpiresAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list open detention occurrences", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

// ListNoticesDue powers the pre-breach alerting sweep: occurrences whose notice
// window has opened but whose notice has not gone out yet.
func (r *occurrenceRepository) ListNoticesDue(
	ctx context.Context,
	req *repositories.ListNoticesDueRequest,
) ([]*detention.DetentionOccurrence, error) {
	cols := buncolgen.DetentionOccurrenceColumns
	entities := make([]*detention.DetentionOccurrence, 0, 32)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(withOccurrenceNames).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
				Where(cols.NotificationStatus.Eq(), detention.NotificationStatusPending).
				Where(cols.NoticeDueAt.IsNotNull()).
				Where(cols.NoticeDueAt.Lte(), req.Before)
		}).
		Order(cols.NoticeDueAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list detention notices due", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

// Upsert writes the occurrence keyed on its stop. Recalculation runs on every
// shipment save, so this is the hot path and must stay idempotent.
func (r *occurrenceRepository) Upsert(
	ctx context.Context,
	entity *detention.DetentionOccurrence,
) (*detention.DetentionOccurrence, error) {
	log := r.l.With(
		zap.String("operation", "Upsert"),
		zap.String("stopId", entity.StopID.String()),
	)

	existing, err := r.GetByStop(ctx, &repositories.GetOccurrenceByStopRequest{
		StopID: entity.StopID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	if existing == nil {
		if _, iErr := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); iErr != nil {
			log.Error("failed to insert detention occurrence", zap.Error(iErr))
			return nil, iErr
		}
		return entity, nil
	}

	entity.ID = existing.ID
	entity.CreatedAt = existing.CreatedAt
	entity.Version = existing.Version

	return r.Update(ctx, entity)
}

func (r *occurrenceRepository) Update(
	ctx context.Context,
	entity *detention.DetentionOccurrence,
) (*detention.DetentionOccurrence, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update detention occurrence", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results, "DetentionOccurrence", entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

// ShipmentAccrued totals what detention has already been booked against a
// shipment so a per-shipment cap can be applied across stops.
func (r *occurrenceRepository) ShipmentAccrued(
	ctx context.Context,
	req *repositories.GetOccurrencesByShipmentRequest,
) (decimal.Decimal, error) {
	cols := buncolgen.DetentionOccurrenceColumns

	var total decimal.NullDecimal
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*detention.DetentionOccurrence)(nil)).
		ColumnExpr("COALESCE(SUM(?), 0)", bun.Safe(cols.BillableAmount.Qualified())).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentID.Eq(), req.ShipmentID)
		}).
		Scan(ctx, &total)
	if err != nil {
		r.l.Error("failed to total shipment detention accrual", zap.Error(err))
		return decimal.Zero, err
	}

	if !total.Valid {
		return decimal.Zero, nil
	}

	return total.Decimal, nil
}
