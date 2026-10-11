package documenttyperepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/documenttypecol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
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

func New(p Params) documenttypeports.DocumentTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-type-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *documenttypeports.ListDocumentTypesRequest,
) *bun.SelectQuery {
	cols := documenttypecol.DocumentTypeColumns
	q = querybuilder.ApplyFilters(
		q,
		documenttypecol.DocumentTypeTable.Alias,
		req.Filter,
		(*documenttype.DocumentType)(nil),
	)

	q = q.Order(cols.CreatedAt.OrderDesc())

	return q.Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset()))
}

func (r *repository) List(
	ctx context.Context,
	req *documenttypeports.ListDocumentTypesRequest,
) (*pagination.ListResult[*documenttype.DocumentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*documenttype.DocumentType], error) {
			log := r.l.With(
				zap.String("operation", "List"),
				zap.Any("request", req),
			)

			entities := make([]*documenttype.DocumentType, 0, req.Filter.Pagination.SafeLimit())
			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.filterQuery(sq, req)
				}).ScanAndCount(ctx)
			if err != nil {
				log.Error("failed to scan and count document types", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*documenttype.DocumentType]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *documenttypeports.ListDocumentTypeConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		documenttypecol.DocumentTypeTable.Alias,
		req.Filter,
		req.Cursor,
		(*documenttype.DocumentType)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *documenttypeports.ListDocumentTypeConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		documenttypecol.DocumentTypeTable.Alias,
		req.Filter,
		(*documenttype.DocumentType)(nil),
	)
}

func applyDocumentTypeColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(documenttypecol.DocumentTypeTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *documenttypeports.ListDocumentTypeConnectionRequest,
) (*pagination.CursorListResult[*documenttype.DocumentType], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*documenttype.DocumentType], error) {
			log := r.l.With(
				zap.String("operation", "ListConnection"),
				zap.Any("request", req),
			)

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*documenttype.DocumentType)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.applyTotalCountFilters(sq, req)
					}).
					Count(ctx)
				if err != nil {
					log.Error("failed to count document types", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*documenttype.DocumentType]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*documenttype.DocumentType) *bun.SelectQuery {
						return dba.
							NewSelect().
							Model(entities).
							Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
								return applyDocumentTypeColumns(sq, req.DocumentTypeColumns)
							})
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return r.applyCursorPageFilters(sq, req)
					},
				})
			if err != nil {
				log.Error("failed to scan document types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *documenttype.DocumentType,
) (*documenttype.DocumentType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*documenttype.DocumentType, error) {
		log := r.l.With(
			zap.String("operation", "Create"),
			zap.String("name", entity.Name),
		)

		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			log.Error("failed to create document type", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *documenttype.DocumentType,
) (*documenttype.DocumentType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*documenttype.DocumentType, error) {
		log := r.l.With(
			zap.String("operation", "Update"),
			zap.String("id", entity.ID.String()),
		)

		ov := entity.Version
		entity.Version++
		cols := documenttypecol.DocumentTypeColumns

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).WherePK().
			Where(cols.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to update document type", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(
			results,
			"DocumentType",
			entity.ID.String(),
		); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req documenttypeports.GetDocumentTypeByIDRequest,
) (*documenttype.DocumentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*documenttype.DocumentType, error) {
		log := r.l.With(
			zap.String("operation", "GetByID"),
			zap.String("id", req.ID.String()),
		)

		entity := new(documenttype.DocumentType)
		cols := documenttypecol.DocumentTypeColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return documenttypecol.DocumentTypeScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get document type", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "DocumentType")
		}

		return entity, nil
	})
}

func (r *repository) GetByCode(
	ctx context.Context,
	req documenttypeports.GetDocumentTypeByCodeRequest,
) (*documenttype.DocumentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*documenttype.DocumentType, error) {
		log := r.l.With(
			zap.String("operation", "GetByCode"),
			zap.String("code", req.Code),
		)

		entity := new(documenttype.DocumentType)
		cols := documenttypecol.DocumentTypeColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return documenttypecol.DocumentTypeScopeTenant(sq, req.TenantInfo).
					Where(cols.Code.Eq(), req.Code)
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get document type by code", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "DocumentType")
		}

		return entity, nil
	})
}

func (r *repository) GetByName(
	ctx context.Context,
	req documenttypeports.GetDocumentTypeByNameRequest,
) (*documenttype.DocumentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*documenttype.DocumentType, error) {
		log := r.l.With(
			zap.String("operation", "GetByName"),
			zap.String("name", req.Name),
		)

		entity := new(documenttype.DocumentType)
		normalizedName := strings.TrimSpace(req.Name)
		cols := documenttypecol.DocumentTypeColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return documenttypecol.DocumentTypeScopeTenant(sq, req.TenantInfo).
					Where(cols.Name.Expr("lower(btrim({})) = lower(btrim(?))"), normalizedName)
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get document type by name", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "DocumentType")
		}

		return entity, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*documenttype.DocumentType], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*documenttype.DocumentType], error) {
			cols := documenttypecol.DocumentTypeColumns
			return dbhelper.SelectOptions[*documenttype.DocumentType](
				ctx,
				r.db.DBForContext(ctx),
				req,
				&dbhelper.SelectOptionsConfig{
					Columns: []string{
						documenttypecol.DocumentTypeColumns.ID.Bare(),
						documenttypecol.DocumentTypeColumns.Code.Bare(),
						documenttypecol.DocumentTypeColumns.Name.Bare(),
						documenttypecol.DocumentTypeColumns.Description.Bare(),
						documenttypecol.DocumentTypeColumns.Color.Bare(),
						documenttypecol.DocumentTypeColumns.DocumentClassification.Bare(),
						documenttypecol.DocumentTypeColumns.DocumentCategory.Bare(),
						documenttypecol.DocumentTypeColumns.IsSystem.Bare(),
					},
					OrgColumn:  documenttypecol.DocumentTypeColumns.OrganizationID.Qualified(),
					BuColumn:   documenttypecol.DocumentTypeColumns.BusinessUnitID.Qualified(),
					EntityName: "DocumentType",
					SearchColumns: []string{
						documenttypecol.DocumentTypeColumns.Code.Qualified(),
						documenttypecol.DocumentTypeColumns.Name.Qualified(),
					},
					QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
						// exclude INVOICE, CREDITMEMO, DEBITMEMO
						// These document types are system document types and should not be set by the user
						return q.WhereGroup("AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
							return sq.Where(
								cols.Code.NotIn(),
								bun.List([]string{"INVOICE", "CREDITMEMO", "DEBITMEMO"}),
							)
						})
					},
				},
			)
		},
	)
}
