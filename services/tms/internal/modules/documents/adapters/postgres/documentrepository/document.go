package documentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultReconciliationLimit = 100

var documentErrors = dberror.Mapping{Entity: "Document"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Quota  platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db    *postgres.Connection
	l     *zap.Logger
	quota platformplanports.QuotaGuard
}

func New(p Params) documentports.DocumentRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.document-repository"),
		quota: quotaservice.OrUnlimited(p.Quota),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *documentports.ListDocumentsRequest,
) (*pagination.ListResult[*document.Document], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*document.Document], error) {
			d := documentdb.Documents
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				d.Where(
					d.IsCurrentVersion.IsTrue(),
					keel.When(req.ResourceID != "", d.ResourceID.Eq(req.ResourceID)),
					keel.When(req.ResourceType != "", d.ResourceType.Eq(req.ResourceType)),
					keel.When(req.Status != "", d.Status.Eq(document.Status(req.Status))),
				),
				&listquery.Request[document.Document]{
					Filter: req.Filter,
					Then:   []keel.Order{d.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list documents", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *documentports.DocumentSelectOptionsRequest,
) (*pagination.ListResult[*document.Document], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*document.Document], error) {
			d := documentdb.Documents
			q := d.Pick(
				d.ID,
				d.FileName,
				d.OriginalName,
				d.Status,
				d.ResourceID,
				d.ResourceType,
				d.DocumentTypeID,
				d.CreatedAt,
			).
				Where(
					d.IsCurrentVersion.IsTrue(),
					d.Status.Ne(document.StatusArchived),
					keel.When(req.ResourceID != "", d.ResourceID.Eq(req.ResourceID)),
					keel.When(req.ResourceType != "", d.ResourceType.Eq(req.ResourceType)),
				).
				Load(d.DocumentType()).
				Order(d.CreatedAt.Desc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						d.OriginalName.ContainsFold(s),
						d.FileName.ContainsFold(s),
						d.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*document.Document, error) {
		entity, err := documentdb.Documents.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, documentErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByStoragePath(
	ctx context.Context,
	req documentports.GetDocumentByStoragePathRequest,
) (*document.Document, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*document.Document, error) {
		d := documentdb.Documents
		entities, err := d.Where(d.StoragePath.Eq(req.StoragePath)).
			Limit(1).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if len(entities) == 0 {
			return nil, documentErrors.Map(keel.ErrNotFound, req.StoragePath)
		}

		return entities[0], nil
	})
}

func (r *repository) GetByResourceID(
	ctx context.Context,
	req *documentports.GetDocumentsByResourceRequest,
) ([]*document.Document, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*document.Document, error) {
		d := documentdb.Documents
		q := d.Where(
			d.ResourceID.Eq(req.ResourceID),
			d.ResourceType.Eq(req.ResourceType),
			d.IsCurrentVersion.IsTrue(),
		).Order(d.CreatedAt.Desc())
		if req.IncludeDocumentType {
			q = q.Load(d.DocumentType())
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get documents by resource", zap.Error(err))
			return nil, err
		}

		return orEmpty(entities), nil
	})
}

func (r *repository) GetByResourceIDs(
	ctx context.Context,
	req *documentports.GetDocumentsByResourceIDsRequest,
) ([]*document.Document, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*document.Document, error) {
		if len(req.ResourceIDs) == 0 {
			return []*document.Document{}, nil
		}

		d := documentdb.Documents
		q := d.Where(
			d.ResourceID.In(req.ResourceIDs...),
			d.ResourceType.Eq(req.ResourceType),
			d.IsCurrentVersion.IsTrue(),
		).Order(d.CreatedAt.Desc())
		if req.IncludeDocumentType {
			q = q.Load(d.DocumentType())
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get documents by resource ids", zap.Error(err))
			return nil, err
		}

		return orEmpty(entities), nil
	})
}

func (r *repository) ListPendingPreviewReconciliation(
	ctx context.Context,
	req *documentports.ListPendingPreviewReconciliationRequest,
) ([]*document.Document, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*document.Document, error) {
		d := documentdb.Documents
		entities, err := d.Where(pendingPreview(d), d.UpdatedAt.Lte(req.OlderThan)).
			Order(d.UpdatedAt.Asc()).
			Limit(reconciliationLimit(req.Limit)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list documents pending preview reconciliation", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) ListPendingPreviewReconciliationTenants(
	ctx context.Context,
	req *documentports.ListPendingPreviewReconciliationRequest,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with document previews to reconcile")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		type tenantRow struct {
			OrganizationID pulid.ID
			BusinessUnitID pulid.ID
		}

		d := documentdb.Documents
		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), d.AllTenants().
			Where(pendingPreview(d), d.UpdatedAt.Lte(req.OlderThan)).
			Group(d.OrganizationID, d.BusinessUnitID).
			Pick(d.OrganizationID, d.BusinessUnitID).
			Order(d.OrganizationID.Asc(), d.BusinessUnitID.Asc()).
			Limit(reconciliationLimit(req.Limit)))
		if err != nil {
			r.l.Error("failed to list document preview reconciliation tenants", zap.Error(err))
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

func pendingPreview(d *documentdb.DocumentsTable) keel.Pred {
	return keel.And(
		d.IsCurrentVersion.IsTrue(),
		d.PreviewStatus.Eq(document.PreviewStatusPending),
		keel.Or(d.PreviewStoragePath.IsNull(), d.PreviewStoragePath.Eq("")),
	)
}

func reconciliationLimit(limit int) int {
	if limit <= 0 {
		return defaultReconciliationLimit
	}

	return limit
}

func (r *repository) Create(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*document.Document, error) {
		err := quotatx.Run(ctx, r.db, r.quota, func(c context.Context) error {
			return documentdb.Documents.Create(c, r.db.Keel(c), entity)
		}, documentQuotaRequests(entity)...)
		if err != nil {
			r.l.Error(
				"failed to create document",
				zap.String("fileName", entity.FileName),
				zap.Error(err),
			)
			return nil, documentErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*document.Document, error) {
		if err := documentdb.Documents.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update document",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, documentErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdatePreview(
	ctx context.Context,
	req *documentports.UpdateDocumentPreviewRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := documentdb.Documents
		n, err := d.Where(d.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			d.PreviewStatus.To(req.PreviewStatus),
			d.PreviewStoragePath.To(req.PreviewStoragePath),
		)
		if err != nil {
			r.l.Error(
				"failed to update document preview",
				zap.Error(err),
				zap.String("id", req.ID.String()),
			)
			return documentErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(n, "Document", req.ID.String())
	})
}

func (r *repository) UpdateIntelligence(
	ctx context.Context,
	req *documentports.UpdateDocumentIntelligenceRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := documentdb.Documents
		documentType := d.DocumentTypeID.ToNull()
		if req.DocumentTypeID != nil {
			documentType = d.DocumentTypeID.To(*req.DocumentTypeID)
		}

		n, err := d.Where(d.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			d.ContentStatus.To(req.ContentStatus),
			d.ContentError.To(req.ContentError),
			d.DetectedKind.To(req.DetectedKind),
			d.HasExtractedText.To(req.HasExtractedText),
			d.ShipmentDraftStatus.To(req.ShipmentDraftStatus),
			documentType,
		)
		if err != nil {
			r.l.Error(
				"failed to update document intelligence",
				zap.Error(err),
				zap.String("id", req.ID.String()),
			)
			return documentErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(n, "Document", req.ID.String())
	})
}

func (r *repository) RecordDetectedSignature(
	ctx context.Context,
	req *documentports.RecordDetectedSignatureRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		d := documentdb.Documents
		signedAt := req.SignedAt
		if req.Status != document.SignatureStatusSigned {
			signedAt = nil
		}

		signed := d.SignedAt.ToNull()
		sameSignedAt := d.SignedAt.IsNull()
		if signedAt != nil {
			signed = d.SignedAt.To(*signedAt)
			sameSignedAt = keel.Not(d.SignedAt.IsDistinctFrom(*signedAt))
		}

		n, err := d.Where(
			d.ID.Eq(req.ID),
			d.SignatureSource.Ne(document.SignatureSourcePerson),
			keel.Not(keel.And(
				d.SignatureStatus.Eq(req.Status),
				sameSignedAt,
				keel.Not(d.SignatureSource.IsDistinctFrom(document.SignatureSourceExtraction)),
			)),
		).Update(
			ctx,
			r.db.Keel(ctx),
			d.SignatureStatus.To(req.Status),
			signed,
			d.SignatureSource.To(document.SignatureSourceExtraction),
			d.SignatureUpdatedByID.ToNull(),
			d.SignatureUpdatedAt.To(req.DetectedAt),
		)
		if err != nil {
			r.l.Error(
				"failed to record detected document signature",
				zap.Error(err),
				zap.String("id", req.ID.String()),
			)
			return false, documentErrors.Map(err, req.ID.String())
		}

		return n > 0, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req documentports.DeleteDocumentRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := documentdb.Documents
		n, err := d.Where(d.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete document",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return documentErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(n, "Document", req.ID.String())
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req documentports.BulkDeleteDocumentRequest,
) ([]*document.Document, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*document.Document, error) {
		if len(req.IDs) == 0 {
			return []*document.Document{}, nil
		}

		d := documentdb.Documents
		entities, err := d.Where(d.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get documents by IDs", zap.Error(err))
			return nil, err
		}

		return orEmpty(entities), nil
	})
}

func (r *repository) ListVersions(
	ctx context.Context,
	req documentports.ListDocumentVersionsRequest,
) ([]*document.Document, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*document.Document, error) {
		d := documentdb.Documents
		entities, err := d.Where(d.LineageID.Eq(req.LineageID)).
			Order(d.VersionNumber.Desc(), d.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		return orEmpty(entities), nil
	})
}

func (r *repository) BulkDelete(
	ctx context.Context,
	req documentports.BulkDeleteDocumentRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(req.IDs) == 0 {
			return nil
		}

		d := documentdb.Documents
		n, err := d.Where(d.ID.In(req.IDs...)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to bulk delete documents", zap.Error(err))
			return documentErrors.Map(err, "")
		}

		r.l.Info("bulk deleted documents", zap.Int64("rowsAffected", n))

		return nil
	})
}

func (r *repository) PromoteVersion(
	ctx context.Context,
	req *documentports.PromoteDocumentVersionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		d := documentdb.Documents

		if _, err := d.Where(d.LineageID.Eq(req.LineageID), d.IsCurrentVersion.IsTrue()).
			Update(ctx, db, d.IsCurrentVersion.To(false)); err != nil {
			return documentErrors.Map(err, req.CurrentDocumentID.String())
		}

		n, err := d.Where(d.ID.Eq(req.CurrentDocumentID), d.LineageID.Eq(req.LineageID)).
			Update(ctx, db, d.IsCurrentVersion.To(true))
		if err != nil {
			return documentErrors.Map(err, req.CurrentDocumentID.String())
		}

		return dberror.CheckAffected(n, "Document", req.CurrentDocumentID.String())
	})
}

func (r *repository) MoveLineageToResource(
	ctx context.Context,
	req *documentports.MoveDocumentLineageRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		d := documentdb.Documents

		current, err := d.Pick(d.LineageID).Where(d.ID.Eq(req.DocumentID)).One(ctx, db)
		if err != nil {
			return documentErrors.Map(err, req.DocumentID.String())
		}

		n, err := d.Where(d.LineageID.Eq(current.LineageID)).Update(
			ctx,
			db,
			d.ResourceID.To(req.ResourceID),
			d.ResourceType.To(req.ResourceType),
		)
		if err != nil {
			return documentErrors.Map(err, req.DocumentID.String())
		}

		return dberror.CheckAffected(n, "Document", req.DocumentID.String())
	})
}

func (r *repository) DeleteByLineageIDs(
	ctx context.Context,
	req documentports.DeleteDocumentLineageRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(req.LineageIDs) == 0 {
			return nil
		}

		d := documentdb.Documents
		_, err := d.Where(d.LineageID.In(req.LineageIDs...)).Delete(ctx, r.db.Keel(ctx))

		return documentErrors.Map(err, "")
	})
}

func documentQuotaRequests(entity *document.Document) []platformplanports.QuotaRequest {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
	size := max(entity.FileSize, 0)

	return []platformplanports.QuotaRequest{
		{TenantInfo: tenantInfo, Meter: platformcatalog.MeterDocumentFileBytes, Quantity: size},
		{TenantInfo: tenantInfo, Meter: platformcatalog.MeterDocumentUploads, Quantity: 1},
		{TenantInfo: tenantInfo, Meter: platformcatalog.MeterDocumentStorageBytes, Quantity: size},
	}
}

func orEmpty(items []*document.Document) []*document.Document {
	if items == nil {
		return []*document.Document{}
	}

	return items
}
