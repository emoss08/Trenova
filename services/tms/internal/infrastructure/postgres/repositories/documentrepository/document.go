package documentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
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
	cols := buncolgen.DocumentColumns

	q = q.Where(cols.IsCurrentVersion.Eq(), true)
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.DocumentTable.Alias,
		req.Filter,
		(*document.Document)(nil),
	)

	if req.ResourceID != "" {
		q = q.Where(cols.ResourceID.Eq(), req.ResourceID)
	}

	if req.ResourceType != "" {
		q = q.Where(cols.ResourceType.Eq(), req.ResourceType)
	}

	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDocumentsRequest,
) (*pagination.ListResult[*document.Document], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*document.Document, 0, req.Filter.Pagination.SafeLimit())
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

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.DocumentSelectOptionsRequest,
) (*pagination.ListResult[*document.Document], error) {
	return dbhelper.SelectOptions[*document.Document](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"file_name",
				"original_name",
				"status",
				"resource_id",
				"resource_type",
				"document_type_id",
				"created_at",
			},
			OrgColumn: "doc.organization_id",
			BuColumn:  "doc.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				cols := buncolgen.DocumentColumns
				q = q.Where(cols.IsCurrentVersion.Eq(), true).
					Where(cols.Status.NotEq(), document.StatusArchived)
				if req.ResourceID != "" {
					q = q.Where(cols.ResourceID.Eq(), req.ResourceID)
				}
				if req.ResourceType != "" {
					q = q.Where(cols.ResourceType.Eq(), req.ResourceType)
				}

				return q.Relation(buncolgen.DocumentRelations.DocumentType).
					Order(cols.CreatedAt.OrderDesc())
			},
			EntityName: "Document",
			SearchColumns: []string{
				"doc.original_name",
				"doc.file_name",
				"doc.description",
			},
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	cols := buncolgen.DocumentColumns
	entity := new(document.Document)

	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentScopeTenant(sq, req.TenantInfo).Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get document", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Document")
	}

	return entity, nil
}

func (r *repository) GetByStoragePath(
	ctx context.Context,
	req repositories.GetDocumentByStoragePathRequest,
) (*document.Document, error) {
	entity := new(document.Document)
	cols := buncolgen.DocumentColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentScopeTenant(sq, req.TenantInfo).
				Where(cols.StoragePath.Eq(), req.StoragePath)
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
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
	cols := buncolgen.DocumentColumns
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentScopeTenant(sq, req.TenantInfo).
				Where(cols.ResourceID.Eq(), req.ResourceID).
				Where(cols.ResourceType.Eq(), req.ResourceType).
				Where(cols.IsCurrentVersion.Eq(), true)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			if req.IncludeDocumentType {
				return sq.Relation(buncolgen.DocumentRelations.DocumentType)
			}

			return sq
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get documents by resource", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) ListPendingPreviewReconciliation(
	ctx context.Context,
	olderThan int64,
	limit int,
) ([]*document.Document, error) {
	if limit <= 0 {
		limit = 100
	}

	entities := make([]*document.Document, 0, limit)
	cols := buncolgen.DocumentColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.IsCurrentVersion.Eq(), true).
		Where(cols.PreviewStatus.Eq(), document.PreviewStatusPending).
		WhereGroup(" OR ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.PreviewStoragePath.IsNull()).
				WhereOr(cols.PreviewStoragePath.Eq(), "")
		}).
		Where(cols.UpdatedAt.Lte(), olderThan).
		Order(cols.UpdatedAt.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list documents pending preview reconciliation", zap.Error(err))
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

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Value("is_current_version", "?", entity.IsCurrentVersion).
		Returning("*").
		Exec(ctx); err != nil {
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
	cols := buncolgen.DocumentColumns

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).WherePK().
		Where(cols.Version.Eq(), ov).
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

func (r *repository) UpdatePreview(
	ctx context.Context,
	req *repositories.UpdateDocumentPreviewRequest,
) error {
	cols := buncolgen.DocumentColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*document.Document)(nil)).
		Set(cols.PreviewStatus.Set(), req.PreviewStatus).
		Set(cols.PreviewStoragePath.Set(), req.PreviewStoragePath).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DocumentScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error(
			"failed to update document preview",
			zap.Error(err),
			zap.String("id", req.ID.String()),
		)
		return err
	}

	if err = dberror.CheckRowsAffected(results, "Document", req.ID.String()); err != nil {
		return err
	}

	return nil
}

func (r *repository) UpdateIntelligence(
	ctx context.Context,
	req *repositories.UpdateDocumentIntelligenceRequest,
) error {
	cols := buncolgen.DocumentColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*document.Document)(nil)).
		Set(cols.ContentStatus.Set(), req.ContentStatus).
		Set(cols.ContentError.Set(), req.ContentError).
		Set(cols.DetectedKind.Set(), req.DetectedKind).
		Set(cols.HasExtractedText.Set(), req.HasExtractedText).
		Set(cols.ShipmentDraftStatus.Set(), req.ShipmentDraftStatus).
		Set(cols.DocumentTypeID.Set(), req.DocumentTypeID).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DocumentScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error(
			"failed to update document intelligence",
			zap.Error(err),
			zap.String("id", req.ID.String()),
		)
		return err
	}

	if err = dberror.CheckRowsAffected(results, "Document", req.ID.String()); err != nil {
		return err
	}

	return nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteDocumentRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	cols := buncolgen.DocumentColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		TableExpr(fmt.Sprintf("%s AS %s", buncolgen.DocumentTable.Name, buncolgen.DocumentTable.Alias)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DocumentScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete document", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "Document", req.ID.String())
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
	cols := buncolgen.DocumentColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.IDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get documents by IDs", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) ListVersions(
	ctx context.Context,
	req repositories.ListDocumentVersionsRequest,
) ([]*document.Document, error) {
	entities := make([]*document.Document, 0)
	cols := buncolgen.DocumentColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentScopeTenant(sq, req.TenantInfo).
				Where(cols.LineageID.Eq(), req.LineageID)
		}).
		Order(cols.VersionNumber.OrderDesc(), cols.CreatedAt.OrderDesc()).
		Scan(ctx)
	if err != nil {
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

	cols := buncolgen.DocumentColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Table(buncolgen.DocumentTable.Name).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DocumentScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.IDs))
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

func (r *repository) PromoteVersion(
	ctx context.Context,
	req *repositories.PromoteDocumentVersionRequest,
) error {
	db := r.db.DBForContext(ctx)
	cols := buncolgen.DocumentColumns
	now := timeutils.NowUnix()

	if _, err := db.NewUpdate().
		TableExpr(fmt.Sprintf("%s AS %s", buncolgen.DocumentTable.Name, buncolgen.DocumentTable.Alias)).
		Set(cols.IsCurrentVersion.Set(), false).
		Set(cols.UpdatedAt.Set(), now).
		Set(cols.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DocumentScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.LineageID.Eq(), req.LineageID).
				Where(cols.IsCurrentVersion.Eq(), true)
		}).
		Exec(ctx); err != nil {
		return err
	}

	result, err := db.NewUpdate().
		TableExpr(fmt.Sprintf("%s AS %s", buncolgen.DocumentTable.Name, buncolgen.DocumentTable.Alias)).
		Set(cols.IsCurrentVersion.Set(), true).
		Set(cols.UpdatedAt.Set(), now).
		Set(cols.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DocumentScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.CurrentDocumentID).
				Where(cols.LineageID.Eq(), req.LineageID)
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(result, "Document", req.CurrentDocumentID.String())
}

func (r *repository) MoveLineageToResource(
	ctx context.Context,
	req *repositories.MoveDocumentLineageRequest,
) error {
	db := r.db.DBForContext(ctx)
	cols := buncolgen.DocumentColumns
	now := timeutils.NowUnix()

	current := new(document.Document)
	if err := db.NewSelect().
		Model(current).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.DocumentID)
		}).
		Scan(ctx); err != nil {
		return dberror.HandleNotFoundError(err, "Document")
	}

	result, err := db.NewUpdate().
		TableExpr(fmt.Sprintf("%s AS %s", buncolgen.DocumentTable.Name, buncolgen.DocumentTable.Alias)).
		Set(cols.ResourceID.Set(), req.ResourceID).
		Set(cols.ResourceType.Set(), req.ResourceType).
		Set(cols.UpdatedAt.Set(), now).
		Set(cols.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DocumentScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.LineageID.Eq(), current.LineageID)
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(result, "Document", req.DocumentID.String())
}

func (r *repository) DeleteByLineageIDs(
	ctx context.Context,
	req repositories.DeleteDocumentLineageRequest,
) error {
	if len(req.LineageIDs) == 0 {
		return nil
	}

	cols := buncolgen.DocumentColumns
	_, err := r.db.DBForContext(ctx).
		NewDelete().
		TableExpr(fmt.Sprintf("%s AS %s", buncolgen.DocumentTable.Name, buncolgen.DocumentTable.Alias)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DocumentScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.LineageID.In(), bun.List(req.LineageIDs))
		}).
		Exec(ctx)
	return err
}
