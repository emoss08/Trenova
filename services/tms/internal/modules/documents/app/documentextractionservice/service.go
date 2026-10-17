//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package documentextractionservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	workflowstarterservice "github.com/emoss08/trenova/internal/infrastructure/temporaljobs/workflowstarter"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentextractioncontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentsearchprojectionports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/ports/documentshipmentdraftports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/serviceerror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	Config              *config.Config
	Metrics             *metrics.Registry
	DocumentControlRepo tenantports.DocumentControlRepository
	DocumentRepo        documentports.DocumentRepository
	ContentRepo         documentcontentports.DocumentContentRepository
	DraftRepo           documentshipmentdraftports.DocumentShipmentDraftRepository
	SearchRepo          customerports.SearchRepository
	SearchProjection    documentsearchprojectionports.DocumentSearchProjectionService
	WorkflowStarter     kernelports.WorkflowStarter
	Plans               platformplanports.PlanService               `optional:"true"`
	AIDocuments         documentaiextractionports.AIDocumentService `optional:"true"`
}

type Service struct {
	logger              *zap.Logger
	metrics             *metrics.Registry
	searchConfig        *config.SearchConfig
	documentControlRepo tenantports.DocumentControlRepository
	documentRepo        documentports.DocumentRepository
	contentRepo         documentcontentports.DocumentContentRepository
	draftRepo           documentshipmentdraftports.DocumentShipmentDraftRepository
	searchRepo          customerports.SearchRepository
	searchProjection    documentsearchprojectionports.DocumentSearchProjectionService
	workflowStarter     kernelports.WorkflowStarter
	plans               platformplanports.PlanService
	aiDocuments         documentaiextractionports.AIDocumentService
}

var (
	_ documentports.DocumentContentService = (*Service)(nil)
	_ documentports.ExtractionStatusReader = (*Service)(nil)
)

func New(p Params) *Service {
	searchProjection := p.SearchProjection
	if searchProjection == nil {
		searchProjection = noopDocumentSearchProjectionService{}
	}

	workflowStarter := p.WorkflowStarter
	if workflowStarter == nil {
		workflowStarter = workflowstarterservice.New(workflowstarterservice.Params{})
	}

	return &Service{
		logger:              p.Logger.Named("service.document-extraction"),
		searchConfig:        p.Config.GetSearchConfig(),
		metrics:             p.Metrics,
		documentControlRepo: p.DocumentControlRepo,
		documentRepo:        p.DocumentRepo,
		contentRepo:         p.ContentRepo,
		draftRepo:           p.DraftRepo,
		searchRepo:          p.SearchRepo,
		searchProjection:    searchProjection,
		workflowStarter:     workflowStarter,
		plans:               p.Plans,
		aiDocuments:         p.AIDocuments,
	}
}

func AsContentService(s *Service) documentports.DocumentContentService { return s }

func AsExtractionStatusReader(s *Service) documentports.ExtractionStatusReader { return s }

func (s *Service) ExtractionStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*documentports.ExtractionStatus, error) {
	control, err := s.documentControlRepo.GetOrCreate(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		return nil, err
	}

	providerAvailable := false
	if s.aiDocuments != nil {
		if providerAvailable, err = s.aiDocuments.ExtractionAvailable(ctx, tenantInfo); err != nil {
			return nil, err
		}
	}

	return &documentports.ExtractionStatus{
		Engine:                 string(control.ExtractionEngine),
		ModelProviderAvailable: providerAvailable,
		ExtractionAvailable:    !control.UsesDocling() || providerAvailable,
	}, nil
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
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	return s.draftRepo.GetByDocumentID(ctx, documentID, tenantInfo)
}

func (s *Service) SearchDocuments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType, resourceID, query string,
) ([]*document.Document, error) {
	if docs, err := s.searchDocumentsViaSearchIndex(
		ctx,
		tenantInfo,
		resourceType,
		resourceID,
		query,
	); err == nil &&
		len(strings.TrimSpace(query)) > 0 {
		return docs, nil
	} else if err != nil {
		s.metrics.Document.RecordSearchQuery("postgres", "fallback")
	}

	return s.contentRepo.SearchByResource(ctx, &documentcontentports.DocumentContentSearchRequest{
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
	doc, err := s.documentRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if !doc.ProcessingProfile.SupportsIntelligence() {
		return errortypes.NewConflictError(
			"Re-extraction is only available for rate confirmation import documents",
		)
	}
	if err = planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilityDocumentIntelligence,
	); err != nil {
		return err
	}

	doc.ContentStatus = document.ContentStatusPending
	doc.ContentError = ""
	doc.HasExtractedText = false
	doc.DetectedKind = ""
	doc.ShipmentDraftStatus = document.ShipmentDraftStatusUnavailable
	if err = s.documentRepo.UpdateIntelligence(
		ctx,
		&documentports.UpdateDocumentIntelligenceRequest{
			ID:                  doc.ID,
			TenantInfo:          tenantInfo,
			ContentStatus:       doc.ContentStatus,
			ContentError:        doc.ContentError,
			DetectedKind:        doc.DetectedKind,
			HasExtractedText:    doc.HasExtractedText,
			ShipmentDraftStatus: doc.ShipmentDraftStatus,
			DocumentTypeID:      doc.DocumentTypeID,
		},
	); err != nil {
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
	if !doc.ProcessingProfile.SupportsIntelligence() {
		s.metrics.Document.RecordExtraction("skipped", "", "processing_profile_disabled")
		return nil
	}
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

	err = documentextractioncontract.Start(
		ctx,
		s.workflowStarter,
		&documentextractioncontract.ProcessDocumentExtractionPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: doc.OrganizationID,
				BusinessUnitID: doc.BusinessUnitID,
				UserID:         userID,
			},
			DocumentID: doc.ID,
		},
	)
	if err != nil {
		if _, ok := errors.AsType[*serviceerror.WorkflowExecutionAlreadyStarted](err); ok {
			return nil
		}

		s.logger.Warn("failed to start document extraction workflow",
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
	documentIndex := s.searchConfig.Meilisearch.Indexes.Documents

	query = strings.TrimSpace(query)
	if query == "" || s.searchRepo == nil || !s.searchRepo.Enabled() || documentIndex == "" {
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

	hits, err := s.searchRepo.Search(ctx, customerports.SearchRequest{
		Index:  documentIndex,
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

	docs, err := s.documentRepo.GetByIDs(ctx, documentports.BulkDeleteDocumentRequest{
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
