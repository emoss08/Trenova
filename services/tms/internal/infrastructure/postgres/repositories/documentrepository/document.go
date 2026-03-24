package documentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
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

func New(p Params) repositories.DocumentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDocumentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"doc",
		req.Filter,
		(*document.Document)(nil),
	)

	if req.ResourceID != "" {
		q = q.Where("doc.resource_id = ?", req.ResourceID)
	}

	if req.ResourceType != "" {
		q = q.Where("doc.resource_type = ?", req.ResourceType)
	}

	if req.Status != "" {
		q = q.Where("doc.status = ?", req.Status)
	}

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDocumentsRequest,
) (*pagination.ListResult[*document.Document], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*document.Document, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count documents", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*document.Document]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(document.Document)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("doc.id = ?", req.ID).
				Where("doc.organization_id = ?", req.TenantInfo.OrgID).
				Where("doc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get document", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Document")
	}

	return entity, nil
}

func (r *repository) GetByResourceID(
	ctx context.Context,
	req *repositories.GetDocumentsByResourceRequest,
) ([]*document.Document, error) {
	log := r.l.With(
		zap.String("operation", "GetByResourceID"),
		zap.String("resourceId", req.ResourceID),
		zap.String("resourceType", req.ResourceType),
	)

	entities := make([]*document.Document, 0)
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("doc.resource_id = ?", req.ResourceID).
				Where("doc.resource_type = ?", req.ResourceType).
				Where("doc.organization_id = ?", req.TenantInfo.OrgID).
				Where("doc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Order("doc.created_at DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get documents by resource", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("fileName", entity.FileName),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create document", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update document", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Document", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteDocumentRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	results, err := r.db.DB().
		NewDelete().
		Model((*document.Document)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("id = ?", req.ID).
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete document", zap.Error(err))
		return err
	}

	rowsAffected, _ := results.RowsAffected()
	if rowsAffected == 0 {
		return errortypes.NewNotFoundError("Document not found within your organization")
	}

	return nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.BulkDeleteDocumentRequest,
) ([]*document.Document, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Int("count", len(req.IDs)),
	)

	entities := make([]*document.Document, 0, len(req.IDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("doc.id IN (?)", bun.In(req.IDs)).
				Where("doc.organization_id = ?", req.TenantInfo.OrgID).
				Where("doc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get documents by IDs", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) BulkDelete(
	ctx context.Context,
	req repositories.BulkDeleteDocumentRequest,
) error {
	log := r.l.With(
		zap.String("operation", "BulkDelete"),
		zap.Int("count", len(req.IDs)),
	)

	results, err := r.db.DB().
		NewDelete().
		Model((*document.Document)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("id IN (?)", bun.In(req.IDs)).
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk delete documents", zap.Error(err))
		return err
	}

	rowsAffected, _ := results.RowsAffected()
	log.Info("bulk deleted documents", zap.Int64("rowsAffected", rowsAffected))

	return nil
}
