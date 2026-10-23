package ratematrixrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/ratematrixcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	cellInsertBatch = 1000
	busyMessage     = "Rate matrix is busy. Retry the request."
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

func New(p Params) ratematrixports.RateMatrixRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratematrix-repository"),
	}
}

func draftDimensions(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := ratematrixcol.RateMatrixDimensionColumns
	return sq.Where(cols.VersionNumber.Eq(), ratematrix.DraftVersion).
		Order(cols.Position.OrderAsc())
}

func matrixTenant(entity *ratematrix.RateMatrix) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *ratematrixports.ListRateMatrixRequest,
) *bun.SelectQuery {
	cols := ratematrixcol.RateMatrixColumns
	q = querybuilder.ApplyFilters(
		q,
		ratematrixcol.RateMatrixTable.Alias,
		req.Filter,
		(*ratematrix.RateMatrix)(nil),
	)

	return q.Apply(ratematrixcol.RateMatrixApplyTenant(req.Filter.TenantInfo)).
		Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset())).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixRequest,
) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
			entities := make([]*ratematrix.RateMatrix, 0, req.Filter.Pagination.SafeLimit())
			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.filterQuery(sq, req)
				}).
				ScanAndCount(ctx)
			if err != nil {
				r.l.Error("failed to scan and count rate matrices", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*ratematrix.RateMatrix]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

func applyRateMatrixColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(ratematrixcol.RateMatrixTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixConnectionRequest,
) (*pagination.CursorListResult[*ratematrix.RateMatrix], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ratematrix.RateMatrix], error) {
			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*ratematrix.RateMatrix)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return querybuilder.ApplyFiltersWithoutSort(
							sq,
							ratematrixcol.RateMatrixTable.Alias,
							req.Filter,
							(*ratematrix.RateMatrix)(nil),
						)
					}).
					Count(ctx)
				if err != nil {
					r.l.Error("failed to count rate matrices", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*ratematrix.RateMatrix]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*ratematrix.RateMatrix) *bun.SelectQuery {
						return dba.
							NewSelect().
							Model(entities).
							Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
								return applyRateMatrixColumns(sq, req.RateMatrixColumns)
							})
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return querybuilder.ApplyCursorFilters(
							sq,
							ratematrixcol.RateMatrixTable.Alias,
							req.Filter,
							req.Cursor,
							(*ratematrix.RateMatrix)(nil),
						)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to scan rate matrices", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *ratematrixports.GetRateMatrixByIDRequest,
) (*ratematrix.RateMatrix, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrix, error) {
		entity := new(ratematrix.RateMatrix)
		cols := ratematrixcol.RateMatrixColumns

		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.RateMatrixID)
			})

		if req.IncludeDimensions {
			q = q.Relation(
				buncolgen.Rel(ratematrixcol.RateMatrixRelations.Dimensions),
				draftDimensions,
			)
		}
		if req.ForUpdate {
			q = q.For("UPDATE")
		}

		if err := q.Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "RateMatrix")
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req *ratematrixports.GetRateMatricesByIDsRequest,
) ([]*ratematrix.RateMatrix, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrix, error) {
		if len(req.RateMatrixIDs) == 0 {
			return []*ratematrix.RateMatrix{}, nil
		}
		cols := ratematrixcol.RateMatrixColumns
		entities := make([]*ratematrix.RateMatrix, 0, len(req.RateMatrixIDs))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.In(), bun.List(req.RateMatrixIDs))
			}).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to get rate matrices by ids", zap.Error(err))
			return nil, err
		}
		return entities, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) (*ratematrix.RateMatrix, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrix, error) {
		err := r.db.WithTx(ctx, kernelports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
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
			r.l.Error(
				"failed to create rate matrix",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(err, busyMessage)
		}

		return entity, nil
	})
}

func (r *repository) updateRow(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) error {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(ratematrixcol.RateMatrixColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(results, "RateMatrix", entity.ID.String())
}

func (r *repository) UpdateHeader(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) (*ratematrix.RateMatrix, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrix, error) {
		dimensions := entity.Dimensions
		if err := r.updateRow(ctx, entity); err != nil {
			r.l.Error("failed to update rate matrix header",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}
		entity.Dimensions = dimensions
		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) (*ratematrix.RateMatrix, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrix, error) {
		dimensionCols := ratematrixcol.RateMatrixDimensionColumns
		dimensions := entity.Dimensions

		err := r.db.WithTx(ctx, kernelports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
			if uErr := r.updateRow(c, entity); uErr != nil {
				return uErr
			}
			entity.Dimensions = dimensions

			if _, dErr := r.db.DBForContext(c).
				NewDelete().
				Model((*ratematrix.RateMatrixDimension)(nil)).
				WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
					return ratematrixcol.RateMatrixDimensionScopeTenantDelete(dq, matrixTenant(entity)).
						Where(dimensionCols.RateMatrixID.Eq(), entity.ID).
						Where(dimensionCols.VersionNumber.Eq(), ratematrix.DraftVersion)
				}).
				Exec(c); dErr != nil {
				return dErr
			}

			stampDimensions(entity, true)

			return r.insertDimensions(c, entity)
		})
		if err != nil {
			r.l.Error("failed to update rate matrix",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(err, busyMessage)
		}

		return entity, nil
	})
}

func (r *repository) insertDimensions(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entity.Dimensions) == 0 {
			return nil
		}

		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Dimensions).
			Returning("*").
			Exec(ctx)

		return err
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req *ratematrixports.GetRateMatrixByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := ratematrixcol.RateMatrixColumns
		results, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*ratematrix.RateMatrix)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return ratematrixcol.RateMatrixScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.RateMatrixID)
			}).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to delete rate matrix",
				zap.String("id", req.RateMatrixID.String()),
				zap.Error(err),
			)
			return err
		}

		return dberror.CheckRowsAffected(results, "RateMatrix", req.RateMatrixID.String())
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
			cols := ratematrixcol.RateMatrixColumns

			return dbhelper.SelectOptions[*ratematrix.RateMatrix](
				ctx,
				r.db.DBForContext(ctx),
				req,
				&dbhelper.SelectOptionsConfig{
					ColumnRefs: []buncolgen.Column{
						cols.ID,
						cols.Code,
						cols.Name,
						cols.Description,
						cols.ValueType,
						cols.Status,
						cols.PublishedVersionNumber,
					},
					OrgColumnRef:     &cols.OrganizationID,
					BuColumnRef:      &cols.BusinessUnitID,
					SearchColumnRefs: []buncolgen.Column{cols.Code, cols.Name},
					EntityName:       "RateMatrix",
					QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
						return q.Where(cols.PublishedVersionNumber.Gt(), 0).
							Where(cols.Status.Ne(), formulatemplate.StatusInactive)
					},
				},
			)
		},
	)
}
