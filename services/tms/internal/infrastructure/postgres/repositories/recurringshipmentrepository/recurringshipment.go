package recurringshipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB        *postgres.Connection
	Logger    *zap.Logger
	Generator seqgen.Generator
}

type repository struct {
	db        *postgres.Connection
	l         *zap.Logger
	generator seqgen.Generator
}

func New(p Params) repositories.RecurringShipmentRepository {
	return &repository{
		db:        p.DB,
		l:         p.Logger.Named("postgres.recurring-shipment-repository"),
		generator: p.Generator,
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListRecurringShipmentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.RecurringShipmentTable.Alias,
		req.Filter,
		(*recurringshipment.RecurringShipment)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRecurringShipmentsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make(
		[]*recurringshipment.RecurringShipment,
		0,
		req.Filter.Pagination.SafeLimit(),
	)
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.RecurringShipmentRelations.Customer).
		Relation(buncolgen.RecurringShipmentRelations.OriginLocation).
		Relation(buncolgen.RecurringShipmentRelations.DestinationLocation).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count recurring shipments", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*recurringshipment.RecurringShipment]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListRecurringShipmentConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.RecurringShipmentTable.Alias,
		req.Filter,
		req.Cursor,
		(*recurringshipment.RecurringShipment)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListRecurringShipmentConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.RecurringShipmentTable.Alias,
		req.Filter,
		(*recurringshipment.RecurringShipment)(nil),
	)
}

func applyRecurringShipmentColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.RecurringShipmentTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListRecurringShipmentConnectionRequest,
) (*pagination.CursorListResult[*recurringshipment.RecurringShipment], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*recurringshipment.RecurringShipment)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count recurring shipments", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*recurringshipment.RecurringShipment]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*recurringshipment.RecurringShipment) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Relation(buncolgen.RecurringShipmentRelations.Customer).
					Relation(buncolgen.RecurringShipmentRelations.OriginLocation).
					Relation(buncolgen.RecurringShipmentRelations.DestinationLocation).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyRecurringShipmentColumns(sq, req.RecurringShipmentColumns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan recurring shipments", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetRecurringShipmentByIDRequest,
) (*recurringshipment.RecurringShipment, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	rsh := buncolgen.RecurringShipmentColumns
	entity := new(recurringshipment.RecurringShipment)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RecurringShipmentScopeTenant(sq, req.TenantInfo).
				Where(rsh.ID.Eq(), req.ID)
		}).
		Relation(buncolgen.RecurringShipmentRelations.Customer).
		Relation(buncolgen.RecurringShipmentRelations.OriginLocation).
		Relation(buncolgen.RecurringShipmentRelations.DestinationLocation).
		Relation(buncolgen.RecurringShipmentRelations.EnteredBy)

	if req.ExpandDetails {
		q = q.Relation(buncolgen.RecurringShipmentRelations.SourceShipment)
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get recurring shipment", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RecurringShipment")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) (*recurringshipment.RecurringShipment, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if err := r.applyDerivedFields(ctx, entity); err != nil {
		return nil, err
	}

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create recurring shipment", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) (*recurringshipment.RecurringShipment, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	if err := r.applyDerivedFields(ctx, entity); err != nil {
		return nil, err
	}

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
		log.Error("failed to update recurring shipment", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results,
		"RecurringShipment",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateRecurringShipmentStatusRequest,
) (*recurringshipment.RecurringShipment, error) {
	log := r.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("id", req.RecurringShipmentID.String()),
		zap.String("status", string(req.Status)),
	)

	entity, err := r.GetByID(ctx, &repositories.GetRecurringShipmentByIDRequest{
		ID:         req.RecurringShipmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	entity.Status = req.Status
	entity.Version = req.Version

	if req.Status == recurringshipment.StatusActive {
		// Resuming a series never backfills paused occurrences — the schedule
		// restarts from the next future slot.
		next, occErr := entity.NextOccurrence(timeutils.NowUnix())
		if occErr != nil {
			log.Error("failed to compute next occurrence on resume", zap.Error(occErr))
			return nil, occErr
		}

		if next == nil {
			entity.Status = recurringshipment.StatusExpired
			entity.NextOccurrenceAt = nil
		} else {
			entity.NextOccurrenceAt = &next.At
		}

		entity.ConsecutiveFailures = 0
	}

	rsh := buncolgen.RecurringShipmentColumns
	ov := req.Version
	entity.Version = ov + 1

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Column(
			rsh.Status.Bare(),
			rsh.NextOccurrenceAt.Bare(),
			rsh.ConsecutiveFailures.Bare(),
			rsh.Version.Bare(),
			rsh.UpdatedAt.Bare(),
		).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update recurring shipment status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results,
		"RecurringShipment",
		req.RecurringShipmentID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.RecurringShipmentSelectOptionsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
	return dbhelper.SelectOptions[*recurringshipment.RecurringShipment](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"name",
				"status",
				"cron_expression",
			},
			OrgColumn:  "rsh.organization_id",
			BuColumn:   "rsh.business_unit_id",
			EntityName: "RecurringShipment",
			SearchColumns: []string{
				"rsh.name",
			},
		},
	)
}

func (r *repository) Match(
	ctx context.Context,
	req *repositories.MatchRecurringShipmentsRequest,
) ([]*recurringshipment.RecurringShipment, error) {
	log := r.l.With(zap.String("operation", "Match"))

	rsh := buncolgen.RecurringShipmentColumns
	entities := make([]*recurringshipment.RecurringShipment, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RecurringShipmentScopeTenant(sq, req.TenantInfo).
				Where(rsh.Status.Eq(), recurringshipment.StatusActive).
				Where(rsh.CustomerID.Eq(), req.CustomerID).
				Where(rsh.OriginLocationID.Eq(), req.OriginLocationID).
				Where(rsh.DestinationLocationID.Eq(), req.DestinationLocationID)
		}).
		Relation(buncolgen.RecurringShipmentRelations.Customer).
		Relation(buncolgen.RecurringShipmentRelations.OriginLocation).
		Relation(buncolgen.RecurringShipmentRelations.DestinationLocation).
		Order(rsh.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		log.Error("failed to match recurring shipments", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) DetectLanePattern(
	ctx context.Context,
	req *repositories.DetectLanePatternRequest,
) (*repositories.LanePatternSummary, error) {
	log := r.l.With(zap.String("operation", "DetectLanePattern"))

	lookbackDays := req.LookbackDays
	if lookbackDays <= 0 {
		lookbackDays = 90
	}

	cutoff := timeutils.DaysAgoUnix(lookbackDays)

	summary := new(repositories.LanePatternSummary)
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("COUNT(*) AS shipment_count").
		ColumnExpr("COALESCE(MIN(sp.created_at), 0) AS first_shipment_at").
		ColumnExpr("COALESCE(MAX(sp.created_at), 0) AS last_shipment_at").
		Where("sp.organization_id = ?", req.TenantInfo.OrgID).
		Where("sp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("sp.customer_id = ?", req.CustomerID).
		Where("sp.status != ?", "Canceled").
		Where("sp.created_at >= ?", cutoff).
		Where(`EXISTS (
			SELECT 1 FROM shipment_moves sm
			JOIN stops stp ON stp.shipment_move_id = sm.id
			WHERE sm.shipment_id = sp.id
			AND stp.type IN ('Pickup', 'SplitPickup')
			AND stp.location_id = ?
		)`, req.OriginLocationID).
		Where(`EXISTS (
			SELECT 1 FROM shipment_moves sm
			JOIN stops stp ON stp.shipment_move_id = sm.id
			WHERE sm.shipment_id = sp.id
			AND stp.type IN ('Delivery', 'SplitDelivery')
			AND stp.location_id = ?
		)`, req.DestinationLocationID).
		Scan(ctx, summary)
	if err != nil {
		log.Error("failed to detect lane pattern", zap.Error(err))
		return nil, err
	}

	minShipments := req.MinShipments
	if minShipments <= 0 {
		minShipments = 3
	}

	if summary.ShipmentCount < minShipments {
		return nil, nil //nolint:nilnil // no pattern is a valid non-error outcome
	}

	return summary, nil
}

func (r *repository) ListDue(
	ctx context.Context,
	req *repositories.ListDueRecurringShipmentsRequest,
) ([]*recurringshipment.RecurringShipment, error) {
	rsh := buncolgen.RecurringShipmentColumns

	entities := make([]*recurringshipment.RecurringShipment, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Where(rsh.Status.Eq(), recurringshipment.StatusActive).
		Where(rsh.AutoGenerate.IsTrue()).
		Where(rsh.NextOccurrenceAt.IsNotNull()).
		Where("rsh.next_occurrence_at - (rsh.lead_time_days::bigint * 86400) <= ?", req.Now).
		Order(rsh.NextOccurrenceAt.OrderAsc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list due recurring shipments", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) ListRuns(
	ctx context.Context,
	req *repositories.ListRecurringShipmentRunsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipmentRun], error) {
	log := r.l.With(
		zap.String("operation", "ListRuns"),
		zap.String("recurringShipmentId", req.RecurringShipmentID.String()),
	)

	rsr := buncolgen.RecurringShipmentRunColumns
	entities := make([]*recurringshipment.RecurringShipmentRun, 0)
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RecurringShipmentRunScopeTenant(sq, req.TenantInfo).
				Where(rsr.RecurringShipmentID.Eq(), req.RecurringShipmentID)
		}).
		Relation(buncolgen.RecurringShipmentRunRelations.GeneratedShipment).
		Relation(buncolgen.RecurringShipmentRunRelations.TriggeredBy).
		Order(rsr.OccurrenceAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to list recurring shipment runs", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*recurringshipment.RecurringShipmentRun]{
		Items: entities,
		Total: total,
	}, nil
}
