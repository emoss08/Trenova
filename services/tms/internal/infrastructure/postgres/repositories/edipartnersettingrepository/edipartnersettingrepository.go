//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edipartnersettingrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.EDIPartnerSettingRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-partner-setting-repository"),
	}
}

func (r *repository) ListPartnerSettingSchemas(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingSchemasRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingSchema], error) {
	entities := make([]*edi.EDIPartnerSettingSchema, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDIPartnerSettingSchemaColumns

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterPartnerSettingSchemasQuery(query, req)
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
	cols := buncolgen.EDIPartnerSettingSchemaColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
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
	cols := buncolgen.EDIPartnerSettingSchemaColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.Standard.Eq(), req.Standard).
		Where(cols.TransactionSet.Eq(), req.TransactionSet).
		Where(cols.Direction.Eq(), req.Direction).
		Where(cols.X12Version.Eq(), req.X12Version).
		Where(cols.Status.Eq(), edi.PartnerSettingStatusActive).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return partnerSettingSchemaTenantScope(sq, req.TenantInfo)
		}).
		OrderExpr(cols.OrganizationID.IsNotNull() + " DESC, " + cols.SchemaVersion.OrderDesc()).
		Limit(1)
	if req.DocumentTypeID.IsNotNil() {
		query = query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(cols.DocumentTypeID.Eq(), req.DocumentTypeID).
				WhereOr(cols.DocumentTypeID.IsNull())
		}).OrderExpr(cols.DocumentTypeID.IsNotNull() + " DESC")
	}
	if req.SchemaVersion > 0 {
		query = query.Where(cols.SchemaVersion.Eq(), req.SchemaVersion)
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
	fieldCols := buncolgen.EDIPartnerSettingFieldColumns

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Join(partnerSettingSchemaJoin()).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterPartnerSettingFieldsQuery(query, req)
		}).
		Order(fieldCols.DisplayOrder.OrderAsc(), fieldCols.Path.OrderAsc()).
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
	cols := buncolgen.EDIPartnerSettingSchemaColumns

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
	if req.DocumentTypeID.IsNotNil() {
		query = query.Where(cols.DocumentTypeID.Eq(), req.DocumentTypeID)
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
			WhereOr(cols.Description.ILike(), term).
			WhereOr(cols.X12Version.ILike(), term)
	})
}

func filterPartnerSettingFieldsQuery(
	query *bun.SelectQuery,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) *bun.SelectQuery {
	query = partnerSettingSchemaTenantScope(query, req.Filter.TenantInfo)
	fieldCols := buncolgen.EDIPartnerSettingFieldColumns
	schemaCols := buncolgen.EDIPartnerSettingSchemaColumns

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
	if strings.TrimSpace(req.PathPrefix) != "" {
		query = query.Where(fieldCols.Path.Like(), strings.TrimSpace(req.PathPrefix)+"%")
	}
	if strings.TrimSpace(req.GroupKey) != "" {
		query = query.Where(fieldCols.GroupKey.Eq(), strings.TrimSpace(req.GroupKey))
	}
	if req.Required != nil {
		query = query.Where(fieldCols.Required.Eq(), *req.Required)
	}
	if req.Secret != nil {
		query = query.Where(fieldCols.Secret.Eq(), *req.Secret)
	}
	if strings.TrimSpace(req.Filter.Query) == "" {
		return query
	}

	term := "%" + strings.TrimSpace(req.Filter.Query) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(fieldCols.Path.ILike(), term).
			WhereOr(fieldCols.Label.ILike(), term).
			WhereOr(fieldCols.Description.ILike(), term).
			WhereOr(fieldCols.GroupKey.ILike(), term)
	})
}

func partnerSettingSchemaTenantScope(
	query *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	cols := buncolgen.EDIPartnerSettingSchemaColumns

	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(
			buncolgen.Expr("({0} = ? AND {1} = ?)", cols.OrganizationID, cols.BusinessUnitID),
			tenantInfo.OrgID,
			tenantInfo.BuID,
		).WhereOr("(" + cols.OrganizationID.IsNull() + " AND " + cols.BusinessUnitID.IsNull() + ")")
	})
}

func partnerSettingSchemaJoin() string {
	schemaCols := buncolgen.EDIPartnerSettingSchemaColumns
	fieldCols := buncolgen.EDIPartnerSettingFieldColumns

	return "JOIN " + buncolgen.EDIPartnerSettingSchemaTable.As(
		buncolgen.EDIPartnerSettingSchemaTable.Alias,
	) +
		" ON " + schemaCols.ID.EqColumn(
		fieldCols.SchemaID,
	)
}
