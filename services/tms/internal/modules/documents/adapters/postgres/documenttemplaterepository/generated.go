package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate/documenttemplatedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/generateddocument"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var generatedErrors = dberror.Mapping{Entity: "GeneratedDocument"}

type GeneratedParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type generatedRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewGeneratedRepository(p GeneratedParams) documenttemplateports.GeneratedDocumentRepository {
	return &generatedRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.generated-document-repository"),
	}
}

func (r *generatedRepository) List(
	ctx context.Context,
	req *documenttemplateports.ListGeneratedDocumentsRequest,
) (*pagination.ListResult[*generateddocument.GeneratedDocument], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*generateddocument.GeneratedDocument], error) {
			g := documenttemplatedb.GeneratedDocuments
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				g.Where(
					keel.When(req.Kind != "", g.Kind.Eq(req.Kind)),
					keel.When(req.ReferenceType != "", g.ReferenceType.Eq(req.ReferenceType)),
					keel.When(
						req.ReferenceID != nil && !req.ReferenceID.IsNil(),
						g.ReferenceID.Eq(derefID(req.ReferenceID)),
					),
				),
				&listquery.Request[generateddocument.GeneratedDocument]{
					Filter: req.Filter,
					Then:   []keel.Order{g.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list generated documents", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

// GetByID loads the artifact with the template version that produced it, which is
// the whole reason the row exists: answering what exactly was sent, months later.
func (r *generatedRepository) GetByID(
	ctx context.Context,
	req *documenttemplateports.GetGeneratedDocumentByIDRequest,
) (*generateddocument.GeneratedDocument, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*generateddocument.GeneratedDocument, error) {
			g := documenttemplatedb.GeneratedDocuments
			entity, err := g.Where(g.ID.Eq(req.GeneratedDocumentID)).
				Load(g.TemplateVersion()).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get generated document", zap.Error(err))
				return nil, generatedErrors.Map(err, req.GeneratedDocumentID.String())
			}

			return entity, nil
		},
	)
}

func (r *generatedRepository) Create(
	ctx context.Context,
	entity *generateddocument.GeneratedDocument,
) (*generateddocument.GeneratedDocument, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*generateddocument.GeneratedDocument, error) {
			err := documenttemplatedb.GeneratedDocuments.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create generated document", zap.Error(err))
				return nil, generatedErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

// Update carries a render from Pending to its outcome. It writes every column: a
// failed render clears file_path and a redelivery clears the previous recipient.
func (r *generatedRepository) Update(
	ctx context.Context,
	entity *generateddocument.GeneratedDocument,
) (*generateddocument.GeneratedDocument, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*generateddocument.GeneratedDocument, error) {
			err := documenttemplatedb.GeneratedDocuments.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update generated document",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, generatedErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func derefID(id *pulid.ID) pulid.ID {
	if id == nil {
		return pulid.Nil
	}

	return *id
}
