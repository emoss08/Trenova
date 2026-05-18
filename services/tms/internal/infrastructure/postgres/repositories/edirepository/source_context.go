package edirepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
)

func (r *repository) ListSourceContextSchemas(
	ctx context.Context,
	req *repositories.ListEDISourceContextSchemasRequest,
) (*pagination.ListResult[*edi.EDISourceContextSchema], error) {
	entities := make([]*edi.EDISourceContextSchema, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterSourceContextSchemasQuery(query, req)
		}).
		OrderExpr(
			"escs.organization_id IS NOT NULL DESC, escs.schema_version DESC, escs.created_at DESC",
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("escs.id = ?", req.ID).
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
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("escs.standard = ?", req.Standard).
		Where("escs.transaction_set = ?", req.TransactionSet).
		Where("escs.direction = ?", req.Direction).
		Where("escs.x12_version = ?", req.X12Version).
		Where("escs.context_key = ?", stringutils.FirstNonEmpty(req.ContextKey, "loadTender")).
		Where("escs.status = ?", edi.SourceContextFieldStatusActive).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sourceContextSchemaTenantScope(sq, req.TenantInfo)
		}).
		OrderExpr("escs.organization_id IS NOT NULL DESC, escs.schema_version DESC").
		Limit(1)
	if req.SchemaVersion > 0 {
		query = query.Where("escs.schema_version = ?", req.SchemaVersion)
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

func (r *repository) searchSourceContextFields(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	entities := make([]*edi.EDISourceContextField, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Join("JOIN edi_source_context_schemas AS escs ON escs.id = escf.schema_id").
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterSourceContextFieldsQuery(query, req)
		}).
		Order("escf.path ASC", "escf.repeat_path ASC").
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
	if req.Standard != "" {
		query = query.Where("escs.standard = ?", req.Standard)
	}
	if req.TransactionSet != "" {
		query = query.Where("escs.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("escs.direction = ?", req.Direction)
	}
	if req.X12Version != "" {
		query = query.Where("escs.x12_version = ?", req.X12Version)
	}
	if req.ContextKey != "" {
		query = query.Where("escs.context_key = ?", req.ContextKey)
	}
	if req.SchemaVersion > 0 {
		query = query.Where("escs.schema_version = ?", req.SchemaVersion)
	}
	if req.Status != "" {
		query = query.Where("escs.status = ?", req.Status)
	}
	if strings.TrimSpace(req.Filter.Query) == "" {
		return query
	}

	term := "%" + strings.TrimSpace(req.Filter.Query) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("escs.name ILIKE ?", term).
			WhereOr("escs.context_key ILIKE ?", term).
			WhereOr("escs.description ILIKE ?", term)
	})
}

func filterSourceContextFieldsQuery(
	query *bun.SelectQuery,
	req *repositories.ListEDISourceContextFieldsRequest,
) *bun.SelectQuery {
	query = sourceContextSchemaTenantScope(query, req.Filter.TenantInfo)
	if req.SchemaID.IsNotNil() {
		query = query.Where("escf.schema_id = ?", req.SchemaID)
	}
	if req.Status != "" {
		query = query.Where("escf.status = ?", req.Status)
	}
	if req.SourceKind != "" {
		query = query.Where("escf.source_kind = ?", req.SourceKind)
	}
	if req.Repeated != nil {
		query = query.Where("escf.repeated = ?", *req.Repeated)
	}
	if strings.TrimSpace(req.PathPrefix) != "" {
		query = query.Where("escf.path LIKE ?", strings.TrimSpace(req.PathPrefix)+"%")
	}
	if strings.TrimSpace(req.Filter.Query) == "" {
		return query
	}

	term := "%" + strings.TrimSpace(req.Filter.Query) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("escf.path ILIKE ?", term).
			WhereOr("escf.display_name ILIKE ?", term).
			WhereOr("escf.description ILIKE ?", term)
	})
}

func sourceContextSchemaTenantScope(
	query *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(
			"(escs.organization_id = ? AND escs.business_unit_id = ?)",
			tenantInfo.OrgID,
			tenantInfo.BuID,
		).WhereOr("(escs.organization_id IS NULL AND escs.business_unit_id IS NULL)")
	})
}
