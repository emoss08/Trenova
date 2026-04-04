package locationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
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

func New(p Params) repositories.LocationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.location-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListLocationRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"loc",
		req.Filter,
		(*location.Location)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListLocationRequest,
) (*pagination.ListResult[*location.Location], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*location.Location, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count locations", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*location.Location]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetLocationByIDRequest,
) (*location.Location, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(location.Location)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Relation("State").
		Relation("LocationCategory").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("loc.id = ?", req.ID).
				Where("loc.organization_id = ?", req.TenantInfo.OrgID).
				Where("loc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get location", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Location")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *location.Location,
) (*location.Location, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create location", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *location.Location,
) (*location.Location, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update location", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Location", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateLocationStatusRequest,
) ([]*location.Location, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*location.Location, 0, len(req.LocationIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("loc.organization_id = ?", req.TenantInfo.OrgID).
				Where("loc.business_unit_id = ?", req.TenantInfo.BuID).
				Where("loc.id IN (?)", bun.List(req.LocationIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update location status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "Location", req.LocationIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetLocationsByIDsRequest,
) ([]*location.Location, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*location.Location, 0, len(req.LocationIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("loc.organization_id = ?", req.TenantInfo.OrgID).
				Where("loc.business_unit_id = ?", req.TenantInfo.BuID).
				Where("loc.id IN (?)", bun.List(req.LocationIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get locations", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Location")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.LocationSelectOptionsRequest,
) (*pagination.ListResult[*location.Location], error) {
	return dbhelper.SelectOptions[*location.Location](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"name",
				"description",
				"status",
				"address_line_1",
				"city",
				"state_id",
				"postal_code",
			},
			OrgColumn: "loc.organization_id",
			BuColumn:  "loc.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("loc.status = ?", domaintypes.StatusActive).Relation("State")
			},
			EntityName: "Location",
			SearchColumns: []string{
				"loc.code",
				"loc.name",
				"loc.description",
			},
		},
	)
}
