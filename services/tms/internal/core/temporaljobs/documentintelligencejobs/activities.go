package documentintelligencejobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gen2brain/go-fitz"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Logger              *zap.Logger
	Config              *config.Config
	Metrics             *metrics.Registry
	DocumentRepo        repositories.DocumentRepository
	DocumentControlRepo repositories.DocumentControlRepository
	DocumentTypeRepo    repositories.DocumentTypeRepository
	ContentRepo         repositories.DocumentContentRepository
	AIExtractionRepo    repositories.DocumentAIExtractionRepository
	DraftRepo           repositories.DocumentShipmentDraftRepository
	AIDocumentService   services.AIDocumentService
	SearchProjection    services.DocumentSearchProjectionService
	Storage             storage.Client
	WorkflowStarter     services.WorkflowStarter
	ParsingRuleRuntime  services.DocumentParsingRuleRuntime
	TemporalClient      client.Client `optional:"true"`
}

type Activities struct {
	logger              *zap.Logger
	cfg                 *config.DocumentIntelligenceConfig
	metrics             *metrics.Registry
	documentRepo        repositories.DocumentRepository
	documentControlRepo repositories.DocumentControlRepository
	documentTypeRepo    repositories.DocumentTypeRepository
	contentRepo         repositories.DocumentContentRepository
	aiExtractionRepo    repositories.DocumentAIExtractionRepository
	draftRepo           repositories.DocumentShipmentDraftRepository
	aiDocumentService   services.AIDocumentService
	searchProjection    services.DocumentSearchProjectionService
	storage             storage.Client
	workflowStarter     services.WorkflowStarter
	parsingRuleRuntime  services.DocumentParsingRuleRuntime
	temporalClient      client.Client
}

//nolint:gocritic // dependency injection param
func NewActivities(p ActivitiesParams) *Activities {
	aiDocumentService := p.AIDocumentService
	if aiDocumentService == nil {
		aiDocumentService = noopAIDocumentService{}
	}

	searchProjection := p.SearchProjection
	if searchProjection == nil {
		searchProjection = noopDocumentSearchProjectionService{}
	}

	workflowStarter := p.WorkflowStarter
	if workflowStarter == nil {
		workflowStarter = noopWorkflowStarter{}
	}

	parsingRuleRuntime := p.ParsingRuleRuntime
	if parsingRuleRuntime == nil {
		parsingRuleRuntime = noopDocumentParsingRuleRuntime{}
	}

	return &Activities{
		logger:              p.Logger.Named("temporal.document-intelligence"),
		cfg:                 p.Config.GetDocumentIntelligenceConfig(),
		metrics:             p.Metrics,
		documentRepo:        p.DocumentRepo,
		documentControlRepo: p.DocumentControlRepo,
		documentTypeRepo:    p.DocumentTypeRepo,
		contentRepo:         p.ContentRepo,
		aiExtractionRepo:    p.AIExtractionRepo,
		draftRepo:           p.DraftRepo,
		aiDocumentService:   aiDocumentService,
		searchProjection:    searchProjection,
		storage:             p.Storage,
		workflowStarter:     workflowStarter,
		parsingRuleRuntime:  parsingRuleRuntime,
		temporalClient:      p.TemporalClient,
	}
}

func (a *Activities) ProcessDocumentIntelligenceActivity(
	ctx context.Context,
	payload *ProcessDocumentIntelligencePayload,
) (*ProcessDocumentIntelligenceResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	doc, err := a.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         payload.DocumentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, temporal.NewNonRetryableApplicationError(
				"Document no longer exists",
				temporaltype.ErrorTypeResourceNotFound.String(),
				err,
			)
		}
		return nil, err
	}
	control, err := a.getDocumentControl(ctx, doc.OrganizationID, doc.BusinessUnitID)
	if err != nil {
		return nil, err
	}
	if !control.EnableDocumentIntelligence {
		a.metrics.Document.RecordExtraction("skipped", "", "control_disabled")
		return &ProcessDocumentIntelligenceResult{
			DocumentID: doc.ID,
			Status:     string(doc.ContentStatus),
			Kind:       doc.DetectedKind,
		}, nil
	}

	content, now, err := a.prepareExtractionContentAndStart(ctx, doc, tenantInfo)
	if err != nil {
		return nil, err
	}

	outcome, err := a.runExtractionPipeline(
		ctx,
		payload,
		doc,
		tenantInfo,
		control,
		content,
	)
	if err != nil {
		return nil, err
	}

	if err = a.persistIndexedContentAndAssociate(
		ctx,
		&PersistIndexedContentPayload{
			Document:       doc,
			TenantInfo:     tenantInfo,
			Control:        control,
			Content:        content,
			Extracted:      outcome.Extracted,
			Classification: outcome.Classification,
			Intelligence:   outcome.Intelligence,
			AIDiagnostics:  outcome.AIDiagnostics,
			Timestamp:      now,
		}); err != nil {
		return nil, err
	}

	if err = a.finalizeProcessDocumentIntelligence(ctx, &FinalizeIntelligenceParams{
		Document:       doc,
		Payload:        payload,
		Content:        content,
		Extracted:      outcome.Extracted,
		Classification: outcome.Classification,
		Intelligence:   outcome.Intelligence,
		AIDiagnostics:  outcome.AIDiagnostics,
		Control:        control,
		TenantInfo:     tenantInfo,
		EnqueueAsyncAI: outcome.EnqueueAsyncAI,
		Timestamp:      now,
	}); err != nil {
		return nil, err
	}

	return &ProcessDocumentIntelligenceResult{
		DocumentID: doc.ID,
		Status:     string(doc.ContentStatus),
		Kind:       outcome.Classification.Kind,
	}, nil
}

func (a *Activities) prepareExtractionContentAndStart(
	ctx context.Context,
	doc *document.Document,
	tenantInfo pagination.TenantInfo,
) (*documentcontent.Content, int64, error) {
	now := timeutils.NowUnix()
	content := &documentcontent.Content{
		DocumentID:      doc.ID,
		OrganizationID:  doc.OrganizationID,
		BusinessUnitID:  doc.BusinessUnitID,
		Status:          documentcontent.StatusExtracting,
		LastExtractedAt: &now,
	}
	if _, err := a.contentRepo.Upsert(ctx, content); err != nil {
		return nil, 0, err
	}

	doc.ContentStatus = document.ContentStatusExtracting
	doc.ContentError = ""
	if err := a.documentRepo.UpdateIntelligence(ctx,
		&repositories.UpdateDocumentIntelligenceRequest{
			ID:                  doc.ID,
			TenantInfo:          tenantInfo,
			ContentStatus:       doc.ContentStatus,
			ContentError:        doc.ContentError,
			DetectedKind:        doc.DetectedKind,
			HasExtractedText:    doc.HasExtractedText,
			ShipmentDraftStatus: doc.ShipmentDraftStatus,
			DocumentTypeID:      doc.DocumentTypeID,
		}); err != nil {
		return nil, 0, err
	}
	a.metrics.Document.RecordExtraction("started", "", "none")
	a.syncSearchProjection(ctx, doc, "")
	return content, now, nil
}

func (a *Activities) finalizeProcessDocumentIntelligence(
	ctx context.Context,
	p *FinalizeIntelligenceParams,
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
	if err = a.documentRepo.UpdateIntelligence(ctx, &repositories.UpdateDocumentIntelligenceRequest{
		ID:                  p.Document.ID,
		TenantInfo:          p.TenantInfo,
		ContentStatus:       p.Document.ContentStatus,
		ContentError:        p.Document.ContentError,
		DetectedKind:        p.Document.DetectedKind,
		HasExtractedText:    p.Document.HasExtractedText,
		ShipmentDraftStatus: p.Document.ShipmentDraftStatus,
		DocumentTypeID:      p.Document.DocumentTypeID,
	}); err != nil {
		return err
	}
	indexedText := p.Extracted.Text
	if !p.Control.EnableFullTextIndexing {
		indexedText = ""
	}
	a.syncSearchProjection(ctx, p.Document, indexedText)

	return a.maybeApplyAsyncAIEnqueueFailure(ctx, p)
}

func (a *Activities) persistIndexedContentAndAssociate(
	ctx context.Context,
	payload *PersistIndexedContentPayload,
) error {
	structured := buildStructuredData(payload.Intelligence, payload.AIDiagnostics)
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
	if err := a.contentRepo.ReplacePages(
		ctx,
		payload.Content,
		buildContentPages(payload.Content, payload.Extracted.Pages),
	); err != nil {
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

func (a *Activities) runExtractionPipeline(
	ctx context.Context,
	payload *ProcessDocumentIntelligencePayload,
	doc *document.Document,
	tenantInfo pagination.TenantInfo,
	control *tenant.DocumentControl,
	content *documentcontent.Content,
) (*ExtractionPipelineOutcome, error) {
	download, err := a.storage.Download(ctx, doc.StoragePath)
	if err != nil {
		return nil, a.markFailed(
			ctx,
			doc,
			content,
			documentupload.FailureDownloadFailed.String(),
			"Failed to download document",
		)
	}
	defer download.Body.Close()

	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(download.Body); err != nil {
		return nil, a.markFailed(
			ctx,
			doc,
			content,
			documentupload.FailureReadFailed.String(),
			"Failed to read document bytes",
		)
	}

	extracted, err := a.extractContent(ctx, doc, buf.Bytes(), control)
	if err != nil {
		return nil, a.markFailed(
			ctx,
			doc,
			content,
			documentupload.FailureExtractionFailed.String(),
			err.Error(),
		)
	}

	features := extractDocumentFeatures(doc.OriginalName, extracted.Pages, extracted.Text)
	fingerprint := detectProviderFingerprint(doc.OriginalName, extracted.Text, features)
	classification := classifyDocumentWithControl(
		doc.OriginalName,
		extracted.Text,
		control,
		features,
		fingerprint,
	)
	intelligence := analyzeDocument(classification, extracted)
	intelligence = a.applyParsingRules(
		ctx,
		tenantInfo,
		doc.OriginalName,
		classification.ProviderFingerprint,
		extracted,
		intelligence,
	)
	result := a.enrichWithAI(
		ctx,
		&EnrichmentPayload{
			Payload:        payload,
			Document:       doc,
			Control:        control,
			Extracted:      extracted,
			Features:       features,
			Fingerprint:    fingerprint,
			Classification: classification,
			Intelligence:   intelligence,
		},
	)

	return &ExtractionPipelineOutcome{
		Extracted:      extracted,
		Classification: classification,
		Intelligence:   intelligence,
		AIDiagnostics:  result.AIDiagnostics,
		EnqueueAsyncAI: result.EnqueueAsyncAI,
	}, nil
}

func (a *Activities) upsertShipmentDraftForProcess(
	ctx context.Context,
	doc *document.Document,
	control *tenant.DocumentControl,
	classification *ClassificationResult,
	intelligence *DocumentIntelligenceAnalysis,
	enqueueAsyncAI bool,
) (document.ShipmentDraftStatus, error) {
	isUsable := canGenerateShipmentDraft(control, doc.ResourceType, classification.Kind) &&
		hasUsableShipmentDraft(intelligence)

	if isUsable {
		draftStatus, err := a.upsertDraft(ctx, doc, classification.Kind, intelligence, true)
		if err != nil {
			return "", err
		}
		a.metrics.Document.RecordShipmentDraft("ready", doc.ResourceType, classification.Kind)
		return draftStatus, nil
	}

	if canGenerateShipmentDraft(control, doc.ResourceType, classification.Kind) && enqueueAsyncAI {
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

	draftStatus, err := a.upsertDraft(ctx, doc, classification.Kind, intelligence, false)
	if err != nil {
		return "", err
	}
	a.metrics.Document.RecordShipmentDraft("unavailable", doc.ResourceType, classification.Kind)
	return draftStatus, nil
}

func (a *Activities) maybeApplyAsyncAIEnqueueFailure(
	ctx context.Context,
	p *FinalizeIntelligenceParams,
) error {
	if !p.EnqueueAsyncAI {
		return nil
	}
	if err := a.startAIExtractionWorkflow(ctx, p.Document, p.Payload.UserID, p.Timestamp); err != nil {
		a.logger.Warn(
			"failed to start async AI extraction workflow",
			zap.String("documentId", p.Document.ID.String()),
			zap.Error(err),
		)
		p.AIDiagnostics.AcceptanceStatus = aiAcceptanceStatusRejected
		p.AIDiagnostics.RejectionReason = "ai_async_enqueue_failed"
		p.Content.StructuredData = buildStructuredData(p.Intelligence, p.AIDiagnostics)
		if _, upsertErr := a.contentRepo.Upsert(ctx, p.Content); upsertErr != nil {
			return upsertErr
		}
	}
	return nil
}

func (a *Activities) ReconcileDocumentIntelligenceActivity(
	ctx context.Context,
	payload *ReconcileDocumentIntelligencePayload,
) (*ReconcileDocumentIntelligenceResult, error) {
	if !a.workflowStarter.Enabled() {
		return &ReconcileDocumentIntelligenceResult{}, nil
	}

	olderThan := time.Now().Add(-time.Duration(payload.OlderThanSeconds) * time.Second).Unix()
	docs, err := a.contentRepo.ListPendingExtraction(ctx, olderThan, a.cfg.GetReconcileBatchSize())
	if err != nil {
		return nil, err
	}

	result := &ReconcileDocumentIntelligenceResult{}
	for _, doc := range docs {
		_, startErr := a.workflowStarter.StartWorkflow(
			ctx,
			client.StartWorkflowOptions{
				ID: fmt.Sprintf(
					"document-intelligence-%s",
					doc.ID.String(),
				),
				TaskQueue:                                temporaltype.DocumentIntelligenceTaskQueue,
				WorkflowExecutionErrorWhenAlreadyStarted: true,
				WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			},
			"ProcessDocumentIntelligenceWorkflow",
			&ProcessDocumentIntelligencePayload{
				BasePayload: temporaltype.BasePayload{
					OrganizationID: doc.OrganizationID,
					BusinessUnitID: doc.BusinessUnitID,
					UserID:         doc.UploadedByID,
				},
				DocumentID: doc.ID,
			},
		)
		if startErr == nil {
			result.Queued++
		}
		a.metrics.Document.RecordReconciliationQueue(startErr == nil)
	}

	return result, nil
}

func (a *Activities) extractContent(
	ctx context.Context,
	doc *document.Document,
	data []byte,
	control *tenant.DocumentControl,
) (*ExtractionResult, error) {
	contentType := strings.ToLower(doc.FileType)
	ext := strings.ToLower(filepath.Ext(doc.OriginalName))

	switch {
	case isPlainTextType(contentType, ext):
		maxChars := a.cfg.GetMaxExtractedChars()
		bounded := data
		if len(bounded) > maxChars {
			bounded = bounded[:maxChars]
		}
		text := stringutils.TruncateAndTrim(string(bounded), maxChars)
		return finalizeExtraction([]*PageExtractionResult{{
			PageNumber: 1,
			SourceKind: documentcontent.SourceKindNative,
			Text:       text,
		}}, a.cfg.GetMaxExtractedChars()), nil
	case isFitzType(contentType, ext):
		return a.extractViaFitz(ctx, data, control.EnableOCR)
	case strings.HasPrefix(contentType, "image/"):
		if !control.EnableOCR {
			return finalizeExtraction([]*PageExtractionResult{{
				PageNumber: 1,
				SourceKind: documentcontent.SourceKindOCR,
			}}, a.cfg.GetMaxExtractedChars()), nil
		}
		page, err := a.runOCRPage(ctx, data, ext, 1)
		if err != nil {
			return nil, err
		}
		return finalizeExtraction([]*PageExtractionResult{page}, a.cfg.GetMaxExtractedChars()), nil
	default:
		return nil, fmt.Errorf("unsupported document type for extraction: %s", doc.FileType)
	}
}

func (a *Activities) extractViaFitz(
	ctx context.Context,
	data []byte,
	enableOCR bool,
) (*ExtractionResult, error) {
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	pageCount := doc.NumPage()
	pages := make([]*PageExtractionResult, 0, pageCount)

	for page := range pageCount {
		recordHeartbeatIfActivity(ctx, page)
		pageText, textErr := doc.Text(page)
		if textErr == nil && strings.TrimSpace(pageText) != "" {
			pages = append(pages, &PageExtractionResult{
				PageNumber: page + 1,
				SourceKind: documentcontent.SourceKindNative,
				Text:       pageText,
			})
			continue
		}

		if !enableOCR || page >= a.cfg.GetMaxOCRPages() {
			pages = append(pages, &PageExtractionResult{
				PageNumber: page + 1,
				SourceKind: documentcontent.SourceKindNative,
				Metadata: map[string]any{
					"extractionMode": "native_skipped",
				},
			})
			continue
		}

		imageBytes, imageErr := doc.ImagePNG(page, 150)
		if imageErr != nil {
			pages = append(pages, &PageExtractionResult{
				PageNumber: page + 1,
				SourceKind: documentcontent.SourceKindOCR,
				Metadata: map[string]any{
					"extractionMode": "ocr_image_failed",
				},
			})
			continue
		}

		ocrPage, ocrErr := a.runOCRPage(ctx, imageBytes, ".png", page+1)
		if ocrErr != nil {
			pages = append(pages, &PageExtractionResult{
				PageNumber: page + 1,
				SourceKind: documentcontent.SourceKindOCR,
				Metadata: map[string]any{
					"extractionMode": "ocr_failed",
					"error":          ocrErr.Error(),
				},
			})
			continue
		}
		pages = append(pages, ocrPage)
	}

	result := finalizeExtraction(pages, a.cfg.GetMaxExtractedChars())
	result.PageCount = pageCount
	return result, nil
}

func (a *Activities) runOCRPage(
	ctx context.Context,
	imageData []byte,
	ext string,
	pageNumber int,
) (*PageExtractionResult, error) {
	page := PageExtractionResult{
		PageNumber: pageNumber,
		SourceKind: documentcontent.SourceKindOCR,
		Metadata:   map[string]any{},
	}

	if width, height, err := readImageDimensions(imageData); err == nil {
		page.Width = width
		page.Height = height
	}

	ocrInput := imageData
	if a.cfg.OCRPreprocessingEnabled() {
		processed, width, height, err := a.preprocessOCRImage(imageData)
		if err == nil {
			ocrInput = processed
			page.Width = width
			page.Height = height
			page.PreprocessingApplied = true
			page.Metadata["preprocessingMode"] = a.cfg.GetOCRPreprocessingMode()
		} else {
			page.Metadata["preprocessingError"] = err.Error()
		}
	}

	text, confidence, err := a.runOCR(ctx, ocrInput, ext)
	if err != nil {
		return &page, err
	}

	page.Text = stringutils.TruncateAndTrim(text, a.cfg.GetMaxExtractedChars())
	page.OCRConfidence = confidence
	if page.Metadata == nil {
		page.Metadata = map[string]any{}
	}
	page.Metadata["ocrLanguage"] = a.cfg.GetOCRLanguage()
	page.Metadata["ocrConfidence"] = confidence

	return &page, nil
}

func (a *Activities) runOCR(
	ctx context.Context,
	imageData []byte,
	ext string,
) (text string, confidence float64, err error) {
	tmpFile, err := os.CreateTemp("", "trenova-ocr-*"+ext)
	if err != nil {
		return "", 0, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.Write(imageData); err != nil {
		tmpFile.Close()
		return "", 0, err
	}
	if err = tmpFile.Close(); err != nil {
		return "", 0, err
	}

	ocrCtx, cancel := context.WithTimeout(ctx, a.cfg.GetOCRTimeout())
	defer cancel()

	// #nosec G204 -- OCR binary path and language come from trusted config; temp path is from os.CreateTemp, not user shell input
	cmd := exec.CommandContext(
		ocrCtx,
		a.cfg.GetOCRCommand(),
		tmpFile.Name(),
		"stdout",
		"-l",
		a.cfg.GetOCRLanguage(),
		"tsv",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ocrCtx.Err() != nil {
			return "", 0, fmt.Errorf(
				"ocr command timed out after %s: %w",
				a.cfg.GetOCRTimeout(),
				ocrCtx.Err(),
			)
		}
		return "", 0, fmt.Errorf(
			"ocr command failed: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}

	text, confidence, err = parseTesseractTSV(string(output))
	if err != nil {
		return "", 0, err
	}

	return text, confidence, nil
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
		&repositories.UpdateDocumentIntelligenceRequest{
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
	a.syncSearchProjection(ctx, doc, "")
	return nil
}

func (a *Activities) syncSearchProjection(
	ctx context.Context,
	doc *document.Document,
	contentText string,
) {
	if err := a.searchProjection.Upsert(ctx, doc, contentText); err != nil {
		a.metrics.Document.RecordSearchProjectionSync(false)
		a.logger.Warn("failed to sync document search projection",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
		return
	}
	a.metrics.Document.RecordSearchProjectionSync(true)
}

//nolint:funlen // multi-step AI enrichment pipeline
func (a *Activities) enrichWithAI(
	ctx context.Context,
	payload *EnrichmentPayload,
) *EnrichmentResult {
	diagnostics := &AIDiagnostics{
		FallbackAnalysis: payload.Intelligence,
		AcceptanceStatus: aiAcceptanceStatusNotAttempted,
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.Document.OrganizationID,
		BuID:   payload.Document.BusinessUnitID,
		UserID: payload.Payload.UserID,
	}
	pages := make([]services.AIDocumentPage, 0, len(payload.Extracted.Pages))
	for _, page := range payload.Extracted.Pages {
		if strings.TrimSpace(page.Text) == "" {
			continue
		}
		pages = append(pages, services.AIDocumentPage{
			PageNumber: page.PageNumber,
			Text: stringutils.TruncateAndTrim(
				page.Text,
				a.cfg.GetAIMaxInputChars()/max(len(payload.Extracted.Pages), 1),
			),
		})
	}

	var route *services.AIRouteResult
	err := a.runWithHeartbeat(ctx, "ai-route", func() error {
		var routeErr error
		route, routeErr = a.aiDocumentService.RouteDocument(ctx, &services.AIRouteRequest{
			TenantInfo: tenantInfo,
			DocumentID: payload.Document.ID,
			FileName:   payload.Document.OriginalName,
			Text: stringutils.TruncateAndTrim(
				payload.Extracted.Text,
				a.cfg.GetAIMaxInputChars(),
			),
			Pages: pages,
			Features: &services.AIDocumentFeatureSet{
				TitleCandidates:  payload.Features.TitleCandidates,
				SectionLabels:    payload.Features.SectionLabels,
				PartyLabels:      payload.Features.PartyLabels,
				ReferenceLabels:  payload.Features.ReferenceLabels,
				MoneySignals:     payload.Features.MoneySignals,
				StopSignals:      payload.Features.StopSignals,
				TermsSignals:     payload.Features.TermsSignals,
				SignatureSignals: payload.Features.SignatureSignals,
			},
			Fingerprint: toAIFingerprintHint(payload.Fingerprint),
		})
		return routeErr
	})
	if err != nil {
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
		return &EnrichmentResult{
			Classification:       payload.Classification,
			DocumentIntelligence: payload.Intelligence,
			AIDiagnostics:        diagnostics,
			EnqueueAsyncAI:       false,
		}
	}
	if route == nil || strings.TrimSpace(route.DocumentKind) == "" {
		diagnostics.RejectionReason = "ai_route_empty"
		return &EnrichmentResult{
			Classification:       payload.Classification,
			DocumentIntelligence: payload.Intelligence,
			AIDiagnostics:        diagnostics,
			EnqueueAsyncAI:       false,
		}
	}

	routedClassification := payload.Classification
	routedClassification.Kind = normalizeRoutedKind(route.DocumentKind)
	routedClassification.Confidence = route.Confidence
	routedClassification.Signals = dedupeStrings(
		append(payload.Classification.Signals, route.Signals...),
	)
	routedClassification.ReviewRequired = normalizeAIReviewStatus(
		route.ReviewStatus,
	) != reviewStatusReady
	routedClassification.Source = normalizeClassifierSource(route.ClassifierSource)
	routedClassification.ProviderFingerprint = stringutils.FirstNonEmpty(
		route.ProviderFingerprint,
		providerName(payload.Fingerprint),
	)
	routedClassification.Reason = stringutils.FirstNonEmpty(
		route.Reason,
		payload.Classification.Reason,
	)
	routedAnalysis := analyzeDocument(routedClassification, payload.Extracted)

	if !strings.EqualFold(routedClassification.Kind, kindRateConfirmation) ||
		!payload.Control.EnableAIAssistedExtraction ||
		!route.ShouldExtract {
		switch {
		case !strings.EqualFold(routedClassification.Kind, kindRateConfirmation):
			diagnostics.RejectionReason = "ai_routed_non_rate_confirmation"
		case !payload.Control.EnableAIAssistedExtraction:
			diagnostics.RejectionReason = "ai_assisted_extraction_disabled"
		case !route.ShouldExtract:
			diagnostics.RejectionReason = "ai_route_declined_extraction"
		}
		return &EnrichmentResult{
			Classification:       routedClassification,
			DocumentIntelligence: routedAnalysis,
			AIDiagnostics:        diagnostics,
			EnqueueAsyncAI:       false,
		}
	}

	diagnostics.AcceptanceStatus = aiAcceptanceStatusPending
	diagnostics.RejectionReason = ""
	return &EnrichmentResult{
		Classification:       routedClassification,
		DocumentIntelligence: routedAnalysis,
		AIDiagnostics:        diagnostics,
		EnqueueAsyncAI:       true,
	}
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
		ticker := time.NewTicker(heartbeatInterval)
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
	control *tenant.DocumentControl,
) error {
	if !control.EnableAutoDocumentTypeAssociate ||
		doc.DocumentTypeID != nil ||
		!canApplyKindToResource(doc.ResourceType, kind) {
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

	inferred, ok := inferDocumentType(kind)
	if !ok {
		a.metrics.Document.RecordTypeAssociation("no_match", kind)
		return nil
	}

	existing, err := a.documentTypeRepo.GetByCode(ctx, repositories.GetDocumentTypeByCodeRequest{
		Code:       inferred.Code,
		TenantInfo: tenantInfo,
	})
	if err == nil {
		doc.DocumentTypeID = &existing.ID
		a.metrics.Document.RecordTypeAssociation("associated_existing_code", kind)
		return nil
	}
	if !errortypes.IsNotFoundError(err) {
		a.metrics.Document.RecordTypeAssociation("lookup_failed", kind)
		return err
	}

	existing, err = a.documentTypeRepo.GetByName(ctx, repositories.GetDocumentTypeByNameRequest{
		Name:       inferred.Name,
		TenantInfo: tenantInfo,
	})
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
		existing, err = a.documentTypeRepo.GetByCode(ctx, repositories.GetDocumentTypeByCodeRequest{
			Code:       inferred.Code,
			TenantInfo: tenantInfo,
		})
		if err == nil {
			doc.DocumentTypeID = &existing.ID
			a.metrics.Document.RecordTypeAssociation("associated_after_conflict", kind)
			return nil
		}
		if !errortypes.IsNotFoundError(err) {
			a.metrics.Document.RecordTypeAssociation("create_failed", kind)
			return createErr
		}

		existing, err = a.documentTypeRepo.GetByName(ctx, repositories.GetDocumentTypeByNameRequest{
			Name:       inferred.Name,
			TenantInfo: tenantInfo,
		})
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

func (a *Activities) getDocumentControl(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*tenant.DocumentControl, error) {
	return a.documentControlRepo.GetOrCreate(ctx, orgID, buID)
}

func (a *Activities) preprocessOCRImage(
	imageData []byte,
) (data []byte, width, height int, err error) {
	img, err := imaging.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, 0, 0, err
	}

	processed := img
	maxDimension := a.cfg.GetOCRMaxImageDimension()
	if maxDimension > 0 {
		bounds := processed.Bounds()
		if bounds.Dx() > maxDimension || bounds.Dy() > maxDimension {
			processed = imaging.Fit(processed, maxDimension, maxDimension, imaging.Lanczos)
		}
	}

	if strings.ToLower(a.cfg.GetOCRPreprocessingMode()) == "standard" {
		processed = imaging.Grayscale(processed)
		processed = imaging.AdjustContrast(processed, ocrPreprocessingContrastBoost)
		processed = imaging.Sharpen(processed, ocrPreprocessingSharpenSigma)
		processed = thresholdImage(processed, ocrPreprocessingBinaryThreshold)
	}

	buf := new(bytes.Buffer)
	if encErr := imaging.Encode(buf, processed, imaging.PNG); encErr != nil {
		return nil, 0, 0, encErr
	}

	bounds := processed.Bounds()
	return buf.Bytes(), bounds.Dx(), bounds.Dy(), nil
}
