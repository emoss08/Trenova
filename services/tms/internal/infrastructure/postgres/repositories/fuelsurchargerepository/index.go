package fuelsurchargerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
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

type IndexParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type indexRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewIndexRepository(p IndexParams) repositories.FuelIndexRepository {
	return &indexRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fuel-index-repository"),
	}
}

func applyFuelIndexColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.FuelIndexTable.All())
	}

	return q.Column(columns...)
}

func (r *indexRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListFuelIndexConnectionRequest,
) (*pagination.CursorListResult[*fuelsurcharge.FuelIndex], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*fuelsurcharge.FuelIndex)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.FuelIndexTable.Alias,
				req.Filter,
				(*fuelsurcharge.FuelIndex)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count fuel indices", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*fuelsurcharge.FuelIndex]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*fuelsurcharge.FuelIndex) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyFuelIndexColumns(sq, req.FuelIndexColumns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.FuelIndexTable.Alias,
					req.Filter,
					req.Cursor,
					(*fuelsurcharge.FuelIndex)(nil),
				)
			},
		})
	if err != nil {
		log.Error("failed to scan fuel indices", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *indexRepository) listBySource(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	source *fuelsurcharge.IndexSource,
) ([]*fuelsurcharge.FuelIndex, error) {
	cols := buncolgen.FuelIndexColumns
	entities := make([]*fuelsurcharge.FuelIndex, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.FuelIndexScopeTenant(sq, tenantInfo).
				Where(cols.IsActive.Eq(), true)
			if source != nil {
				sq = sq.Where(cols.Source.Eq(), *source)
			}
			return sq
		}).
		Order(cols.Code.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *indexRepository) ListActiveEIA(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*fuelsurcharge.FuelIndex, error) {
	source := fuelsurcharge.IndexSourceEIA
	entities, err := r.listBySource(ctx, tenantInfo, &source)
	if err != nil {
		r.l.Error("failed to list active EIA fuel indices", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *indexRepository) ListActive(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*fuelsurcharge.FuelIndex, error) {
	entities, err := r.listBySource(ctx, tenantInfo, nil)
	if err != nil {
		r.l.Error("failed to list active fuel indices", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *indexRepository) GetByID(
	ctx context.Context,
	req *repositories.GetFuelIndexByIDRequest,
) (*fuelsurcharge.FuelIndex, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.FuelIndexID.String()),
	)

	cols := buncolgen.FuelIndexColumns
	entity := new(fuelsurcharge.FuelIndex)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelIndexScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.FuelIndexID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get fuel index", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "FuelIndex")
	}

	return entity, nil
}

func (r *indexRepository) Create(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndex,
) (*fuelsurcharge.FuelIndex, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create fuel index", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *indexRepository) Update(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndex,
) (*fuelsurcharge.FuelIndex, error) {
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
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update fuel index", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "FuelIndex", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *indexRepository) Delete(
	ctx context.Context,
	req *repositories.GetFuelIndexByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.FuelIndexID.String()),
	)

	cols := buncolgen.FuelIndexColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*fuelsurcharge.FuelIndex)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.FuelIndexScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.FuelIndexID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete fuel index", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "FuelIndex", req.FuelIndexID.String())
}

func (r *indexRepository) SelectOptions(
	ctx context.Context,
	req *repositories.FuelIndexSelectOptionsRequest,
) (*pagination.ListResult[*fuelsurcharge.FuelIndex], error) {
	cols := buncolgen.FuelIndexColumns
	return dbhelper.SelectOptions[*fuelsurcharge.FuelIndex](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Name,
				cols.Code,
				cols.Description,
				cols.Source,
				cols.EIASeriesID,
				cols.IsActive,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.IsActive.Eq(), true).
					Order(cols.Code.OrderAsc())
			},
			EntityName: "FuelIndex",
			SearchColumnRefs: []buncolgen.Column{
				cols.Name,
				cols.Code,
				cols.Description,
			},
		},
	)
}
