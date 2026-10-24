//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edipartnersettingrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var schemaErrors = dberror.Mapping{Entity: "EDIPartnerSettingSchema"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediports.EDIPartnerSettingRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-partner-setting-repository"),
	}
}

func (r *repository) ListPartnerSettingSchemas(
	ctx context.Context,
	req *ediports.ListEDIPartnerSettingSchemasRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingSchema], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIPartnerSettingSchema], error) {
			return listquery.Offset(
				ctx,
				r.db.Keel(ctx),
				schemasQuery(req),
				req.Filter.Pagination,
			)
		},
	)
}

func schemasQuery(
	req *ediports.ListEDIPartnerSettingSchemasRequest,
) *keel.Query[edi.EDIPartnerSettingSchema] {
	ps := edidb.EDIPartnerSettingSchemas
	search := strings.TrimSpace(req.Filter.Query)

	return ps.Where(
		visibleSchema(req.Filter.TenantInfo),
		keel.When(req.Standard != "", ps.Standard.Eq(req.Standard)),
		keel.When(req.TransactionSet != "", ps.TransactionSet.Eq(req.TransactionSet)),
		keel.When(req.Direction != "", ps.Direction.Eq(req.Direction)),
		keel.When(req.X12Version != "", ps.X12Version.Eq(req.X12Version)),
		keel.When(req.DocumentTypeID.IsNotNil(), ps.DocumentTypeID.Eq(req.DocumentTypeID)),
		keel.When(req.SchemaVersion > 0, ps.SchemaVersion.Eq(req.SchemaVersion)),
		keel.When(req.Status != "", ps.Status.Eq(req.Status)),
		keel.When(search != "", keel.Or(
			ps.Name.ContainsFold(search),
			ps.Description.ContainsFold(search),
			ps.X12Version.ContainsFold(search),
		)),
	).Order(
		keel.Desc(ps.OrganizationID.IsNotNull()),
		ps.SchemaVersion.Desc(),
		ps.CreatedAt.Desc(),
	)
}

func (r *repository) GetPartnerSettingSchema(
	ctx context.Context,
	req ediports.GetEDIPartnerSettingSchemaRequest,
) (*edi.EDIPartnerSettingSchema, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIPartnerSettingSchema, error) {
		ps := edidb.EDIPartnerSettingSchemas
		entity, err := ps.Where(ps.ID.Eq(req.ID), visibleSchema(req.TenantInfo)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, schemaErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetActivePartnerSettingSchema(
	ctx context.Context,
	req ediports.GetActiveEDIPartnerSettingSchemaRequest,
) (*edi.EDIPartnerSettingSchema, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIPartnerSettingSchema, error) {
		entity, err := activeSchemaQuery(&req).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, schemaErrors.Map(err, string(req.TransactionSet))
		}

		return entity, nil
	})
}

func activeSchemaQuery(
	req *ediports.GetActiveEDIPartnerSettingSchemaRequest,
) *keel.Query[edi.EDIPartnerSettingSchema] {
	ps := edidb.EDIPartnerSettingSchemas
	q := ps.Where(
		ps.Standard.Eq(req.Standard),
		ps.TransactionSet.Eq(req.TransactionSet),
		ps.Direction.Eq(req.Direction),
		ps.X12Version.Eq(req.X12Version),
		ps.Status.Eq(edi.PartnerSettingStatusActive),
		visibleSchema(req.TenantInfo),
		keel.When(req.SchemaVersion > 0, ps.SchemaVersion.Eq(req.SchemaVersion)),
	).Order(keel.Desc(ps.OrganizationID.IsNotNull()), ps.SchemaVersion.Desc())
	if req.DocumentTypeID.IsNotNil() {
		q.Where(keel.Or(ps.DocumentTypeID.Eq(req.DocumentTypeID), ps.DocumentTypeID.IsNull())).
			Order(keel.Desc(ps.DocumentTypeID.IsNotNull()))
	}

	return q
}

func (r *repository) ListPartnerSettingFields(
	ctx context.Context,
	req *ediports.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return r.searchPartnerSettingFields(ctx, req)
}

func (r *repository) SearchPartnerSettingFields(
	ctx context.Context,
	req *ediports.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return r.searchPartnerSettingFields(ctx, req)
}

func (r *repository) SelectPartnerSettingFieldOptions(
	ctx context.Context,
	req *ediports.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return r.searchPartnerSettingFields(ctx, req)
}

func (r *repository) searchPartnerSettingFields(
	ctx context.Context,
	req *ediports.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
			return listquery.Offset(ctx, r.db.Keel(ctx), fieldsQuery(req), req.Filter.Pagination)
		},
	)
}

func fieldsQuery(
	req *ediports.ListEDIPartnerSettingFieldsRequest,
) *keel.Query[edi.EDIPartnerSettingField] {
	pf := edidb.EDIPartnerSettingFields
	schema := pf.Schema()
	pathPrefix := strings.TrimSpace(req.PathPrefix)
	groupKey := strings.TrimSpace(req.GroupKey)
	search := strings.TrimSpace(req.Filter.Query)

	return pf.Where(
		visibleSchemaOf(schema, req.Filter.TenantInfo),
		keel.When(req.SchemaID.IsNotNil(), pf.SchemaID.Eq(req.SchemaID)),
		keel.When(req.Standard != "", schema.Standard.Eq(req.Standard)),
		keel.When(req.TransactionSet != "", schema.TransactionSet.Eq(req.TransactionSet)),
		keel.When(req.Direction != "", schema.Direction.Eq(req.Direction)),
		keel.When(req.Status != "", pf.Status.Eq(req.Status)),
		keel.When(pathPrefix != "", pf.Path.HasPrefix(pathPrefix)),
		keel.When(groupKey != "", pf.GroupKey.Eq(groupKey)),
		keel.When(req.Required != nil, pf.Required.Eq(req.Required != nil && *req.Required)),
		keel.When(req.Secret != nil, pf.Secret.Eq(req.Secret != nil && *req.Secret)),
		keel.When(search != "", keel.Or(
			pf.Path.ContainsFold(search),
			pf.Label.ContainsFold(search),
			pf.Description.ContainsFold(search),
			pf.GroupKey.ContainsFold(search),
		)),
	).Order(pf.DisplayOrder.Asc(), pf.Path.Asc())
}

func visibleSchema(tenant pagination.TenantInfo) keel.Pred {
	return visibleSchemaOf(edidb.EDIPartnerSettingSchemas, tenant)
}

func visibleSchemaOf(
	ps *edidb.EDIPartnerSettingSchemasTable,
	tenant pagination.TenantInfo,
) keel.Pred {
	return keel.Or(
		keel.And(ps.OrganizationID.Eq(tenant.OrgID), ps.BusinessUnitID.Eq(tenant.BuID)),
		keel.And(ps.OrganizationID.IsNull(), ps.BusinessUnitID.IsNull()),
	)
}
