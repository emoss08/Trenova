package documentcontentrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.DocumentContentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-content-repository"),
	}
}

func (r *repository) GetByDocumentID(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentcontent.Content, error) {
	entity := new(documentcontent.Content)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dc.document_id = ?", documentID).
		Where("dc.organization_id = ?", tenantInfo.OrgID).
		Where("dc.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Document content")
	}

	return entity, nil
}

func (r *repository) ListPagesByDocumentID(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]*documentcontent.Page, error) {
	items := make([]*documentcontent.Page, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dcp.document_id = ?", documentID).
		Where("dcp.organization_id = ?", tenantInfo.OrgID).
		Where("dcp.business_unit_id = ?", tenantInfo.BuID).
		Order("dcp.page_number ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *documentcontent.Content,
) (*documentcontent.Content, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(`CONFLICT ("document_id", "organization_id", "business_unit_id") DO UPDATE`).
		Set("status = EXCLUDED.status").
		Set("content_text = EXCLUDED.content_text").
		Set("page_count = EXCLUDED.page_count").
		Set("source_kind = EXCLUDED.source_kind").
		Set("detected_language = EXCLUDED.detected_language").
		Set("detected_document_kind = EXCLUDED.detected_document_kind").
		Set("classification_confidence = EXCLUDED.classification_confidence").
		Set("structured_data = EXCLUDED.structured_data").
		Set("failure_code = EXCLUDED.failure_code").
		Set("failure_message = EXCLUDED.failure_message").
		Set("last_extracted_at = EXCLUDED.last_extracted_at").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ReplacePages(
	ctx context.Context,
	content *documentcontent.Content,
	pages []*documentcontent.Page,
) error {
	db := r.db.DBForContext(ctx)

	if _, err := db.NewDelete().
		Table("document_content_pages").
		Where("document_content_id = ?", content.ID).
		Where("organization_id = ?", content.OrganizationID).
		Where("business_unit_id = ?", content.BusinessUnitID).
		Exec(ctx); err != nil {
		return err
	}

	if len(pages) == 0 {
		return nil
	}

	for _, page := range pages {
		page.DocumentContentID = content.ID
		page.DocumentID = content.DocumentID
		page.OrganizationID = content.OrganizationID
		page.BusinessUnitID = content.BusinessUnitID
	}

	_, err := db.NewInsert().
		Model(&pages).
		Exec(ctx)
	return err
}

func (r *repository) ListPendingExtraction(
	ctx context.Context,
	olderThan int64,
	limit int,
) ([]*document.Document, error) {
	if limit <= 0 {
		limit = 100
	}

	items := make([]*document.Document, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Column("doc.*").
		Join(`LEFT JOIN document_contents AS dc
			ON dc.document_id = doc.id
			AND dc.organization_id = doc.organization_id
			AND dc.business_unit_id = doc.business_unit_id`).
		Where("doc.status = ?", document.StatusActive).
		Where(`
			(dc.document_id IS NULL OR dc.status IN (?, ?) OR doc.content_status IN (?, ?))`,
			documentcontent.StatusPending,
			documentcontent.StatusExtracting,
			document.ContentStatusPending,
			document.ContentStatusExtracting,
		).
		Where("doc.updated_at <= ?", olderThan).
		Order("doc.updated_at ASC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *repository) SearchByResource(
	ctx context.Context,
	req *repositories.DocumentContentSearchRequest,
) ([]*document.Document, error) {
	items := make([]*document.Document, 0)
	query := strings.TrimSpace(req.Query)

	selectQuery := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Column("doc.*").
		Join(`LEFT JOIN document_contents AS dc
			ON dc.document_id = doc.id
			AND dc.organization_id = doc.organization_id
			AND dc.business_unit_id = doc.business_unit_id`).
		Where("doc.organization_id = ?", req.TenantInfo.OrgID).
		Where("doc.business_unit_id = ?", req.TenantInfo.BuID).
		Where("doc.resource_id = ?", req.ResourceID).
		Where("doc.resource_type = ?", req.ResourceType)

	if query != "" {
		selectQuery = selectQuery.Where(`
			doc.search_vector @@ websearch_to_tsquery('english', ?)
			OR dc.search_vector @@ websearch_to_tsquery('english', ?)`,
			query,
			query,
		).OrderExpr(`
			GREATEST(
				ts_rank_cd(doc.search_vector, websearch_to_tsquery('english', ?)),
				COALESCE(ts_rank_cd(dc.search_vector, websearch_to_tsquery('english', ?)), 0)
			) DESC`,
			query,
			query,
		)
	} else {
		selectQuery = selectQuery.Order("doc.created_at DESC")
	}

	if req.Limit > 0 {
		selectQuery = selectQuery.Limit(req.Limit)
	}

	if err := selectQuery.Scan(ctx); err != nil {
		return nil, err
	}

	return items, nil
}
