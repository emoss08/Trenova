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

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
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

func (o *observed) Create(ctx context.Context, entity *document.Document) (*document.Document, error) {
	created, err := o.DocumentRepository.Create(ctx, entity)
	if err == nil {
		o.notify(ctx, created)
	}
	return created, err
}

func (o *observed) Update(ctx context.Context, entity *document.Document) (*document.Document, error) {
	updated, err := o.DocumentRepository.Update(ctx, entity)
	if err == nil {
		o.notify(ctx, updated)
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

func (o *observed) PromoteVersion(
	ctx context.Context,
	req *documentports.PromoteDocumentVersionRequest,
) error {
	before := o.lineage(ctx, req.TenantInfo, req.LineageID)
	err := o.DocumentRepository.PromoteVersion(ctx, req)
	if err == nil {
		o.notify(ctx, before...)
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
		o.notify(ctx, append(before, moved)...)
	}
	return err
}

func (o *observed) Delete(ctx context.Context, req documentports.DeleteDocumentRequest) error {
	before := o.load(ctx, req.ID, req.TenantInfo)
	err := o.DocumentRepository.Delete(ctx, req)
	if err == nil {
		o.notify(ctx, before...)
	}
	return err
}

func (o *observed) BulkDelete(ctx context.Context, req documentports.BulkDeleteDocumentRequest) error {
	before, loadErr := o.DocumentRepository.GetByIDs(ctx, req)
	if loadErr != nil {
		o.l.Warn("could not read documents before deleting them", zap.Error(loadErr))
	}
	err := o.DocumentRepository.BulkDelete(ctx, req)
	if err == nil {
		o.notify(ctx, before...)
	}
	return err
}

func (o *observed) DeleteByLineageIDs(
	ctx context.Context,
	req documentports.DeleteDocumentLineageRequest,
) error {
	before := make([]*document.Document, 0, len(req.LineageIDs))
	for _, lineageID := range req.LineageIDs {
		before = append(before, o.lineage(ctx, req.TenantInfo, lineageID)...)
	}
	err := o.DocumentRepository.DeleteByLineageIDs(ctx, req)
	if err == nil {
		o.notify(ctx, before...)
	}
	return err
}

func (o *observed) load(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) []*document.Document {
	doc, err := o.DocumentRepository.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil
	}
	return []*document.Document{doc}
}

func (o *observed) lineage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	lineageID pulid.ID,
) []*document.Document {
	versions, err := o.DocumentRepository.ListVersions(ctx, documentports.ListDocumentVersionsRequest{
		LineageID:  lineageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil
	}
	if len(versions) > 1 {
		return versions[:1]
	}
	return versions
}

func (o *observed) notify(ctx context.Context, docs ...*document.Document) {
	changes := make([]*documentports.DocumentChange, 0, 1)
	for _, doc := range docs {
		if doc == nil || doc.ResourceType == "" || doc.ResourceID == "" {
			continue
		}
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
