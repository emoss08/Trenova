package documentintelligencejobs

import (
	"bytes"
	"context"
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
	DraftRepo           repositories.DocumentShipmentDraftRepository
	AIDocumentService   services.AIDocumentService
	SearchProjection    services.DocumentSearchProjectionService
	Storage             storage.Client
	WorkflowStarter     services.WorkflowStarter
	ParsingRuleRuntime  services.DocumentParsingRuleRuntime
}

type Activities struct {
	logger              *zap.Logger
	cfg                 *config.DocumentIntelligenceConfig
	metrics             *metrics.Registry
	documentRepo        repositories.DocumentRepository
	documentControlRepo repositories.DocumentControlRepository
	documentTypeRepo    repositories.DocumentTypeRepository
	contentRepo         repositories.DocumentContentRepository
	draftRepo           repositories.DocumentShipmentDraftRepository
	aiDocumentService   services.AIDocumentService
	searchProjection    services.DocumentSearchProjectionService
	storage             storage.Client
	workflowStarter     services.WorkflowStarter
	parsingRuleRuntime  services.DocumentParsingRuleRuntime
}

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
		draftRepo:           p.DraftRepo,
		aiDocumentService:   aiDocumentService,
		searchProjection:    searchProjection,
		storage:             p.Storage,
		workflowStarter:     workflowStarter,
		parsingRuleRuntime:  parsingRuleRuntime,
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

	now := time.Now().Unix()
	content := &documentcontent.Content{
		DocumentID:      doc.ID,
		OrganizationID:  doc.OrganizationID,
		BusinessUnitID:  doc.BusinessUnitID,
		Status:          documentcontent.StatusExtracting,
		LastExtractedAt: &now,
	}
	if _, err = a.contentRepo.Upsert(ctx, content); err != nil {
		return nil, err
	}

	doc.ContentStatus = document.ContentStatusExtracting
	doc.ContentError = ""
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
		return nil, err
	}
	a.metrics.Document.RecordExtraction("started", "", "none")
	a.syncSearchProjection(ctx, doc, "")

	download, err := a.storage.Download(ctx, doc.StoragePath)
	if err != nil {
		return nil, a.markFailed(ctx, doc, content, "DOWNLOAD_FAILED", "Failed to download document")
	}
	defer download.Body.Close()

	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(download.Body); err != nil {
		return nil, a.markFailed(ctx, doc, content, "READ_FAILED", "Failed to read document bytes")
	}

	extracted, err := a.extractContent(ctx, doc, buf.Bytes(), control)
	if err != nil {
		return nil, a.markFailed(ctx, doc, content, "EXTRACTION_FAILED", err.Error())
	}

	features := extractDocumentFeatures(doc.OriginalName, extracted.Pages, extracted.Text)
	fingerprint := detectProviderFingerprint(doc.OriginalName, extracted.Text, features)
	classification := classifyDocumentWithControl(doc.OriginalName, extracted.Text, extracted.Pages, control, features, fingerprint)
	intelligence := analyzeDocument(classification, extracted)
	intelligence = a.applyParsingRules(ctx, tenantInfo, doc.OriginalName, classification.ProviderFingerprint, extracted, intelligence)
	classification, intelligence = a.enrichWithAI(ctx, payload, doc, control, extracted, features, fingerprint, classification, intelligence)
	structured := buildStructuredData(intelligence)
	content.Status = documentcontent.StatusIndexed
	content.ContentText = extracted.Text
	content.PageCount = extracted.PageCount
	content.SourceKind = extracted.SourceKind
	content.DetectedLanguage = "en"
	content.DetectedDocumentKind = classification.Kind
	content.ClassificationConfidence = classification.Confidence
	content.StructuredData = structured
	content.FailureCode = ""
	content.FailureMessage = ""
	content.LastExtractedAt = &now
	if _, err = a.contentRepo.Upsert(ctx, content); err != nil {
		return nil, err
	}
	if err = a.contentRepo.ReplacePages(ctx, content, buildContentPages(content, extracted.Pages)); err != nil {
		return nil, err
	}
	a.metrics.Document.RecordExtraction("succeeded", extracted.SourceKind, "none")

	doc.ContentStatus = document.ContentStatusIndexed
	doc.ContentError = ""
	doc.HasExtractedText = strings.TrimSpace(extracted.Text) != ""
	doc.DetectedKind = classification.Kind
	if err = a.associateDocumentType(ctx, doc, tenantInfo, classification.Kind, control); err != nil {
		return nil, err
	}

	draftStatus := document.ShipmentDraftStatusUnavailable
	if canGenerateShipmentDraft(control, doc.ResourceType, classification.Kind) &&
		intelligence.ReviewStatus == "Ready" {
		draft := &documentshipmentdraft.Draft{
			DocumentID:     doc.ID,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			Status:         documentshipmentdraft.StatusReady,
			DocumentKind:   classification.Kind,
			Confidence:     intelligence.OverallConfidence,
			DraftData:      intelligence.ToMap(),
		}
		if _, err = a.draftRepo.Upsert(ctx, draft); err != nil {
			return nil, err
		}
		draftStatus = document.ShipmentDraftStatusReady
		a.metrics.Document.RecordShipmentDraft("ready", doc.ResourceType, classification.Kind)
	} else {
		draftState := documentshipmentdraft.StatusUnavailable
		if canGenerateShipmentDraft(control, doc.ResourceType, classification.Kind) {
			draftStatus = document.ShipmentDraftStatusPending
		}
		if canGenerateShipmentDraft(control, doc.ResourceType, classification.Kind) {
			draftState = documentshipmentdraft.StatusPending
		}
		if _, err = a.draftRepo.Upsert(ctx, &documentshipmentdraft.Draft{
			DocumentID:     doc.ID,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			Status:         draftState,
			DocumentKind:   classification.Kind,
			Confidence:     intelligence.OverallConfidence,
			DraftData:      intelligence.ToMap(),
		}); err != nil {
			return nil, err
		}
		a.metrics.Document.RecordShipmentDraft("unavailable", doc.ResourceType, classification.Kind)
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
		return nil, err
	}
	indexedText := extracted.Text
	if !control.EnableFullTextIndexing {
		indexedText = ""
	}
	a.syncSearchProjection(ctx, doc, indexedText)

	return &ProcessDocumentIntelligenceResult{
		DocumentID: doc.ID,
		Status:     string(doc.ContentStatus),
		Kind:       classification.Kind,
	}, nil
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
				ID:                                       fmt.Sprintf("document-intelligence-%s", doc.ID.String()),
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

type extractionResult struct {
	Text       string
	PageCount  int
	SourceKind documentcontent.SourceKind
	Pages      []pageExtractionResult
}

type pageExtractionResult struct {
	PageNumber           int
	SourceKind           documentcontent.SourceKind
	Text                 string
	OCRConfidence        float64
	PreprocessingApplied bool
	Width                int
	Height               int
	Metadata             map[string]any
}

func (a *Activities) extractContent(
	ctx context.Context,
	doc *document.Document,
	data []byte,
	control *tenant.DocumentControl,
) (*extractionResult, error) {
	contentType := strings.ToLower(doc.FileType)
	ext := strings.ToLower(filepath.Ext(doc.OriginalName))

	switch {
	case isPlainTextType(contentType, ext):
		text := truncateExtractedText(string(data), a.cfg.GetMaxExtractedChars())
		return finalizeExtraction([]pageExtractionResult{{
			PageNumber: 1,
			SourceKind: documentcontent.SourceKindNative,
			Text:       text,
		}}, a.cfg.GetMaxExtractedChars()), nil
	case isFitzType(contentType, ext):
		return a.extractViaFitz(ctx, data, control.EnableOCR)
	case strings.HasPrefix(contentType, "image/"):
		if !control.EnableOCR {
			return finalizeExtraction([]pageExtractionResult{{
				PageNumber: 1,
				SourceKind: documentcontent.SourceKindOCR,
			}}, a.cfg.GetMaxExtractedChars()), nil
		}
		page, err := a.runOCRPage(ctx, data, ext, 1)
		if err != nil {
			return nil, err
		}
		return finalizeExtraction([]pageExtractionResult{page}, a.cfg.GetMaxExtractedChars()), nil
	default:
		return nil, fmt.Errorf("unsupported document type for extraction: %s", doc.FileType)
	}
}

func (a *Activities) extractViaFitz(ctx context.Context, data []byte, enableOCR bool) (*extractionResult, error) {
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	pageCount := doc.NumPage()
	pages := make([]pageExtractionResult, 0, pageCount)

	for page := range pageCount {
		activity.RecordHeartbeat(ctx, page)
		pageText, textErr := doc.Text(page)
		if textErr == nil && strings.TrimSpace(pageText) != "" {
			pages = append(pages, pageExtractionResult{
				PageNumber: page + 1,
				SourceKind: documentcontent.SourceKindNative,
				Text:       pageText,
			})
			continue
		}

		if !enableOCR || page >= a.cfg.GetMaxOCRPages() {
			pages = append(pages, pageExtractionResult{
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
			pages = append(pages, pageExtractionResult{
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
			pages = append(pages, pageExtractionResult{
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
) (pageExtractionResult, error) {
	page := pageExtractionResult{
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
		return page, err
	}

	page.Text = truncateExtractedText(text, a.cfg.GetMaxExtractedChars())
	page.OCRConfidence = confidence
	if page.Metadata == nil {
		page.Metadata = map[string]any{}
	}
	page.Metadata["ocrLanguage"] = a.cfg.GetOCRLanguage()
	page.Metadata["ocrConfidence"] = confidence

	return page, nil
}

func (a *Activities) runOCR(ctx context.Context, imageData []byte, ext string) (string, float64, error) {
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

	cmd := exec.CommandContext(ocrCtx, a.cfg.GetOCRCommand(), tmpFile.Name(), "stdout", "-l", a.cfg.GetOCRLanguage(), "tsv")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ocrCtx.Err() != nil {
			return "", 0, fmt.Errorf("ocr command timed out after %s: %w", a.cfg.GetOCRTimeout(), ocrCtx.Err())
		}
		return "", 0, fmt.Errorf("ocr command failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	text, confidence, parseErr := parseTesseractTSV(string(output))
	if parseErr != nil {
		return "", 0, parseErr
	}

	return text, confidence, nil
}

func (a *Activities) markFailed(
	ctx context.Context,
	doc *document.Document,
	content *documentcontent.Content,
	code, message string,
) error {
	now := time.Now().Unix()
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
	if err := a.documentRepo.UpdateIntelligence(ctx, &repositories.UpdateDocumentIntelligenceRequest{
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

func truncateExtractedText(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 || len(text) <= max {
		return text
	}
	return text[:max]
}

type classificationResult struct {
	Kind                string
	Confidence          float64
	Signals             []string
	ReviewRequired      bool
	Source              string
	ProviderFingerprint string
	Reason              string
}

type documentFeatureSet struct {
	TitleCandidates  []string
	SectionLabels    []string
	PartyLabels      []string
	ReferenceLabels  []string
	MoneySignals     []string
	StopSignals      []string
	TermsSignals     []string
	SignatureSignals []string
}

type providerFingerprint struct {
	Provider   string
	KindHint   string
	Confidence float64
	Signals    []string
}

type reviewField struct {
	Label           string
	Value           string
	Confidence      float64
	Excerpt         string
	EvidenceExcerpt string
	PageNumber      int
	ReviewRequired  bool
	Conflict        bool
	Source          string
}

type reviewConflict struct {
	Key             string
	Label           string
	Values          []string
	PageNumbers     []int
	EvidenceExcerpt string
	Source          string
}

type intelligenceStop struct {
	Sequence            int
	Role                string
	Name                string
	AddressLine1        string
	AddressLine2        string
	City                string
	State               string
	PostalCode          string
	Date                string
	TimeWindow          string
	AppointmentRequired bool
	PageNumber          int
	EvidenceExcerpt     string
	Confidence          float64
	ReviewRequired      bool
	Source              string
}

type documentIntelligenceAnalysis struct {
	Kind                 string
	OverallConfidence    float64
	ReviewStatus         string
	MissingFields        []string
	Signals              []string
	ClassifierSource     string
	ProviderFingerprint  string
	ClassificationReason string
	ParsingRuleMetadata  *services.DocumentParsingRuleMetadata
	Conflicts            []reviewConflict
	Fields               map[string]reviewField
	Stops                []intelligenceStop
	RawExcerpt           string
}

func (a documentIntelligenceAnalysis) ToMap() map[string]any {
	fields := make(map[string]any, len(a.Fields))
	for key, field := range a.Fields {
		fields[key] = map[string]any{
			"label":           field.Label,
			"value":           field.Value,
			"confidence":      field.Confidence,
			"excerpt":         field.Excerpt,
			"evidenceExcerpt": field.EvidenceExcerpt,
			"pageNumber":      field.PageNumber,
			"reviewRequired":  field.ReviewRequired,
			"conflict":        field.Conflict,
			"source":          field.Source,
		}
	}

	conflicts := make([]map[string]any, 0, len(a.Conflicts))
	for _, conflict := range a.Conflicts {
		conflicts = append(conflicts, map[string]any{
			"key":             conflict.Key,
			"label":           conflict.Label,
			"values":          conflict.Values,
			"pageNumbers":     conflict.PageNumbers,
			"evidenceExcerpt": conflict.EvidenceExcerpt,
			"source":          conflict.Source,
		})
	}

	stops := make([]map[string]any, 0, len(a.Stops))
	for _, stop := range a.Stops {
		stops = append(stops, map[string]any{
			"sequence":            stop.Sequence,
			"role":                stop.Role,
			"name":                stop.Name,
			"addressLine1":        stop.AddressLine1,
			"addressLine2":        stop.AddressLine2,
			"city":                stop.City,
			"state":               stop.State,
			"postalCode":          stop.PostalCode,
			"date":                stop.Date,
			"timeWindow":          stop.TimeWindow,
			"appointmentRequired": stop.AppointmentRequired,
			"pageNumber":          stop.PageNumber,
			"evidenceExcerpt":     stop.EvidenceExcerpt,
			"confidence":          stop.Confidence,
			"reviewRequired":      stop.ReviewRequired,
			"source":              stop.Source,
		})
	}

	return map[string]any{
		"kind":                 a.Kind,
		"overallConfidence":    a.OverallConfidence,
		"reviewStatus":         a.ReviewStatus,
		"missingFields":        a.MissingFields,
		"signals":              a.Signals,
		"classifierSource":     a.ClassifierSource,
		"providerFingerprint":  a.ProviderFingerprint,
		"classificationReason": a.ClassificationReason,
		"parsingRuleMetadata":  a.ParsingRuleMetadata,
		"conflicts":            conflicts,
		"fields":               fields,
		"stops":                stops,
		"rawExcerpt":           a.RawExcerpt,
	}
}

func classifyDocumentWithControl(
	name, text string,
	pages []pageExtractionResult,
	control *tenant.DocumentControl,
	features documentFeatureSet,
	fingerprint *providerFingerprint,
) classificationResult {
	if control == nil || !control.EnableAutoClassification {
		return classificationResult{
			Kind:           "Other",
			Confidence:     0,
			Signals:        []string{"auto classification disabled"},
			ReviewRequired: true,
			Source:         "disabled",
			Reason:         "automatic classification disabled by document controls",
		}
	}
	return classifyDocumentWithFeatures(name, text, pages, features, fingerprint)
}

func classifyDocument(name, text string) classificationResult {
	return classifyDocumentWithFeatures(
		name,
		text,
		nil,
		extractDocumentFeatures(name, nil, text),
		detectProviderFingerprint(name, text, extractDocumentFeatures(name, nil, text)),
	)
}

func classifyDocumentWithFeatures(
	name, text string,
	_ []pageExtractionResult,
	features documentFeatureSet,
	fingerprint *providerFingerprint,
) classificationResult {
	corpus := strings.ToLower(name + "\n" + text)

	candidates := []classificationResult{
		scoreRateConfirmation(corpus, features, fingerprint),
		scoreBillOfLading(corpus, features, fingerprint),
		scoreProofOfDelivery(corpus, features, fingerprint),
	}

	best := classificationResult{
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
		return classificationResult{
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
	features documentFeatureSet,
	fingerprint *providerFingerprint,
) classificationResult {
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
	if containsAny(corpus, "line haul", "flat rate", "fuel surcharge", "quick pay", "cash advance") {
		score += 0.15
		signals = append(signals, "carrier rate terms")
	}
	if containsAny(corpus, "service for load #", "load #", "carrier load number") {
		score += 0.1
		signals = append(signals, "load number")
	}
	if containsAny(corpus, "load confirmation is subject to the terms", "this load confirmation is") {
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
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, "RateConfirmation") {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return classificationResult{
		Kind:                "RateConfirmation",
		Confidence:          score,
		Signals:             dedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: providerName(fingerprint),
		Reason:              "rate and load-confirmation evidence detected",
	}
}

func scoreBillOfLading(
	corpus string,
	features documentFeatureSet,
	fingerprint *providerFingerprint,
) classificationResult {
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
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, "BillOfLading") {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return classificationResult{
		Kind:                "BillOfLading",
		Confidence:          score,
		Signals:             dedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: providerName(fingerprint),
		Reason:              "bill-of-lading shipping evidence detected",
	}
}

func scoreProofOfDelivery(
	corpus string,
	features documentFeatureSet,
	fingerprint *providerFingerprint,
) classificationResult {
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
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, "ProofOfDelivery") {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return classificationResult{
		Kind:                "ProofOfDelivery",
		Confidence:          score,
		Signals:             dedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: providerName(fingerprint),
		Reason:              "delivery completion evidence detected",
	}
}

func scoreInvoice(
	corpus string,
	features documentFeatureSet,
	fingerprint *providerFingerprint,
) classificationResult {
	score := 0.0
	signals := make([]string, 0, 4)

	if containsAny(corpus, "\ninvoice", "invoice #", "invoice number") {
		score += 0.6
		signals = append(signals, "invoice phrase")
	}
	if containsAny(corpus, "amount due", "total due", "balance due") {
		score += 0.2
		signals = append(signals, "amount due")
	}
	if containsAny(corpus, "invoice date", "due date") {
		score += 0.1
		signals = append(signals, "invoice dates")
	}
	if len(features.MoneySignals) > 0 {
		score += 0.05
		signals = append(signals, "money signals")
	}
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, "Invoice") {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return classificationResult{
		Kind:                "Invoice",
		Confidence:          score,
		Signals:             dedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: providerName(fingerprint),
		Reason:              "invoice and billing evidence detected",
	}
}

func buildStructuredData(intelligence documentIntelligenceAnalysis) map[string]any {
	data := map[string]any{
		"schemaVersion": 5,
		"intelligence":  intelligence.ToMap(),
	}
	return data
}

func extractDocumentFeatures(
	name string,
	pages []pageExtractionResult,
	text string,
) documentFeatureSet {
	corpus := strings.ToLower(name + "\n" + text)
	lines := splitNormalizedLines(text)
	features := documentFeatureSet{
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
			recordLineFeatures(&features, line)
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

func recordLineFeatures(features *documentFeatureSet, line string) {
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
	features documentFeatureSet,
) *providerFingerprint {
	corpus := strings.ToLower(name + "\n" + text)
	registry := []providerFingerprint{
		{
			Provider:   "CHRobinson",
			KindHint:   "RateConfirmation",
			Confidence: 0.95,
			Signals:    []string{"ch robinson fingerprint", "carrier load confirmation format"},
		},
		{
			Provider:   "TQL",
			KindHint:   "RateConfirmation",
			Confidence: 0.9,
			Signals:    []string{"tql fingerprint"},
		},
		{
			Provider:   "Echo",
			KindHint:   "RateConfirmation",
			Confidence: 0.9,
			Signals:    []string{"echo fingerprint"},
		},
		{
			Provider:   "UberFreight",
			KindHint:   "RateConfirmation",
			Confidence: 0.9,
			Signals:    []string{"uber freight fingerprint"},
		},
	}

	for _, candidate := range registry {
		switch candidate.Provider {
		case "CHRobinson":
			if containsAny(corpus, "c.h. robinson", "ch robinson", "navispherecarrier", "carrier load confirmation", "contract addendum and carrier load confirmation") {
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
		return &providerFingerprint{
			Provider:   "GenericBrokerLoadConfirmation",
			KindHint:   "RateConfirmation",
			Confidence: 0.7,
			Signals:    []string{"generic broker load confirmation fingerprint"},
		}
	}

	return nil
}

func providerName(fingerprint *providerFingerprint) string {
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

func (a *Activities) enrichWithAI(
	ctx context.Context,
	payload *ProcessDocumentIntelligencePayload,
	doc *document.Document,
	control *tenant.DocumentControl,
	extracted *extractionResult,
	features documentFeatureSet,
	fingerprint *providerFingerprint,
	classification classificationResult,
	intelligence documentIntelligenceAnalysis,
) (classificationResult, documentIntelligenceAnalysis) {
	if extracted == nil {
		return classification, intelligence
	}
	if !a.cfg.AIEnabled() || !control.EnableAIAssistedClassification {
		return classification, intelligence
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  doc.OrganizationID,
		BuID:   doc.BusinessUnitID,
		UserID: payload.UserID,
	}
	pages := make([]services.AIDocumentPage, 0, len(extracted.Pages))
	for _, page := range extracted.Pages {
		if strings.TrimSpace(page.Text) == "" {
			continue
		}
		pages = append(pages, services.AIDocumentPage{
			PageNumber: page.PageNumber,
			Text:       truncateExtractedText(page.Text, a.cfg.GetAIMaxInputChars()/max(len(extracted.Pages), 1)),
		})
	}

	route, err := a.aiDocumentService.RouteDocument(ctx, &services.AIRouteRequest{
		TenantInfo: tenantInfo,
		DocumentID: doc.ID,
		FileName:   doc.OriginalName,
		Text:       truncateExtractedText(extracted.Text, a.cfg.GetAIMaxInputChars()),
		Pages:      pages,
		Features: &services.AIDocumentFeatureSet{
			TitleCandidates:  features.TitleCandidates,
			SectionLabels:    features.SectionLabels,
			PartyLabels:      features.PartyLabels,
			ReferenceLabels:  features.ReferenceLabels,
			MoneySignals:     features.MoneySignals,
			StopSignals:      features.StopSignals,
			TermsSignals:     features.TermsSignals,
			SignatureSignals: features.SignatureSignals,
		},
		Fingerprint: toAIFingerprintHint(fingerprint),
	})
	if err != nil {
		a.logger.Warn("ai route failed", zap.String("documentId", doc.ID.String()), zap.Error(err))
		return classification, intelligence
	}
	if route == nil || strings.TrimSpace(route.DocumentKind) == "" {
		return classification, intelligence
	}
	routedClassification := classification
	routedClassification.Kind = normalizeRoutedKind(route.DocumentKind)
	routedClassification.Confidence = route.Confidence
	routedClassification.Signals = dedupeStrings(append(classification.Signals, route.Signals...))
	routedClassification.ReviewRequired = normalizeAIReviewStatus(route.ReviewStatus) != "Ready"
	routedClassification.Source = normalizeClassifierSource(route.ClassifierSource)
	routedClassification.ProviderFingerprint = firstNonEmpty(route.ProviderFingerprint, providerName(fingerprint))
	routedClassification.Reason = firstNonEmpty(route.Reason, classification.Reason)
	routedAnalysis := analyzeDocument(routedClassification, extracted)

	if !strings.EqualFold(routedClassification.Kind, "RateConfirmation") || !control.EnableAIAssistedExtraction || !route.ShouldExtract {
		return routedClassification, routedAnalysis
	}

	aiExtract, err := a.aiDocumentService.ExtractRateConfirmation(ctx, &services.AIExtractRequest{
		TenantInfo: tenantInfo,
		DocumentID: doc.ID,
		FileName:   doc.OriginalName,
		Text:       truncateExtractedText(extracted.Text, a.cfg.GetAIMaxInputChars()),
		Pages:      pages,
	})
	if err != nil {
		a.logger.Warn("ai extraction failed", zap.String("documentId", doc.ID.String()), zap.Error(err))
		return classification, intelligence
	}
	if aiExtract == nil {
		return routedClassification, routedAnalysis
	}

	merged, ok := mergeAIAnalysis(routedAnalysis, aiExtract)
	if !ok {
		return routedClassification, routedAnalysis
	}

	routedClassification.Kind = merged.Kind
	routedClassification.Confidence = merged.OverallConfidence
	routedClassification.Signals = dedupeStrings(append(routedClassification.Signals, route.Signals...))
	routedClassification.ReviewRequired = merged.ReviewStatus != "Ready"
	routedClassification.Source = "ai"
	routedClassification.ProviderFingerprint = firstNonEmpty(route.ProviderFingerprint, providerName(fingerprint))
	routedClassification.Reason = firstNonEmpty(route.Reason, routedClassification.Reason)

	return routedClassification, merged
}

func mergeAIAnalysis(
	fallback documentIntelligenceAnalysis,
	aiExtract *services.AIExtractResult,
) (documentIntelligenceAnalysis, bool) {
	if aiExtract == nil || !strings.EqualFold(aiExtract.DocumentKind, "RateConfirmation") {
		return fallback, false
	}
	if !validateAIExtract(aiExtract) {
		return fallback, false
	}

	merged := fallback
	merged.Kind = "RateConfirmation"
	merged.OverallConfidence = clampConfidence(aiExtract.OverallConfidence)
	merged.ReviewStatus = normalizeAIReviewStatus(aiExtract.ReviewStatus)
	merged.MissingFields = dedupeStrings(aiExtract.MissingFields)
	merged.Signals = dedupeStrings(aiExtract.Signals)
	merged.Fields = make(map[string]reviewField, len(aiExtract.Fields))
	for key, field := range aiExtract.Fields {
		merged.Fields[key] = reviewField{
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

	merged.Stops = make([]intelligenceStop, 0, len(aiExtract.Stops))
	for _, stop := range aiExtract.Stops {
		merged.Stops = append(merged.Stops, intelligenceStop{
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

	merged.Conflicts = make([]reviewConflict, 0, len(aiExtract.Conflicts))
	for _, conflict := range aiExtract.Conflicts {
		merged.Conflicts = append(merged.Conflicts, reviewConflict{
			Key:             conflict.Key,
			Label:           conflict.Label,
			Values:          conflict.Values,
			PageNumbers:     conflict.PageNumbers,
			EvidenceExcerpt: conflict.EvidenceExcerpt,
			Source:          normalizeAISource(conflict.Source),
		})
	}

	return merged, true
}

func validateAIExtract(result *services.AIExtractResult) bool {
	if result == nil || !strings.EqualFold(result.DocumentKind, "RateConfirmation") {
		return false
	}
	requiredFields := []string{"shipper", "consignee", "rate"}
	for _, key := range requiredFields {
		field, ok := result.Fields[key]
		if !ok || strings.TrimSpace(field.Value) == "" || field.PageNumber <= 0 {
			return false
		}
	}

	hasPickup := false
	hasDelivery := false
	for _, stop := range result.Stops {
		if stop.PageNumber <= 0 || strings.TrimSpace(stop.EvidenceExcerpt) == "" {
			return false
		}
		switch strings.ToLower(strings.TrimSpace(stop.Role)) {
		case "pickup":
			hasPickup = true
		case "delivery":
			hasDelivery = true
		}
	}
	return hasPickup && hasDelivery
}

func isPotentialRateConfirmation(name, text, kind string) bool {
	if strings.EqualFold(kind, "RateConfirmation") {
		return true
	}
	corpus := strings.ToLower(name + "\n" + text)
	return containsAny(corpus, "rate confirmation", "load confirmation", "rate con", "ratecon", "load tender")
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
		return "Ready"
	case "unavailable":
		return "Unavailable"
	default:
		return "NeedsReview"
	}
}

func normalizeRoutedKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "rateconfirmation", "rate_confirmation":
		return "RateConfirmation"
	case "billoflading", "bill_of_lading":
		return "BillOfLading"
	case "proofofdelivery", "proof_of_delivery":
		return "ProofOfDelivery"
	default:
		return "Other"
	}
}

func toAIFingerprintHint(fingerprint *providerFingerprint) *services.AIDocumentFingerprintHint {
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

type inferredDocumentType struct {
	Code           string
	Name           string
	Category       documenttype.DocumentCategory
	Classification documenttype.DocumentClassification
	Color          string
}

func inferDocumentType(kind string) (*inferredDocumentType, bool) {
	switch kind {
	case "RateConfirmation":
		return &inferredDocumentType{
			Code:           "RATECONF",
			Name:           "Rate Confirmation",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#0f766e",
		}, true
	case "BillOfLading":
		return &inferredDocumentType{
			Code:           "BOL",
			Name:           "Bill of Lading",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#f59e0b",
		}, true
	case "ProofOfDelivery":
		return &inferredDocumentType{
			Code:           "POD",
			Name:           "Proof of Delivery",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#8b5cf6",
		}, true
	case "Invoice":
		return &inferredDocumentType{
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
	case "RateConfirmation", "BillOfLading", "ProofOfDelivery", "Invoice":
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
		kind == "RateConfirmation"
}

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

var (
	rateRegex         = regexp.MustCompile(`(?im)^(?:rate|freight charge|line haul|amount due|total)\s*[:#-]\s*([$]?[0-9,]+(?:\.[0-9]{2})?)$`)
	referenceRegex    = regexp.MustCompile(`(?im)^(?:load|reference|order|confirmation|pro|bol|invoice)(?:\s+(?:number|#))?\s*[:#-]\s*([A-Za-z0-9-]+)$`)
	bolReferenceRegex = regexp.MustCompile(`(?im)^(?:bill of lading|bol|b\/l|pro|reference|order|load)(?:\s+(?:number|#|no\.?))\s*[:#-]?\s*([A-Za-z0-9-]+)$`)
	podReferenceRegex = regexp.MustCompile(`(?im)^(?:pod|pro|reference|order|load)(?:\s+(?:number|#|no\.?))\s*[:#-]?\s*([A-Za-z0-9-]+)$`)
	pickupRegex       = regexp.MustCompile(`(?im)^(pickup|ship)(?:\s+(?:date|window|location|address))?\s*[:\-]\s*(.+)$`)
	deliveryRegex     = regexp.MustCompile(`(?im)^(delivery|drop)(?:\s+(?:date|window|location|address))?\s*[:\-]\s*(.+)$`)
	shipperRegex      = regexp.MustCompile(`(?im)^shipper(?:\s+name)?\s*[:\-]\s*(.+)$`)
	consigneeRegex    = regexp.MustCompile(`(?im)^(consignee|receiver)(?:\s+name)?\s*[:\-]\s*(.+)$`)
	commodityRegex    = regexp.MustCompile(`(?im)^(commodity|product|description)\s*[:\-]\s*(.+)$`)
	equipmentRegex    = regexp.MustCompile(`(?i)\b(van|reefer|flatbed|step deck|power only|hotshot|conestoga|dry van)\b`)
	weightRegex       = regexp.MustCompile(`(?i)([0-9,]+)\s*(lbs|pounds)`)
	pieceCountRegex   = regexp.MustCompile(`(?im)^(?:pieces?|piece count|total pieces|packages?|package count|units?|cartons?)\s*[:#-]?\s*([0-9,]+)\b`)
	instructionsRegex = regexp.MustCompile(`(?im)^(instructions|notes|special instructions)\s*[:\-]\s*(.+)$`)
	invoiceDateRegex  = regexp.MustCompile(`(?im)^invoice date\s*[:\-]\s*(.+)$`)
	dueDateRegex      = regexp.MustCompile(`(?im)^due date\s*[:\-]\s*(.+)$`)
	totalDueRegex     = regexp.MustCompile(`(?im)^(?:amount due|total due|balance due)\s*[:#-]\s*([$]?[0-9,]+(?:\.[0-9]{2})?)$`)
	signatureRegex    = regexp.MustCompile(`(?im)^(?:receiver|consignee) signature\s*[:\-]\s*(.+)$`)
	dateLabelRegex    = regexp.MustCompile(`(?i)\b(?:date|pickup|delivery)\b`)
	timeWindowRegex   = regexp.MustCompile(`(?i)\b([0-9]{1,2}[:][0-9]{2}\s*(?:am|pm)?\s*[-–]\s*[0-9]{1,2}[:][0-9]{2}\s*(?:am|pm)?)\b`)
	dateValueRegex    = regexp.MustCompile(`(?i)\b(?:\d{1,2}[/-]\d{1,2}(?:[/-]\d{2,4})?|(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\s+\d{1,2}(?:,\s*\d{4})?)\b`)
	addressLineRegex  = regexp.MustCompile(`(?i)\b\d{1,6}\s+[a-z0-9][a-z0-9.\-# ]+\b`)
	cityStateZipRegex = regexp.MustCompile(`(?i)\b([a-z .'-]+),\s*([a-z]{2})\s+(\d{5}(?:-\d{4})?)\b`)
)

func analyzeDocument(
	classification classificationResult,
	extracted *extractionResult,
) documentIntelligenceAnalysis {
	if extracted == nil {
		extracted = &extractionResult{}
	}
	text := ""
	text = extracted.Text
	analysis := documentIntelligenceAnalysis{
		Kind:                 classification.Kind,
		ReviewStatus:         "NeedsReview",
		MissingFields:        []string{},
		Signals:              append([]string{}, classification.Signals...),
		ClassifierSource:     classification.Source,
		ProviderFingerprint:  classification.ProviderFingerprint,
		ClassificationReason: classification.Reason,
		Conflicts:            []reviewConflict{},
		Fields:               make(map[string]reviewField),
		Stops:                []intelligenceStop{},
		RawExcerpt:           truncateExtractedText(strings.ReplaceAll(text, "\r", ""), 2000),
	}

	required := requiredFieldsForKind(classification.Kind)

	switch classification.Kind {
	case "RateConfirmation":
		analysis.Stops = extractRateConfirmationStops(extracted.Pages)
		addFieldFromPages(analysis.Fields, &analysis.Signals, "shipper", "Shipper", shipperRegex, extracted.Pages, 0.9)
		addFieldFromPages(analysis.Fields, &analysis.Signals, "consignee", "Consignee", consigneeRegex, extracted.Pages, 0.9)
		addStopTimingField(analysis.Fields, &analysis.Signals, "pickupWindow", "Pickup Window", firstStopByRole(analysis.Stops, "pickup"), 0.88)
		addStopTimingField(analysis.Fields, &analysis.Signals, "deliveryWindow", "Delivery Window", firstStopByRole(analysis.Stops, "delivery"), 0.88)
		addFieldFromPages(analysis.Fields, &analysis.Signals, "referenceNumber", "Reference Number", referenceRegex, extracted.Pages, 0.8)
		addFieldFromPages(analysis.Fields, &analysis.Signals, "commodity", "Commodity", commodityRegex, extracted.Pages, 0.78)
		addFieldFromPages(analysis.Fields, &analysis.Signals, "instructions", "Instructions", instructionsRegex, extracted.Pages, 0.72)
		addCurrencyFieldFromPages(analysis.Fields, &analysis.Signals, "rate", "Rate", rateRegex, extracted.Pages, 0.92, "rate amount")
		addRegexValueFieldFromPages(analysis.Fields, &analysis.Signals, "equipmentType", "Equipment Type", equipmentRegex, extracted.Pages, 0.82, "equipment type", false)
		addWeightFieldFromPages(analysis.Fields, &analysis.Signals, extracted.Pages, "weight")
		analysis.Conflicts = append(analysis.Conflicts, collectFieldConflicts(analysis.Fields)...)
		analysis.Conflicts = append(analysis.Conflicts, collectStopConflicts(analysis.Stops)...)
	case "BillOfLading":
		addFieldFromSectionLabels(analysis.Fields, &analysis.Signals, "shipper", "Shipper", extracted.Pages, []string{"ship from", "shipper", "shipper name", "shipper information"}, 0.93, "shipper", false, extractEntityNameFromSection)
		addFieldFromSectionLabels(analysis.Fields, &analysis.Signals, "consignee", "Consignee", extracted.Pages, []string{"ship to", "consignee", "receiver", "delivery to"}, 0.93, "consignee", false, extractEntityNameFromSection)
		addFieldFromSectionLabels(analysis.Fields, &analysis.Signals, "commodity", "Commodity", extracted.Pages, []string{"commodity", "description", "product", "articles"}, 0.86, "commodity", false, extractCommodityFromSection)
		addRegexValueFieldFromPages(analysis.Fields, &analysis.Signals, "referenceNumber", "BOL / Reference Number", bolReferenceRegex, extracted.Pages, 0.85, "reference number", false)
		addRegexValueFieldFromPages(analysis.Fields, &analysis.Signals, "pieceCount", "Pieces / Packages", pieceCountRegex, extracted.Pages, 0.76, "piece count", true)
		addWeightFieldFromPages(analysis.Fields, &analysis.Signals, extracted.Pages, "weight")
		analysis.Conflicts = append(analysis.Conflicts, collectFieldConflicts(analysis.Fields)...)
	case "ProofOfDelivery":
		addFieldFromSectionLabels(analysis.Fields, &analysis.Signals, "consignee", "Consignee", extracted.Pages, []string{"consignee", "receiver name", "delivery to", "delivered to", "received by"}, 0.91, "consignee", false, extractEntityNameFromSection)
		addFieldFromSectionLabels(analysis.Fields, &analysis.Signals, "deliveryWindow", "Delivery", extracted.Pages, []string{"delivery date", "delivered on", "received on", "date delivered"}, 0.89, "delivery", false, extractDeliveryFieldFromSection)
		if _, ok := analysis.Fields["deliveryWindow"]; !ok {
			addFieldFromPages(analysis.Fields, &analysis.Signals, "deliveryWindow", "Delivery", deliveryRegex, extracted.Pages, 0.86)
		}
		addFieldFromSectionLabels(analysis.Fields, &analysis.Signals, "signature", "Signature", extracted.Pages, []string{"receiver signature", "consignee signature", "signature", "received by", "signed by"}, 0.82, "signature", false, extractSignatureFromSection)
		addRegexValueFieldFromPages(analysis.Fields, &analysis.Signals, "referenceNumber", "Reference Number", podReferenceRegex, extracted.Pages, 0.82, "reference number", false)
		addFieldFromSectionLabels(analysis.Fields, &analysis.Signals, "receiptNotes", "Receipt Notes", extracted.Pages, []string{"remarks", "exceptions", "received in good order", "delivery status"}, 0.72, "receipt notes", true, extractFreeformSectionValue)
		analysis.Conflicts = append(analysis.Conflicts, collectFieldConflicts(analysis.Fields)...)
	case "Invoice":
		addFieldFromPages(analysis.Fields, &analysis.Signals, "referenceNumber", "Invoice Number", referenceRegex, extracted.Pages, 0.84)
		addFieldFromPages(analysis.Fields, &analysis.Signals, "invoiceDate", "Invoice Date", invoiceDateRegex, extracted.Pages, 0.88)
		addFieldFromPages(analysis.Fields, &analysis.Signals, "dueDate", "Due Date", dueDateRegex, extracted.Pages, 0.88)
		addFieldFromPages(analysis.Fields, &analysis.Signals, "shipper", "Bill To / Shipper", shipperRegex, extracted.Pages, 0.72)
		addCurrencyFieldFromPages(analysis.Fields, &analysis.Signals, "totalDue", "Total Due", totalDueRegex, extracted.Pages, 0.93, "total due")
	}

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
	if classification.Kind == "RateConfirmation" {
		if !hasStopRole(analysis.Stops, "pickup") {
			analysis.MissingFields = appendIfMissing(analysis.MissingFields, "Pickup Stop")
		}
		if !hasStopRole(analysis.Stops, "delivery") {
			analysis.MissingFields = appendIfMissing(analysis.MissingFields, "Delivery Stop")
		}
	}

	if len(analysis.Fields) == 0 {
		analysis.ReviewStatus = "Unavailable"
		return analysis
	}

	if classification.Kind == "RateConfirmation" {
		if analysis.OverallConfidence >= 0.82 &&
			len(analysis.MissingFields) == 0 &&
			len(analysis.Conflicts) == 0 &&
			hasStopRole(analysis.Stops, "pickup") &&
			hasStopRole(analysis.Stops, "delivery") &&
			!hasReviewRequiredStop(analysis.Stops) {
			analysis.ReviewStatus = "Ready"
		}
		return analysis
	}

	if classification.Kind == "BillOfLading" {
		if analysis.OverallConfidence >= 0.82 && len(analysis.MissingFields) == 0 && len(analysis.Conflicts) == 0 {
			analysis.ReviewStatus = "Ready"
		}
		return analysis
	}

	if classification.Kind == "ProofOfDelivery" {
		if analysis.OverallConfidence >= 0.82 &&
			len(analysis.MissingFields) == 0 &&
			len(analysis.Conflicts) == 0 &&
			!analysis.Fields["deliveryWindow"].ReviewRequired &&
			!analysis.Fields["signature"].ReviewRequired {
			analysis.ReviewStatus = "Ready"
		}
		return analysis
	}

	if analysis.OverallConfidence >= 0.82 && len(analysis.MissingFields) <= 1 && len(analysis.Conflicts) == 0 {
		analysis.ReviewStatus = "Ready"
	}

	return analysis
}

func requiredFieldsForKind(kind string) []struct {
	key   string
	label string
} {
	switch kind {
	case "RateConfirmation":
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
	case "BillOfLading":
		return []struct {
			key   string
			label string
		}{
			{key: "shipper", label: "Shipper"},
			{key: "consignee", label: "Consignee"},
			{key: "commodity", label: "Commodity"},
			{key: "referenceNumber", label: "Reference Number"},
		}
	case "ProofOfDelivery":
		return []struct {
			key   string
			label string
		}{
			{key: "consignee", label: "Consignee"},
			{key: "deliveryWindow", label: "Delivery"},
			{key: "signature", label: "Signature"},
		}
	case "Invoice":
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

func addFieldFromRegex(
	fields map[string]reviewField,
	signals *[]string,
	key, label string,
	re *regexp.Regexp,
	text string,
	confidence float64,
) {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[len(match)-1])
	if value == "" {
		return
	}

	fields[key] = reviewField{
		Label:          label,
		Value:          value,
		Confidence:     confidence,
		Excerpt:        strings.TrimSpace(match[0]),
		ReviewRequired: confidence < 0.8,
		Source:         "deterministic",
	}
	if signals != nil {
		*signals = append(*signals, strings.ToLower(label))
	}
}

func addFieldFromPages(
	fields map[string]reviewField,
	signals *[]string,
	key, label string,
	re *regexp.Regexp,
	pages []pageExtractionResult,
	confidence float64,
) {
	pageNumber, match := firstPageMatch(re, pages)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[len(match)-1])
	if value == "" {
		return
	}

	fields[key] = reviewField{
		Label:           label,
		Value:           value,
		Confidence:      pageAdjustedConfidence(confidence, pageNumber, pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  confidence < 0.8,
		Conflict:        hasConflictingMatches(re, value, pages),
		Source:          "deterministic",
	}
	if signals != nil {
		*signals = append(*signals, strings.ToLower(label))
	}
}

func addCurrencyField(
	fields map[string]reviewField,
	signals *[]string,
	key, label string,
	re *regexp.Regexp,
	text string,
	confidence float64,
	signal string,
) {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[1])
	if value == "" {
		return
	}

	fields[key] = reviewField{
		Label:          label,
		Value:          value,
		Confidence:     confidence,
		Excerpt:        strings.TrimSpace(match[0]),
		ReviewRequired: confidence < 0.8,
		Source:         "deterministic",
	}
	if signals != nil && signal != "" {
		*signals = append(*signals, signal)
	}
}

func addCurrencyFieldFromPages(
	fields map[string]reviewField,
	signals *[]string,
	key, label string,
	re *regexp.Regexp,
	pages []pageExtractionResult,
	confidence float64,
	signal string,
) {
	pageNumber, match := firstPageMatch(re, pages)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[1])
	if value == "" {
		return
	}

	fields[key] = reviewField{
		Label:           label,
		Value:           value,
		Confidence:      pageAdjustedConfidence(confidence, pageNumber, pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  confidence < 0.8,
		Conflict:        hasConflictingMatches(re, value, pages),
		Source:          "deterministic",
	}
	if signals != nil && signal != "" {
		*signals = append(*signals, signal)
	}
}

func addRegexValueField(
	fields map[string]reviewField,
	signals *[]string,
	key, label string,
	re *regexp.Regexp,
	text string,
	confidence float64,
	signal string,
	reviewRequired bool,
) {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[1])
	if value == "" {
		return
	}

	fields[key] = reviewField{
		Label:          label,
		Value:          value,
		Confidence:     confidence,
		Excerpt:        strings.TrimSpace(match[0]),
		ReviewRequired: reviewRequired,
		Source:         "deterministic",
	}
	if signals != nil && signal != "" {
		*signals = append(*signals, signal)
	}
}

func addRegexValueFieldFromPages(
	fields map[string]reviewField,
	signals *[]string,
	key, label string,
	re *regexp.Regexp,
	pages []pageExtractionResult,
	confidence float64,
	signal string,
	reviewRequired bool,
) {
	pageNumber, match := firstPageMatch(re, pages)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[1])
	if value == "" {
		return
	}

	fields[key] = reviewField{
		Label:           label,
		Value:           value,
		Confidence:      pageAdjustedConfidence(confidence, pageNumber, pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  reviewRequired,
		Conflict:        hasConflictingMatches(re, value, pages),
		Source:          "deterministic",
	}
	if signals != nil && signal != "" {
		*signals = append(*signals, signal)
	}
}

func addWeightField(
	fields map[string]reviewField,
	signals *[]string,
	text string,
	signal string,
) {
	match := weightRegex.FindStringSubmatch(text)
	if len(match) < 2 {
		return
	}

	fields["weight"] = reviewField{
		Label:          "Weight",
		Value:          strings.TrimSpace(match[1]) + " lbs",
		Confidence:     0.8,
		Excerpt:        strings.TrimSpace(match[0]),
		ReviewRequired: false,
		Source:         "deterministic",
	}
	if signals != nil && signal != "" {
		*signals = append(*signals, signal)
	}
}

func addWeightFieldFromPages(
	fields map[string]reviewField,
	signals *[]string,
	pages []pageExtractionResult,
	signal string,
) {
	pageNumber, match := firstPageMatch(weightRegex, pages)
	if len(match) < 2 {
		return
	}

	fields["weight"] = reviewField{
		Label:           "Weight",
		Value:           strings.TrimSpace(match[1]) + " lbs",
		Confidence:      pageAdjustedConfidence(0.8, pageNumber, pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  false,
		Source:          "deterministic",
	}
	if signals != nil && signal != "" {
		*signals = append(*signals, signal)
	}
}

func addStopTimingField(
	fields map[string]reviewField,
	signals *[]string,
	key, label string,
	stop *intelligenceStop,
	confidence float64,
) {
	if stop == nil {
		return
	}

	value := strings.TrimSpace(strings.Join(filterNonEmpty(stop.Date, stop.TimeWindow), " "))
	if value == "" {
		return
	}

	fields[key] = reviewField{
		Label:           label,
		Value:           value,
		Confidence:      clampConfidence((confidence + stop.Confidence) / 2),
		Excerpt:         stop.EvidenceExcerpt,
		EvidenceExcerpt: stop.EvidenceExcerpt,
		PageNumber:      stop.PageNumber,
		ReviewRequired:  stop.ReviewRequired,
		Source:          stop.Source,
	}
	if signals != nil {
		*signals = append(*signals, strings.ToLower(label))
	}
}

type pageSectionMatch struct {
	PageNumber int
	Value      string
	Excerpt    string
}

func addFieldFromSectionLabels(
	fields map[string]reviewField,
	signals *[]string,
	key, label string,
	pages []pageExtractionResult,
	labels []string,
	confidence float64,
	signal string,
	reviewRequired bool,
	extractor func(string, []string) string,
) {
	matches := findSectionMatches(pages, labels, extractor)
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

	fields[key] = reviewField{
		Label:           label,
		Value:           selected.Value,
		Confidence:      pageAdjustedConfidence(confidence, selected.PageNumber, pages),
		Excerpt:         selected.Excerpt,
		EvidenceExcerpt: selected.Excerpt,
		PageNumber:      selected.PageNumber,
		ReviewRequired:  reviewRequired || conflict || normalizeSectionValue(selected.Value) == "",
		Conflict:        conflict,
		Source:          "deterministic",
	}
	if signals != nil && signal != "" {
		*signals = append(*signals, signal)
	}
}

func findSectionMatches(
	pages []pageExtractionResult,
	labels []string,
	extractor func(string, []string) string,
) []pageSectionMatch {
	matches := make([]pageSectionMatch, 0)
	for _, page := range pages {
		lines := splitNormalizedLines(page.Text)
		for idx, line := range lines {
			if !matchesSectionLabel(line, labels) {
				continue
			}
			block := collectSectionBlock(lines, idx)
			value := strings.TrimSpace(extractor(line, block))
			if value == "" {
				continue
			}
			matches = append(matches, pageSectionMatch{
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
	boundaries := []string{
		"ship from", "ship to", "shipper", "consignee", "receiver", "delivery to", "delivered to",
		"pickup", "delivery", "received by", "signed by", "receiver signature", "consignee signature",
		"bill of lading", "bol", "reference", "load", "pro", "commodity", "description", "weight",
		"pieces", "packages", "remarks", "exceptions", "carrier", "invoice", "bill to",
	}
	for _, boundary := range boundaries {
		if normalized == boundary || strings.HasPrefix(normalized, boundary+" ") {
			return true
		}
	}
	return false
}

func extractEntityNameFromSection(header string, block []string) string {
	if value := extractSectionHeaderValue(header); value != "" && !looksLikeAddress(value) && !cityStateZipRegex.MatchString(value) {
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
	if value := extractSectionHeaderValue(header); value != "" && !dateValueRegex.MatchString(value) {
		return value
	}
	for _, line := range block[1:] {
		if line == "" || dateValueRegex.MatchString(line) || strings.Contains(strings.ToLower(line), "date") {
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

func dedupeSectionMatches(matches []pageSectionMatch) []pageSectionMatch {
	deduped := make([]pageSectionMatch, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		key := normalizeSectionValue(match.Value) + "|" + strconv.Itoa(match.PageNumber)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, match)
	}
	return deduped
}

func extractRateConfirmationStops(pages []pageExtractionResult) []intelligenceStop {
	stops := make([]intelligenceStop, 0, 2)

	for _, page := range pages {
		lines := splitNormalizedLines(page.Text)
		for idx, line := range lines {
			role, ok := detectStopRole(line)
			if !ok {
				continue
			}

			stop := intelligenceStop{
				Sequence:        len(stops) + 1,
				Role:            role,
				PageNumber:      page.PageNumber,
				EvidenceExcerpt: collectStopExcerpt(lines, idx),
				Confidence:      baseStopConfidence(page),
				ReviewRequired:  false,
				Source:          "deterministic",
			}

			if labelValue := extractLabelValue(line); labelValue != "" {
				switch {
				case looksLikeAddress(labelValue):
					stop.AddressLine1 = labelValue
				case dateLabelRegex.MatchString(line) && stop.Date == "":
					stop.Date = firstRegexValue(dateValueRegex, labelValue)
					stop.TimeWindow = firstRegexValue(timeWindowRegex, labelValue)
				default:
					stop.Name = labelValue
				}
			}

			for _, candidate := range lines[idx+1:] {
				if candidate == "" {
					break
				}
				if _, nextStop := detectStopRole(candidate); nextStop {
					break
				}

				switch {
				case stop.Name == "" && !looksLikeAddress(candidate) && !cityStateZipRegex.MatchString(candidate) && !dateLabelRegex.MatchString(candidate):
					stop.Name = strings.TrimSpace(candidate)
				case stop.AddressLine1 == "" && looksLikeAddress(candidate):
					stop.AddressLine1 = strings.TrimSpace(candidate)
				case stop.City == "" && cityStateZipRegex.MatchString(candidate):
					city, state, postalCode := extractCityStateZip(candidate)
					stop.City = city
					stop.State = state
					stop.PostalCode = postalCode
				case stop.Date == "" && dateLabelRegex.MatchString(candidate):
					stop.Date = firstRegexValue(dateValueRegex, candidate)
					stop.TimeWindow = firstRegexValue(timeWindowRegex, candidate)
				case stop.TimeWindow == "":
					stop.TimeWindow = firstRegexValue(timeWindowRegex, candidate)
				case stop.AddressLine2 == "" && looksLikeAddress(candidate) && strings.TrimSpace(candidate) != stop.AddressLine1:
					stop.AddressLine2 = strings.TrimSpace(candidate)
				}
			}

			if stop.Date == "" {
				stop.Date = firstRegexValue(dateValueRegex, stop.EvidenceExcerpt)
			}
			if stop.TimeWindow == "" {
				stop.TimeWindow = firstRegexValue(timeWindowRegex, stop.EvidenceExcerpt)
			}
			stop.AppointmentRequired = strings.Contains(strings.ToLower(stop.EvidenceExcerpt), "appointment")

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

func collectFieldConflicts(fields map[string]reviewField) []reviewConflict {
	conflicts := make([]reviewConflict, 0)
	for key, field := range fields {
		if !field.Conflict {
			continue
		}
		conflicts = append(conflicts, reviewConflict{
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

func collectStopConflicts(stops []intelligenceStop) []reviewConflict {
	conflicts := make([]reviewConflict, 0)

	for _, role := range []string{"pickup", "delivery"} {
		addresses := make(map[string][]intelligenceStop)
		dates := make(map[string][]intelligenceStop)
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
			conflicts = append(conflicts, reviewConflict{
				Key:             role + "Address",
				Label:           roleLabel(role) + " Address",
				Values:          mapKeys(addresses),
				PageNumbers:     stopPages(stops, role),
				EvidenceExcerpt: firstStopExcerpt(stops, role),
				Source:          "deterministic",
			})
		}
		if len(dates) > 1 {
			conflicts = append(conflicts, reviewConflict{
				Key:             role + "Date",
				Label:           roleLabel(role) + " Date",
				Values:          mapKeys(dates),
				PageNumbers:     stopPages(stops, role),
				EvidenceExcerpt: firstStopExcerpt(stops, role),
				Source:          "deterministic",
			})
		}
	}

	return conflicts
}

func hasStopRole(stops []intelligenceStop, role string) bool {
	for _, stop := range stops {
		if stop.Role == role {
			return true
		}
	}
	return false
}

func hasReviewRequiredStop(stops []intelligenceStop) bool {
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
	switch {
	case strings.HasPrefix(lower, "pickup"):
		if strings.HasPrefix(lower, "pickup date") || strings.HasPrefix(lower, "pickup window") {
			return "", false
		}
		return "pickup", true
	case strings.HasPrefix(lower, "delivery"), strings.HasPrefix(lower, "drop"):
		if strings.HasPrefix(lower, "delivery date") || strings.HasPrefix(lower, "delivery window") {
			return "", false
		}
		return "delivery", true
	default:
		return "", false
	}
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

func extractCityStateZip(line string) (string, string, string) {
	match := cityStateZipRegex.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 4 {
		return "", "", ""
	}
	return strings.TrimSpace(match[1]), strings.ToUpper(strings.TrimSpace(match[2])), strings.TrimSpace(match[3])
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

func baseStopConfidence(page pageExtractionResult) float64 {
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

func stopPages(stops []intelligenceStop, role string) []int {
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

func firstStopExcerpt(stops []intelligenceStop, role string) string {
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
	case "pickup":
		return "Pickup"
	case "delivery":
		return "Delivery"
	default:
		return strings.ToUpper(role[:1]) + role[1:]
	}
}

func firstStopByRole(stops []intelligenceStop, role string) *intelligenceStop {
	for idx := range stops {
		if stops[idx].Role == role {
			return &stops[idx]
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

func buildContentPages(content *documentcontent.Content, pages []pageExtractionResult) []*documentcontent.Page {
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

func finalizeExtraction(pages []pageExtractionResult, maxExtractedChars int) *extractionResult {
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
		default:
			nativeCount++
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
			pages[idx].Metadata["documentAverageConfidence"] = clampConfidence(weightedConfidence / weightedPages)
		}
	}

	return &extractionResult{
		Text:       truncateExtractedText(strings.Join(textParts, "\n\n"), maxExtractedChars),
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

func firstPageMatch(re *regexp.Regexp, pages []pageExtractionResult) (int, []string) {
	for _, page := range pages {
		match := re.FindStringSubmatch(strings.ReplaceAll(page.Text, "\r", ""))
		if len(match) > 0 {
			return page.PageNumber, match
		}
	}
	return 0, nil
}

func hasConflictingMatches(re *regexp.Regexp, selected string, pages []pageExtractionResult) bool {
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

func pageAdjustedConfidence(base float64, pageNumber int, pages []pageExtractionResult) float64 {
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

func readImageDimensions(imageData []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func (a *Activities) preprocessOCRImage(imageData []byte) ([]byte, int, int, error) {
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

	switch strings.ToLower(a.cfg.GetOCRPreprocessingMode()) {
	case "standard":
		processed = imaging.Grayscale(processed)
		processed = imaging.AdjustContrast(processed, 25)
		processed = imaging.Sharpen(processed, 1.5)
		processed = thresholdImage(processed, 170)
	}

	buf := new(bytes.Buffer)
	if err := imaging.Encode(buf, processed, imaging.PNG); err != nil {
		return nil, 0, 0, err
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
			luma := uint8(((299 * r) + (587 * g) + (114 * b) + 500) / 1000 >> 8)
			if luma >= threshold {
				dst.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(a >> 8)})
				continue
			}
			dst.SetNRGBA(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: uint8(a >> 8)})
		}
	}

	return dst
}

func parseTesseractTSV(output string) (string, float64, error) {
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

	joined := strings.TrimSpace(strings.Join(lineTexts, "\n"))
	confidence := 0.0
	if confidenceCount > 0 {
		confidence = clampConfidence(totalConfidence / confidenceCount)
	}

	return joined, confidence, nil
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
