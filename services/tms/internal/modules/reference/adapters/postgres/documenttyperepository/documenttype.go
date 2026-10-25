package documenttyperepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var systemDocumentTypeCodes = []string{"INVOICE", "CREDITMEMO", "DEBITMEMO"}

var documentTypeErrors = dberror.Mapping{Entity: "DocumentType"}

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

func (r *repository) List(
	ctx context.Context,
	req *documenttypeports.ListDocumentTypesRequest,
) (*pagination.ListResult[*documenttype.DocumentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*documenttype.DocumentType], error) {
			dt := documenttypedb.DocumentTypes
			result, err := listquery.List(ctx, r.db.Keel(ctx), dt.Query(),
				&listquery.Request[documenttype.DocumentType]{
					Filter: req.Filter,
					Then:   []keel.Order{dt.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list document types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *documenttypeports.ListDocumentTypeConnectionRequest,
) (*pagination.CursorListResult[*documenttype.DocumentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*documenttype.DocumentType], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				documenttypedb.DocumentTypes.Query(),
				&listquery.Request[documenttype.DocumentType]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.DocumentTypeColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list document types", zap.Error(err))
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
		if err := documenttypedb.DocumentTypes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create document type",
				zap.String("name", entity.Name),
				zap.Error(err),
			)
			return nil, documentTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *documenttype.DocumentType,
) (*documenttype.DocumentType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*documenttype.DocumentType, error) {
		if err := documenttypedb.DocumentTypes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update document type",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, documentTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req documenttypeports.GetDocumentTypeByIDRequest,
) (*documenttype.DocumentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*documenttype.DocumentType, error) {
		dt := documenttypedb.DocumentTypes
		entity, err := dt.Where(dt.ID.Eq(req.ID)).
			Cache(resultcache.Reference()...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, documentTypeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByCode(
	ctx context.Context,
	req documenttypeports.GetDocumentTypeByCodeRequest,
) (*documenttype.DocumentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*documenttype.DocumentType, error) {
		dt := documenttypedb.DocumentTypes
		entity, err := dt.Where(dt.Code.Lower().Eq(strings.ToLower(req.Code))).
			Order(dt.CreatedAt.Asc()).
			Cache(resultcache.Reference()...).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, documentTypeErrors.Map(err, req.Code)
		}

		return entity, nil
	})
}

func (r *repository) GetByName(
	ctx context.Context,
	req documenttypeports.GetDocumentTypeByNameRequest,
) (*documenttype.DocumentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*documenttype.DocumentType, error) {
		dt := documenttypedb.DocumentTypes
		name := keel.DynLit(strings.TrimSpace(req.Name)).Cast(keel.TypeString)
		entity, err := dt.Where(
			keel.DynFunc("lower", keel.DynFunc("btrim", dt.Name)).
				Eq(keel.DynFunc("lower", keel.DynFunc("btrim", name))),
		).
			Order(dt.CreatedAt.Asc()).
			Cache(resultcache.Reference()...).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, documentTypeErrors.Map(err, req.Name)
		}

		return entity, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*documenttype.DocumentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*documenttype.DocumentType], error) {
			dt := documenttypedb.DocumentTypes
			q := dt.Pick(
				dt.ID,
				dt.Code,
				dt.Name,
				dt.Description,
				dt.Color,
				dt.DocumentClassification,
				dt.DocumentCategory,
				dt.IsSystem,
			).
				Where(dt.Code.NotIn(systemDocumentTypeCodes...)).
				Cache(resultcache.Reference()...)

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req,
				func(s string) keel.Pred {
					return keel.Or(dt.Code.ContainsFold(s), dt.Name.ContainsFold(s))
				},
			)
		},
	)
}
