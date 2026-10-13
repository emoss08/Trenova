package factoringcompanyrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcompanycol"
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

func New(p Params) factoringcompanyports.FactoringCompanyRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.factoring-company-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *factoringcompanyports.ListFactoringCompanyRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		factoringcompanycol.FactoringCompanyTable.Alias,
		req.Filter,
		(*factoringcompany.FactoringCompany)(nil),
	)

	return q.Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset()))
}

func (r *repository) List(
	ctx context.Context,
	req *factoringcompanyports.ListFactoringCompanyRequest,
) (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
			log := r.l.With(
				zap.String("operation", "List"),
				zap.Any("request", req),
			)

			entities := make(
				[]*factoringcompany.FactoringCompany,
				0,
				req.Filter.Pagination.SafeLimit(),
			)
			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.filterQuery(sq, req)
				}).ScanAndCount(ctx)
			if err != nil {
				log.Error("failed to scan and count factoring companies", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*factoringcompany.FactoringCompany]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *factoringcompanyports.ListFactoringCompanyConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		factoringcompanycol.FactoringCompanyTable.Alias,
		req.Filter,
		req.Cursor,
		(*factoringcompany.FactoringCompany)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *factoringcompanyports.ListFactoringCompanyConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		factoringcompanycol.FactoringCompanyTable.Alias,
		req.Filter,
		(*factoringcompany.FactoringCompany)(nil),
	)
}

func applyFactoringCompanyColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(factoringcompanycol.FactoringCompanyTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *factoringcompanyports.ListFactoringCompanyConnectionRequest,
) (*pagination.CursorListResult[*factoringcompany.FactoringCompany], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoringcompany.FactoringCompany], error) {
			log := r.l.With(
				zap.String("operation", "ListConnection"),
				zap.Any("request", req),
			)

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*factoringcompany.FactoringCompany)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.applyTotalCountFilters(sq, req)
					}).
					Count(ctx)
				if err != nil {
					log.Error("failed to count factoring companies", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*factoringcompany.FactoringCompany]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*factoringcompany.FactoringCompany) *bun.SelectQuery {
						return dba.
							NewSelect().
							Model(entities).
							Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
								return applyFactoringCompanyColumns(sq, req.FactoringCompanyColumns)
							})
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return r.applyCursorPageFilters(sq, req)
					},
				})
			if err != nil {
				log.Error("failed to scan factoring companies", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req factoringcompanyports.GetFactoringCompanyByIDRequest,
) (*factoringcompany.FactoringCompany, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringcompany.FactoringCompany, error) {
			log := r.l.With(
				zap.String("operation", "GetByID"),
				zap.String("id", req.ID.String()),
			)

			entity := new(factoringcompany.FactoringCompany)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = factoringcompanycol.FactoringCompanyScopeTenant(sq, req.TenantInfo).
						Where(factoringcompanycol.FactoringCompanyColumns.ID.Eq(), req.ID)
					if req.IncludeState {
						sq = sq.Relation("State")
					}
					return sq
				}).
				Scan(ctx)
			if err != nil {
				log.Error("failed to get factoring company", zap.Error(err))
				return nil, dberror.HandleNotFoundError(err, "FactoringCompany")
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *factoringcompany.FactoringCompany,
) (*factoringcompany.FactoringCompany, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringcompany.FactoringCompany, error) {
			log := r.l.With(
				zap.String("operation", "Create"),
			)

			if _, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx); err != nil {
				log.Error("failed to create factoring company", zap.Error(err))
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *factoringcompany.FactoringCompany,
) (*factoringcompany.FactoringCompany, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringcompany.FactoringCompany, error) {
			log := r.l.With(
				zap.String("operation", "Update"),
				zap.String("id", entity.ID.String()),
			)

			ov := entity.Version
			entity.Version++

			results, err := r.db.DBForContext(ctx).
				NewUpdate().
				Model(entity).
				ExcludeColumn(factoringcompanycol.FactoringCompanyColumns.CreatedAt.Bare()).
				WherePK().
				Where("version = ?", ov).
				Returning("*").
				Exec(ctx)
			if err != nil {
				log.Error("failed to update factoring company", zap.Error(err))
				return nil, err
			}

			if err = dberror.CheckRowsAffected(
				results,
				"FactoringCompany",
				entity.ID.String(),
			); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *factoringcompanyports.FactoringCompanySelectOptionsRequest,
) (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
			cols := factoringcompanycol.FactoringCompanyColumns
			return dbhelper.SelectOptions[*factoringcompany.FactoringCompany](
				ctx,
				r.db.DBForContext(ctx),
				req.SelectQueryRequest,
				&dbhelper.SelectOptionsConfig{
					Columns: []string{
						cols.ID.Bare(),
						cols.Code.Bare(),
						cols.Name.Bare(),
						cols.City.Bare(),
						cols.Status.Bare(),
					},
					OrgColumn: cols.OrganizationID.Qualified(),
					BuColumn:  cols.BusinessUnitID.Qualified(),
					QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
						return q.Where(cols.Status.Eq(), domaintypes.StatusActive)
					},
					EntityName: "FactoringCompany",
					SearchColumns: []string{
						cols.Code.Qualified(),
						cols.Name.Qualified(),
					},
				},
			)
		},
	)
}

func (r *repository) CountAssignedCustomers(
	ctx context.Context,
	req factoringcompanyports.CountAssignedCustomersRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := customercol.CustomerBillingProfileColumns
		count, err := customercol.CustomerBillingProfileScopeTenant(
			r.db.DBForContext(ctx).
				NewSelect().
				Model((*customer.CustomerBillingProfile)(nil)),
			req.TenantInfo,
		).
			Where(cols.FactoringCompanyID.Eq(), req.FactoringCompanyID).
			Count(ctx)
		if err != nil {
			return 0, fmt.Errorf("count customers assigned to a factoring company: %w", err)
		}
		return int(count), nil
	})
}
