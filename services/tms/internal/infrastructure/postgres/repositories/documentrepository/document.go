package documentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("doc.storage_path = ?", req.StoragePath).
		Where("doc.organization_id = ?", req.TenantInfo.OrgID).
		Where("doc.business_unit_id = ?", req.TenantInfo.BuID).
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
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("doc.resource_id = ?", req.ResourceID).
				Where("doc.resource_type = ?", req.ResourceType).
				Where("doc.is_current_version = ?", true).
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

func (r *repository) ListPendingPreviewReconciliation(
	ctx context.Context,
	olderThan int64,
	limit int,
) ([]*document.Document, error) {
	if limit <= 0 {
		limit = 100
	}

	entities := make([]*document.Document, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("doc.is_current_version = ?", true).
		Where("doc.preview_status = ?", document.PreviewStatusPending).
		Where("(doc.preview_storage_path IS NULL OR doc.preview_storage_path = '')").
		Where("doc.updated_at <= ?", olderThan).
		Order("doc.updated_at ASC").
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

func (r *repository) UpdatePreview(
	ctx context.Context,
	req *repositories.UpdateDocumentPreviewRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*document.Document)(nil)).
		Set("preview_status = ?", req.PreviewStatus).
		Set("preview_storage_path = ?", req.PreviewStoragePath).
		Set("updated_at = ?", timeutils.NowUnix()).
		Set("version = version + 1").
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
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
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*document.Document)(nil)).
		Set("content_status = ?", req.ContentStatus).
		Set("content_error = ?", req.ContentError).
		Set("detected_kind = ?", req.DetectedKind).
		Set("has_extracted_text = ?", req.HasExtractedText).
		Set("shipment_draft_status = ?", req.ShipmentDraftStatus).
		Set("document_type_id = ?", req.DocumentTypeID).
		Set("updated_at = ?", timeutils.NowUnix()).
		Set("version = version + 1").
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
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

	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Table("documents").
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
	err := r.db.DBForContext(ctx).
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

func (r *repository) ListVersions(
	ctx context.Context,
	req repositories.ListDocumentVersionsRequest,
) ([]*document.Document, error) {
	entities := make([]*document.Document, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("doc.lineage_id = ?", req.LineageID).
		Where("doc.organization_id = ?", req.TenantInfo.OrgID).
		Where("doc.business_unit_id = ?", req.TenantInfo.BuID).
		Order("doc.version_number DESC, doc.created_at DESC").
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

	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Table("documents").
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

func (r *repository) PromoteVersion(
	ctx context.Context,
	req *repositories.PromoteDocumentVersionRequest,
) error {
	db := r.db.DBForContext(ctx)

	if _, err := db.NewUpdate().
		Table("documents").
		Set("is_current_version = false").
		Set("updated_at = ?", timeutils.NowUnix()).
		Set("version = version + 1").
		Where("lineage_id = ?", req.LineageID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Where("is_current_version = ?", true).
		Exec(ctx); err != nil {
		return err
	}

	result, err := db.NewUpdate().
		Table("documents").
		Set("is_current_version = true").
		Set("updated_at = ?", timeutils.NowUnix()).
		Set("version = version + 1").
		Where("id = ?", req.CurrentDocumentID).
		Where("lineage_id = ?", req.LineageID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
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

	current := new(document.Document)
	if err := db.NewSelect().
		Model(current).
		Where("doc.id = ?", req.DocumentID).
		Where("doc.organization_id = ?", req.TenantInfo.OrgID).
		Where("doc.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx); err != nil {
		return dberror.HandleNotFoundError(err, "Document")
	}

	result, err := db.NewUpdate().
		Table("documents").
		Set("resource_id = ?", req.ResourceID).
		Set("resource_type = ?", req.ResourceType).
		Set("updated_at = ?", timeutils.NowUnix()).
		Set("version = version + 1").
		Where("lineage_id = ?", current.LineageID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
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

	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Table("documents").
		Where("lineage_id IN (?)", bun.In(req.LineageIDs)).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	return err
}
