package documentcontentrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	cols := buncolgen.ContentColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ContentScopeTenant(sq, tenantInfo).
				Where(cols.DocumentID.Eq(), documentID)
		}).
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
	cols := buncolgen.PageColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PageScopeTenant(sq, tenantInfo).
				Where(cols.DocumentID.Eq(), documentID)
		}).
		Order(cols.PageNumber.OrderAsc()).
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
	cols := buncolgen.ContentColumns
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(`CONFLICT ("document_id", "organization_id", "business_unit_id") DO UPDATE`).
		Set(cols.Status.SetExcluded()).
		Set(cols.ContentText.SetExcluded()).
		Set(cols.PageCount.SetExcluded()).
		Set(cols.SourceKind.SetExcluded()).
		Set(cols.DetectedLanguage.SetExcluded()).
		Set(cols.DetectedDocumentKind.SetExcluded()).
		Set(cols.ClassificationConfidence.SetExcluded()).
		Set(cols.StructuredData.SetExcluded()).
		Set(cols.FailureCode.SetExcluded()).
		Set(cols.FailureMessage.SetExcluded()).
		Set(cols.LastExtractedAt.SetExcluded()).
		Set(cols.UpdatedAt.SetExcluded()).
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
	cols := buncolgen.PageColumns

	if _, err := db.NewDelete().
		Table(buncolgen.PageTable.Name).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.PageScopeTenantDelete(sq, pagination.TenantInfo{
				OrgID: content.OrganizationID,
				BuID:  content.BusinessUnitID,
			}).
				Where(cols.DocumentContentID.Eq(), content.ID)
		}).
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
	docCols := buncolgen.DocumentColumns
	contentCols := buncolgen.ContentColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		ColumnExpr(buncolgen.DocumentTable.Alias+".*").
		Join(`LEFT JOIN document_contents AS dc
			ON dc.document_id = doc.id
			AND dc.organization_id = doc.organization_id
			AND dc.business_unit_id = doc.business_unit_id`).
		Where(docCols.Status.Eq(), document.StatusActive).
		WhereGroup(" OR ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(contentCols.DocumentID.IsNull()).
				WhereOr(contentCols.Status.In(), bun.List([]documentcontent.Status{
					documentcontent.StatusPending,
					documentcontent.StatusExtracting,
				})).
				WhereOr(docCols.ContentStatus.In(), bun.List([]document.ContentStatus{
					document.ContentStatusPending,
					document.ContentStatusExtracting,
				}))
		}).
		Where(docCols.UpdatedAt.Lte(), olderThan).
		Order(docCols.UpdatedAt.OrderAsc()).
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
	docCols := buncolgen.DocumentColumns

	selectQuery := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		ColumnExpr(buncolgen.DocumentTable.Alias+".*").
		Join(`LEFT JOIN document_contents AS dc
			ON dc.document_id = doc.id
			AND dc.organization_id = doc.organization_id
			AND dc.business_unit_id = doc.business_unit_id`).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentScopeTenant(sq, req.TenantInfo).
				Where(docCols.IsCurrentVersion.Eq(), true).
				Where(docCols.ResourceID.Eq(), req.ResourceID).
				Where(docCols.ResourceType.Eq(), req.ResourceType)
		})

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
		selectQuery = selectQuery.Order(docCols.CreatedAt.OrderDesc())
	}

	if req.Limit > 0 {
		selectQuery = selectQuery.Limit(req.Limit)
	}

	if err := selectQuery.Scan(ctx); err != nil {
		return nil, err
	}

	return items, nil
}
