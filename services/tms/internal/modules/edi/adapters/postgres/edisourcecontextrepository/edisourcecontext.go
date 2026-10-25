//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edisourcecontextrepository

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
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultContextKey = "loadTender"

var schemaErrors = dberror.Mapping{Entity: "EDISourceContextSchema"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediports.EDISourceContextRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-source-context-repository"),
	}
}

func (r *repository) ListSourceContextSchemas(
	ctx context.Context,
	req *ediports.ListEDISourceContextSchemasRequest,
) (*pagination.ListResult[*edi.EDISourceContextSchema], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDISourceContextSchema], error) {
			return listquery.Offset(ctx, r.db.Keel(ctx), schemasQuery(req), req.Filter.Pagination)
		},
	)
}

func schemasQuery(
	req *ediports.ListEDISourceContextSchemasRequest,
) *keel.Query[*edi.EDISourceContextSchema] {
	sc := edidb.EDISourceContextSchemas
	search := strings.TrimSpace(req.Filter.Query)

	return sc.Where(
		visibleSchema(sc, req.Filter.TenantInfo),
		keel.When(req.Standard != "", sc.Standard.Eq(req.Standard)),
		keel.When(req.TransactionSet != "", sc.TransactionSet.Eq(req.TransactionSet)),
		keel.When(req.Direction != "", sc.Direction.Eq(req.Direction)),
		keel.When(req.X12Version != "", sc.X12Version.Eq(req.X12Version)),
		keel.When(req.ContextKey != "", sc.ContextKey.Eq(req.ContextKey)),
		keel.When(req.SchemaVersion > 0, sc.SchemaVersion.Eq(req.SchemaVersion)),
		keel.When(req.Status != "", sc.Status.Eq(req.Status)),
		keel.When(search != "", keel.Or(
			sc.Name.ContainsFold(search),
			sc.ContextKey.ContainsFold(search),
			sc.Description.ContainsFold(search),
		)),
	).Order(
		keel.Desc(sc.OrganizationID.IsNotNull()),
		sc.SchemaVersion.Desc(),
		sc.CreatedAt.Desc(),
	)
}

func (r *repository) GetSourceContextSchema(
	ctx context.Context,
	req ediports.GetEDISourceContextSchemaRequest,
) (*edi.EDISourceContextSchema, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDISourceContextSchema, error) {
		sc := edidb.EDISourceContextSchemas
		entity, err := sc.Where(sc.ID.Eq(req.ID), visibleSchema(sc, req.TenantInfo)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, schemaErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetActiveSourceContextSchema(
	ctx context.Context,
	req ediports.GetActiveEDISourceContextSchemaRequest,
) (*edi.EDISourceContextSchema, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDISourceContextSchema, error) {
		sc := edidb.EDISourceContextSchemas
		entity, err := sc.Where(
			sc.Standard.Eq(req.Standard),
			sc.TransactionSet.Eq(req.TransactionSet),
			sc.Direction.Eq(req.Direction),
			sc.X12Version.Eq(req.X12Version),
			sc.ContextKey.Eq(stringutils.FirstNonEmpty(req.ContextKey, defaultContextKey)),
			sc.Status.Eq(edi.SourceContextFieldStatusActive),
			visibleSchema(sc, req.TenantInfo),
			keel.When(req.SchemaVersion > 0, sc.SchemaVersion.Eq(req.SchemaVersion)),
		).
			Order(keel.Desc(sc.OrganizationID.IsNotNull()), sc.SchemaVersion.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, schemaErrors.Map(err, string(req.TransactionSet))
		}

		return entity, nil
	})
}

func (r *repository) ListSourceContextFields(
	ctx context.Context,
	req *ediports.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return r.searchSourceContextFields(ctx, req)
}

func (r *repository) SearchSourceContextFields(
	ctx context.Context,
	req *ediports.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return r.searchSourceContextFields(ctx, req)
}

func (r *repository) SelectSourceContextFieldOptions(
	ctx context.Context,
	req *ediports.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return r.searchSourceContextFields(ctx, req)
}

func (r *repository) searchSourceContextFields(
	ctx context.Context,
	req *ediports.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDISourceContextField], error) {
			return listquery.Offset(ctx, r.db.Keel(ctx), fieldsQuery(req), req.Filter.Pagination)
		},
	)
}

func fieldsQuery(
	req *ediports.ListEDISourceContextFieldsRequest,
) *keel.Query[*edi.EDISourceContextField] {
	sf := edidb.EDISourceContextFields
	schema := sf.Schema()
	pathPrefix := strings.TrimSpace(req.PathPrefix)
	search := strings.TrimSpace(req.Filter.Query)

	return sf.Where(
		visibleSchema(schema, req.Filter.TenantInfo),
		keel.When(req.SchemaID.IsNotNil(), sf.SchemaID.Eq(req.SchemaID)),
		keel.When(req.Standard != "", schema.Standard.Eq(req.Standard)),
		keel.When(req.TransactionSet != "", schema.TransactionSet.Eq(req.TransactionSet)),
		keel.When(req.Direction != "", schema.Direction.Eq(req.Direction)),
		keel.When(req.Status != "", sf.Status.Eq(req.Status)),
		keel.When(req.SourceKind != "", sf.SourceKind.Eq(req.SourceKind)),
		keel.When(req.Repeated != nil, sf.Repeated.Eq(req.Repeated != nil && *req.Repeated)),
		keel.When(pathPrefix != "", sf.Path.HasPrefix(pathPrefix)),
		keel.When(search != "", keel.Or(
			sf.Path.ContainsFold(search),
			sf.DisplayName.ContainsFold(search),
			sf.Description.ContainsFold(search),
			listquery.TextContainsFold(sf.DataType, search),
		)),
	).Order(sf.Path.Asc(), sf.RepeatPath.Asc())
}

func visibleSchema(
	sc *edidb.EDISourceContextSchemasTable,
	tenant pagination.TenantInfo,
) keel.Pred {
	return keel.Or(
		keel.And(sc.OrganizationID.Eq(tenant.OrgID), sc.BusinessUnitID.Eq(tenant.BuID)),
		keel.And(sc.OrganizationID.IsNull(), sc.BusinessUnitID.IsNull()),
	)
}
