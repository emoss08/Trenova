package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type GeneratedParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type generatedRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewGeneratedRepository(p GeneratedParams) repositories.GeneratedDocumentRepository {
	return &generatedRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.generated-document-repository"),
	}
}

func (r *generatedRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListGeneratedDocumentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.GeneratedDocumentTable.Alias,
		req.Filter,
		(*documenttemplate.GeneratedDocument)(nil),
	)

	cols := buncolgen.GeneratedDocumentColumns

	if req.Kind != "" {
		q = q.Where(cols.Kind.Eq(), req.Kind)
	}

	if req.ReferenceType != "" {
		q = q.Where(cols.ReferenceType.Eq(), req.ReferenceType)
	}

	if req.ReferenceID != nil && !req.ReferenceID.IsNil() {
		q = q.Where(cols.ReferenceID.Eq(), *req.ReferenceID)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *generatedRepository) List(
	ctx context.Context,
	req *repositories.ListGeneratedDocumentsRequest,
) (*pagination.ListResult[*documenttemplate.GeneratedDocument], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*documenttemplate.GeneratedDocument, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		Order(buncolgen.GeneratedDocumentColumns.CreatedAt.OrderDesc()).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count generated documents", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*documenttemplate.GeneratedDocument]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID loads the artifact with the template version that produced it, which is
// the whole reason the row exists: answering what exactly was sent, months later.
func (r *generatedRepository) GetByID(
	ctx context.Context,
	req *repositories.GetGeneratedDocumentByIDRequest,
) (*documenttemplate.GeneratedDocument, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.GeneratedDocumentID.String()),
	)

	entity := new(documenttemplate.GeneratedDocument)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.GeneratedDocumentRelations.TemplateVersion).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.GeneratedDocumentScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.GeneratedDocumentColumns.ID.Eq(), req.GeneratedDocumentID)
		}).
		Scan(ctx); err != nil {
		log.Error("failed to get generated document", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "GeneratedDocument")
	}

	return entity, nil
}

func (r *generatedRepository) Create(
	ctx context.Context,
	entity *documenttemplate.GeneratedDocument,
) (*documenttemplate.GeneratedDocument, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create generated document", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// Update carries a render from Pending to its outcome. It does not use OmitZero:
// a failed render clears file_path and a redelivery clears the previous
// recipient, and OmitZero would keep the stale value.
func (r *generatedRepository) Update(
	ctx context.Context,
	entity *documenttemplate.GeneratedDocument,
) (*documenttemplate.GeneratedDocument, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	cols := buncolgen.GeneratedDocumentColumns
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update generated document", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results, "GeneratedDocument", entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}
