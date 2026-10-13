package factoringrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	noaEntity       = "FactoringNoa"
	openNoaIndex    = "uq_factoring_noas_open"
	maxExpiringNoas = 500
)

type noaRepository struct {
	db kernelports.DBConnection
	l  *zap.Logger
}

func NewNoaRepository(p Params) factoringports.NoaRepository {
	return &noaRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-noa"),
	}
}

func (r *noaRepository) Create(ctx context.Context, entity *factoring.Noa) (*factoring.Noa, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Noa, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, translateNoaError(err, "create factoring notice of assignment")
		}

		return entity, nil
	})
}

func (r *noaRepository) Update(ctx context.Context, entity *factoring.Noa) (*factoring.Noa, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Noa, error) {
		previousVersion := entity.Version
		entity.Version++

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			ExcludeColumn(
				factoringcol.NoaColumns.CreatedAt.Bare(),
				factoringcol.NoaColumns.CustomerID.Bare(),
				factoringcol.NoaColumns.FactoringCompanyID.Bare(),
			).
			WherePK().
			Where(factoringcol.NoaColumns.Version.Eq(), previousVersion).
			Returning("*").
			Exec(ctx)
		if err != nil {
			entity.Version = previousVersion
			return nil, translateNoaError(err, "update factoring notice of assignment")
		}
		if err = dberror.CheckRowsAffected(res, noaEntity, entity.ID.String()); err != nil {
			entity.Version = previousVersion
			return nil, err
		}

		return entity, nil
	})
}

func (r *noaRepository) GetByID(
	ctx context.Context,
	req factoringports.GetNoaRequest,
) (*factoring.Noa, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Noa, error) {
		entity := new(factoring.Noa)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.NoaScopeTenant(sq, req.TenantInfo).
					Where(factoringcol.NoaColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, noaEntity)
		}

		return entity, nil
	})
}

func (r *noaRepository) GetOpen(
	ctx context.Context,
	req factoringports.GetOpenNoaRequest,
) (*factoring.Noa, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Noa, error) {
		cols := factoringcol.NoaColumns
		entity := new(factoring.Noa)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.NoaScopeTenant(sq, req.TenantInfo).
					Where(cols.CustomerID.Eq(), req.CustomerID).
					Where(cols.FactoringCompanyID.Eq(), req.FactoringCompanyID).
					Where(cols.Status.Ne(), factoring.NoaStatusReleased)
			}).
			Limit(1).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, noaEntity)
		}

		return entity, nil
	})
}

func (r *noaRepository) ListOpen(
	ctx context.Context,
	req *factoringports.ListOpenNoasRequest,
) (map[factoringports.NoaPair]*factoring.Noa, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[factoringports.NoaPair]*factoring.Noa, error) {
			result := make(map[factoringports.NoaPair]*factoring.Noa, len(req.Pairs))
			if len(req.Pairs) == 0 {
				return result, nil
			}

			cols := factoringcol.NoaColumns
			rows := make([]*factoring.Noa, 0, len(req.Pairs))
			if err := r.db.DBForContext(ctx).NewSelect().
				Model(&rows).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return factoringcol.NoaScopeTenant(sq, req.TenantInfo).
						Where(cols.Status.Ne(), factoring.NoaStatusReleased).
						Where(
							"("+cols.CustomerID.Qualified()+", "+cols.FactoringCompanyID.Qualified()+") IN (?)",
							bun.In(pairValues(req.Pairs)),
						)
				}).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("list open notices of assignment: %w", err)
			}

			for _, row := range rows {
				result[factoringports.NoaPair{
					CustomerID:         row.CustomerID,
					FactoringCompanyID: row.FactoringCompanyID,
				}] = row
			}

			return result, nil
		},
	)
}

func (r *noaRepository) ListConnection(
	ctx context.Context,
	req *factoringports.ListNoaConnectionRequest,
) (*pagination.CursorListResult[*factoring.Noa], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.Noa], error) {
			dba := r.db.DBForContext(ctx)
			alias := factoringcol.NoaTable.Alias

			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.NewSelect().
					Model((*factoring.Noa)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.scope(
							querybuilder.ApplyFiltersWithoutSort(sq, alias, req.Filter, (*factoring.Noa)(nil)),
							req,
						)
					}).
					Count(ctx)
				if err != nil {
					return nil, fmt.Errorf("count notices of assignment: %w", err)
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*factoring.Noa]{
				Filter:     req.Filter,
				Cursor:     req.Cursor,
				TotalCount: totalCount,
				Query: func(entities *[]*factoring.Noa) *bun.SelectQuery {
					return dba.NewSelect().
						Model(entities).
						ColumnExpr(factoringcol.NoaTable.All())
				},
				Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
					q, applyErr := querybuilder.ApplyCursorFilters(
						sq,
						alias,
						req.Filter,
						req.Cursor,
						(*factoring.Noa)(nil),
					)
					if applyErr != nil {
						return nil, applyErr
					}
					return r.scope(q, req), nil
				},
			})
			if err != nil {
				return nil, fmt.Errorf("list notices of assignment: %w", err)
			}

			return result, nil
		},
	)
}

func (r *noaRepository) scope(
	q *bun.SelectQuery,
	req *factoringports.ListNoaConnectionRequest,
) *bun.SelectQuery {
	cols := factoringcol.NoaColumns
	if !req.CustomerID.IsNil() {
		q = q.Where(cols.CustomerID.Eq(), req.CustomerID)
	}
	if !req.FactoringCompanyID.IsNil() {
		q = q.Where(cols.FactoringCompanyID.Eq(), req.FactoringCompanyID)
	}
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	return q
}

func (r *noaRepository) ListExpiring(
	ctx context.Context,
	req factoringports.ListExpiringNoasRequest,
) ([]*factoring.Noa, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoring.Noa, error) {
		cols := factoringcol.NoaColumns
		limit := req.Limit
		if limit <= 0 || limit > maxExpiringNoas {
			limit = maxExpiringNoas
		}

		rows := make([]*factoring.Noa, 0, limit)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.NoaScopeTenant(sq, req.TenantInfo).
					Where(cols.Status.Eq(), factoring.NoaStatusAcknowledged).
					Where(cols.ExpiresAt.IsNotNull()).
					Where(cols.ExpiresAt.Gt(), req.At).
					Where(cols.ExpiresAt.Lte(), req.Before).
					Where(cols.ExpiryWarnedAt.IsNull())
			}).
			Order(cols.ExpiresAt.OrderAsc(), cols.ID.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list expiring notices of assignment: %w", err)
		}

		return rows, nil
	})
}

func (r *noaRepository) ClaimExpiryWarning(
	ctx context.Context,
	req factoringports.ClaimNoaExpiryWarningRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		cols := factoringcol.NoaColumns
		res, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*factoring.Noa)(nil)).
			Set(cols.ExpiryWarnedAt.Set(), req.At).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return factoringcol.NoaScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.ExpiryWarnedAt.IsNull())
			}).
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("claim notice expiry warning: %w", err)
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("claim notice expiry warning: %w", err)
		}

		return rows > 0, nil
	})
}

func pairValues(pairs []factoringports.NoaPair) [][]any {
	values := make([][]any, 0, len(pairs))
	for _, pair := range pairs {
		values = append(values, []any{pair.CustomerID, pair.FactoringCompanyID})
	}
	return values
}

func translateNoaError(err error, action string) error {
	if dberror.IsUniqueConstraintViolation(err) && dberror.ExtractConstraintName(err) == openNoaIndex {
		return errortypes.NewConflictError(
			"This customer already has an open notice of assignment for this factor",
		)
	}
	return fmt.Errorf("%s: %w", action, err)
}
