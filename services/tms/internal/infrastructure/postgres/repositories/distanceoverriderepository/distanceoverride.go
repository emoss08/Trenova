package distanceoverriderepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.DistanceOverrideRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.distance-override-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDistanceOverrideRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"diso",
		req.Filter,
		(*distanceoverride.DistanceOverride)(nil),
	)

	q.Relation("Customer").
		Relation("OriginLocation").
		Relation("OriginLocation.State").
		Relation("DestinationLocation").
		Relation("DestinationLocation.State")

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDistanceOverrideRequest,
) (*pagination.ListResult[*distanceoverride.DistanceOverride], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*distanceoverride.DistanceOverride, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("IntermediateStops", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("dios.stop_order ASC")
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count distance overrides", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*distanceoverride.DistanceOverride]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListDistanceOverrideConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.DistanceOverrideTable.Alias,
		req.Filter,
		req.Cursor,
		(*distanceoverride.DistanceOverride)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListDistanceOverrideConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.DistanceOverrideTable.Alias,
		req.Filter,
		(*distanceoverride.DistanceOverride)(nil),
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListDistanceOverrideConnectionRequest,
) (*pagination.CursorListResult[*distanceoverride.DistanceOverride], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*distanceoverride.DistanceOverride)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count distance overrides", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*distanceoverride.DistanceOverride]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*distanceoverride.DistanceOverride) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.DistanceOverrideTable.All()).
					Relation("Customer").
					Relation("OriginLocation").
					Relation("OriginLocation.State").
					Relation("DestinationLocation").
					Relation("DestinationLocation.State").
					Relation("IntermediateStops", func(q *bun.SelectQuery) *bun.SelectQuery {
						return q.OrderExpr("dios.stop_order ASC")
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan distance overrides", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDistanceOverrideByIDRequest,
) (*distanceoverride.DistanceOverride, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(distanceoverride.DistanceOverride)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("OriginLocation").
		Relation("DestinationLocation").
		Relation("Customer").
		Relation("IntermediateStops", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("dios.stop_order ASC")
		}).
		Relation("IntermediateStops.Location").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("diso.id = ?", req.ID).
				Where("diso.organization_id = ?", req.TenantInfo.OrgID).
				Where("diso.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get distance override", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DistanceOverride")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *distanceoverride.DistanceOverride,
) (*distanceoverride.DistanceOverride, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
	)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		if _, insertErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); insertErr != nil {
			log.Error("failed to create distance override", zap.Error(insertErr))
			return insertErr
		}

		if len(entity.IntermediateStops) > 0 {
			entity.NormalizeIntermediateStops()
			if _, insertErr := r.db.DBForContext(c).
				NewInsert().
				Model(&entity.IntermediateStops).
				Exec(c); insertErr != nil {
				log.Error("failed to create distance override stops", zap.Error(insertErr))
				return insertErr
			}
		}

		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Distance override is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *distanceoverride.DistanceOverride,
) (*distanceoverride.DistanceOverride, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		results, updateErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(c)
		if updateErr != nil {
			log.Error("failed to update distance override", zap.Error(updateErr))
			return updateErr
		}

		if checkErr := dberror.CheckRowsAffected(
			results,
			"DistanceOverride",
			entity.ID.String(),
		); checkErr != nil {
			return checkErr
		}

		if _, deleteErr := r.db.DBForContext(c).NewDelete().
			Model((*distanceoverride.DistanceOverrideStop)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return dq.Where("distance_override_id = ?", entity.ID).
					Where("organization_id = ?", entity.OrganizationID).
					Where("business_unit_id = ?", entity.BusinessUnitID)
			}).
			Exec(c); deleteErr != nil {
			log.Error("failed to replace distance override stops", zap.Error(deleteErr))
			return deleteErr
		}

		if len(entity.IntermediateStops) > 0 {
			entity.NormalizeIntermediateStops()
			if _, insertErr := r.db.DBForContext(c).
				NewInsert().
				Model(&entity.IntermediateStops).
				Exec(c); insertErr != nil {
				log.Error("failed to insert distance override stops", zap.Error(insertErr))
				return insertErr
			}
		}

		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Distance override is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteDistanceOverrideRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	result, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*distanceoverride.DistanceOverride)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("diso.id = ?", req.ID).
				Where("diso.organization_id = ?", req.TenantInfo.OrgID).
				Where("diso.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete distance override", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "DistanceOverride", req.ID.String())
}

func (r *repository) GetByRouteSignature(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	routeSignature string,
) (*distanceoverride.DistanceOverride, error) {
	entity := new(distanceoverride.DistanceOverride)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("diso.organization_id = ?", tenantInfo.OrgID).
		Where("diso.business_unit_id = ?", tenantInfo.BuID).
		Where("diso.route_signature = ?", routeSignature).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DistanceOverride")
	}

	return entity, nil
}
