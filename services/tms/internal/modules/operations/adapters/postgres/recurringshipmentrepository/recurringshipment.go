package recurringshipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/recurringshipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/recurringshipmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/dbscope"
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
	Quota     platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db        *postgres.Connection
	l         *zap.Logger
	generator seqgen.Generator
	quota     platformplanports.QuotaGuard
}

func New(p Params) recurringshipmentports.RecurringShipmentRepository {
	return &repository{
		db:        p.DB,
		l:         p.Logger.Named("postgres.recurring-shipment-repository"),
		generator: p.Generator,
		quota:     quotaservice.OrUnlimited(p.Quota),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *recurringshipmentports.ListRecurringShipmentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		recurringshipmentcol.RecurringShipmentTable.Alias,
		req.Filter,
		(*recurringshipment.RecurringShipment)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *recurringshipmentports.ListRecurringShipmentsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
		log := r.l.With(zap.String("operation", "List"))

		entities := make(
			[]*recurringshipment.RecurringShipment,
			0,
			req.Filter.Pagination.SafeLimit(),
		)
		total, err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Relation(recurringshipmentcol.RecurringShipmentRelations.Customer).
			Relation(recurringshipmentcol.RecurringShipmentRelations.OriginLocation).
			Relation(recurringshipmentcol.RecurringShipmentRelations.DestinationLocation).
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
	})
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *recurringshipmentports.ListRecurringShipmentConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		recurringshipmentcol.RecurringShipmentTable.Alias,
		req.Filter,
		req.Cursor,
		(*recurringshipment.RecurringShipment)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *recurringshipmentports.ListRecurringShipmentConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		recurringshipmentcol.RecurringShipmentTable.Alias,
		req.Filter,
		(*recurringshipment.RecurringShipment)(nil),
	)
}

func applyRecurringShipmentColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(recurringshipmentcol.RecurringShipmentTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *recurringshipmentports.ListRecurringShipmentConnectionRequest,
) (*pagination.CursorListResult[*recurringshipment.RecurringShipment], error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagination.CursorListResult[*recurringshipment.RecurringShipment], error) {
		log := r.l.With(zap.String("operation", "ListConnection"))

		dba := r.db.DBForContext(ctx)
		var totalCount *int
		if req.Cursor.IncludeTotalCount {
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
			totalCount = &total
		}

		result, err := dbhelper.CursorList(
			ctx,
			dbhelper.CursorListParams[*recurringshipment.RecurringShipment]{
				Filter:     req.Filter,
				Cursor:     req.Cursor,
				TotalCount: totalCount,
				Query: func(entities *[]*recurringshipment.RecurringShipment) *bun.SelectQuery {
					return dba.
						NewSelect().
						Model(entities).
						Relation(recurringshipmentcol.RecurringShipmentRelations.Customer).
						Relation(recurringshipmentcol.RecurringShipmentRelations.OriginLocation).
						Relation(recurringshipmentcol.RecurringShipmentRelations.DestinationLocation).
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
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req *recurringshipmentports.GetRecurringShipmentByIDRequest,
) (*recurringshipment.RecurringShipment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*recurringshipment.RecurringShipment, error) {
		log := r.l.With(
			zap.String("operation", "GetByID"),
			zap.String("id", req.ID.String()),
		)

		rsh := recurringshipmentcol.RecurringShipmentColumns
		entity := new(recurringshipment.RecurringShipment)
		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return recurringshipmentcol.RecurringShipmentScopeTenant(sq, req.TenantInfo).
					Where(rsh.ID.Eq(), req.ID)
			}).
			Relation(recurringshipmentcol.RecurringShipmentRelations.Customer).
			Relation(recurringshipmentcol.RecurringShipmentRelations.OriginLocation).
			Relation(recurringshipmentcol.RecurringShipmentRelations.DestinationLocation).
			Relation(recurringshipmentcol.RecurringShipmentRelations.EnteredBy)

		if req.ExpandDetails {
			q = q.Relation(recurringshipmentcol.RecurringShipmentRelations.SourceShipment)
		}

		if err := q.Scan(ctx); err != nil {
			log.Error("failed to get recurring shipment", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "RecurringShipment")
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) (*recurringshipment.RecurringShipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*recurringshipment.RecurringShipment, error) {
		log := r.l.With(zap.String("operation", "Create"))

		if err := r.applyDerivedFields(ctx, entity); err != nil {
			return nil, err
		}

		if err := r.insertSeries(ctx, entity); err != nil {
			log.Error("failed to create recurring shipment", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) insertSeries(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) error {
	return quotatx.Run(ctx, r.db, r.quota, func(c context.Context) error {
		_, err := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c)
		return err
	}, platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		Meter:    platformcatalog.MeterRecurringShipmentSeries,
		Quantity: 1,
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) (*recurringshipment.RecurringShipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*recurringshipment.RecurringShipment, error) {
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
	})
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *recurringshipmentports.UpdateRecurringShipmentStatusRequest,
) (*recurringshipment.RecurringShipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*recurringshipment.RecurringShipment, error) {
		log := r.l.With(
			zap.String("operation", "UpdateStatus"),
			zap.String("id", req.RecurringShipmentID.String()),
			zap.String("status", string(req.Status)),
		)

		entity, err := r.GetByID(ctx, &recurringshipmentports.GetRecurringShipmentByIDRequest{
			ID:         req.RecurringShipmentID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}

		entity.Version = req.Version
		if err = entity.ApplyStatus(req.Status, timeutils.NowUnix()); err != nil {
			log.Error("failed to compute next occurrence on resume", zap.Error(err))
			return nil, err
		}

		rsh := recurringshipmentcol.RecurringShipmentColumns
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
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *recurringshipmentports.RecurringShipmentSelectOptionsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
		return dbhelper.SelectOptions[*recurringshipment.RecurringShipment](
			ctx,
			r.db.DBForContext(ctx),
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
	})
}

func (r *repository) Match(
	ctx context.Context,
	req *recurringshipmentports.MatchRecurringShipmentsRequest,
) ([]*recurringshipment.RecurringShipment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*recurringshipment.RecurringShipment, error) {
		log := r.l.With(zap.String("operation", "Match"))

		rsh := recurringshipmentcol.RecurringShipmentColumns
		entities := make([]*recurringshipment.RecurringShipment, 0)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return recurringshipmentcol.RecurringShipmentScopeTenant(sq, req.TenantInfo).
					Where(rsh.Status.Eq(), recurringshipment.StatusActive).
					Where(rsh.CustomerID.Eq(), req.CustomerID).
					Where(rsh.OriginLocationID.Eq(), req.OriginLocationID).
					Where(rsh.DestinationLocationID.Eq(), req.DestinationLocationID)
			}).
			Relation(recurringshipmentcol.RecurringShipmentRelations.Customer).
			Relation(recurringshipmentcol.RecurringShipmentRelations.OriginLocation).
			Relation(recurringshipmentcol.RecurringShipmentRelations.DestinationLocation).
			Order(rsh.CreatedAt.OrderAsc()).
			Scan(ctx)
		if err != nil {
			log.Error("failed to match recurring shipments", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) DetectLanePattern(
	ctx context.Context,
	req *recurringshipmentports.DetectLanePatternRequest,
) (*recurringshipmentports.LanePatternSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*recurringshipmentports.LanePatternSummary, error) {
		log := r.l.With(zap.String("operation", "DetectLanePattern"))

		lookbackDays := req.LookbackDays
		if lookbackDays <= 0 {
			lookbackDays = 90
		}

		cutoff := timeutils.DaysAgoUnix(lookbackDays)

		summary := new(recurringshipmentports.LanePatternSummary)
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
	})
}

func (r *repository) ListDue(
	ctx context.Context,
	req *recurringshipmentports.ListDueRecurringShipmentsRequest,
) ([]*recurringshipment.RecurringShipment, error) {
	ctx = dbscope.WithSystem(ctx, "list recurring shipments due across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*recurringshipment.RecurringShipment, error) {
		rsh := recurringshipmentcol.RecurringShipmentColumns

		entities := make([]*recurringshipment.RecurringShipment, 0)
		q := r.db.DBForContext(ctx).
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
	})
}

func (r *repository) ListRuns(
	ctx context.Context,
	req *recurringshipmentports.ListRecurringShipmentRunsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipmentRun], error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*pagination.ListResult[*recurringshipment.RecurringShipmentRun], error) {
		log := r.l.With(
			zap.String("operation", "ListRuns"),
			zap.String("recurringShipmentId", req.RecurringShipmentID.String()),
		)

		rsr := recurringshipmentcol.RecurringShipmentRunColumns
		entities := make([]*recurringshipment.RecurringShipmentRun, 0)
		total, err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return recurringshipmentcol.RecurringShipmentRunScopeTenant(sq, req.TenantInfo).
					Where(rsr.RecurringShipmentID.Eq(), req.RecurringShipmentID)
			}).
			Relation(recurringshipmentcol.RecurringShipmentRunRelations.GeneratedShipment).
			Relation(recurringshipmentcol.RecurringShipmentRunRelations.TriggeredBy).
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
	})
}
