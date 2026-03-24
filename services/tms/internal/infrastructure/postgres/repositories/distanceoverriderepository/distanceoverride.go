package distanceoverriderepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
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

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDistanceOverrideRequest,
) (*pagination.ListResult[*distanceoverride.DistanceOverride], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*distanceoverride.DistanceOverride, 0, req.Filter.Pagination.Limit)
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
		if _, insertErr := r.db.DBForContext(c).NewInsert().Model(entity).Returning("*").Exec(c); insertErr != nil {
			log.Error("failed to create distance override", zap.Error(insertErr))
			return insertErr
		}

		if len(entity.IntermediateStops) > 0 {
			entity.NormalizeIntermediateStops()
			if _, insertErr := r.db.DBForContext(c).NewInsert().Model(&entity.IntermediateStops).Exec(c); insertErr != nil {
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

		if checkErr := dberror.CheckRowsAffected(results, "DistanceOverride", entity.ID.String()); checkErr != nil {
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
			if _, insertErr := r.db.DBForContext(c).NewInsert().Model(&entity.IntermediateStops).Exec(c); insertErr != nil {
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
