package trailerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
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

func NewRepository(p Params) repositories.TrailerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.trailer-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.TrailerFilterOptions,
) *bun.SelectQuery {
	if opts.IncludeEquipmentDetails {
		q = q.Relation("EquipmentType").Relation("EquipmentManufacturer")
	}

	if opts.IncludeFleetDetails {
		q = q.Relation("FleetCode")
	}

	if opts.Status != "" {
		status, err := domain.EquipmentStatusFromString(opts.Status)
		if err != nil {
			r.l.Error("invalid status", zap.Error(err), zap.String("status", opts.Status))
			return q
		}

		q = q.Where("tr.status = ?", status)
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListTrailerRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"tr",
		req.Filter,
		(*trailer.Trailer)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.FilterOptions)
	})

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListTrailerRequest,
) (*pagination.ListResult[*trailer.Trailer], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*trailer.Trailer, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan trailers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*trailer.Trailer]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetTrailerByIDRequest,
) (*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(trailer.Trailer)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("tr.id = ?", req.ID).
				Where("tr.organization_id = ?", req.OrgID).
				Where("tr.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Trailer")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *trailer.Trailer,
) (*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert trailer", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *trailer.Trailer,
) (*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("tr.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update trailer", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Trailer", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}
