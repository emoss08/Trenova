package documentextractiontest

import (
	"context"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/ports/documentshipmentdraftports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type MemoryDocumentRepository struct {
	documentports.DocumentRepository

	mu      sync.Mutex
	Doc     *document.Document
	Updates []documentports.UpdateDocumentIntelligenceRequest
}

func (r *MemoryDocumentRepository) GetByID(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if req.ID != r.Doc.ID {
		return nil, errortypes.NewNotFoundError("document not found")
	}
	clone := *r.Doc

	return &clone, nil
}

func (r *MemoryDocumentRepository) UpdateIntelligence(
	_ context.Context,
	req *documentports.UpdateDocumentIntelligenceRequest,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Updates = append(r.Updates, *req)
	r.Doc.ContentStatus = req.ContentStatus
	r.Doc.ContentError = req.ContentError
	r.Doc.DetectedKind = req.DetectedKind
	r.Doc.HasExtractedText = req.HasExtractedText
	r.Doc.ShipmentDraftStatus = req.ShipmentDraftStatus
	r.Doc.DocumentTypeID = req.DocumentTypeID

	return nil
}

func (r *MemoryDocumentRepository) Last() documentports.UpdateDocumentIntelligenceRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Updates[len(r.Updates)-1]
}

type MemoryContentRepository struct {
	documentcontentports.DocumentContentRepository

	mu      sync.Mutex
	Content *documentcontent.Content
	Pages   []*documentcontent.Page
	Writes  []documentcontent.Content
}

func (r *MemoryContentRepository) GetByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Content == nil {
		return nil, errortypes.NewNotFoundError("content not found")
	}
	clone := *r.Content

	return &clone, nil
}

func (r *MemoryContentRepository) ListPagesByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*documentcontent.Page, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*documentcontent.Page, 0, len(r.Pages))
	for _, page := range r.Pages {
		clone := *page
		out = append(out, &clone)
	}

	return out, nil
}

func (r *MemoryContentRepository) ReplacePages(
	_ context.Context,
	_ *documentcontent.Content,
	pages []*documentcontent.Page,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Pages = make([]*documentcontent.Page, 0, len(pages))
	for _, page := range pages {
		stored := *page
		stored.Metadata = jsonRoundTrip(page.Metadata)
		r.Pages = append(r.Pages, &stored)
	}

	return nil
}

func (r *MemoryContentRepository) Upsert(
	_ context.Context,
	entity *documentcontent.Content,
) (*documentcontent.Content, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Content != nil {
		entity.ID = r.Content.ID
	}
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("dc_")
	}
	stored := *entity
	r.Content = &stored
	r.Writes = append(r.Writes, stored)

	return entity, nil
}

type MemoryDraftRepository struct {
	documentshipmentdraftports.DocumentShipmentDraftRepository

	mu     sync.Mutex
	Drafts []documentshipmentdraft.DocumentShipmentDraft
}

func (r *MemoryDraftRepository) GetByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.Drafts) == 0 {
		return nil, errortypes.NewNotFoundError("draft not found")
	}
	clone := r.Drafts[len(r.Drafts)-1]

	return &clone, nil
}

func (r *MemoryDraftRepository) Upsert(
	_ context.Context,
	entity *documentshipmentdraft.DocumentShipmentDraft,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Drafts = append(r.Drafts, *entity)

	return entity, nil
}

func jsonRoundTrip(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	encoded, err := sonic.Marshal(metadata)
	if err != nil {
		panic(err)
	}
	var decoded map[string]any
	if err = sonic.Unmarshal(encoded, &decoded); err != nil {
		panic(err)
	}

	return decoded
}
