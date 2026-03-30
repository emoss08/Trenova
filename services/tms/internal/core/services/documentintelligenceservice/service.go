package documentintelligenceservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	workflowstarterservice "github.com/emoss08/trenova/internal/core/services/workflowstarter"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentintelligencejobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	Config              *config.Config
	Metrics             *metrics.Registry
	DocumentControlRepo repositories.DocumentControlRepository
	DocumentRepo        repositories.DocumentRepository
	ContentRepo         repositories.DocumentContentRepository
	DraftRepo           repositories.DocumentShipmentDraftRepository
	SearchRepo          repositories.SearchRepository
	SearchProjection    serviceports.DocumentSearchProjectionService
	WorkflowStarter     serviceports.WorkflowStarter
}

type Service struct {
	logger              *zap.Logger
	documentIndex       string
	metrics             *metrics.Registry
	documentControlRepo repositories.DocumentControlRepository
	documentRepo        repositories.DocumentRepository
	contentRepo         repositories.DocumentContentRepository
	draftRepo           repositories.DocumentShipmentDraftRepository
	searchRepo          repositories.SearchRepository
	searchProjection    serviceports.DocumentSearchProjectionService
	workflowStarter     serviceports.WorkflowStarter
}

var _ serviceports.DocumentContentService = (*Service)(nil)

func New(p Params) serviceports.DocumentContentService {
	searchProjection := p.SearchProjection
	if searchProjection == nil {
		searchProjection = noopDocumentSearchProjectionService{}
	}

	workflowStarter := p.WorkflowStarter
	if workflowStarter == nil {
		workflowStarter = workflowstarterservice.New(workflowstarterservice.Params{})
	}

	return &Service{
		logger:              p.Logger.Named("service.document-intelligence"),
		documentIndex:       p.Config.GetSearchConfig().Meilisearch.Indexes.Documents,
		metrics:             p.Metrics,
		documentControlRepo: p.DocumentControlRepo,
		documentRepo:        p.DocumentRepo,
		contentRepo:         p.ContentRepo,
		draftRepo:           p.DraftRepo,
		searchRepo:          p.SearchRepo,
		searchProjection:    searchProjection,
		workflowStarter:     workflowStarter,
	}
}

func (s *Service) GetContent(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentcontent.Content, error) {
	content, err := s.contentRepo.GetByDocumentID(ctx, documentID, tenantInfo)
	if err != nil {
		return nil, err
	}

	pages, err := s.contentRepo.ListPagesByDocumentID(ctx, documentID, tenantInfo)
	if err != nil {
		return nil, err
	}
	content.Pages = pages

	return content, nil
}

func (s *Service) GetShipmentDraft(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentshipmentdraft.Draft, error) {
	return s.draftRepo.GetByDocumentID(ctx, documentID, tenantInfo)
}

func (s *Service) SearchDocuments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType, resourceID, query string,
) ([]*document.Document, error) {
	if docs, err := s.searchDocumentsViaSearchIndex(ctx, tenantInfo, resourceType, resourceID, query); err == nil && len(strings.TrimSpace(query)) > 0 {
		return docs, nil
	} else if err != nil {
		s.metrics.Document.RecordSearchQuery("postgres", "fallback")
	}

	return s.contentRepo.SearchByResource(ctx, &repositories.DocumentContentSearchRequest{
		TenantInfo:   tenantInfo,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Query:        strings.TrimSpace(query),
		Limit:        250,
	})
}

func (s *Service) Reextract(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	doc, err := s.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	doc.ContentStatus = document.ContentStatusPending
	doc.ContentError = ""
	doc.HasExtractedText = false
	doc.DetectedKind = ""
	doc.ShipmentDraftStatus = document.ShipmentDraftStatusUnavailable
	if err = s.documentRepo.UpdateIntelligence(ctx, &repositories.UpdateDocumentIntelligenceRequest{
		ID:                  doc.ID,
		TenantInfo:          tenantInfo,
		ContentStatus:       doc.ContentStatus,
		ContentError:        doc.ContentError,
		DetectedKind:        doc.DetectedKind,
		HasExtractedText:    doc.HasExtractedText,
		ShipmentDraftStatus: doc.ShipmentDraftStatus,
		DocumentTypeID:      doc.DocumentTypeID,
	}); err != nil {
		return err
	}
	if err = s.searchProjection.Upsert(ctx, doc, ""); err != nil {
		s.logger.Warn("failed to sync search projection during reextract",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
	}

	content := &documentcontent.Content{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		Status:         documentcontent.StatusPending,
	}
	if _, err = s.contentRepo.Upsert(ctx, content); err != nil {
		return err
	}
	if err = s.contentRepo.ReplacePages(ctx, content, nil); err != nil {
		return err
	}

	return s.EnqueueExtraction(ctx, doc, tenantInfo.UserID)
}

func (s *Service) EnqueueExtraction(
	ctx context.Context,
	doc *document.Document,
	userID pulid.ID,
) error {
	if !s.workflowStarter.Enabled() {
		return nil
	}
	control, err := s.documentControlRepo.GetOrCreate(ctx, doc.OrganizationID, doc.BusinessUnitID)
	if err != nil {
		return err
	}
	if !control.EnableDocumentIntelligence {
		s.metrics.Document.RecordExtraction("skipped", "", "control_disabled")
		return nil
	}

	_, err = s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("document-intelligence-%s", doc.ID.String()),
			TaskQueue:                                temporaltype.DocumentIntelligenceTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			StaticSummary:                            fmt.Sprintf("Extracting content for document %s", doc.ID),
		},
		"ProcessDocumentIntelligenceWorkflow",
		&documentintelligencejobs.ProcessDocumentIntelligencePayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: doc.OrganizationID,
				BusinessUnitID: doc.BusinessUnitID,
				UserID:         userID,
			},
			DocumentID: doc.ID,
		},
	)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return nil
		}
		s.logger.Warn("failed to start document intelligence workflow",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
		s.metrics.Document.RecordReconciliationQueue(false)
	} else {
		s.metrics.Document.RecordReconciliationQueue(true)
	}

	return nil
}

func (s *Service) searchDocumentsViaSearchIndex(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType, resourceID, query string,
) ([]*document.Document, error) {
	query = strings.TrimSpace(query)
	if query == "" || s.searchRepo == nil || !s.searchRepo.Enabled() || s.documentIndex == "" {
		s.metrics.Document.RecordSearchQuery("postgres", "direct")
		return nil, nil
	}

	filter := fmt.Sprintf(
		`organization_id = "%s" AND business_unit_id = "%s" AND resource_type = "%s" AND resource_id = "%s"`,
		tenantInfo.OrgID,
		tenantInfo.BuID,
		resourceType,
		resourceID,
	)
	hits, err := s.searchRepo.Search(ctx, repositories.SearchRequest{
		Index:  s.documentIndex,
		Query:  query,
		Limit:  250,
		Filter: filter,
	})
	if err != nil {
		s.logger.Warn("document intelligence meilisearch query failed", zap.Error(err))
		s.metrics.Document.RecordSearchQuery("meilisearch", "error")
		return nil, err
	}
	s.metrics.Document.RecordSearchQuery("meilisearch", "success")

	ids := make([]pulid.ID, 0, len(hits))
	order := make([]string, 0, len(hits))
	for _, hit := range hits {
		rawID, ok := hit["id"]
		if !ok {
			continue
		}
		parsedID, parseErr := pulid.Parse(fmt.Sprintf("%v", rawID))
		if parseErr != nil {
			continue
		}
		ids = append(ids, parsedID)
		order = append(order, parsedID.String())
	}
	if len(ids) == 0 {
		return []*document.Document{}, nil
	}

	docs, err := s.documentRepo.GetByIDs(ctx, repositories.BulkDeleteDocumentRequest{
		IDs:        ids,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	byID := make(map[string]*document.Document, len(docs))
	for _, doc := range docs {
		byID[doc.ID.String()] = doc
	}

	ordered := make([]*document.Document, 0, len(order))
	for _, id := range order {
		if doc, ok := byID[id]; ok {
			ordered = append(ordered, doc)
		}
	}
	if len(ordered) == 0 {
		return []*document.Document{}, nil
	}

	return ordered, nil
}
