//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edirepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

func (r *repository) ListPartnerSettingSchemas(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingSchemasRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingSchema], error) {
	entities := make([]*edi.EDIPartnerSettingSchema, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterPartnerSettingSchemasQuery(query, req)
		}).
		OrderExpr(
			"epss.organization_id IS NOT NULL DESC, epss.schema_version DESC, epss.created_at DESC",
		).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDIPartnerSettingSchema]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetPartnerSettingSchema(
	ctx context.Context,
	req repositories.GetEDIPartnerSettingSchemaRequest,
) (*edi.EDIPartnerSettingSchema, error) {
	entity := new(edi.EDIPartnerSettingSchema)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("epss.id = ?", req.ID).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return partnerSettingSchemaTenantScope(query, req.TenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartnerSettingSchema")
	}
	return entity, nil
}

func (r *repository) GetActivePartnerSettingSchema(
	ctx context.Context,
	req repositories.GetActiveEDIPartnerSettingSchemaRequest,
) (*edi.EDIPartnerSettingSchema, error) {
	entity := new(edi.EDIPartnerSettingSchema)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("epss.standard = ?", req.Standard).
		Where("epss.transaction_set = ?", req.TransactionSet).
		Where("epss.direction = ?", req.Direction).
		Where("epss.x12_version = ?", req.X12Version).
		Where("epss.status = ?", edi.PartnerSettingStatusActive).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return partnerSettingSchemaTenantScope(sq, req.TenantInfo)
		}).
		OrderExpr("epss.organization_id IS NOT NULL DESC, epss.schema_version DESC").
		Limit(1)
	if req.DocumentTypeID.IsNotNil() {
		query = query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr("epss.document_type_id = ?", req.DocumentTypeID).
				WhereOr("epss.document_type_id IS NULL")
		}).OrderExpr("epss.document_type_id IS NOT NULL DESC")
	}
	if req.SchemaVersion > 0 {
		query = query.Where("epss.schema_version = ?", req.SchemaVersion)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartnerSettingSchema")
	}
	return entity, nil
}

func (r *repository) ListPartnerSettingFields(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return r.searchPartnerSettingFields(ctx, req)
}

func (r *repository) SearchPartnerSettingFields(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return r.searchPartnerSettingFields(ctx, req)
}

func (r *repository) SelectPartnerSettingFieldOptions(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return r.searchPartnerSettingFields(ctx, req)
}

func (r *repository) searchPartnerSettingFields(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	entities := make([]*edi.EDIPartnerSettingField, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Join("JOIN edi_partner_setting_schemas AS epss ON epss.id = epsf.schema_id").
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterPartnerSettingFieldsQuery(query, req)
		}).
		Order("epsf.display_order ASC", "epsf.path ASC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDIPartnerSettingField]{
		Items: entities,
		Total: total,
	}, nil
}

func filterPartnerSettingSchemasQuery(
	query *bun.SelectQuery,
	req *repositories.ListEDIPartnerSettingSchemasRequest,
) *bun.SelectQuery {
	query = partnerSettingSchemaTenantScope(query, req.Filter.TenantInfo)
	if req.Standard != "" {
		query = query.Where("epss.standard = ?", req.Standard)
	}
	if req.TransactionSet != "" {
		query = query.Where("epss.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("epss.direction = ?", req.Direction)
	}
	if req.X12Version != "" {
		query = query.Where("epss.x12_version = ?", req.X12Version)
	}
	if req.DocumentTypeID.IsNotNil() {
		query = query.Where("epss.document_type_id = ?", req.DocumentTypeID)
	}
	if req.SchemaVersion > 0 {
		query = query.Where("epss.schema_version = ?", req.SchemaVersion)
	}
	if req.Status != "" {
		query = query.Where("epss.status = ?", req.Status)
	}
	if strings.TrimSpace(req.Filter.Query) == "" {
		return query
	}

	term := "%" + strings.TrimSpace(req.Filter.Query) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("epss.name ILIKE ?", term).
			WhereOr("epss.description ILIKE ?", term).
			WhereOr("epss.x12_version ILIKE ?", term)
	})
}

func filterPartnerSettingFieldsQuery(
	query *bun.SelectQuery,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) *bun.SelectQuery {
	query = partnerSettingSchemaTenantScope(query, req.Filter.TenantInfo)
	if req.SchemaID.IsNotNil() {
		query = query.Where("epsf.schema_id = ?", req.SchemaID)
	}
	if req.Standard != "" {
		query = query.Where("epss.standard = ?", req.Standard)
	}
	if req.TransactionSet != "" {
		query = query.Where("epss.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("epss.direction = ?", req.Direction)
	}
	if req.Status != "" {
		query = query.Where("epsf.status = ?", req.Status)
	}
	if strings.TrimSpace(req.PathPrefix) != "" {
		query = query.Where("epsf.path LIKE ?", strings.TrimSpace(req.PathPrefix)+"%")
	}
	if strings.TrimSpace(req.GroupKey) != "" {
		query = query.Where("epsf.group_key = ?", strings.TrimSpace(req.GroupKey))
	}
	if req.Required != nil {
		query = query.Where("epsf.required = ?", *req.Required)
	}
	if req.Secret != nil {
		query = query.Where("epsf.secret = ?", *req.Secret)
	}
	if strings.TrimSpace(req.Filter.Query) == "" {
		return query
	}

	term := "%" + strings.TrimSpace(req.Filter.Query) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("epsf.path ILIKE ?", term).
			WhereOr("epsf.label ILIKE ?", term).
			WhereOr("epsf.description ILIKE ?", term).
			WhereOr("epsf.group_key ILIKE ?", term)
	})
}

func partnerSettingSchemaTenantScope(
	query *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(
			"(epss.organization_id = ? AND epss.business_unit_id = ?)",
			tenantInfo.OrgID,
			tenantInfo.BuID,
		).WhereOr("(epss.organization_id IS NULL AND epss.business_unit_id IS NULL)")
	})
}
