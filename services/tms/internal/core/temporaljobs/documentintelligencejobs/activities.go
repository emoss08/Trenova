package documentintelligencejobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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
	"github.com/emoss08/trenova/shared/intutils"
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

const (
	reviewStatusReady    = "Ready"
	stopRoleShipper      = "shipper"
	stopRoleConsignee    = "consignee"
	stopRolePickup       = "pickup"
	stopRoleDelivery     = "delivery"
	kindRateConfirmation = "RateConfirmation"
	kindBillOfLading     = "BillOfLading"
	kindProofOfDelivery  = "ProofOfDelivery"
	kindInvoice          = "Invoice"
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

	if err = a.finalizeProcessDocumentIntelligence(
		ctx,
		doc,
		payload,
		content,
		outcome.Extracted,
		outcome.Classification,
		outcome.Intelligence,
		outcome.AIDiagnostics,
		control,
		tenantInfo,
		outcome.EnqueueAsyncAI,
		now,
	); err != nil {
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
	doc *document.Document,
	payload *ProcessDocumentIntelligencePayload,
	content *documentcontent.Content,
	extracted *ExtractionResult,
	classification *ClassificationResult,
	intelligence *DocumentIntelligenceAnalysis,
	aiDiagnostics *AIDiagnostics,
	control *tenant.DocumentControl,
	tenantInfo pagination.TenantInfo,
	enqueueAsyncAI bool,
	now int64,
) error {
	draftStatus, err := a.upsertShipmentDraftForProcess(
		ctx,
		doc,
		control,
		classification,
		intelligence,
		enqueueAsyncAI,
	)
	if err != nil {
		return err
	}

	doc.ShipmentDraftStatus = draftStatus
	if err = a.documentRepo.UpdateIntelligence(ctx, &repositories.UpdateDocumentIntelligenceRequest{
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
	indexedText := extracted.Text
	if !control.EnableFullTextIndexing {
		indexedText = ""
	}
	a.syncSearchProjection(ctx, doc, indexedText)

	return a.maybeApplyAsyncAIEnqueueFailure(
		ctx,
		doc,
		payload,
		content,
		intelligence,
		aiDiagnostics,
		enqueueAsyncAI,
		now,
	)
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
	draftIsUsable := canGenerateShipmentDraft(control, doc.ResourceType, classification.Kind) &&
		hasUsableShipmentDraft(intelligence)
	if draftIsUsable {
		draft := &documentshipmentdraft.Draft{
			DocumentID:     doc.ID,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			Status:         documentshipmentdraft.StatusReady,
			DocumentKind:   classification.Kind,
			Confidence:     intelligence.OverallConfidence,
			DraftData:      intelligence.ToMap(),
		}
		if _, err := a.draftRepo.Upsert(ctx, draft); err != nil {
			return "", err
		}
		a.metrics.Document.RecordShipmentDraft("ready", doc.ResourceType, classification.Kind)
		return document.ShipmentDraftStatusReady, nil
	}

	draftState := documentshipmentdraft.StatusUnavailable
	draftStatus := document.ShipmentDraftStatusUnavailable
	if canGenerateShipmentDraft(control, doc.ResourceType, classification.Kind) && enqueueAsyncAI {
		draftStatus = document.ShipmentDraftStatusPending
		draftState = documentshipmentdraft.StatusPending
	}
	if _, err := a.draftRepo.Upsert(ctx, &documentshipmentdraft.Draft{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		Status:         draftState,
		DocumentKind:   classification.Kind,
		Confidence:     intelligence.OverallConfidence,
		DraftData:      intelligence.ToMap(),
	}); err != nil {
		return "", err
	}
	a.metrics.Document.RecordShipmentDraft("unavailable", doc.ResourceType, classification.Kind)
	return draftStatus, nil
}

func (a *Activities) maybeApplyAsyncAIEnqueueFailure(
	ctx context.Context,
	doc *document.Document,
	payload *ProcessDocumentIntelligencePayload,
	content *documentcontent.Content,
	intelligence *DocumentIntelligenceAnalysis,
	aiDiagnostics *AIDiagnostics,
	enqueueAsyncAI bool,
	now int64,
) error {
	if !enqueueAsyncAI {
		return nil
	}
	if err := a.startAIExtractionWorkflow(ctx, doc, payload.UserID, now); err != nil {
		a.logger.Warn(
			"failed to start async AI extraction workflow",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
		aiDiagnostics.AcceptanceStatus = aiAcceptanceStatusRejected
		aiDiagnostics.RejectionReason = "ai_async_enqueue_failed"
		content.StructuredData = buildStructuredData(intelligence, aiDiagnostics)
		if _, upsertErr := a.contentRepo.Upsert(ctx, content); upsertErr != nil {
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
		text := stringutils.TruncateAndTrim(string(data), a.cfg.GetMaxExtractedChars())
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
	if _, err := a.draftRepo.Upsert(ctx, &documentshipmentdraft.Draft{
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

func isPlainTextType(contentType, ext string) bool {
	return strings.HasPrefix(contentType, "text/") ||
		ext == ".txt" ||
		ext == ".csv" ||
		ext == ".json" ||
		ext == ".xml" ||
		ext == ".html"
}

func isFitzType(contentType, ext string) bool {
	switch {
	case contentType == "application/pdf":
		return true
	case ext == ".pdf" || ext == ".docx" || ext == ".xlsx" || ext == ".pptx" || ext == ".epub":
		return true
	default:
		return false
	}
}

func classifyDocumentWithControl(
	name, text string,
	control *tenant.DocumentControl,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	if control == nil || !control.EnableAutoClassification {
		return &ClassificationResult{
			Kind:           "Other",
			Confidence:     0,
			Signals:        []string{"auto classification disabled"},
			ReviewRequired: true,
			Source:         "disabled",
			Reason:         "automatic classification disabled by document controls",
		}
	}
	return classifyDocumentWithFeatures(name, text, features, fingerprint)
}

func classifyDocumentWithFeatures(
	name, text string,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	corpus := strings.ToLower(name + "\n" + text)

	candidates := []*ClassificationResult{
		scoreRateConfirmation(corpus, features, fingerprint),
		scoreBillOfLading(corpus, features, fingerprint),
		scoreProofOfDelivery(corpus, features, fingerprint),
	}

	best := &ClassificationResult{
		Kind:           "Other",
		Confidence:     0.3,
		Signals:        []string{"no strong classification signals"},
		ReviewRequired: true,
		Source:         "deterministic",
		Reason:         "no strong document-kind evidence detected",
	}

	for _, candidate := range candidates {
		if candidate.Confidence > best.Confidence {
			best = candidate
		}
	}

	if best.Kind == "Other" || best.Confidence < 0.55 {
		return &ClassificationResult{
			Kind:                "Other",
			Confidence:          clampConfidence(best.Confidence),
			Signals:             best.Signals,
			ReviewRequired:      true,
			Source:              "deterministic",
			ProviderFingerprint: providerName(fingerprint),
			Reason:              best.Reason,
		}
	}

	best.Confidence = clampConfidence(best.Confidence)
	best.ReviewRequired = best.Confidence < 0.8
	if best.Source == "" {
		best.Source = "deterministic"
	}
	if best.ProviderFingerprint == "" {
		best.ProviderFingerprint = providerName(fingerprint)
	}

	return best
}

func scoreRateConfirmation(
	corpus string,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	score := 0.0
	signals := make([]string, 0, 10)

	if containsAny(
		corpus,
		"rate confirmation",
		"load confirmation",
		"carrier load confirmation",
		"contract addendum and carrier load confirmation",
		"ratecon",
	) {
		score += 0.5
		signals = append(signals, "rate confirmation phrase")
	}
	if rateRegex.MatchString(corpus) {
		score += 0.15
		signals = append(signals, "rate amount")
	}
	if pickupRegex.MatchString(corpus) {
		score += 0.1
		signals = append(signals, "pickup details")
	}
	if deliveryRegex.MatchString(corpus) {
		score += 0.1
		signals = append(signals, "delivery details")
	}
	if equipmentRegex.MatchString(corpus) {
		score += 0.075
		signals = append(signals, "equipment type")
	}
	if referenceRegex.MatchString(corpus) {
		score += 0.075
		signals = append(signals, "reference number")
	}
	if containsAny(
		corpus,
		"line haul",
		"flat rate",
		"fuel surcharge",
		"quick pay",
		"cash advance",
	) {
		score += 0.15
		signals = append(signals, "carrier rate terms")
	}
	if containsAny(corpus, "service for load #", "load #", "carrier load number") {
		score += 0.1
		signals = append(signals, "load number")
	}
	if containsAny(
		corpus,
		"load confirmation is subject to the terms",
		"this load confirmation is",
	) {
		score += 0.1
		signals = append(signals, "load confirmation terms")
	}
	if len(features.MoneySignals) > 0 {
		score += 0.1
		signals = append(signals, "money signals")
	}
	if len(features.StopSignals) > 0 {
		score += 0.05
		signals = append(signals, "stop signals")
	}
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, kindRateConfirmation) {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return &ClassificationResult{
		Kind:                kindRateConfirmation,
		Confidence:          score,
		Signals:             dedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: providerName(fingerprint),
		Reason:              "rate and load-confirmation evidence detected",
	}
}

func scoreBillOfLading(
	corpus string,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	score := 0.0
	signals := make([]string, 0, 4)

	if containsAny(corpus, "bill of lading", "straight bill") {
		score += 0.65
		signals = append(signals, "bill of lading phrase")
	}
	if containsAny(corpus, "shipper", "consignee") {
		score += 0.05
		signals = append(signals, "shipper/consignee labels")
	}
	if containsAny(corpus, "bol", "pickup number") {
		score += 0.1
		signals = append(signals, "bol reference")
	}
	if containsAny(corpus, "rate confirmation", "load confirmation", "carrier load confirmation") {
		score -= 0.25
	}
	if len(features.SignatureSignals) > 0 {
		score += 0.08
		signals = append(signals, "signature signals")
	}
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, kindBillOfLading) {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return &ClassificationResult{
		Kind:                kindBillOfLading,
		Confidence:          score,
		Signals:             dedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: providerName(fingerprint),
		Reason:              "bill-of-lading shipping evidence detected",
	}
}

func scoreProofOfDelivery(
	corpus string,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	score := 0.0
	signals := make([]string, 0, 4)

	if containsAny(corpus, "proof of delivery", "delivery receipt", "received in good order") {
		score += 0.7
		signals = append(signals, "proof of delivery phrase")
	}
	if containsAny(corpus, "delivered", "consignee signature", "receiver signature") {
		score += 0.15
		signals = append(signals, "delivery confirmation language")
	}
	if len(features.SignatureSignals) > 0 {
		score += 0.08
		signals = append(signals, "signature signals")
	}
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, kindProofOfDelivery) {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return &ClassificationResult{
		Kind:                kindProofOfDelivery,
		Confidence:          score,
		Signals:             dedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: providerName(fingerprint),
		Reason:              "delivery completion evidence detected",
	}
}

func buildStructuredData(
	intelligence *DocumentIntelligenceAnalysis,
	aiDiagnostics *AIDiagnostics,
) map[string]any {
	data := map[string]any{
		"schemaVersion": 6,
		"intelligence":  intelligence.ToMap(),
		"aiDiagnostics": aiDiagnostics.ToMap(),
	}
	return data
}

func extractDocumentFeatures(
	name string,
	pages []*PageExtractionResult,
	text string,
) *DocumentFeatureSet {
	corpus := strings.ToLower(name + "\n" + text)
	lines := splitNormalizedLines(text)
	features := &DocumentFeatureSet{
		TitleCandidates:  make([]string, 0, 4),
		SectionLabels:    make([]string, 0, 12),
		PartyLabels:      make([]string, 0, 8),
		ReferenceLabels:  make([]string, 0, 8),
		MoneySignals:     make([]string, 0, 8),
		StopSignals:      make([]string, 0, 8),
		TermsSignals:     make([]string, 0, 8),
		SignatureSignals: make([]string, 0, 6),
	}

	for _, page := range pages {
		pageLines := splitNormalizedLines(page.Text)
		for i, line := range pageLines {
			if i < 3 && looksLikeTitle(line) {
				features.TitleCandidates = append(features.TitleCandidates, line)
			}
			recordLineFeatures(features, line)
		}
	}

	if len(features.TitleCandidates) == 0 {
		for i, line := range lines {
			if i >= 6 {
				break
			}
			if looksLikeTitle(line) {
				features.TitleCandidates = append(features.TitleCandidates, line)
			}
		}
	}

	if containsAny(corpus, "line haul", "flat rate", "fuel surcharge", "amount due", "total due") {
		features.MoneySignals = append(features.MoneySignals, "billing terms")
	}
	if containsAny(corpus, "pickup", "delivery", "shipper", "receiver", "consignee") {
		features.StopSignals = append(features.StopSignals, "stop/party labels")
	}
	if containsAny(corpus, "signature", "received in good order", "proof of delivery") {
		features.SignatureSignals = append(features.SignatureSignals, "signature language")
	}
	if containsAny(corpus, "load confirmation", "subject to the terms", "contract addendum") {
		features.TermsSignals = append(features.TermsSignals, "carrier contract terms")
	}

	features.TitleCandidates = dedupeStrings(features.TitleCandidates)
	features.SectionLabels = dedupeStrings(features.SectionLabels)
	features.PartyLabels = dedupeStrings(features.PartyLabels)
	features.ReferenceLabels = dedupeStrings(features.ReferenceLabels)
	features.MoneySignals = dedupeStrings(features.MoneySignals)
	features.StopSignals = dedupeStrings(features.StopSignals)
	features.TermsSignals = dedupeStrings(features.TermsSignals)
	features.SignatureSignals = dedupeStrings(features.SignatureSignals)

	return features
}

func recordLineFeatures(features *DocumentFeatureSet, line string) {
	normalized := strings.ToLower(strings.TrimSpace(line))
	if normalized == "" {
		return
	}
	if strings.Contains(normalized, ":") || strings.HasSuffix(normalized, "#") {
		if looksLikeSectionLabel(normalized) {
			features.SectionLabels = append(features.SectionLabels, normalized)
		}
	}
	switch {
	case containsAny(normalized, "shipper", "consignee", "receiver", "bill to"):
		features.PartyLabels = append(features.PartyLabels, normalized)
	case containsAny(normalized, "load #", "ref #", "reference", "confirmation", "invoice #", "bol"):
		features.ReferenceLabels = append(features.ReferenceLabels, normalized)
	case containsAny(normalized, "rate", "line haul", "fuel surcharge", "amount due", "total"):
		features.MoneySignals = append(features.MoneySignals, normalized)
	case containsAny(normalized, "pickup", "delivery", "scheduled delivery", "pick up date", "delivery date"):
		features.StopSignals = append(features.StopSignals, normalized)
	case containsAny(normalized, "signature", "received", "proof of delivery"):
		features.SignatureSignals = append(features.SignatureSignals, normalized)
	case containsAny(normalized, "load confirmation", "subject to the terms", "agreement", "contract addendum"):
		features.TermsSignals = append(features.TermsSignals, normalized)
	}
}

func detectProviderFingerprint(
	name, text string,
	features *DocumentFeatureSet,
) *ProviderFingerprint {
	corpus := strings.ToLower(name + "\n" + text)
	registry := []ProviderFingerprint{
		{
			Provider:   "CHRobinson",
			KindHint:   kindRateConfirmation,
			Confidence: 0.95,
			Signals:    []string{"ch robinson fingerprint", "carrier load confirmation format"},
		},
		{
			Provider:   "TQL",
			KindHint:   kindRateConfirmation,
			Confidence: 0.9,
			Signals:    []string{"tql fingerprint"},
		},
		{
			Provider:   "Echo",
			KindHint:   kindRateConfirmation,
			Confidence: 0.9,
			Signals:    []string{"echo fingerprint"},
		},
		{
			Provider:   "UberFreight",
			KindHint:   kindRateConfirmation,
			Confidence: 0.9,
			Signals:    []string{"uber freight fingerprint"},
		},
	}

	for _, candidate := range registry {
		switch candidate.Provider {
		case "CHRobinson":
			if containsAny(
				corpus,
				"c.h. robinson",
				"ch robinson",
				"navispherecarrier",
				"carrier load confirmation",
				"contract addendum and carrier load confirmation",
			) {
				return &candidate
			}
		case "TQL":
			if containsAny(corpus, "tql", "total quality logistics") {
				return &candidate
			}
		case "Echo":
			if containsAny(corpus, "echo global logistics", "echo logistics") {
				return &candidate
			}
		case "UberFreight":
			if containsAny(corpus, "uber freight") {
				return &candidate
			}
		}
	}

	if len(features.TermsSignals) > 0 && containsAny(corpus, "load confirmation", "carrier load") {
		return &ProviderFingerprint{
			Provider:   "GenericBrokerLoadConfirmation",
			KindHint:   kindRateConfirmation,
			Confidence: 0.7,
			Signals:    []string{"generic broker load confirmation fingerprint"},
		}
	}

	return nil
}

func providerName(fingerprint *ProviderFingerprint) string {
	if fingerprint == nil {
		return ""
	}
	return fingerprint.Provider
}

func looksLikeTitle(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || len(trimmed) > 120 {
		return false
	}
	lower := strings.ToLower(trimmed)
	return containsAny(lower,
		"rate confirmation",
		"load confirmation",
		"bill of lading",
		"proof of delivery",
		"invoice",
		"carrier load confirmation",
	)
}

func looksLikeSectionLabel(line string) bool {
	return containsAny(line,
		"shipper",
		"receiver",
		"consignee",
		"pickup",
		"delivery",
		"rate",
		"reference",
		"commodity",
		"instructions",
		"invoice",
	)
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
	routedClassification.ProviderFingerprint = firstNonEmpty(
		route.ProviderFingerprint,
		providerName(payload.Fingerprint),
	)
	routedClassification.Reason = firstNonEmpty(route.Reason, payload.Classification.Reason)
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

func (a *Activities) runWithHeartbeat(ctx context.Context, stage string, fn func() error) error {
	recordHeartbeatIfActivity(ctx, stage)

	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
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

	return fn()
}

func recordHeartbeatIfActivity(ctx context.Context, details ...any) {
	if !activity.IsActivity(ctx) {
		return
	}

	activity.RecordHeartbeat(ctx, details...)
}

func mergeAIAnalysis(
	fallback *DocumentIntelligenceAnalysis,
	aiExtract *services.AIExtractResult,
) (merged *DocumentIntelligenceAnalysis, accepted bool, rejectionReason string) {
	normalized := normalizeAIExtractResult(aiExtract)
	rejectionReason = validateAIExtract(normalized)
	if rejectionReason != "" {
		return fallback, false, rejectionReason
	}

	merged = analysisFromAIExtract(normalized)
	merged.ClassifierSource = fallback.ClassifierSource
	merged.ProviderFingerprint = fallback.ProviderFingerprint
	merged.ClassificationReason = fallback.ClassificationReason
	merged.ParsingRuleMetadata = fallback.ParsingRuleMetadata
	merged.RawExcerpt = fallback.RawExcerpt
	return merged, true, ""
}

func analysisFromAIExtract(aiExtract *services.AIExtractResult) *DocumentIntelligenceAnalysis {
	aiExtract = normalizeAIExtractResult(aiExtract)
	if aiExtract == nil {
		return &DocumentIntelligenceAnalysis{
			Kind:          kindRateConfirmation,
			MissingFields: []string{},
			Signals:       []string{},
			Fields:        map[string]*ReviewField{},
			Stops:         []*IntelligenceStop{},
			Conflicts:     []*ReviewConflict{},
		}
	}

	analysis := &DocumentIntelligenceAnalysis{
		Kind:              kindRateConfirmation,
		OverallConfidence: clampConfidence(aiExtract.OverallConfidence),
		ReviewStatus:      normalizeAIReviewStatus(aiExtract.ReviewStatus),
		ClassifierSource:  "ai",
		MissingFields:     dedupeStrings(aiExtract.MissingFields),
		Signals:           dedupeStrings(aiExtract.Signals),
		Fields:            make(map[string]*ReviewField, len(aiExtract.Fields)),
		Stops:             make([]*IntelligenceStop, 0, len(aiExtract.Stops)),
		Conflicts:         make([]*ReviewConflict, 0, len(aiExtract.Conflicts)),
	}
	for key, field := range aiExtract.Fields {
		analysis.Fields[key] = &ReviewField{
			Label:           field.Label,
			Value:           field.Value,
			Confidence:      clampConfidence(field.Confidence),
			Excerpt:         field.EvidenceExcerpt,
			EvidenceExcerpt: field.EvidenceExcerpt,
			PageNumber:      field.PageNumber,
			ReviewRequired:  field.ReviewRequired,
			Conflict:        field.Conflict,
			Source:          normalizeAISource(field.Source),
		}
	}

	for _, stop := range aiExtract.Stops {
		analysis.Stops = append(analysis.Stops, &IntelligenceStop{
			Sequence:            stop.Sequence,
			Role:                stop.Role,
			Name:                stop.Name,
			AddressLine1:        stop.AddressLine1,
			AddressLine2:        stop.AddressLine2,
			City:                stop.City,
			State:               stop.State,
			PostalCode:          stop.PostalCode,
			Date:                stop.Date,
			TimeWindow:          stop.TimeWindow,
			AppointmentRequired: stop.AppointmentRequired,
			PageNumber:          stop.PageNumber,
			EvidenceExcerpt:     stop.EvidenceExcerpt,
			Confidence:          clampConfidence(stop.Confidence),
			ReviewRequired:      stop.ReviewRequired,
			Source:              normalizeAISource(stop.Source),
		})
	}

	for _, conflict := range aiExtract.Conflicts {
		analysis.Conflicts = append(analysis.Conflicts, &ReviewConflict{
			Key:             conflict.Key,
			Label:           conflict.Label,
			Values:          conflict.Values,
			PageNumbers:     conflict.PageNumbers,
			EvidenceExcerpt: conflict.EvidenceExcerpt,
			Source:          normalizeAISource(conflict.Source),
		})
	}

	return analysis
}

func normalizeAIExtractResult(result *services.AIExtractResult) *services.AIExtractResult {
	if result == nil {
		return nil
	}

	normalized := &services.AIExtractResult{
		DocumentKind:      normalizeRoutedKind(result.DocumentKind),
		OverallConfidence: result.OverallConfidence,
		ReviewStatus:      normalizeAIReviewStatus(result.ReviewStatus),
		MissingFields:     append([]string{}, result.MissingFields...),
		Signals:           append([]string{}, result.Signals...),
		Fields:            make(map[string]services.AIDocumentField, len(result.Fields)+3),
		Stops:             make([]*services.AIDocumentStop, 0, len(result.Stops)),
		Conflicts:         append([]*services.AIDocumentConflict{}, result.Conflicts...),
	}

	for key, field := range result.Fields {
		canonicalKey := normalizeAIFieldKey(key)
		if canonicalKey == "" {
			canonicalKey = normalizeAIFieldKey(field.Label)
		}
		if canonicalKey == "" {
			continue
		}

		field.Label = strings.TrimSpace(field.Label)
		field.Value = strings.TrimSpace(field.Value)
		field.Source = normalizeAISource(field.Source)
		if existing, ok := normalized.Fields[canonicalKey]; !ok ||
			field.PageNumber > 0 && existing.PageNumber <= 0 {
			normalized.Fields[canonicalKey] = field
		}
	}

	for _, stop := range result.Stops {
		stop.Role = normalizeAIStopRole(stop.Role)
		stop.Name = strings.TrimSpace(stop.Name)
		stop.AddressLine1 = strings.TrimSpace(stop.AddressLine1)
		stop.AddressLine2 = strings.TrimSpace(stop.AddressLine2)
		stop.City = strings.TrimSpace(stop.City)
		stop.State = strings.TrimSpace(stop.State)
		stop.PostalCode = strings.TrimSpace(stop.PostalCode)
		stop.Date = strings.TrimSpace(stop.Date)
		stop.TimeWindow = strings.TrimSpace(stop.TimeWindow)
		stop.Source = normalizeAISource(stop.Source)
		normalized.Stops = append(normalized.Stops, stop)
	}

	ensureCanonicalAIField(
		normalized,
		"rate",
		[]string{
			"rate",
			"totalrate",
			"linehaul",
			"linehaulrate",
			"freightcharge",
			"total",
			"amountdue",
		},
	)
	ensureCanonicalAIFieldFromStop(normalized, stopRoleShipper, "Shipper", stopRolePickup)
	ensureCanonicalAIFieldFromStop(normalized, stopRoleConsignee, "Consignee", stopRoleDelivery)

	return normalized
}

func normalizeAIFieldKey(value string) string {
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "", "/", "")
	normalized := replacer.Replace(strings.TrimSpace(strings.ToLower(value)))
	switch normalized {
	case "rate", "totalrate", "linehaul", "linehaulrate", "freightcharge", "total", "amountdue":
		return "rate"
	case "shipper", "shippername", "shipfrom", "originname":
		return stopRoleShipper
	case "consignee", "receiver", "receivername", "deliveryto", "shipto", "destinationname":
		return stopRoleConsignee
	default:
		return normalized
	}
}

func normalizeAIStopRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "pickup", "origin", "shipper":
		return stopRolePickup
	case "delivery", "destination", "receiver", "consignee", "drop":
		return stopRoleDelivery
	default:
		return strings.TrimSpace(strings.ToLower(role))
	}
}

func ensureCanonicalAIField(result *services.AIExtractResult, key string, aliases []string) {
	if result == nil {
		return
	}
	if field, ok := result.Fields[key]; ok && strings.TrimSpace(field.Value) != "" {
		if field.PageNumber > 0 {
			return
		}
	}

	for _, alias := range aliases {
		field, ok := result.Fields[alias]
		if !ok || strings.TrimSpace(field.Value) == "" {
			continue
		}
		field.Source = normalizeAISource(field.Source)
		if strings.TrimSpace(field.Label) == "" {
			field.Label = key
		}
		result.Fields[key] = field
		return
	}
}

func ensureCanonicalAIFieldFromStop(result *services.AIExtractResult, key, label, role string) {
	if result == nil {
		return
	}
	if field, ok := result.Fields[key]; ok && strings.TrimSpace(field.Value) != "" &&
		field.PageNumber > 0 {
		return
	}

	for _, stop := range result.Stops {
		if stop.Role != role || strings.TrimSpace(stop.Name) == "" || stop.PageNumber <= 0 {
			continue
		}
		result.Fields[key] = services.AIDocumentField{
			Label:           label,
			Value:           stop.Name,
			Confidence:      clampConfidence(stop.Confidence),
			EvidenceExcerpt: stop.EvidenceExcerpt,
			PageNumber:      stop.PageNumber,
			ReviewRequired:  stop.ReviewRequired,
			Conflict:        false,
			Source:          normalizeAISource(stop.Source),
		}
		return
	}
}

func validateAIExtract(result *services.AIExtractResult) string {
	if result == nil || !strings.EqualFold(result.DocumentKind, kindRateConfirmation) {
		return "ai_candidate_invalid_document_kind"
	}
	requiredFields := []string{stopRoleShipper, stopRoleConsignee, "rate"}
	for _, key := range requiredFields {
		field, ok := result.Fields[key]
		if !ok || strings.TrimSpace(field.Value) == "" || field.PageNumber <= 0 {
			return "ai_candidate_missing_required_field_" + key
		}
	}

	hasPickup := false
	hasDelivery := false
	for _, stop := range result.Stops {
		if stop.PageNumber <= 0 || strings.TrimSpace(stop.EvidenceExcerpt) == "" {
			return "ai_candidate_invalid_stop_metadata"
		}
		switch strings.ToLower(strings.TrimSpace(stop.Role)) {
		case stopRolePickup:
			hasPickup = true
		case stopRoleDelivery:
			hasDelivery = true
		}
	}
	if !hasPickup {
		return "ai_candidate_missing_pickup_stop"
	}
	if !hasDelivery {
		return "ai_candidate_missing_delivery_stop"
	}
	return ""
}

func normalizeAISource(source string) string {
	if strings.TrimSpace(source) == "" {
		return "ai"
	}
	return source
}

func normalizeClassifierSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "ai", "template", "hybrid", "deterministic":
		return strings.TrimSpace(strings.ToLower(source))
	default:
		return "ai"
	}
}

func normalizeAIReviewStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "ready":
		return reviewStatusReady
	case "unavailable":
		return "Unavailable"
	default:
		return "NeedsReview"
	}
}

func normalizeRoutedKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "rateconfirmation", "rate_confirmation":
		return kindRateConfirmation
	case "billoflading", "bill_of_lading":
		return kindBillOfLading
	case "proofofdelivery", "proof_of_delivery":
		return kindProofOfDelivery
	default:
		return "Other"
	}
}

func toAIFingerprintHint(fingerprint *ProviderFingerprint) *services.AIDocumentFingerprintHint {
	if fingerprint == nil {
		return nil
	}
	return &services.AIDocumentFingerprintHint{
		Provider:   fingerprint.Provider,
		KindHint:   fingerprint.KindHint,
		Confidence: fingerprint.Confidence,
		Signals:    append([]string{}, fingerprint.Signals...),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func inferDocumentType(kind string) (*InferredDocumentType, bool) {
	switch kind {
	case kindRateConfirmation:
		return &InferredDocumentType{
			Code:           "RATECONF",
			Name:           "Rate Confirmation",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#0f766e",
		}, true
	case kindBillOfLading:
		return &InferredDocumentType{
			Code:           "BOL",
			Name:           "Bill of Lading",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#f59e0b",
		}, true
	case kindProofOfDelivery:
		return &InferredDocumentType{
			Code:           "POD",
			Name:           "Proof of Delivery",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#8b5cf6",
		}, true
	case kindInvoice:
		return &InferredDocumentType{
			Code:           "INVOICE",
			Name:           "Invoice",
			Category:       documenttype.CategoryInvoice,
			Classification: documenttype.ClassificationPublic,
			Color:          "#3b82f6",
		}, true
	default:
		return nil, false
	}
}

func canApplyKindToResource(resourceType, kind string) bool {
	switch kind {
	case kindRateConfirmation, kindBillOfLading, kindProofOfDelivery, kindInvoice:
		return strings.EqualFold(resourceType, "shipment")
	default:
		return true
	}
}

func canGenerateShipmentDraft(
	control *tenant.DocumentControl,
	resourceType, kind string,
) bool {
	return control != nil &&
		control.EnableShipmentDraftExtraction &&
		control.AllowsShipmentDraftResource(resourceType) &&
		kind == kindRateConfirmation
}

func hasUsableShipmentDraft(intelligence *DocumentIntelligenceAnalysis) bool {
	if strings.EqualFold(strings.TrimSpace(intelligence.ReviewStatus), reviewStatusReady) {
		return true
	}

	if hasMeaningfulStopForRole(intelligence.Stops, stopRolePickup) &&
		hasMeaningfulStopForRole(intelligence.Stops, stopRoleDelivery) {
		return true
	}

	return hasMeaningfulField(intelligence.Fields, stopRoleShipper) &&
		hasMeaningfulField(intelligence.Fields, stopRoleConsignee) &&
		hasMeaningfulField(intelligence.Fields, "rate")
}

func hasMeaningfulStopForRole(stops []*IntelligenceStop, role string) bool {
	for _, i := range stops {
		if !strings.EqualFold(strings.TrimSpace(i.Role), role) {
			continue
		}
		if hasReviewableStopData(i) {
			return true
		}
	}
	return false
}

func hasReviewableStopData(stop *IntelligenceStop) bool {
	return strings.TrimSpace(stop.Name) != "" ||
		strings.TrimSpace(stop.AddressLine1) != "" ||
		strings.TrimSpace(stop.AddressLine2) != "" ||
		strings.TrimSpace(stop.City) != "" ||
		strings.TrimSpace(stop.State) != "" ||
		strings.TrimSpace(stop.PostalCode) != "" ||
		strings.TrimSpace(stop.Date) != "" ||
		strings.TrimSpace(stop.TimeWindow) != ""
}

func hasMeaningfulField(fields map[string]*ReviewField, key string) bool {
	field, ok := fields[key]
	if !ok {
		return false
	}
	return strings.TrimSpace(field.Value) != ""
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

func analyzeDocument(
	classification *ClassificationResult,
	extracted *ExtractionResult,
) *DocumentIntelligenceAnalysis {
	if extracted == nil {
		extracted = &ExtractionResult{}
	}

	text := extracted.Text
	analysis := &DocumentIntelligenceAnalysis{
		Kind:                 classification.Kind,
		ReviewStatus:         "NeedsReview",
		MissingFields:        []string{},
		Signals:              append([]string{}, classification.Signals...),
		ClassifierSource:     classification.Source,
		ProviderFingerprint:  classification.ProviderFingerprint,
		ClassificationReason: classification.Reason,
		Conflicts:            []*ReviewConflict{},
		Fields:               make(map[string]*ReviewField),
		Stops:                []*IntelligenceStop{},
		RawExcerpt:           stringutils.TruncateAndTrim(strings.ReplaceAll(text, "\r", ""), 2000),
	}

	required := requiredFieldsForKind(classification.Kind)

	switch classification.Kind {
	case kindRateConfirmation:
		analyzeRateConfirmation(analysis, extracted)
	case kindBillOfLading:
		analyzeBillOfLading(analysis, extracted)
	case kindProofOfDelivery:
		analyzeProofOfDelivery(analysis, extracted)
	case kindInvoice:
		analyzeInvoice(analysis, extracted)
	}

	finalizeAnalysis(analysis, classification, required)
	return analysis
}

func finalizeAnalysis(
	analysis *DocumentIntelligenceAnalysis,
	classification *ClassificationResult,
	required []struct {
		key   string
		label string
	},
) {
	totalConfidence := classification.Confidence
	fieldCount := 1.0
	for _, field := range analysis.Fields {
		totalConfidence += field.Confidence
		fieldCount++
	}
	analysis.OverallConfidence = clampConfidence(totalConfidence / fieldCount)

	for _, field := range required {
		if _, ok := analysis.Fields[field.key]; !ok {
			analysis.MissingFields = append(analysis.MissingFields, field.label)
		}
	}
	if classification.Kind == kindRateConfirmation {
		if !hasStopRole(analysis.Stops, stopRolePickup) {
			analysis.MissingFields = appendIfMissing(analysis.MissingFields, "Pickup Stop")
		}
		if !hasStopRole(analysis.Stops, stopRoleDelivery) {
			analysis.MissingFields = appendIfMissing(analysis.MissingFields, "Delivery Stop")
		}
	}

	if len(analysis.Fields) == 0 {
		analysis.ReviewStatus = "Unavailable"
		return
	}

	analysis.ReviewStatus = resolveReviewStatus(analysis, classification.Kind)
}

func resolveReviewStatus(
	analysis *DocumentIntelligenceAnalysis,
	kind string,
) string {
	noConflicts := len(analysis.Conflicts) == 0
	noMissing := len(analysis.MissingFields) == 0
	highConfidence := analysis.OverallConfidence >= 0.82
	baseReady := highConfidence && noMissing && noConflicts

	switch kind {
	case kindRateConfirmation:
		if baseReady &&
			hasStopRole(analysis.Stops, stopRolePickup) &&
			hasStopRole(analysis.Stops, stopRoleDelivery) &&
			!hasReviewRequiredStop(analysis.Stops) {
			return reviewStatusReady
		}
	case kindBillOfLading:
		if baseReady {
			return reviewStatusReady
		}
	case kindProofOfDelivery:
		if baseReady &&
			!analysis.Fields["deliveryWindow"].ReviewRequired &&
			!analysis.Fields["signature"].ReviewRequired {
			return reviewStatusReady
		}
	default:
		if highConfidence && len(analysis.MissingFields) <= 1 && noConflicts {
			return reviewStatusReady
		}
	}

	return "NeedsReview"
}

func analyzeRateConfirmation(analysis *DocumentIntelligenceAnalysis, extracted *ExtractionResult) {
	analysis.Stops = extractRateConfirmationStops(extracted.Pages)
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "shipper", Label: "Shipper", Regex: shipperRegex,
		Pages: extracted.Pages, Confidence: 0.9,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "consignee", Label: "Consignee", Regex: consigneeRegex,
		Pages: extracted.Pages, Confidence: 0.9,
	})
	addStopTimingField(&AddStopTimingFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "pickupWindow", Label: "Pickup Window",
		Stop: firstStopByRole(analysis.Stops, stopRolePickup), Confidence: 0.88,
	})
	addStopTimingField(&AddStopTimingFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "deliveryWindow", Label: "Delivery Window",
		Stop: firstStopByRole(analysis.Stops, stopRoleDelivery), Confidence: 0.88,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "referenceNumber", Label: "Reference Number", Regex: referenceRegex,
		Pages: extracted.Pages, Confidence: 0.8,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "commodity", Label: "Commodity", Regex: commodityRegex,
		Pages: extracted.Pages, Confidence: 0.78,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "instructions", Label: "Instructions", Regex: instructionsRegex,
		Pages: extracted.Pages, Confidence: 0.72,
	})
	addCurrencyFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "rate", Label: "Rate", Regex: rateRegex,
		Pages: extracted.Pages, Confidence: 0.92, Signal: "rate amount",
	})
	addRegexValueFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "equipmentType", Label: "Equipment Type", Regex: equipmentRegex,
		Pages: extracted.Pages, Confidence: 0.82,
		Signal: "equipment type", ReviewRequired: false,
	})
	addWeightFieldFromPages(&AddWeightFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Text: "weight", Signal: "weight",
	}, extracted.Pages)
	analysis.Conflicts = append(analysis.Conflicts, collectFieldConflicts(analysis.Fields)...)
	analysis.Conflicts = append(analysis.Conflicts, collectStopConflicts(analysis.Stops)...)
}

func analyzeBillOfLading(analysis *DocumentIntelligenceAnalysis, extracted *ExtractionResult) {
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "shipper", Label: "Shipper", Pages: extracted.Pages,
		Labels:     []string{"ship from", "shipper", "shipper name", "shipper information"},
		Confidence: 0.93, Signal: "shipper",
		ReviewRequired: false, Extractor: extractEntityNameFromSection,
	})
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "consignee", Label: "Consignee", Pages: extracted.Pages,
		Labels:     []string{"ship to", "consignee", "receiver", "delivery to"},
		Confidence: 0.93, Signal: "consignee",
		ReviewRequired: false, Extractor: extractEntityNameFromSection,
	})
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "commodity", Label: "Commodity", Pages: extracted.Pages,
		Labels:     []string{"commodity", "description", "product", "articles"},
		Confidence: 0.86, Signal: "commodity",
		ReviewRequired: false, Extractor: extractCommodityFromSection,
	})
	addRegexValueFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "referenceNumber", Label: "BOL / Reference Number",
		Regex: bolReferenceRegex, Pages: extracted.Pages,
		Confidence: 0.85, Signal: "reference number", ReviewRequired: false,
	})
	addRegexValueFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "pieceCount", Label: "Pieces / Packages",
		Regex: pieceCountRegex, Pages: extracted.Pages,
		Confidence: 0.76, Signal: "piece count", ReviewRequired: true,
	})
	addWeightFieldFromPages(&AddWeightFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Text: "weight", Signal: "weight",
	}, extracted.Pages)
	analysis.Conflicts = append(analysis.Conflicts, collectFieldConflicts(analysis.Fields)...)
}

func analyzeProofOfDelivery(analysis *DocumentIntelligenceAnalysis, extracted *ExtractionResult) {
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "consignee", Label: "Consignee", Pages: extracted.Pages,
		Labels: []string{
			"consignee", "receiver name", "delivery to", "delivered to", "received by",
		},
		Confidence: 0.91, Signal: "consignee",
		ReviewRequired: false, Extractor: extractEntityNameFromSection,
	})
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "deliveryWindow", Label: "Delivery", Pages: extracted.Pages,
		Labels: []string{
			"delivery date", "delivered on", "received on", "date delivered",
		},
		Confidence: 0.89, Signal: "delivery",
		ReviewRequired: false, Extractor: extractDeliveryFieldFromSection,
	})
	if _, ok := analysis.Fields["deliveryWindow"]; !ok {
		addFieldFromPages(&RegexValueFieldParams{
			Fields: analysis.Fields, Signals: &analysis.Signals,
			Key: "deliveryWindow", Label: "Delivery", Regex: deliveryRegex,
			Pages: extracted.Pages, Confidence: 0.86,
		})
	}
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "signature", Label: "Signature", Pages: extracted.Pages,
		Labels: []string{
			"receiver signature", "consignee signature", "signature",
			"received by", "signed by",
		},
		Confidence: 0.82, Signal: "signature",
		ReviewRequired: false, Extractor: extractSignatureFromSection,
	})
	addRegexValueFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "referenceNumber", Label: "Reference Number",
		Regex: podReferenceRegex, Pages: extracted.Pages,
		Confidence: 0.82, Signal: "reference number", ReviewRequired: false,
	})
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "receiptNotes", Label: "Receipt Notes", Pages: extracted.Pages,
		Labels: []string{
			"remarks", "exceptions", "received in good order", "delivery status",
		},
		Confidence: 0.72, Signal: "receipt notes",
		ReviewRequired: true, Extractor: extractFreeformSectionValue,
	})
	analysis.Conflicts = append(analysis.Conflicts, collectFieldConflicts(analysis.Fields)...)
}

func analyzeInvoice(analysis *DocumentIntelligenceAnalysis, extracted *ExtractionResult) {
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "referenceNumber", Label: "Invoice Number", Regex: referenceRegex,
		Pages: extracted.Pages, Confidence: 0.84,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "invoiceDate", Label: "Invoice Date", Regex: invoiceDateRegex,
		Pages: extracted.Pages, Confidence: 0.88,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "dueDate", Label: "Due Date", Regex: dueDateRegex,
		Pages: extracted.Pages, Confidence: 0.88,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "shipper", Label: "Bill To / Shipper", Regex: shipperRegex,
		Pages: extracted.Pages, Confidence: 0.72,
	})
	addCurrencyFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "totalDue", Label: "Total Due", Regex: totalDueRegex,
		Pages: extracted.Pages, Confidence: 0.93, Signal: "total due",
	})
}

func requiredFieldsForKind(kind string) []struct {
	key   string
	label string
} {
	switch kind {
	case kindRateConfirmation:
		return []struct {
			key   string
			label string
		}{
			{key: "shipper", label: "Shipper"},
			{key: "consignee", label: "Consignee"},
			{key: "pickupWindow", label: "Pickup Window"},
			{key: "deliveryWindow", label: "Delivery Window"},
			{key: "rate", label: "Rate"},
		}
	case kindBillOfLading:
		return []struct {
			key   string
			label string
		}{
			{key: "shipper", label: "Shipper"},
			{key: "consignee", label: "Consignee"},
			{key: "commodity", label: "Commodity"},
			{key: "referenceNumber", label: "Reference Number"},
		}
	case kindProofOfDelivery:
		return []struct {
			key   string
			label string
		}{
			{key: "consignee", label: "Consignee"},
			{key: "deliveryWindow", label: "Delivery"},
			{key: "signature", label: "Signature"},
		}
	case kindInvoice:
		return []struct {
			key   string
			label string
		}{
			{key: "referenceNumber", label: "Invoice Number"},
			{key: "invoiceDate", label: "Invoice Date"},
			{key: "dueDate", label: "Due Date"},
			{key: "totalDue", label: "Total Due"},
		}
	default:
		return nil
	}
}

func addFieldFromPages(params *RegexValueFieldParams) {
	pageNumber, match := firstPageMatch(params.Regex, params.Pages)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[len(match)-1])
	if value == "" {
		return
	}

	params.Fields[params.Key] = &ReviewField{
		Label:           params.Label,
		Value:           value,
		Confidence:      pageAdjustedConfidence(params.Confidence, pageNumber, params.Pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  params.Confidence < 0.8,
		Conflict:        hasConflictingMatches(params.Regex, value, params.Pages),
		Source:          "deterministic",
	}
	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func addCurrencyFieldFromPages(params *RegexValueFieldParams) {
	pageNumber, match := firstPageMatch(params.Regex, params.Pages)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[1])
	if value == "" {
		return
	}

	params.Fields[params.Key] = &ReviewField{
		Label:           params.Label,
		Value:           value,
		Confidence:      pageAdjustedConfidence(params.Confidence, pageNumber, params.Pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  params.Confidence < 0.8,
		Conflict:        hasConflictingMatches(params.Regex, value, params.Pages),
		Source:          "deterministic",
	}
	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func addRegexValueFieldFromPages(params *RegexValueFieldParams) {
	pageNumber, match := firstPageMatch(params.Regex, params.Pages)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[1])
	if value == "" {
		return
	}

	params.Fields[params.Key] = &ReviewField{
		Label:           params.Label,
		Value:           value,
		Confidence:      pageAdjustedConfidence(params.Confidence, pageNumber, params.Pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  params.ReviewRequired,
		Conflict:        hasConflictingMatches(params.Regex, value, params.Pages),
		Source:          "deterministic",
	}
	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func addWeightFieldFromPages(params *AddWeightFieldParams, pages []*PageExtractionResult) {
	pageNumber, match := firstPageMatch(weightRegex, pages)
	if len(match) < 2 {
		return
	}

	params.Fields["weight"] = &ReviewField{
		Label:           "Weight",
		Value:           fmt.Sprintf("%s lbs", strings.TrimSpace(match[1])),
		Confidence:      pageAdjustedConfidence(0.8, pageNumber, pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  false,
		Source:          "deterministic",
	}

	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func addStopTimingField(params *AddStopTimingFieldParams) {
	if params.Stop == nil {
		return
	}

	value := strings.TrimSpace(
		strings.Join(filterNonEmpty(params.Stop.Date, params.Stop.TimeWindow), " "),
	)
	if value == "" {
		return
	}

	params.Fields[params.Key] = &ReviewField{
		Label:           params.Label,
		Value:           value,
		Confidence:      clampConfidence((params.Confidence + params.Stop.Confidence) / 2),
		Excerpt:         params.Stop.EvidenceExcerpt,
		EvidenceExcerpt: params.Stop.EvidenceExcerpt,
		PageNumber:      params.Stop.PageNumber,
		ReviewRequired:  params.Stop.ReviewRequired,
		Source:          params.Stop.Source,
	}
	if params.Signals != nil {
		*params.Signals = append(*params.Signals, strings.ToLower(params.Label))
	}
}

func addFieldFromSectionLabels(params *AddFieldFromSectionLabelsParams) {
	matches := findSectionMatches(params)
	if len(matches) == 0 {
		return
	}

	selected := matches[0]
	conflict := false
	normalizedSelected := normalizeSectionValue(selected.Value)
	for _, match := range matches[1:] {
		if normalizeSectionValue(match.Value) != normalizedSelected {
			conflict = true
			break
		}
	}

	params.Fields[params.Key] = &ReviewField{
		Label: params.Label,
		Value: selected.Value,
		Confidence: pageAdjustedConfidence(
			params.Confidence,
			selected.PageNumber,
			params.Pages,
		),
		Excerpt:         selected.Excerpt,
		EvidenceExcerpt: selected.Excerpt,
		PageNumber:      selected.PageNumber,
		ReviewRequired: params.ReviewRequired || conflict ||
			normalizeSectionValue(selected.Value) == "",
		Conflict: conflict,
		Source:   "deterministic",
	}
	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func findSectionMatches(
	params *AddFieldFromSectionLabelsParams,
) []PageSectionMatch {
	matches := make([]PageSectionMatch, 0)
	for _, page := range params.Pages {
		lines := splitNormalizedLines(page.Text)
		for idx, line := range lines {
			if !matchesSectionLabel(line, params.Labels) {
				continue
			}
			block := collectSectionBlock(lines, idx)
			value := strings.TrimSpace(params.Extractor(line, block))
			if value == "" {
				continue
			}
			matches = append(matches, PageSectionMatch{
				PageNumber: page.PageNumber,
				Value:      value,
				Excerpt:    strings.Join(block, "\n"),
			})
		}
	}

	return dedupeSectionMatches(matches)
}

func matchesSectionLabel(line string, labels []string) bool {
	normalized := normalizeSectionLabel(line)
	for _, label := range labels {
		want := normalizeSectionLabel(label)
		if normalized == want || strings.HasPrefix(normalized, want+" ") {
			return true
		}
	}
	return false
}

func normalizeSectionLabel(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	lower = strings.TrimSuffix(lower, ":")
	lower = strings.ReplaceAll(lower, "-", " ")
	lower = strings.ReplaceAll(lower, "_", " ")
	lower = strings.Join(strings.Fields(lower), " ")
	return lower
}

func normalizeSectionValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func collectSectionBlock(lines []string, idx int) []string {
	end := idx + 6
	if end > len(lines) {
		end = len(lines)
	}

	block := make([]string, 0, end-idx)
	for pos := idx; pos < end; pos++ {
		line := strings.TrimSpace(lines[pos])
		if pos > idx && line == "" {
			break
		}
		if pos > idx && isLikelyBoundaryLine(line) {
			break
		}
		block = append(block, line)
	}
	return block
}

func isLikelyBoundaryLine(line string) bool {
	if line == "" {
		return false
	}
	normalized := normalizeSectionLabel(line)
	for _, boundary := range Boundaries {
		if normalized == boundary || strings.HasPrefix(normalized, boundary+" ") {
			return true
		}
	}
	return false
}

func extractEntityNameFromSection(header string, block []string) string {
	if value := extractSectionHeaderValue(header); value != "" && !looksLikeAddress(value) &&
		!cityStateZipRegex.MatchString(value) {
		return value
	}
	for _, line := range block[1:] {
		switch {
		case line == "":
			continue
		case looksLikeAddress(line):
			continue
		case cityStateZipRegex.MatchString(line):
			continue
		case dateValueRegex.MatchString(line):
			continue
		case strings.Contains(strings.ToLower(line), "signature"):
			continue
		default:
			return line
		}
	}
	if len(block) > 1 {
		return strings.TrimSpace(block[1])
	}
	return ""
}

func extractCommodityFromSection(header string, block []string) string {
	if value := extractSectionHeaderValue(header); value != "" {
		return value
	}
	for _, line := range block[1:] {
		if line == "" || looksLikeAddress(line) || cityStateZipRegex.MatchString(line) {
			continue
		}
		return line
	}
	return ""
}

func extractDeliveryFieldFromSection(header string, block []string) string {
	candidates := append([]string{header}, block[1:]...)
	for _, candidate := range candidates {
		date := firstRegexValue(dateValueRegex, candidate)
		window := firstRegexValue(timeWindowRegex, candidate)
		value := strings.TrimSpace(strings.Join(filterNonEmpty(date, window), " "))
		if value != "" {
			return value
		}
	}
	if value := extractSectionHeaderValue(header); value != "" {
		return value
	}
	for _, line := range block[1:] {
		if line != "" {
			return line
		}
	}
	return ""
}

func extractSignatureFromSection(header string, block []string) string {
	if value := extractSectionHeaderValue(header); value != "" &&
		!dateValueRegex.MatchString(value) {
		return value
	}
	for _, line := range block[1:] {
		if line == "" || dateValueRegex.MatchString(line) ||
			strings.Contains(strings.ToLower(line), "date") {
			continue
		}
		return line
	}
	return ""
}

func extractFreeformSectionValue(header string, block []string) string {
	if value := extractSectionHeaderValue(header); value != "" {
		return value
	}
	for _, line := range block[1:] {
		if line != "" {
			return line
		}
	}
	return ""
}

func extractSectionHeaderValue(header string) string {
	parts := strings.SplitN(header, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func dedupeSectionMatches(matches []PageSectionMatch) []PageSectionMatch {
	deduped := make([]PageSectionMatch, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		key := fmt.Sprintf("%s|%d", normalizeSectionValue(match.Value), match.PageNumber)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, match)
	}
	return deduped
}

func extractRateConfirmationStops(pages []*PageExtractionResult) []*IntelligenceStop {
	stops := make([]*IntelligenceStop, 0, 2)

	for _, page := range pages {
		lines := splitNormalizedLines(page.Text)
		for idx, line := range lines {
			role, ok := detectStopRole(line)
			if !ok {
				continue
			}

			stop := &IntelligenceStop{
				Sequence:        len(stops) + 1,
				Role:            role,
				PageNumber:      page.PageNumber,
				EvidenceExcerpt: collectStopExcerpt(lines, idx),
				Confidence:      baseStopConfidence(page),
				ReviewRequired:  false,
				Source:          "deterministic",
			}

			block := collectRateConfirmationStopBlock(lines, idx)
			if len(block) > 0 {
				stop.EvidenceExcerpt = strings.Join(block, "\n")
			}
			populateRateConfirmationStop(stop, block)
			if !hasMeaningfulStopData(stop) {
				continue
			}

			if stop.AddressLine1 == "" || stop.Date == "" {
				stop.ReviewRequired = true
				stop.Confidence = clampConfidence(stop.Confidence - 0.18)
			}
			if stop.City == "" || stop.State == "" {
				stop.ReviewRequired = true
				stop.Confidence = clampConfidence(stop.Confidence - 0.08)
			}

			stops = append(stops, stop)
		}
	}

	return stops
}

func collectFieldConflicts(fields map[string]*ReviewField) []*ReviewConflict {
	conflicts := make([]*ReviewConflict, 0)
	for key, field := range fields {
		if !field.Conflict {
			continue
		}
		conflicts = append(conflicts, &ReviewConflict{
			Key:             key,
			Label:           field.Label,
			Values:          []string{field.Value},
			PageNumbers:     nonZeroPageNumbers(field.PageNumber),
			EvidenceExcerpt: field.EvidenceExcerpt,
			Source:          field.Source,
		})
	}
	return conflicts
}

func collectStopConflicts(stops []*IntelligenceStop) []*ReviewConflict {
	conflicts := make([]*ReviewConflict, 0)

	for _, role := range []string{stopRolePickup, stopRoleDelivery} {
		addresses := make(map[string][]*IntelligenceStop)
		dates := make(map[string][]*IntelligenceStop)
		for _, stop := range stops {
			if stop.Role != role {
				continue
			}
			if address := strings.TrimSpace(strings.ToLower(stop.AddressLine1)); address != "" {
				addresses[address] = append(addresses[address], stop)
			}
			if date := strings.TrimSpace(strings.ToLower(stop.Date)); date != "" {
				dates[date] = append(dates[date], stop)
			}
		}

		if len(addresses) > 1 {
			conflicts = append(conflicts, &ReviewConflict{
				Key:             fmt.Sprintf("%sAddress", role),
				Label:           fmt.Sprintf("%s Address", roleLabel(role)),
				Values:          mapKeys(addresses),
				PageNumbers:     stopPages(stops, role),
				EvidenceExcerpt: firstStopExcerpt(stops, role),
				Source:          "deterministic",
			})
		}
		if len(dates) > 1 {
			conflicts = append(conflicts, &ReviewConflict{
				Key:             fmt.Sprintf("%sDate", role),
				Label:           fmt.Sprintf("%s Date", roleLabel(role)),
				Values:          mapKeys(dates),
				PageNumbers:     stopPages(stops, role),
				EvidenceExcerpt: firstStopExcerpt(stops, role),
				Source:          "deterministic",
			})
		}
	}

	return conflicts
}

func hasStopRole(stops []*IntelligenceStop, role string) bool {
	for _, stop := range stops {
		if stop.Role == role {
			return true
		}
	}
	return false
}

func hasReviewRequiredStop(stops []*IntelligenceStop) bool {
	for _, stop := range stops {
		if stop.ReviewRequired {
			return true
		}
	}
	return false
}

func appendIfMissing(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func splitNormalizedLines(text string) []string {
	rawLines := strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}

func detectStopRole(line string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(line))
	if isStopMetadataLine(lower) {
		return "", false
	}
	switch {
	case stopSectionRegex.MatchString(lower):
		switch {
		case strings.HasPrefix(lower, "shipper"),
			strings.HasPrefix(lower, "pickup"),
			strings.HasPrefix(lower, "origin"):
			return stopRolePickup, true
		case strings.HasPrefix(lower, "receiver"),
			strings.HasPrefix(lower, "consignee"),
			strings.HasPrefix(lower, "delivery"),
			strings.HasPrefix(lower, "drop"),
			strings.HasPrefix(lower, "destination"):
			return stopRoleDelivery, true
		}
		return "", false
	case strings.HasPrefix(lower, "pickup"):
		if strings.HasPrefix(lower, "pickup date") || strings.HasPrefix(lower, "pickup window") {
			return "", false
		}
		return stopRolePickup, true
	case strings.HasPrefix(lower, "delivery"), strings.HasPrefix(lower, "drop"):
		if strings.HasPrefix(lower, "delivery date") ||
			strings.HasPrefix(lower, "delivery window") {
			return "", false
		}
		return stopRoleDelivery, true
	default:
		return "", false
	}
}

func collectRateConfirmationStopBlock(lines []string, idx int) []string {
	end := idx + 36
	if end > len(lines) {
		end = len(lines)
	}

	block := make([]string, 0, end-idx)
	blankRun := 0
	for pos := idx; pos < end; pos++ {
		line := strings.TrimSpace(lines[pos])

		if pos > idx {
			if _, nextStop := detectStopRole(line); nextStop {
				break
			}
			if strings.HasPrefix(line, "--- PAGE ") {
				break
			}
		}

		if line == "" {
			blankRun++
			if blankRun > 5 && hasStopSignal(block) {
				break
			}
			continue
		}

		blankRun = 0
		block = append(block, line)
	}

	return block
}

func populateRateConfirmationStop(stop *IntelligenceStop, block []string) {
	if stop == nil || len(block) == 0 {
		return
	}

	header := block[0]
	if labelValue := extractLabelValue(header); labelValue != "" &&
		!isStopMetadataLine(strings.ToLower(labelValue)) {
		switch {
		case looksLikeAddress(labelValue):
			stop.AddressLine1 = labelValue
		case dateLabelRegex.MatchString(header):
			if stop.Date == "" {
				stop.Date = firstRegexValue(dateValueRegex, labelValue)
			}
			if stop.TimeWindow == "" {
				stop.TimeWindow = firstRegexValue(timeWindowRegex, labelValue)
			}
		default:
			stop.Name = labelValue
		}
	}

	if stop.Date == "" {
		stop.Date = findLastRegexValue(dateValueRegex, block)
	}
	if stop.TimeWindow == "" {
		stop.TimeWindow = findLastStopTimeValue(block)
	}

	cityIdx, city, state, postalCode := findLastCityStateZip(block)
	if cityIdx >= 0 {
		stop.City = city
		stop.State = state
		stop.PostalCode = postalCode
		if stop.AddressLine1 == "" {
			stop.AddressLine1, stop.AddressLine2 = extractAddressBeforeCity(block, cityIdx)
		}
		if stop.Name == "" {
			stop.Name = extractStopNameBeforeIndex(block, cityIdx)
		}
	}

	if stop.AddressLine1 == "" {
		stop.AddressLine1 = findLastAddressLine(block)
	}
	if stop.Name == "" {
		stop.Name = extractStopNameBeforeIndex(block, len(block))
	}

	stop.Name = sanitizeStopName(stop.Name)
	stop.AddressLine1 = strings.TrimSpace(stop.AddressLine1)
	stop.AddressLine2 = strings.TrimSpace(stop.AddressLine2)
	stop.Date = strings.TrimSpace(stop.Date)
	stop.TimeWindow = strings.TrimSpace(stop.TimeWindow)
	stop.AppointmentRequired = strings.Contains(
		strings.ToLower(strings.Join(block, "\n")),
		"appointment",
	) ||
		strings.Contains(strings.ToLower(stop.TimeWindow), "appt")
}

func hasStopSignal(block []string) bool {
	for _, line := range block {
		if looksLikeAddress(line) || cityStateZipRegex.MatchString(line) ||
			dateValueRegex.MatchString(line) ||
			timeWindowRegex.MatchString(line) {
			return true
		}
	}
	return false
}

func findLastRegexValue(re *regexp.Regexp, block []string) string {
	for idx := len(block) - 1; idx >= 0; idx-- {
		if value := firstRegexValue(re, block[idx]); value != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func findLastStopTimeValue(block []string) string {
	if value := findLastRegexValue(timeWindowRegex, block); value != "" {
		return value
	}
	return findLastRegexValue(appointmentRegex, block)
}

func findLastCityStateZip(
	block []string,
) (foundIdx int, foundCity, foundState, foundZip string) {
	for i := len(block) - 1; i >= 0; i-- {
		if !cityStateZipRegex.MatchString(block[i]) {
			continue
		}
		city, state, postalCode := extractCityStateZip(block[i])
		if city != "" && state != "" {
			return i, city, state, postalCode
		}
	}
	return -1, "", "", ""
}

func extractAddressBeforeCity(
	block []string, cityIdx int,
) (addr1, addr2 string) { //nolint:unparam // addr2 reserved for future suite/apt lines
	if cityIdx <= 0 || cityIdx > len(block) {
		return "", ""
	}

	prevIdx, prevLine := previousMeaningfulStopLine(block, cityIdx-1)
	if prevIdx < 0 {
		return "", ""
	}

	if looksLikeAddress(prevLine) {
		return prevLine, ""
	}
	if isStreetFragment(prevLine) {
		numberIdx, numberLine := previousMeaningfulStopLine(block, prevIdx-1)
		if numberIdx >= 0 && isNumericAddressPrefix(numberLine) {
			return strings.TrimSpace(numberLine + " " + prevLine), ""
		}
	}

	return "", ""
}

func extractStopNameBeforeIndex(block []string, limit int) string {
	if limit > len(block) {
		limit = len(block)
	}
	for idx := limit - 1; idx >= 0; idx-- {
		line := strings.TrimSpace(block[idx])
		if !isUsableStopName(line) {
			continue
		}
		return line
	}
	return ""
}

func previousMeaningfulStopLine(
	block []string, start int,
) (foundIdx int, foundLine string) {
	for i := start; i >= 0; i-- {
		l := strings.TrimSpace(block[i])
		if l == "" || isStopMetadataLine(strings.ToLower(l)) ||
			phoneLineRegex.MatchString(l) {
			continue
		}
		return i, l
	}
	return -1, ""
}

func findLastAddressLine(block []string) string {
	for idx := len(block) - 1; idx >= 0; idx-- {
		line := strings.TrimSpace(block[idx])
		if line == "" || isStopMetadataLine(strings.ToLower(line)) {
			continue
		}
		if looksLikeAddress(line) {
			return line
		}
		if isStreetFragment(line) {
			if prevIdx, prevLine := previousMeaningfulStopLine(block, idx-1); prevIdx >= 0 &&
				isNumericAddressPrefix(prevLine) {
				return strings.TrimSpace(prevLine + " " + line)
			}
		}
	}
	return ""
}

func sanitizeStopName(name string) string {
	trimmed := strings.TrimSpace(name)
	if !isUsableStopName(trimmed) {
		return ""
	}
	return trimmed
}

func hasMeaningfulStopData(stop *IntelligenceStop) bool {
	return strings.TrimSpace(stop.Name) != "" ||
		strings.TrimSpace(stop.AddressLine1) != "" ||
		(strings.TrimSpace(stop.City) != "" && strings.TrimSpace(stop.State) != "") ||
		strings.TrimSpace(stop.Date) != "" ||
		strings.TrimSpace(stop.TimeWindow) != ""
}

func isUsableStopName(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if isStopMetadataLine(lower) || phoneLineRegex.MatchString(trimmed) ||
		looksLikeAddress(trimmed) ||
		cityStateZipRegex.MatchString(trimmed) ||
		dateValueRegex.MatchString(trimmed) ||
		isStreetFragment(trimmed) ||
		isNumericAddressPrefix(trimmed) {
		return false
	}
	return true
}

func isStopMetadataLine(lower string) bool {
	normalized := normalizeSectionLabel(lower)
	if normalized == "" {
		return false
	}

	switch {
	case normalized == "shipper instructions",
		normalized == "receiver instructions",
		normalized == "address",
		normalized == "phone",
		normalized == "ref #",
		normalized == "ref",
		normalized == "commodity",
		normalized == "est wgt",
		normalized == "units",
		normalized == "count",
		normalized == "pallets",
		normalized == "temp",
		normalized == "driver name",
		normalized == "trailer #",
		normalized == "tractor #",
		normalized == "pickup#",
		normalized == "delivery#",
		normalized == "appointment#",
		normalized == "pick up date",
		normalized == "pick up time",
		normalized == "pickup date",
		normalized == "pickup time",
		normalized == "delivery date",
		normalized == "delivery time":
		return true
	case strings.HasPrefix(normalized, "please "),
		strings.HasPrefix(normalized, "scheduled "),
		strings.HasPrefix(normalized, "page "),
		strings.HasPrefix(normalized, "this load was booked"),
		strings.HasPrefix(normalized, "thank you"),
		normalized == "loose(s)":
		return true
	default:
		return false
	}
}

func isStreetFragment(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || looksLikeAddress(trimmed) || cityStateZipRegex.MatchString(trimmed) {
		return false
	}
	if isNumericAddressPrefix(trimmed) || phoneLineRegex.MatchString(trimmed) ||
		dateValueRegex.MatchString(trimmed) {
		return false
	}
	lower := strings.ToLower(trimmed)
	if isStopMetadataLine(lower) {
		return false
	}

	streetKeywords := []string{
		"street",
		"st",
		"road",
		"rd",
		"drive",
		"dr",
		"avenue",
		"ave",
		"boulevard",
		"blvd",
		"lane",
		"ln",
		"court",
		"ct",
		"circle",
		"cir",
		"way",
		"parkway",
		"pkwy",
		"highway",
		"hwy",
		"suite",
		"ste",
	}
	for _, keyword := range streetKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func isNumericAddressPrefix(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func collectStopExcerpt(lines []string, idx int) string {
	end := idx + 4
	if end > len(lines) {
		end = len(lines)
	}
	chunk := make([]string, 0, end-idx)
	for _, line := range lines[idx:end] {
		if strings.TrimSpace(line) == "" {
			break
		}
		chunk = append(chunk, line)
	}
	return strings.Join(chunk, "\n")
}

func extractLabelValue(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func looksLikeAddress(value string) bool {
	return addressLineRegex.MatchString(strings.TrimSpace(value))
}

func extractCityStateZip(line string) (city, state, postalCode string) {
	match := cityStateZipRegex.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 4 {
		return "", "", ""
	}
	return strings.TrimSpace(
			match[1],
		), strings.ToUpper(
			strings.TrimSpace(match[2]),
		), strings.TrimSpace(
			match[3],
		)
}

func firstRegexValue(re *regexp.Regexp, text string) string {
	match := re.FindStringSubmatch(text)
	if len(match) == 0 {
		return ""
	}
	if len(match) == 1 {
		return strings.TrimSpace(match[0])
	}
	return strings.TrimSpace(match[1])
}

func baseStopConfidence(page *PageExtractionResult) float64 {
	if page.SourceKind == documentcontent.SourceKindOCR {
		return clampConfidence((0.72 + page.OCRConfidence) / 2)
	}
	return 0.9
}

func nonZeroPageNumbers(pageNumber int) []int {
	if pageNumber <= 0 {
		return []int{}
	}
	return []int{pageNumber}
}

func stopPages(stops []*IntelligenceStop, role string) []int {
	pages := make([]int, 0)
	seen := make(map[int]struct{})
	for _, stop := range stops {
		if stop.Role != role || stop.PageNumber <= 0 {
			continue
		}
		if _, ok := seen[stop.PageNumber]; ok {
			continue
		}
		seen[stop.PageNumber] = struct{}{}
		pages = append(pages, stop.PageNumber)
	}
	return pages
}

func firstStopExcerpt(stops []*IntelligenceStop, role string) string {
	for _, stop := range stops {
		if stop.Role == role && stop.EvidenceExcerpt != "" {
			return stop.EvidenceExcerpt
		}
	}
	return ""
}

func mapKeys[T any](items map[string][]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	return keys
}

func roleLabel(role string) string {
	if role == "" {
		return "Unknown"
	}
	switch role {
	case stopRolePickup:
		return "Pickup"
	case stopRoleDelivery:
		return "Delivery"
	default:
		return strings.ToUpper(role[:1]) + role[1:]
	}
}

func firstStopByRole(stops []*IntelligenceStop, role string) *IntelligenceStop {
	for idx := range stops {
		if stops[idx].Role == role {
			return stops[idx]
		}
	}
	return nil
}

func filterNonEmpty(items ...string) []string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func buildContentPages(
	content *documentcontent.Content,
	pages []*PageExtractionResult,
) []*documentcontent.Page {
	items := make([]*documentcontent.Page, 0, len(pages))
	for _, page := range pages {
		items = append(items, &documentcontent.Page{
			DocumentContentID:    content.ID,
			DocumentID:           content.DocumentID,
			OrganizationID:       content.OrganizationID,
			BusinessUnitID:       content.BusinessUnitID,
			PageNumber:           page.PageNumber,
			SourceKind:           page.SourceKind,
			ExtractedText:        page.Text,
			OCRConfidence:        page.OCRConfidence,
			PreprocessingApplied: page.PreprocessingApplied,
			Width:                page.Width,
			Height:               page.Height,
			Metadata:             defaultMetadata(page.Metadata),
		})
	}

	return items
}

func finalizeExtraction(pages []*PageExtractionResult, maxExtractedChars int) *ExtractionResult {
	textParts := make([]string, 0, len(pages))
	pageCount := len(pages)
	nativeCount := 0
	ocrCount := 0
	weightedConfidence := 0.0
	weightedPages := 0.0

	for _, page := range pages {
		if trimmed := strings.TrimSpace(page.Text); trimmed != "" {
			textParts = append(textParts, trimmed)
		}
		switch page.SourceKind {
		case documentcontent.SourceKindOCR:
			ocrCount++
			weightedConfidence += page.OCRConfidence
			weightedPages++
		case documentcontent.SourceKindNative:
			nativeCount++
			weightedConfidence += 0.99
			weightedPages++
		case documentcontent.SourceKindMixed:
			nativeCount++
			ocrCount++
			weightedConfidence += (page.OCRConfidence + 0.99) / 2
			weightedPages++
		}
	}

	sourceKind := documentcontent.SourceKindNative
	switch {
	case ocrCount > 0 && nativeCount > 0:
		sourceKind = documentcontent.SourceKindMixed
	case ocrCount > 0:
		sourceKind = documentcontent.SourceKindOCR
	}

	if weightedPages > 0 && len(pages) > 0 {
		for idx := range pages {
			if pages[idx].Metadata == nil {
				pages[idx].Metadata = map[string]any{}
			}
			pages[idx].Metadata["documentAverageConfidence"] = clampConfidence(
				weightedConfidence / weightedPages,
			)
		}
	}

	return &ExtractionResult{
		Text:       stringutils.TruncateAndTrim(strings.Join(textParts, "\n\n"), maxExtractedChars),
		PageCount:  pageCount,
		SourceKind: sourceKind,
		Pages:      pages,
	}
}

func defaultMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func firstPageMatch(
	re *regexp.Regexp, pages []*PageExtractionResult,
) (pageNum int, groups []string) {
	for _, page := range pages {
		m := re.FindStringSubmatch(strings.ReplaceAll(page.Text, "\r", ""))
		if len(m) > 0 {
			return page.PageNumber, m
		}
	}
	return 0, nil
}

func hasConflictingMatches(re *regexp.Regexp, selected string, pages []*PageExtractionResult) bool {
	normalizedSelected := strings.TrimSpace(strings.ToLower(selected))
	for _, page := range pages {
		matches := re.FindAllStringSubmatch(strings.ReplaceAll(page.Text, "\r", ""), -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			candidate := strings.TrimSpace(strings.ToLower(match[len(match)-1]))
			if candidate != "" && candidate != normalizedSelected {
				return true
			}
		}
	}
	return false
}

func pageAdjustedConfidence(base float64, pageNumber int, pages []*PageExtractionResult) float64 {
	if pageNumber <= 0 {
		return clampConfidence(base)
	}
	for _, page := range pages {
		if page.PageNumber != pageNumber {
			continue
		}
		if page.SourceKind == documentcontent.SourceKindOCR {
			return clampConfidence((base + page.OCRConfidence) / 2)
		}
		return clampConfidence((base + 0.99) / 2)
	}
	return clampConfidence(base)
}

func readImageDimensions(imageData []byte) (width, height int, err error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
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
		processed = imaging.AdjustContrast(processed, 25)
		processed = imaging.Sharpen(processed, 1.5)
		processed = thresholdImage(processed, 170)
	}

	buf := new(bytes.Buffer)
	if encErr := imaging.Encode(buf, processed, imaging.PNG); encErr != nil {
		return nil, 0, 0, encErr
	}

	bounds := processed.Bounds()
	return buf.Bytes(), bounds.Dx(), bounds.Dy(), nil
}

func thresholdImage(img image.Image, threshold uint8) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			lumaVal := ((299 * r) + (587 * g) + (114 * b) + 500) / 1000 >> 8
			luma := intutils.SafeUint32ToUint8(lumaVal)
			alpha := intutils.SafeUint32ToUint8(a >> 8)
			if luma >= threshold {
				dst.SetNRGBA(x, y, color.NRGBA{
					R: 255, G: 255, B: 255, A: alpha,
				})
				continue
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: 0, G: 0, B: 0, A: alpha,
			})
		}
	}

	return dst
}

//nolint:gocognit,funlen // TSV parsing with line-grouping state machine
func parseTesseractTSV(
	output string,
) (parsed string, avgConfidence float64, parseErr error) { //nolint:unparam // error return reserved for future format validation
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	if len(lines) <= 1 {
		return "", 0, nil
	}

	type lineKey struct {
		page  int
		block int
		par   int
		line  int
	}

	lineTexts := make([]string, 0)
	currentKey := lineKey{}
	currentWords := make([]string, 0, 8)
	totalConfidence := 0.0
	confidenceCount := 0.0
	seenHeader := false

	flush := func() {
		if len(currentWords) == 0 {
			return
		}
		lineTexts = append(lineTexts, strings.Join(currentWords, " "))
		currentWords = currentWords[:0]
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		cols := strings.Split(raw, "\t")
		if len(cols) < 12 {
			continue
		}
		if !seenHeader {
			seenHeader = true
			if strings.EqualFold(cols[0], "level") {
				continue
			}
		}

		level, err := strconv.Atoi(cols[0])
		if err != nil || level != 5 {
			continue
		}

		key := lineKey{}
		if key.page, err = strconv.Atoi(cols[1]); err != nil {
			continue
		}
		if key.block, err = strconv.Atoi(cols[2]); err != nil {
			continue
		}
		if key.par, err = strconv.Atoi(cols[3]); err != nil {
			continue
		}
		if key.line, err = strconv.Atoi(cols[4]); err != nil {
			continue
		}

		if currentKey != (lineKey{}) && key != currentKey {
			flush()
		}
		currentKey = key

		text := strings.TrimSpace(cols[11])
		if text == "" {
			continue
		}
		currentWords = append(currentWords, text)

		conf, err := strconv.ParseFloat(cols[10], 64)
		if err == nil && conf >= 0 {
			totalConfidence += conf / 100
			confidenceCount++
		}
	}

	flush()

	parsed = strings.TrimSpace(strings.Join(lineTexts, "\n"))
	if confidenceCount > 0 {
		avgConfidence = clampConfidence(totalConfidence / confidenceCount)
	}

	return parsed, avgConfidence, nil
}

func containsAny(corpus string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(corpus, needle) {
			return true
		}
	}
	return false
}

func clampConfidence(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 0.99:
		return 0.99
	default:
		return value
	}
}
