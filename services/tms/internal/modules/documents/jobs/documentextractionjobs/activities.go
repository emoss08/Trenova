package documentextractionjobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentaiextractionjobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionsupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/ports/documentshipmentdraftports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Logger                   *zap.Logger
	AIConfig                 *aiconfig.Config
	DocumentConversionConfig *documentconversionconfig.Config
	Metrics                  *metrics.Registry
	DocumentRepo             documentports.DocumentRepository
	DocumentTypeRepo         documenttypeports.DocumentTypeRepository
	ContentRepo              documentcontentports.DocumentContentRepository
	DraftRepo                documentshipmentdraftports.DocumentShipmentDraftRepository
	AIDocumentService        documentaiextractionports.AIDocumentService
	Storage                  kernelports.Client
	WorkflowStarter          kernelports.WorkflowStarter
	Encryption               *encryptionservice.Service
	Imaging                  kernelports.Imaging
	Converter                kernelports.DocumentConverter       `optional:"true"`
	Triage                   inboundmessageports.TriageEventSink `optional:"true"`
	Realtime                 realtimeports.RealtimeService       `optional:"true"`
	Support                  *documentextractionsupport.Service
}

type Activities struct {
	logger            *zap.Logger
	cfg               *aiconfig.Config
	metrics           *metrics.Registry
	documentRepo      documentports.DocumentRepository
	documentTypeRepo  documenttypeports.DocumentTypeRepository
	contentRepo       documentcontentports.DocumentContentRepository
	draftRepo         documentshipmentdraftports.DocumentShipmentDraftRepository
	aiDocumentService documentaiextractionports.AIDocumentService
	storage           kernelports.Client
	workflowStarter   kernelports.WorkflowStarter
	encryption        *encryptionservice.Service
	imaging           kernelports.Imaging
	converter         kernelports.DocumentConverter
	conversionCfg     *documentconversionconfig.Config
	triage            inboundmessageports.TriageEventSink
	realtime          realtimeports.RealtimeService
	support           *documentextractionsupport.Service
}

//nolint:gocritic // dependency injection param
func NewActivities(p ActivitiesParams) *Activities {
	aiDocumentService := p.AIDocumentService
	if aiDocumentService == nil {
		aiDocumentService = documentextractionsupport.NoopAIDocumentService{}
	}

	workflowStarter := p.WorkflowStarter
	if workflowStarter == nil {
		workflowStarter = noopWorkflowStarter{}
	}

	return &Activities{
		logger:            p.Logger.Named("temporal.document-extraction"),
		cfg:               p.AIConfig,
		metrics:           p.Metrics,
		documentRepo:      p.DocumentRepo,
		documentTypeRepo:  p.DocumentTypeRepo,
		contentRepo:       p.ContentRepo,
		draftRepo:         p.DraftRepo,
		aiDocumentService: aiDocumentService,
		storage:           p.Storage,
		workflowStarter:   workflowStarter,
		encryption:        p.Encryption,
		triage:            p.Triage,
		realtime:          p.Realtime,
		imaging:           p.Imaging,
		converter:         p.Converter,
		conversionCfg:     p.DocumentConversionConfig,
		support:           p.Support,
	}
}

func (a *Activities) markExtractionStarted(
	ctx context.Context,
	doc *document.Document,
	tenantInfo pagination.TenantInfo,
) error {
	now := timeutils.NowUnix()
	content := &documentcontent.Content{
		DocumentID:      doc.ID,
		OrganizationID:  doc.OrganizationID,
		BusinessUnitID:  doc.BusinessUnitID,
		Status:          documentcontent.StatusExtracting,
		LastExtractedAt: &now,
	}
	if _, err := a.contentRepo.Upsert(ctx, content); err != nil {
		return err
	}

	doc.ContentStatus = document.ContentStatusExtracting
	doc.ContentError = ""
	if err := a.documentRepo.UpdateIntelligence(ctx,
		&documentports.UpdateDocumentIntelligenceRequest{
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
	a.metrics.Document.RecordExtraction("started", "", "none")
	a.support.SyncSearchProjection(ctx, doc, "")
	return nil
}

func (a *Activities) finalizeProcessDocumentIntelligence(
	ctx context.Context,
	p *documentanalysis.FinalizeIntelligenceParams,
) error {
	draftStatus, err := a.upsertShipmentDraftForProcess(
		ctx,
		p.Document,
		p.Control,
		p.Classification,
		p.Intelligence,
		p.EnqueueAsyncAI,
	)
	if err != nil {
		return err
	}

	p.Document.ShipmentDraftStatus = draftStatus
	if err = a.documentRepo.UpdateIntelligence(
		ctx,
		&documentports.UpdateDocumentIntelligenceRequest{
			ID:                  p.Document.ID,
			TenantInfo:          p.TenantInfo,
			ContentStatus:       p.Document.ContentStatus,
			ContentError:        p.Document.ContentError,
			DetectedKind:        p.Document.DetectedKind,
			HasExtractedText:    p.Document.HasExtractedText,
			ShipmentDraftStatus: p.Document.ShipmentDraftStatus,
			DocumentTypeID:      p.Document.DocumentTypeID,
		},
	); err != nil {
		return err
	}
	a.recordDetectedSignature(ctx, p.Document, p.TenantInfo, p.Signature)
	indexedText := p.Extracted.Text
	if !p.Control.EnableFullTextIndexing {
		indexedText = ""
	}
	a.support.SyncSearchProjection(ctx, p.Document, indexedText)
	inboundmessageports.AnnounceDocumentAttached(ctx, a.triage, p.Document)

	return a.maybeApplyAsyncAIEnqueueFailure(ctx, p)
}

func (a *Activities) persistIndexedContentAndAssociate(
	ctx context.Context,
	payload *documentanalysis.PersistIndexedContentPayload,
) error {
	structured := documentanalysis.BuildStructuredData(payload.Intelligence, payload.AIDiagnostics)
	payload.Content.Status = documentcontent.StatusIndexed
	payload.Content.ContentText = payload.Extracted.Text
	payload.Content.PageCount = payload.Extracted.PageCount
	payload.Content.SourceKind = payload.Extracted.SourceKind
	payload.Content.DetectedLanguage = "en"
	payload.Content.DetectedDocumentKind = payload.Classification.Kind
	payload.Content.ClassificationConfidence = payload.Classification.Confidence
	payload.Content.StructuredData = structured
	payload.Content.FailureCode = ""
	payload.Content.FailureMessage = ""
	payload.Content.LastExtractedAt = &payload.Timestamp

	if _, err := a.contentRepo.Upsert(ctx, payload.Content); err != nil {
		return err
	}
	a.metrics.Document.RecordExtraction("succeeded", payload.Extracted.SourceKind, "none")

	payload.Document.ContentStatus = document.ContentStatusIndexed
	payload.Document.ContentError = ""
	payload.Document.HasExtractedText = strings.TrimSpace(payload.Extracted.Text) != ""
	payload.Document.DetectedKind = payload.Classification.Kind
	if err := a.associateDocumentType(
		ctx,
		payload.Document,
		payload.TenantInfo,
		payload.Classification.Kind,
		payload.Control,
	); err != nil {
		return err
	}
	return nil
}

func (a *Activities) upsertShipmentDraftForProcess(
	ctx context.Context,
	doc *document.Document,
	control *documentcontrol.DocumentControl,
	classification *documentanalysis.ClassificationResult,
	intelligence *documentanalysis.DocumentIntelligenceAnalysis,
	enqueueAsyncAI bool,
) (document.ShipmentDraftStatus, error) {
	isUsable := documentaiextractionjobs.CanGenerateShipmentDraft(
		control,
		doc.ResourceType,
		classification.Kind,
	) &&
		documentaiextractionjobs.HasUsableShipmentDraft(intelligence)

	if isUsable {
		draftStatus, err := a.support.UpsertDraft(ctx, doc, classification.Kind, intelligence, true)
		if err != nil {
			return "", err
		}
		a.metrics.Document.RecordShipmentDraft("ready", doc.ResourceType, classification.Kind)
		return draftStatus, nil
	}

	if documentaiextractionjobs.CanGenerateShipmentDraft(
		control,
		doc.ResourceType,
		classification.Kind,
	) &&
		enqueueAsyncAI {
		if _, err := a.draftRepo.Upsert(ctx, &documentshipmentdraft.DocumentShipmentDraft{
			DocumentID:     doc.ID,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			Status:         documentshipmentdraft.StatusPending,
			DocumentKind:   classification.Kind,
			Confidence:     intelligence.OverallConfidence,
			DraftData:      intelligence.ToMap(),
		}); err != nil {
			return "", err
		}
		a.metrics.Document.RecordShipmentDraft("unavailable", doc.ResourceType, classification.Kind)
		return document.ShipmentDraftStatusPending, nil
	}

	draftStatus, err := a.support.UpsertDraft(ctx, doc, classification.Kind, intelligence, false)
	if err != nil {
		return "", err
	}
	a.metrics.Document.RecordShipmentDraft("unavailable", doc.ResourceType, classification.Kind)
	return draftStatus, nil
}

func (a *Activities) maybeApplyAsyncAIEnqueueFailure(
	ctx context.Context,
	p *documentanalysis.FinalizeIntelligenceParams,
) error {
	if !p.EnqueueAsyncAI {
		return nil
	}
	if err := a.startAIExtractionWorkflow(
		ctx,
		p.Document,
		p.Payload.UserID,
		p.Timestamp,
	); err != nil {
		a.logger.Warn(
			"failed to start async AI extraction workflow",
			zap.String("documentId", p.Document.ID.String()),
			zap.Error(err),
		)
		p.AIDiagnostics.AcceptanceStatus = documentanalysis.AIAcceptanceStatusRejected
		p.AIDiagnostics.RejectionReason = "ai_async_enqueue_failed"
		p.Content.StructuredData = documentanalysis.BuildStructuredData(
			p.Intelligence,
			p.AIDiagnostics,
		)
		if _, upsertErr := a.contentRepo.Upsert(ctx, p.Content); upsertErr != nil {
			return upsertErr
		}
	}
	return nil
}

func (a *Activities) ListDocumentIntelligenceTenantsActivity(
	ctx context.Context,
	payload *documentanalysis.ReconcileDocumentIntelligencePayload,
) (*documentanalysis.ListDocumentIntelligenceTenantsResult, error) {
	olderThan := time.Now().Add(-time.Duration(payload.OlderThanSeconds) * time.Second).Unix()
	tenants, err := a.contentRepo.ListPendingExtractionTenants(
		ctx,
		&documentcontentports.ListPendingDocumentExtractionRequest{
			OlderThan: olderThan,
			Limit:     temporaljobs.DefaultTenantScanLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	return &documentanalysis.ListDocumentIntelligenceTenantsResult{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, a.cfg.GetReconcileBatchSize()),
	}, nil
}

func (a *Activities) markFailed(
	ctx context.Context,
	doc *document.Document,
	content *documentcontent.Content,
	code, message string,
) error {
	now := timeutils.NowUnix()
	content.Status = documentcontent.StatusFailed
	content.FailureCode = code
	content.FailureMessage = message
	content.LastExtractedAt = &now
	if _, err := a.contentRepo.Upsert(ctx, content); err != nil {
		return err
	}
	if _, err := a.draftRepo.Upsert(ctx, &documentshipmentdraft.DocumentShipmentDraft{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		Status:         documentshipmentdraft.StatusFailed,
		DocumentKind:   doc.DetectedKind,
		Confidence:     0,
		DraftData:      map[string]any{},
		FailureCode:    code,
		FailureMessage: message,
	}); err != nil {
		return err
	}
	a.metrics.Document.RecordExtraction("failed", "", code)
	a.metrics.Document.RecordShipmentDraft("failed", doc.ResourceType, doc.DetectedKind)

	doc.ContentStatus = document.ContentStatusFailed
	doc.ContentError = message
	doc.HasExtractedText = false
	doc.DetectedKind = ""
	doc.ShipmentDraftStatus = document.ShipmentDraftStatusFailed
	if err := a.documentRepo.UpdateIntelligence(
		ctx,
		&documentports.UpdateDocumentIntelligenceRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: doc.OrganizationID,
				BuID:  doc.BusinessUnitID,
			},
			ContentStatus:       doc.ContentStatus,
			ContentError:        doc.ContentError,
			DetectedKind:        doc.DetectedKind,
			HasExtractedText:    doc.HasExtractedText,
			ShipmentDraftStatus: doc.ShipmentDraftStatus,
			DocumentTypeID:      doc.DocumentTypeID,
		}); err != nil {
		return err
	}
	a.support.SyncSearchProjection(ctx, doc, "")
	return nil
}

// enrichWithAI asks the model to route the document. A transient failure is
// returned for Temporal to retry the activity, the way the provider's answer
// says; on the last attempt, or for a failure no retry would change, the
// document keeps what the deterministic pass found.
//
//nolint:funlen // multi-step AI enrichment pipeline
func (a *Activities) enrichWithAI(
	ctx context.Context,
	payload *documentanalysis.EnrichmentPayload,
) (*documentanalysis.EnrichmentResult, error) {
	diagnostics := &documentanalysis.AIDiagnostics{
		FallbackAnalysis: payload.Intelligence,
		AcceptanceStatus: documentanalysis.AIAcceptanceStatusNotAttempted,
	}
	if !payload.Document.ProcessingProfile.ExtractsWithAI() {
		diagnostics.RejectionReason = "profile_text_only"
		return &documentanalysis.EnrichmentResult{
			Classification:       payload.Classification,
			DocumentIntelligence: payload.Intelligence,
			AIDiagnostics:        diagnostics,
			EnqueueAsyncAI:       false,
		}, nil
	}
	if !a.cfg.DocumentExtractionEnabled() {
		diagnostics.RejectionReason = "ai_disabled"
		return &documentanalysis.EnrichmentResult{
			Classification:       payload.Classification,
			DocumentIntelligence: payload.Intelligence,
			AIDiagnostics:        diagnostics,
			EnqueueAsyncAI:       false,
		}, nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.Document.OrganizationID,
		BuID:   payload.Document.BusinessUnitID,
		UserID: payload.Payload.UserID,
	}
	pages := make([]documentaiextractionports.AIDocumentPage, 0, len(payload.Extracted.Pages))
	pageLimit := a.cfg.GetMaxInputChars() / max(len(payload.Extracted.Pages), 1)
	for _, page := range payload.Extracted.Pages {
		if strings.TrimSpace(page.Text) == "" && strings.TrimSpace(page.Markdown) == "" {
			continue
		}
		pages = append(
			pages,
			documentaiextractionjobs.ModelPage(
				page.PageNumber,
				page.Text,
				page.Markdown,
				pageLimit,
			),
		)
	}

	var route *documentaiextractionports.AIRouteResult
	err := a.runWithHeartbeat(ctx, "ai-route", func() error {
		var routeErr error
		route, routeErr = a.aiDocumentService.RouteDocument(
			ctx,
			&documentaiextractionports.AIRouteRequest{
				TenantInfo: tenantInfo,
				DocumentID: payload.Document.ID,
				FileName:   payload.Document.OriginalName,
				Text: stringutils.TruncateAndTrim(
					payload.Extracted.Text,
					a.cfg.GetMaxInputChars(),
				),
				Pages: pages,
				Features: &documentaiextractionports.AIDocumentFeatureSet{
					TitleCandidates:  payload.Features.TitleCandidates,
					SectionLabels:    payload.Features.SectionLabels,
					PartyLabels:      payload.Features.PartyLabels,
					ReferenceLabels:  payload.Features.ReferenceLabels,
					MoneySignals:     payload.Features.MoneySignals,
					StopSignals:      payload.Features.StopSignals,
					TermsSignals:     payload.Features.TermsSignals,
					SignatureSignals: payload.Features.SignatureSignals,
				},
				Fingerprint: documentaiextractionjobs.ToAIFingerprintHint(payload.Fingerprint),
			},
		)
		return routeErr
	})
	if err != nil {
		if modelcall.Transient(err) &&
			!modelcall.FinalAttempt(
				ctx,
				documentaiextractionjobs.DefaultRetryPolicy.MaximumAttempts,
			) {
			return nil, modelcall.Classify(err)
		}
		a.logger.Warn(
			"ai route failed",
			zap.String("documentId", payload.Document.ID.String()),
			zap.Error(err),
		)
		if errors.Is(err, context.DeadlineExceeded) {
			diagnostics.RejectionReason = "ai_route_timeout"
		} else {
			diagnostics.RejectionReason = "ai_route_failed"
		}
		return &documentanalysis.EnrichmentResult{
			Classification:       payload.Classification,
			DocumentIntelligence: payload.Intelligence,
			AIDiagnostics:        diagnostics,
			EnqueueAsyncAI:       false,
		}, nil
	}
	if route == nil || strings.TrimSpace(route.DocumentKind) == "" {
		diagnostics.RejectionReason = "ai_route_empty"
		return &documentanalysis.EnrichmentResult{
			Classification:       payload.Classification,
			DocumentIntelligence: payload.Intelligence,
			AIDiagnostics:        diagnostics,
			EnqueueAsyncAI:       false,
		}, nil
	}

	routedClassification := payload.Classification
	routedClassification.Kind = documentaiextractionjobs.NormalizeRoutedKind(route.DocumentKind)
	routedClassification.Confidence = route.Confidence
	routedClassification.Signals = sliceutils.DedupeStrings(
		append(payload.Classification.Signals, route.Signals...),
	)
	routedClassification.ReviewRequired = documentaiextractionjobs.NormalizeAIReviewStatus(
		route.ReviewStatus,
	) != documentanalysis.ReviewStatusReady
	routedClassification.Source = documentaiextractionjobs.NormalizeClassifierSource(
		route.ClassifierSource,
	)
	routedClassification.ProviderFingerprint = stringutils.FirstNonEmpty(
		route.ProviderFingerprint,
		documentanalysis.ProviderName(payload.Fingerprint),
	)
	routedClassification.Reason = stringutils.FirstNonEmpty(
		route.Reason,
		payload.Classification.Reason,
	)
	routedAnalysis := documentanalysis.AnalyzeDocument(routedClassification, payload.Extracted)

	if !strings.EqualFold(routedClassification.Kind, documentanalysis.KindRateConfirmation) ||
		!payload.Control.EnableAIAssistedExtraction ||
		!route.ShouldExtract {
		switch {
		case !strings.EqualFold(routedClassification.Kind, documentanalysis.KindRateConfirmation):
			diagnostics.RejectionReason = "ai_routed_non_rate_confirmation"
		case !payload.Control.EnableAIAssistedExtraction:
			diagnostics.RejectionReason = "ai_assisted_extraction_disabled"
		case !route.ShouldExtract:
			diagnostics.RejectionReason = "ai_route_declined_extraction"
		}
		return &documentanalysis.EnrichmentResult{
			Classification:       routedClassification,
			DocumentIntelligence: routedAnalysis,
			AIDiagnostics:        diagnostics,
			Signature:            route.Signature,
			EnqueueAsyncAI:       false,
		}, nil
	}

	diagnostics.AcceptanceStatus = documentanalysis.AIAcceptanceStatusPending
	diagnostics.RejectionReason = ""
	return &documentanalysis.EnrichmentResult{
		Classification:       routedClassification,
		DocumentIntelligence: routedAnalysis,
		AIDiagnostics:        diagnostics,
		Signature:            route.Signature,
		EnqueueAsyncAI:       true,
	}, nil
}

func (a *Activities) runWithHeartbeat(
	ctx context.Context,
	stage string,
	fn func() error,
) (retErr error) {
	recordHeartbeatIfActivity(ctx, stage)

	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(documentanalysis.HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				recordHeartbeatIfActivity(ctx, stage)
			}
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("panic during %s: %v", stage, r)
		}
	}()

	return fn()
}

func recordHeartbeatIfActivity(ctx context.Context, details ...any) {
	if !activity.IsActivity(ctx) {
		return
	}

	activity.RecordHeartbeat(ctx, details...)
}

//nolint:funlen // sequential fallback lookups for document type
func (a *Activities) associateDocumentType(
	ctx context.Context,
	doc *document.Document,
	tenantInfo pagination.TenantInfo,
	kind string,
	control *documentcontrol.DocumentControl,
) error {
	if !control.EnableAutoDocumentTypeAssociate ||
		doc.DocumentTypeID != nil ||
		!documentaiextractionjobs.CanApplyKindToResource(doc.ResourceType, kind) {
		switch {
		case !control.EnableAutoDocumentTypeAssociate:
			a.metrics.Document.RecordTypeAssociation("disabled", kind)
		case doc.DocumentTypeID != nil:
			a.metrics.Document.RecordTypeAssociation("already_set", kind)
		default:
			a.metrics.Document.RecordTypeAssociation("resource_mismatch", kind)
		}
		return nil
	}

	inferred, ok := documentaiextractionjobs.InferDocumentType(kind)
	if !ok {
		a.metrics.Document.RecordTypeAssociation("no_match", kind)
		return nil
	}

	existing, err := a.documentTypeRepo.GetByCode(
		ctx,
		documenttypeports.GetDocumentTypeByCodeRequest{
			Code:       inferred.Code,
			TenantInfo: tenantInfo,
		},
	)
	if err == nil {
		doc.DocumentTypeID = &existing.ID
		a.metrics.Document.RecordTypeAssociation("associated_existing_code", kind)
		return nil
	}
	if !errortypes.IsNotFoundError(err) {
		a.metrics.Document.RecordTypeAssociation("lookup_failed", kind)
		return err
	}

	existing, err = a.documentTypeRepo.GetByName(
		ctx,
		documenttypeports.GetDocumentTypeByNameRequest{
			Name:       inferred.Name,
			TenantInfo: tenantInfo,
		},
	)
	if err == nil {
		doc.DocumentTypeID = &existing.ID
		a.metrics.Document.RecordTypeAssociation("associated_existing_name", kind)
		return nil
	}
	if !errortypes.IsNotFoundError(err) {
		a.metrics.Document.RecordTypeAssociation("lookup_failed", kind)
		return err
	}

	if !control.EnableAutoCreateDocumentTypes {
		a.metrics.Document.RecordTypeAssociation("create_disabled", kind)
		return nil
	}

	created, createErr := a.documentTypeRepo.Create(ctx, &documenttype.DocumentType{
		ID:                     pulid.MustNew("dt_"),
		OrganizationID:         doc.OrganizationID,
		BusinessUnitID:         doc.BusinessUnitID,
		Code:                   inferred.Code,
		Name:                   inferred.Name,
		Color:                  inferred.Color,
		IsSystem:               true,
		DocumentCategory:       inferred.Category,
		DocumentClassification: inferred.Classification,
	})
	if createErr != nil {
		existing, err = a.documentTypeRepo.GetByCode(
			ctx,
			documenttypeports.GetDocumentTypeByCodeRequest{
				Code:       inferred.Code,
				TenantInfo: tenantInfo,
			},
		)
		if err == nil {
			doc.DocumentTypeID = &existing.ID
			a.metrics.Document.RecordTypeAssociation("associated_after_conflict", kind)
			return nil
		}
		if !errortypes.IsNotFoundError(err) {
			a.metrics.Document.RecordTypeAssociation("create_failed", kind)
			return createErr
		}

		existing, err = a.documentTypeRepo.GetByName(
			ctx,
			documenttypeports.GetDocumentTypeByNameRequest{
				Name:       inferred.Name,
				TenantInfo: tenantInfo,
			},
		)
		if err != nil {
			a.metrics.Document.RecordTypeAssociation("create_failed", kind)
			return createErr
		}
		doc.DocumentTypeID = &existing.ID
		a.metrics.Document.RecordTypeAssociation("associated_after_conflict", kind)
		return nil
	}

	doc.DocumentTypeID = &created.ID
	a.metrics.Document.RecordTypeAssociation("created", kind)
	return nil
}
