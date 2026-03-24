package trailerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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

func New(p Params) repositories.TrailerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.trailer-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListTrailersRequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "filterQuery"),
		zap.Any("request", req),
	)

	rel := buncolgen.TrailerRelations
	if req.IncludeEquipmentDetails {
		q = q.Relation(rel.EquipmentType).
			Relation(rel.EquipmentManufacturer)
	}

	if req.IncludeFleetDetails {
		q = q.Relation(rel.FleetCode)
	}

	q = querybuilder.ApplyFilters(
		q,
		buncolgen.TrailerTable.Alias,
		req.Filter,
		(*trailer.Trailer)(nil),
	)

	if req.Status != "" {
		status, err := domaintypes.EquipmentStatusFromString(req.Status)
		if err != nil {
			log.Error("failed to parse equipment status", zap.Error(err))
			return q
		}

		q = q.Where(buncolgen.TrailerColumns.Status.Eq(), status)
	}

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListTrailersRequest,
) (*pagination.ListResult[*trailer.Trailer], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*trailer.Trailer, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(applyLastKnownLocationJoin).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count trailers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*trailer.Trailer]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *trailer.Trailer,
) (*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create trailer", zap.Error(err))
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
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.TrailerColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update trailer", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Trailer", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetTrailerByIDRequest,
) (*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(trailer.Trailer)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Apply(applyLastKnownLocationJoin).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TrailerScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.TrailerColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get trailer", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Trailer")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*trailer.Trailer], error) {
	cols := buncolgen.TrailerColumns

	return dbhelper.SelectOptions[*trailer.Trailer](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				cols.ID.Bare(),
				cols.Status.Bare(),
				cols.Code.Bare(),
			},
			OrgColumn: cols.OrganizationID.Qualified(),
			BuColumn:  cols.BusinessUnitID.Qualified(),
			SearchColumns: []string{
				cols.Code.Qualified(),
			},
			EntityName: "Trailer",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(
					cols.Status.Eq(),
					domaintypes.EquipmentStatusAvailable,
				)
			},
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetTrailersByIDsRequest,
) ([]*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*trailer.Trailer, 0, len(req.TrailerIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(applyLastKnownLocationJoin).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TrailerScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.TrailerColumns.ID.In(), bun.List(req.TrailerIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get trailers", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Trailer")
	}

	return entities, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateTrailerStatusRequest,
) ([]*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*trailer.Trailer, 0, len(req.TrailerIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TrailerScopeTenantUpdate(uq, req.TenantInfo).
				Where(buncolgen.TrailerColumns.ID.In(), bun.List(req.TrailerIDs))
		}).
		Set(buncolgen.TrailerColumns.Status.Set(), req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update trailer status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "Trailer", req.TrailerIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func applyLastKnownLocationJoin(q *bun.SelectQuery) *bun.SelectQuery {
	return q.
		Column("tr.*").
		ColumnExpr("ec.current_location_id AS last_known_location_id").
		ColumnExpr("COALESCE(lkl.name, '') AS last_known_location_name").
		Join("LEFT JOIN equipment_continuity AS ec ON ec.equipment_id = tr.id AND ec.equipment_type = ? AND ec.organization_id = tr.organization_id AND ec.business_unit_id = tr.business_unit_id AND ec.is_current = TRUE", "Trailer").
		Join("LEFT JOIN locations AS lkl ON lkl.id = ec.current_location_id AND lkl.organization_id = ec.organization_id AND lkl.business_unit_id = ec.business_unit_id")
}
