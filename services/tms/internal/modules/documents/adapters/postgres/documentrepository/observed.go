package documentrepository

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ObservedParams struct {
	fx.In

	Repo      documentports.DocumentRepository
	Observers []documentports.DocumentChangeObserver `group:"document_change_observers"`
	Logger    *zap.Logger
}

type observed struct {
	documentports.DocumentRepository
	observers []documentports.DocumentChangeObserver
	l         *zap.Logger
}

func Observe(p ObservedParams) documentports.DocumentRepository {
	observers := make([]documentports.DocumentChangeObserver, 0, len(p.Observers))
	for _, o := range p.Observers {
		if o != nil {
			observers = append(observers, o)
		}
	}
	if len(observers) == 0 {
		return p.Repo
	}
	return &observed{
		DocumentRepository: p.Repo,
		observers:          observers,
		l:                  p.Logger.Named("repository.document-observed"),
	}
}

func (o *observed) Create(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	created, err := o.DocumentRepository.Create(ctx, entity)
	if err == nil {
		o.notify(ctx, created)
	}
	return created, err
}

func (o *observed) Update(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	updated, err := o.DocumentRepository.Update(ctx, entity)
	if err == nil {
		var retired []retirement
		if updated != nil && !inUse(updated) {
			retired = append(retired, retirement{doc: updated})
		}
		o.publish(ctx, []*document.Document{updated}, retired)
	}
	return updated, err
}

func (o *observed) UpdateIntelligence(
	ctx context.Context,
	req *documentports.UpdateDocumentIntelligenceRequest,
) error {
	before := o.load(ctx, req.ID, req.TenantInfo)
	err := o.DocumentRepository.UpdateIntelligence(ctx, req)
	if err == nil {
		o.notify(ctx, before...)
	}
	return err
}

func (o *observed) RecordDetectedSignature(
	ctx context.Context,
	req *documentports.RecordDetectedSignatureRequest,
) (bool, error) {
	changed, err := o.DocumentRepository.RecordDetectedSignature(ctx, req)
	if err == nil && changed {
		o.notify(ctx, o.load(ctx, req.ID, req.TenantInfo)...)
	}
	return changed, err
}

func (o *observed) RecordDeliveryFacts(
	ctx context.Context,
	req *documentports.RecordDeliveryFactsRequest,
) (*document.Document, error) {
	written, err := o.DocumentRepository.RecordDeliveryFacts(ctx, req)
	if err == nil && written != nil {
		o.notify(ctx, written)
	}
	return written, err
}

func (o *observed) PromoteVersion(
	ctx context.Context,
	req *documentports.PromoteDocumentVersionRequest,
) error {
	versions := o.versions(ctx, req.TenantInfo, req.LineageID)
	err := o.DocumentRepository.PromoteVersion(ctx, req)
	if err == nil {
		retired := make([]retirement, 0, 1)
		for _, version := range versions {
			if version.IsCurrentVersion && version.ID != req.CurrentDocumentID {
				retired = append(
					retired,
					retirement{doc: version, replacedBy: req.CurrentDocumentID},
				)
			}
		}
		o.publish(ctx, newest(versions), retired)
	}
	return err
}

func (o *observed) MoveLineageToResource(
	ctx context.Context,
	req *documentports.MoveDocumentLineageRequest,
) error {
	before := o.load(ctx, req.DocumentID, req.TenantInfo)
	err := o.DocumentRepository.MoveLineageToResource(ctx, req)
	if err == nil {
		moved := &document.Document{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			ResourceType:   req.ResourceType,
			ResourceID:     req.ResourceID,
		}
		o.publish(ctx, append(before, moved), retire(before))
	}
	return err
}

func (o *observed) Delete(ctx context.Context, req documentports.DeleteDocumentRequest) error {
	before := o.load(ctx, req.ID, req.TenantInfo)
	err := o.DocumentRepository.Delete(ctx, req)
	if err == nil {
		o.publish(ctx, before, retire(before))
	}
	return err
}

func (o *observed) BulkDelete(
	ctx context.Context,
	req documentports.BulkDeleteDocumentRequest,
) error {
	before, loadErr := o.GetByIDs(ctx, documentports.GetDocumentsByIDsRequest{
		IDs:        req.IDs,
		TenantInfo: req.TenantInfo,
	})
	if loadErr != nil {
		o.l.Warn("could not read documents before deleting them", zap.Error(loadErr))
	}
	err := o.DocumentRepository.BulkDelete(ctx, req)
	if err == nil {
		o.publish(ctx, before, retire(before))
	}
	return err
}

func (o *observed) DeleteByLineageIDs(
	ctx context.Context,
	req documentports.DeleteDocumentLineageRequest,
) error {
	before := make([]*document.Document, 0, len(req.LineageIDs))
	versions := make([]*document.Document, 0, len(req.LineageIDs))
	for _, lineageID := range req.LineageIDs {
		lineage := o.versions(ctx, req.TenantInfo, lineageID)
		before = append(before, newest(lineage)...)
		versions = append(versions, lineage...)
	}
	err := o.DocumentRepository.DeleteByLineageIDs(ctx, req)
	if err == nil {
		o.publish(ctx, before, retire(versions))
	}
	return err
}

func (o *observed) load(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) []*document.Document {
	doc, err := o.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil
	}
	return []*document.Document{doc}
}

func (o *observed) versions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	lineageID pulid.ID,
) []*document.Document {
	versions, err := o.ListVersions(
		ctx,
		documentports.ListDocumentVersionsRequest{
			LineageID:  lineageID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil
	}
	return versions
}

func newest(versions []*document.Document) []*document.Document {
	if len(versions) > 1 {
		return versions[:1]
	}
	return versions
}

type retirement struct {
	doc        *document.Document
	replacedBy pulid.ID
}

func retire(docs []*document.Document) []retirement {
	out := make([]retirement, 0, len(docs))
	for _, doc := range docs {
		out = append(out, retirement{doc: doc})
	}
	return out
}

func inUse(doc *document.Document) bool {
	return doc.IsCurrentVersion &&
		doc.Status != document.StatusRejected &&
		doc.Status != document.StatusArchived
}

func (o *observed) notify(ctx context.Context, docs ...*document.Document) {
	o.publish(ctx, docs, nil)
}

func (o *observed) publish(
	ctx context.Context,
	docs []*document.Document,
	retired []retirement,
) {
	changes := make([]*documentports.DocumentChange, 0, 1)
	changeFor := func(doc *document.Document) *documentports.DocumentChange {
		tenant := pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID}
		idx := slices.IndexFunc(changes, func(c *documentports.DocumentChange) bool {
			return c.ResourceType == doc.ResourceType && c.TenantInfo == tenant
		})
		if idx < 0 {
			changes = append(changes, &documentports.DocumentChange{
				TenantInfo:   tenant,
				ResourceType: doc.ResourceType,
			})
			idx = len(changes) - 1
		}
		if !slices.Contains(changes[idx].ResourceIDs, doc.ResourceID) {
			changes[idx].ResourceIDs = append(changes[idx].ResourceIDs, doc.ResourceID)
		}
		return changes[idx]
	}
	for _, doc := range docs {
		if doc == nil || doc.ResourceType == "" || doc.ResourceID == "" {
			continue
		}
		changeFor(doc)
	}
	for _, gone := range retired {
		if gone.doc == nil || gone.doc.ID.IsNil() || gone.doc.ResourceType == "" ||
			gone.doc.ResourceID == "" {
			continue
		}
		change := changeFor(gone.doc)
		change.Retired = append(change.Retired, documentports.RetiredDocument{
			ID:         gone.doc.ID,
			ResourceID: gone.doc.ResourceID,
			ReplacedBy: gone.replacedBy,
		})
	}
	if len(changes) == 0 {
		return
	}
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		for _, change := range changes {
			for _, observer := range o.observers {
				observer.DocumentsChanged(committed, change)
			}
		}
	})
}
