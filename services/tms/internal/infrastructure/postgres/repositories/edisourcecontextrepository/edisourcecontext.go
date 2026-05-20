//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edisourcecontextrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
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

func New(p Params) repositories.EDISourceContextRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-source-context-repository"),
	}
}

func (r *repository) ListSourceContextSchemas(
	ctx context.Context,
	req *repositories.ListEDISourceContextSchemasRequest,
) (*pagination.ListResult[*edi.EDISourceContextSchema], error) {
	entities := make([]*edi.EDISourceContextSchema, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDISourceContextSchemaColumns

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterSourceContextSchemasQuery(query, req)
		}).
		OrderExpr(
			cols.OrganizationID.IsNotNull() + " DESC, " +
				cols.SchemaVersion.OrderDesc() + ", " +
				cols.CreatedAt.OrderDesc(),
		).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDISourceContextSchema]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetSourceContextSchema(
	ctx context.Context,
	req repositories.GetEDISourceContextSchemaRequest,
) (*edi.EDISourceContextSchema, error) {
	entity := new(edi.EDISourceContextSchema)
	cols := buncolgen.EDISourceContextSchemaColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return sourceContextSchemaTenantScope(query, req.TenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDISourceContextSchema")
	}
	return entity, nil
}

//nolint:gocritic // EDI repository request structs are passed by value consistently.
func (r *repository) GetActiveSourceContextSchema(
	ctx context.Context,
	req repositories.GetActiveEDISourceContextSchemaRequest,
) (*edi.EDISourceContextSchema, error) {
	entity := new(edi.EDISourceContextSchema)
	cols := buncolgen.EDISourceContextSchemaColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.Standard.Eq(), req.Standard).
		Where(cols.TransactionSet.Eq(), req.TransactionSet).
		Where(cols.Direction.Eq(), req.Direction).
		Where(cols.X12Version.Eq(), req.X12Version).
		Where(cols.ContextKey.Eq(), stringutils.FirstNonEmpty(req.ContextKey, "loadTender")).
		Where(cols.Status.Eq(), edi.SourceContextFieldStatusActive).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sourceContextSchemaTenantScope(sq, req.TenantInfo)
		}).
		OrderExpr(cols.OrganizationID.IsNotNull() + " DESC, " + cols.SchemaVersion.OrderDesc()).
		Limit(1)
	if req.SchemaVersion > 0 {
		query = query.Where(cols.SchemaVersion.Eq(), req.SchemaVersion)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDISourceContextSchema")
	}
	return entity, nil
}

func (r *repository) ListSourceContextFields(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return r.searchSourceContextFields(ctx, req)
}

func (r *repository) SearchSourceContextFields(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return r.searchSourceContextFields(ctx, req)
}

func (r *repository) SelectSourceContextFieldOptions(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return r.searchSourceContextFields(ctx, req)
}

func (r *repository) searchSourceContextFields(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	entities := make([]*edi.EDISourceContextField, 0, req.Filter.Pagination.SafeLimit())
	fieldCols := buncolgen.EDISourceContextFieldColumns

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Join(sourceContextSchemaJoin()).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterSourceContextFieldsQuery(query, req)
		}).
		Order(fieldCols.Path.OrderAsc(), fieldCols.RepeatPath.OrderAsc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDISourceContextField]{
		Items: entities,
		Total: total,
	}, nil
}

func filterSourceContextSchemasQuery(
	query *bun.SelectQuery,
	req *repositories.ListEDISourceContextSchemasRequest,
) *bun.SelectQuery {
	query = sourceContextSchemaTenantScope(query, req.Filter.TenantInfo)
	cols := buncolgen.EDISourceContextSchemaColumns

	if req.Standard != "" {
		query = query.Where(cols.Standard.Eq(), req.Standard)
	}
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.X12Version != "" {
		query = query.Where(cols.X12Version.Eq(), req.X12Version)
	}
	if req.ContextKey != "" {
		query = query.Where(cols.ContextKey.Eq(), req.ContextKey)
	}
	if req.SchemaVersion > 0 {
		query = query.Where(cols.SchemaVersion.Eq(), req.SchemaVersion)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if strings.TrimSpace(req.Filter.Query) == "" {
		return query
	}

	term := "%" + strings.TrimSpace(req.Filter.Query) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(cols.Name.ILike(), term).
			WhereOr(cols.ContextKey.ILike(), term).
			WhereOr(cols.Description.ILike(), term)
	})
}

func filterSourceContextFieldsQuery(
	query *bun.SelectQuery,
	req *repositories.ListEDISourceContextFieldsRequest,
) *bun.SelectQuery {
	query = sourceContextSchemaTenantScope(query, req.Filter.TenantInfo)
	fieldCols := buncolgen.EDISourceContextFieldColumns
	schemaCols := buncolgen.EDISourceContextSchemaColumns

	if req.SchemaID.IsNotNil() {
		query = query.Where(fieldCols.SchemaID.Eq(), req.SchemaID)
	}
	if req.Standard != "" {
		query = query.Where(schemaCols.Standard.Eq(), req.Standard)
	}
	if req.TransactionSet != "" {
		query = query.Where(schemaCols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(schemaCols.Direction.Eq(), req.Direction)
	}
	if req.Status != "" {
		query = query.Where(fieldCols.Status.Eq(), req.Status)
	}
	if req.SourceKind != "" {
		query = query.Where(fieldCols.SourceKind.Eq(), req.SourceKind)
	}
	if req.Repeated != nil {
		query = query.Where(fieldCols.Repeated.Eq(), *req.Repeated)
	}
	if strings.TrimSpace(req.PathPrefix) != "" {
		query = query.Where(fieldCols.Path.Like(), strings.TrimSpace(req.PathPrefix)+"%")
	}
	if strings.TrimSpace(req.Filter.Query) == "" {
		return query
	}

	term := "%" + strings.TrimSpace(req.Filter.Query) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(fieldCols.Path.ILike(), term).
			WhereOr(fieldCols.DisplayName.ILike(), term).
			WhereOr(fieldCols.Description.ILike(), term).
			WhereOr(fieldCols.DataType.TextILike(), term)
	})
}

func sourceContextSchemaTenantScope(
	query *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	cols := buncolgen.EDISourceContextSchemaColumns

	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(
			buncolgen.Expr("({0} = ? AND {1} = ?)", cols.OrganizationID, cols.BusinessUnitID),
			tenantInfo.OrgID,
			tenantInfo.BuID,
		).WhereOr("(" + cols.OrganizationID.IsNull() + " AND " + cols.BusinessUnitID.IsNull() + ")")
	})
}

func sourceContextSchemaJoin() string {
	schemaCols := buncolgen.EDISourceContextSchemaColumns
	fieldCols := buncolgen.EDISourceContextFieldColumns

	return "JOIN " + buncolgen.EDISourceContextSchemaTable.As(
		buncolgen.EDISourceContextSchemaTable.Alias,
	) +
		" ON " + schemaCols.ID.EqColumn(
		fieldCols.SchemaID,
	)
}
