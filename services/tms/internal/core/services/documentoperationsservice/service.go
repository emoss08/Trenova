//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package documentoperationsservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	workflowstarterservice "github.com/emoss08/trenova/internal/core/services/workflowstarter"
	"github.com/emoss08/trenova/internal/core/temporaljobs/thumbnailjobs/thumbnailcontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentsearchprojectionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
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

	Logger           *zap.Logger
	DocumentRepo     documentports.DocumentRepository
	SessionRepo      documentuploadports.DocumentUploadSessionRepository
	ContentService   documentports.DocumentContentService
	SearchProjection documentsearchprojectionports.DocumentSearchProjectionService
	WorkflowStarter  kernelports.WorkflowStarter
}

type Service struct {
	logger           *zap.Logger
	documentRepo     documentports.DocumentRepository
	sessionRepo      documentuploadports.DocumentUploadSessionRepository
	contentService   documentports.DocumentContentService
	searchProjection documentsearchprojectionports.DocumentSearchProjectionService
	workflowStarter  kernelports.WorkflowStarter
}

type WorkflowReference struct {
	Kind       string `json:"kind"`
	WorkflowID string `json:"workflowId"`
}

type Diagnostics struct {
	Document      *document.Document                           `json:"document"`
	Versions      []*document.Document                         `json:"versions"`
	Sessions      []*documentupload.DocumentUploadSession      `json:"sessions"`
	Content       *documentcontent.Content                     `json:"content,omitempty"`
	ShipmentDraft *documentshipmentdraft.DocumentShipmentDraft `json:"shipmentDraft,omitempty"`
	LastErrors    []string                                     `json:"lastErrors"`
	WorkflowRefs  []WorkflowReference                          `json:"workflowRefs"`
}

func New(p Params) *Service {
	workflowStarter := p.WorkflowStarter
	if workflowStarter == nil {
		workflowStarter = workflowstarterservice.New(workflowstarterservice.Params{})
	}

	return &Service{
		logger:           p.Logger.Named("service.document-operations"),
		documentRepo:     p.DocumentRepo,
		sessionRepo:      p.SessionRepo,
		contentService:   p.ContentService,
		searchProjection: p.SearchProjection,
		workflowStarter:  workflowStarter,
	}
}

func (s *Service) GetDiagnostics(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*Diagnostics, error) {
	doc, err := s.documentRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	versions, err := s.documentRepo.ListVersions(ctx, documentports.ListDocumentVersionsRequest{
		LineageID:  doc.LineageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	sessions, err := s.sessionRepo.ListRelated(
		ctx,
		&documentuploadports.ListRelatedDocumentUploadSessionsRequest{
			TenantInfo: tenantInfo,
			DocumentID: doc.ID,
			LineageID:  doc.LineageID,
		},
	)
	if err != nil {
		return nil, err
	}

	content, err := s.contentService.GetContent(ctx, doc.ID, tenantInfo)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	if errortypes.IsNotFoundError(err) {
		content = nil
	}

	shipmentDraft, err := s.contentService.GetShipmentDraft(ctx, doc.ID, tenantInfo)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	if errortypes.IsNotFoundError(err) {
		shipmentDraft = nil
	}

	return &Diagnostics{
		Document:      doc,
		Versions:      versions,
		Sessions:      sessions,
		Content:       content,
		ShipmentDraft: shipmentDraft,
		LastErrors:    collectErrors(doc, sessions),
		WorkflowRefs:  workflowReferences(doc, versions, sessions),
	}, nil
}

func (s *Service) Reextract(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	return s.contentService.Reextract(ctx, documentID, tenantInfo)
}

func (s *Service) ResyncSearch(
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

	contentText := ""
	content, err := s.contentService.GetContent(ctx, doc.ID, tenantInfo)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}
	if content != nil {
		contentText = content.ContentText
	}

	if err = s.searchProjection.Upsert(ctx, doc, contentText); err != nil {
		return err
	}

	return nil
}

func (s *Service) RegeneratePreview(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	if !s.workflowStarter.Enabled() {
		return errortypes.NewConflictError("Thumbnail regeneration is unavailable")
	}

	doc, err := s.documentRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if !document.SupportsPreview(doc.FileType) {
		return errortypes.NewConflictError(
			"Preview generation is not supported for this document type",
		)
	}

	payload := &thumbnailcontract.GenerateThumbnailPayload{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		StoragePath:    doc.StoragePath,
		ContentType:    doc.FileType,
		ResourceType:   doc.ResourceType,
	}

	_, err = s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("thumbnail-%s", doc.ID.String()),
			TaskQueue:                                temporaltype.ThumbnailTaskQueue,
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary: fmt.Sprintf(
				"Generating thumbnail for document %s",
				doc.ID,
			),
		},
		"GenerateThumbnailWorkflow",
		payload,
	)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if !errors.As(err, &alreadyStarted) {
			return err
		}
	}

	return s.documentRepo.UpdatePreview(ctx, &documentports.UpdateDocumentPreviewRequest{
		ID:                 doc.ID,
		TenantInfo:         tenantInfo,
		PreviewStatus:      document.PreviewStatusPending,
		PreviewStoragePath: "",
	})
}

func collectErrors(
	doc *document.Document,
	sessions []*documentupload.DocumentUploadSession,
) []string {
	errorsList := make([]string, 0, 1+len(sessions))
	if doc.ContentError != "" {
		errorsList = append(errorsList, fmt.Sprintf("content: %s", doc.ContentError))
	}

	for _, session := range sessions {
		if session == nil || session.FailureCode == "" && session.FailureMessage == "" {
			continue
		}

		msg := session.FailureMessage
		if msg == "" {
			msg = session.FailureCode
		} else if session.FailureCode != "" {
			msg = fmt.Sprintf("%s: %s", session.FailureCode, session.FailureMessage)
		}

		errorsList = append(errorsList, fmt.Sprintf("upload session %s: %s", session.ID, msg))
	}

	return slices.Compact(errorsList)
}

func workflowReferences(
	doc *document.Document,
	versions []*document.Document,
	sessions []*documentupload.DocumentUploadSession,
) []WorkflowReference {
	refs := make([]WorkflowReference, 0, len(versions)*2+len(sessions))
	seen := make(map[string]struct{}, len(versions)*2+len(sessions))

	add := func(kind, workflowID string) {
		if workflowID == "" {
			return
		}
		key := kind + ":" + workflowID
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		refs = append(refs, WorkflowReference{Kind: kind, WorkflowID: workflowID})
	}

	if doc != nil {
		add("thumbnail", fmt.Sprintf("thumbnail-%s", doc.ID))
		add("document_intelligence", fmt.Sprintf("document-intelligence-%s", doc.ID))
	}

	for _, version := range versions {
		if version == nil || doc != nil && version.ID == doc.ID {
			continue
		}
		add("thumbnail", fmt.Sprintf("thumbnail-%s", version.ID))
		add("document_intelligence", fmt.Sprintf("document-intelligence-%s", version.ID))
	}

	for _, session := range sessions {
		if session == nil {
			continue
		}
		add("upload_finalize", fmt.Sprintf("document-upload-finalize-%s", session.ID))
	}

	return refs
}
