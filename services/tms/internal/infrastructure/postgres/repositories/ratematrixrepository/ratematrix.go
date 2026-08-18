package ratematrixrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports"
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

// cellInsertBatch bounds how many cells go in one insert. A class tariff runs
// to hundreds of thousands of rows, and one statement that size holds locks far
// longer than the work deserves.
const cellInsertBatch = 1000

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.RateMatrixRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratematrix-repository"),
	}
}

func orderDimensions(sq *bun.SelectQuery) *bun.SelectQuery {
	return sq.Order(buncolgen.RateMatrixDimensionColumns.Position.OrderAsc())
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListRateMatrixRequest,
) *bun.SelectQuery {
	cols := buncolgen.RateMatrixColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.RateMatrixTable.Alias,
		req.Filter,
		(*ratematrix.RateMatrix)(nil),
	)

	return q.Apply(buncolgen.RateMatrixApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRateMatrixRequest,
) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*ratematrix.RateMatrix, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count rate matrices", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*ratematrix.RateMatrix]{Items: entities, Total: total}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateMatrixConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.RateMatrixTable.Alias,
		req.Filter,
		req.Cursor,
		(*ratematrix.RateMatrix)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateMatrixConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.RateMatrixTable.Alias,
		req.Filter,
		(*ratematrix.RateMatrix)(nil),
	)
}

func applyRateMatrixColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.RateMatrixTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListRateMatrixConnectionRequest,
) (*pagination.CursorListResult[*ratematrix.RateMatrix], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*ratematrix.RateMatrix)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count rate matrices", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*ratematrix.RateMatrix]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*ratematrix.RateMatrix) *bun.SelectQuery {
			return dba.
				NewSelect().
				Model(entities).
				Relation(buncolgen.Rel(buncolgen.RateMatrixRelations.FormulaTemplate)).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return applyRateMatrixColumns(sq, req.RateMatrixColumns)
				})
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return r.applyCursorPageFilters(sq, req)
		},
	})
	if err != nil {
		log.Error("failed to scan rate matrices", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetRateMatrixByIDRequest,
) (*ratematrix.RateMatrix, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.RateMatrixID.String()),
	)

	entity := new(ratematrix.RateMatrix)
	cols := buncolgen.RateMatrixColumns

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateMatrixScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateMatrixID)
		})

	if req.IncludeDimensions {
		q = q.Relation(
			buncolgen.Rel(buncolgen.RateMatrixRelations.Dimensions),
			orderDimensions,
		)
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get rate matrix", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RateMatrix")
	}

	return entity, nil
}

// GetByCode resolves the name a formula expression refers to.
//
// Expressions name a matrix by code rather than by id, the way they already
// name rate tables, so an organization can rebuild a tariff without rewriting
// every formula that reads it.
func (r *repository) GetByCode(
	ctx context.Context,
	req *repositories.GetRateMatrixByCodeRequest,
) (*ratematrix.RateMatrix, error) {
	log := r.l.With(zap.String("operation", "GetByCode"), zap.String("code", req.Code))

	entity := new(ratematrix.RateMatrix)
	cols := buncolgen.RateMatrixColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.Rel(buncolgen.RateMatrixRelations.Dimensions), orderDimensions).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateMatrixScopeTenant(sq, req.TenantInfo).
				Where(cols.Code.LowerLike(), req.Code).
				Where(cols.Status.Eq(), domaintypes.StatusActive)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get rate matrix by code", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RateMatrix")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) (*ratematrix.RateMatrix, error) {
	log := r.l.With(zap.String("operation", "Create"), zap.String("code", entity.Code))

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		stampDimensions(entity, false)

		return r.insertDimensions(c, entity)
	})
	if err != nil {
		log.Error("failed to create rate matrix", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate matrix is busy. Retry the request.",
		)
	}

	return entity, nil
}

// Update rewrites the header and the axes. Cells are left alone: a matrix can
// hold hundreds of thousands of them, and they are replaced through
// ReplaceCells rather than dragged along with every rename.
func (r *repository) Update(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) (*ratematrix.RateMatrix, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	cols := buncolgen.RateMatrixColumns
	dimensionCols := buncolgen.RateMatrixDimensionColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(cols.Version.Eq(), ov).
			OmitZero().
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		if uErr = dberror.CheckRowsAffected(
			results,
			"RateMatrix",
			entity.ID.String(),
		); uErr != nil {
			return uErr
		}

		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*ratematrix.RateMatrixDimension)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.RateMatrixDimensionScopeTenantDelete(dq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).Where(dimensionCols.RateMatrixID.Eq(), entity.ID)
			}).
			Exec(c); dErr != nil {
			return dErr
		}

		stampDimensions(entity, true)

		return r.insertDimensions(c, entity)
	})
	if err != nil {
		log.Error("failed to update rate matrix", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate matrix is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) insertDimensions(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) error {
	if len(entity.Dimensions) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Dimensions).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.GetRateMatrixByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.RateMatrixID.String()),
	)

	cols := buncolgen.RateMatrixColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*ratematrix.RateMatrix)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.RateMatrixScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateMatrixID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete rate matrix", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "RateMatrix", req.RateMatrixID.String())
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
	cols := buncolgen.RateMatrixColumns

	return dbhelper.SelectOptions[*ratematrix.RateMatrix](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Code,
				cols.Name,
				cols.Description,
				cols.FormulaTemplateID,
				cols.Currency,
				cols.Status,
			},
			OrgColumnRef:     &cols.OrganizationID,
			BuColumnRef:      &cols.BusinessUnitID,
			SearchColumnRefs: []buncolgen.Column{cols.Code, cols.Name},
			EntityName:       "RateMatrix",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), domaintypes.StatusActive)
			},
		},
	)
}
