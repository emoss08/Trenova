package documentcontentrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent/documentcontentdb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultPendingLimit = 100

var contentErrors = dberror.Mapping{Entity: "Document content"}

var pageErrors = dberror.Mapping{Entity: "Document content page"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) documentcontentports.DocumentContentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-content-repository"),
	}
}

func (r *repository) GetByDocumentID(
	ctx context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) (*documentcontent.Content, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*documentcontent.Content, error) {
		dc := documentcontentdb.Contents
		entity, err := dc.Where(dc.DocumentID.Eq(documentID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, contentErrors.Map(err, documentID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListPagesByDocumentID(
	ctx context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) ([]*documentcontent.Page, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*documentcontent.Page, error) {
		p := documentcontentdb.Pages
		items, err := p.Where(p.DocumentID.Eq(documentID)).
			Order(p.PageNumber.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if items == nil {
			items = make([]*documentcontent.Page, 0)
		}

		return items, nil
	})
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *documentcontent.Content,
) (*documentcontent.Content, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*documentcontent.Content, error) {
		dc := documentcontentdb.Contents
		db := r.db.Keel(ctx)
		inserted, err := dc.Upsert(
			ctx,
			db,
			[]*documentcontent.Content{entity},
			keel.OnConflict(dc.DocumentID, dc.OrganizationID, dc.BusinessUnitID).Update(
				dc.Status,
				dc.ContentText,
				dc.PageCount,
				dc.SourceKind,
				dc.DetectedLanguage,
				dc.DetectedDocumentKind,
				dc.ClassificationConfidence,
				dc.StructuredData,
				dc.FailureCode,
				dc.FailureMessage,
				dc.LastExtractedAt,
				dc.ConverterVersion,
				dc.RawObjectKey,
			),
		)
		if err != nil {
			return nil, contentErrors.Map(err, entity.DocumentID.String())
		}
		if len(inserted) == 1 && inserted[0] {
			return entity, nil
		}

		stored, err := dc.Where(dc.DocumentID.Eq(entity.DocumentID)).One(ctx, db)
		if err != nil {
			return nil, contentErrors.Map(err, entity.DocumentID.String())
		}
		*entity = *stored

		return entity, nil
	})
}

func (r *repository) ReplacePages(
	ctx context.Context,
	content *documentcontent.Content,
	pages []*documentcontent.Page,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		p := documentcontentdb.Pages

		if _, err := p.Where(p.DocumentContentID.Eq(content.ID)).Delete(ctx, db); err != nil {
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

		if err := p.CreateMany(ctx, db, pages); err != nil {
			return pageErrors.Map(err, content.ID.String())
		}

		return nil
	})
}

func (r *repository) ListPendingExtraction(
	ctx context.Context,
	req *documentcontentports.ListPendingDocumentExtractionRequest,
) ([]*document.Document, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*document.Document, error) {
		items, err := pendingExtractionQuery(req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		return items, nil
	})
}

func (r *repository) ListPendingExtractionTenants(
	ctx context.Context,
	req *documentcontentports.ListPendingDocumentExtractionRequest,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with documents awaiting extraction")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		type tenantRow struct {
			OrganizationID pulid.ID
			BusinessUnitID pulid.ID
		}

		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), pendingTenantsQuery(req))
		if err != nil {
			return nil, err
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}

		return tenants, nil
	})
}

func pendingExtractionQuery(
	req *documentcontentports.ListPendingDocumentExtractionRequest,
) *keel.Query[*document.Document] {
	d := documentdb.Documents
	dc := documentcontentdb.Contents

	return d.LeftJoin(dc, dc.DocumentID.EqCol(d.ID)).
		Where(pendingExtraction(d, dc)...).
		Where(d.UpdatedAt.Lte(req.OlderThan)).
		Order(d.UpdatedAt.Asc()).
		Limit(pendingLimit(req.Limit))
}

func pendingTenantsQuery(
	req *documentcontentports.ListPendingDocumentExtractionRequest,
) *keel.Query[*document.Document] {
	d := documentdb.Documents
	dc := documentcontentdb.Contents

	return d.AllTenants().
		LeftJoin(
			dc,
			dc.DocumentID.EqCol(d.ID),
			dc.OrganizationID.EqCol(d.OrganizationID),
			dc.BusinessUnitID.EqCol(d.BusinessUnitID),
		).
		Where(pendingExtraction(d, dc)...).
		Where(d.UpdatedAt.Lte(req.OlderThan)).
		Group(d.OrganizationID, d.BusinessUnitID).
		Pick(d.OrganizationID, d.BusinessUnitID).
		Order(d.OrganizationID.Asc(), d.BusinessUnitID.Asc()).
		Limit(pendingLimit(req.Limit))
}

func pendingLimit(limit int) int {
	if limit <= 0 {
		return defaultPendingLimit
	}

	return limit
}

func pendingExtraction(
	d *documentdb.DocumentsTable,
	dc *documentcontentdb.ContentsTable,
) []keel.Pred {
	return []keel.Pred{
		d.Status.Eq(document.StatusActive),
		d.ProcessingProfile.In(document.IntelligenceProcessingProfiles()...),
		keel.Or(
			dc.DocumentID.IsNull(),
			dc.Status.In(documentcontent.StatusPending, documentcontent.StatusExtracting),
			d.ContentStatus.In(document.ContentStatusPending, document.ContentStatusExtracting),
		),
	}
}

func (r *repository) SearchByResource(
	ctx context.Context,
	req *documentcontentports.DocumentContentSearchRequest,
) ([]*document.Document, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*document.Document, error) {
		query := strings.TrimSpace(req.Query)
		d := documentdb.Documents
		dc := documentcontentdb.Contents

		q := d.LeftJoin(dc, dc.DocumentID.EqCol(d.ID)).
			Where(
				d.IsCurrentVersion.IsTrue(),
				d.ResourceID.Eq(req.ResourceID),
				d.ResourceType.Eq(req.ResourceType),
			)

		if query == "" {
			q = q.Order(d.CreatedAt.Desc())
		} else {
			search := keel.SearchConfig("simple", "english")
			q = q.Where(keel.Or(d.SearchVector.Match(query, search), dc.SearchVector.Match(query, search))).
				Order(keel.Float64Of(d.SearchVector.RankCD(query, search)).
					Greatest(keel.Float64Of(dc.SearchVector.RankCD(query, search)).Coalesce(0)).
					Desc())
		}

		var (
			items []*document.Document
			err   error
		)
		if req.Limit > 0 {
			items, err = q.Limit(req.Limit).List(ctx, r.db.Keel(ctx))
		} else {
			items, err = q.All(ctx, r.db.Keel(ctx))
		}
		if err != nil {
			return nil, err
		}
		if items == nil {
			items = make([]*document.Document, 0)
		}

		return items, nil
	})
}
