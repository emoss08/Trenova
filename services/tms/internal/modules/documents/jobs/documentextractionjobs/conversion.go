package documentextractionjobs

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentextractioncontract"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentintelligencecontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

const (
	conversionPollInterval = 2 * time.Second
	maxConversionResubmits = 3

	reasonNoModelProvider       = "no_model_provider"
	reasonConverterUnconfigured = "converter_not_configured"

	extractionUnavailableMessage = "Document extraction needs an AI model provider. " +
		"Add one in AI settings, then extract the document again."
	converterUnavailableMessage = "Document extraction is not set up on this server: " +
		"the document converter is not configured."
)

type ExtractionRoute string

const (
	ExtractionRouteSkip    ExtractionRoute = "skip"
	ExtractionRouteLegacy  ExtractionRoute = "legacy"
	ExtractionRouteDocling ExtractionRoute = "docling"
	ExtractionRouteStopped ExtractionRoute = "stopped"
)

type PlanDocumentExtractionResult struct {
	Route                    ExtractionRoute                    `json:"route"`
	OCR                      bool                               `json:"ocr"`
	ConversionTimeoutSeconds int64                              `json:"conversionTimeoutSeconds"`
	Result                   *ProcessDocumentIntelligenceResult `json:"result,omitempty"`
}

type ConvertDocumentInput struct {
	temporaltype.BasePayload

	DocumentID pulid.ID `json:"documentId"`
	OCR        bool     `json:"ocr"`
}

type ConvertDocumentResult struct {
	DocumentID pulid.ID `json:"documentId"`
	ContentID  pulid.ID `json:"contentId"`
	PageCount  int      `json:"pageCount"`
	Converted  bool     `json:"converted"`
	Status     string   `json:"status"`
}

type AnalyzeConvertedDocumentInput struct {
	temporaltype.BasePayload

	DocumentID pulid.ID `json:"documentId"`
}

type conversionHeartbeat struct {
	TaskID string `json:"taskId"`
}

type conversionFormat int

const (
	formatUnsupported conversionFormat = iota
	formatLegacyOffice
	formatPlainText
	formatPDF
	formatImage
	formatOffice
	formatEmail
)

var legacyOfficeContentTypes = map[string]struct{}{
	"application/msword":            {},
	"application/vnd.ms-excel":      {},
	"application/vnd.ms-powerpoint": {},
	"application/rtf":               {},
	"application/x-rtf":             {},
	"text/rtf":                      {},
	"text/richtext":                 {},
}

var convertibleImageTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/jpg":  {},
	"image/tiff": {},
	"image/bmp":  {},
	"image/webp": {},
}

func conversionFormatOf(contentType, fileName string) conversionFormat {
	mediaType, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(contentType)), ";")
	mediaType = strings.TrimSpace(mediaType)
	ext := strings.ToLower(filepath.Ext(fileName))

	if _, legacy := legacyOfficeContentTypes[mediaType]; legacy {
		return formatLegacyOffice
	}
	switch ext {
	case ".doc", ".xls", ".ppt", ".rtf":
		return formatLegacyOffice
	}

	switch {
	case isPDFType(mediaType, ext):
		return formatPDF
	case mediaType == "message/rfc822" || ext == ".eml":
		return formatEmail
	case isPlainTextType(mediaType, ext):
		return formatPlainText
	}
	if _, ok := convertibleImageTypes[mediaType]; ok {
		return formatImage
	}
	if format, ok := kernelports.OfficeFormatOf(mediaType, fileName); ok &&
		format != kernelports.OfficeFormatEPUB {
		return formatOffice
	}

	return formatUnsupported
}

func (a *Activities) PlanDocumentExtractionActivity(
	ctx context.Context,
	payload *documentextractioncontract.ProcessDocumentExtractionPayload,
) (*PlanDocumentExtractionResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	doc, err := a.loadDocument(ctx, payload.DocumentID, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !doc.ProcessingProfile.SupportsIntelligence() {
		a.metrics.Document.RecordExtraction("skipped", "", "processing_profile_disabled")
		return stoppedPlan(ExtractionRouteSkip, doc), nil
	}

	control, err := a.getDocumentControl(ctx, doc.OrganizationID, doc.BusinessUnitID)
	if err != nil {
		return nil, err
	}
	if !control.EnableDocumentIntelligence {
		a.metrics.Document.RecordExtraction("skipped", "", "control_disabled")
		return stoppedPlan(ExtractionRouteSkip, doc), nil
	}
	if !control.UsesDocling() {
		return &PlanDocumentExtractionResult{Route: ExtractionRouteLegacy}, nil
	}

	return a.planDocling(ctx, doc, control, tenantInfo)
}

func (a *Activities) planDocling(
	ctx context.Context,
	doc *document.Document,
	control *tenant.DocumentControl,
	tenantInfo pagination.TenantInfo,
) (*PlanDocumentExtractionResult, error) {
	available, err := a.aiDocumentService.ExtractionAvailable(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !available {
		if err = a.markExtractionUnavailable(ctx, doc, &extractionUnavailable{
			code:    documentupload.FailureExtractionUnavailable,
			reason:  reasonNoModelProvider,
			message: extractionUnavailableMessage,
		}); err != nil {
			return nil, err
		}
		return stoppedPlan(ExtractionRouteStopped, doc), nil
	}

	format := conversionFormatOf(doc.FileType, doc.OriginalName)
	switch format {
	case formatLegacyOffice, formatUnsupported:
		content := &documentcontent.Content{
			DocumentID:     doc.ID,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			Converter:      documentcontent.ConverterDocling,
		}
		if err = a.markFailed(
			ctx,
			doc,
			content,
			documentupload.FailureUnsupportedFormat.String(),
			unsupportedFormatMessage(format, doc.FileType),
		); err != nil {
			return nil, err
		}
		return stoppedPlan(ExtractionRouteStopped, doc), nil
	case formatPlainText:
	case formatPDF, formatImage, formatOffice, formatEmail:
		if a.converter == nil || !a.converter.Enabled() {
			if err = a.markExtractionUnavailable(ctx, doc, &extractionUnavailable{
				code:    documentupload.FailureConverterUnavailable,
				reason:  reasonConverterUnconfigured,
				message: converterUnavailableMessage,
			}); err != nil {
				return nil, err
			}
			return stoppedPlan(ExtractionRouteStopped, doc), nil
		}
	}

	if _, _, err = a.prepareExtractionContentAndStart(ctx, doc, tenantInfo); err != nil {
		return nil, err
	}

	return &PlanDocumentExtractionResult{
		Route:                    ExtractionRouteDocling,
		OCR:                      control.EnableOCR,
		ConversionTimeoutSeconds: int64(a.conversionCfg.GetConversionTimeout().Seconds()),
	}, nil
}

func unsupportedFormatMessage(format conversionFormat, fileType string) string {
	if format == formatLegacyOffice {
		return "Legacy Office and RTF files are not read. Save the file as PDF, DOCX or XLSX and upload it again."
	}

	return fmt.Sprintf("Files of type %q cannot be read for extraction.", fileType)
}

func stoppedPlan(route ExtractionRoute, doc *document.Document) *PlanDocumentExtractionResult {
	return &PlanDocumentExtractionResult{
		Route: route,
		Result: &ProcessDocumentIntelligenceResult{
			DocumentID: doc.ID,
			Status:     string(doc.ContentStatus),
			Kind:       doc.DetectedKind,
		},
	}
}

func (a *Activities) loadDocument(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*document.Document, error) {
	doc, err := a.documentRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
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

	return doc, nil
}

type extractionUnavailable struct {
	code    documentupload.FailureCode
	reason  string
	message string
}

func (a *Activities) markExtractionUnavailable(
	ctx context.Context,
	doc *document.Document,
	why *extractionUnavailable,
) error {
	now := timeutils.NowUnix()
	analysis := &DocumentIntelligenceAnalysis{
		Kind:          "",
		ReviewStatus:  reviewStatusUnavailable,
		MissingFields: []string{},
		Signals:       []string{},
		Conflicts:     []*ReviewConflict{},
		Fields:        make(map[string]*ReviewField),
		Stops:         []*IntelligenceStop{},
	}
	content := &documentcontent.Content{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		Status:         documentcontent.StatusIndexed,
		Converter:      documentcontent.ConverterDocling,
		StructuredData: buildStructuredData(analysis, &AIDiagnostics{
			FallbackAnalysis: analysis,
			AcceptanceStatus: aiAcceptanceStatusNotAttempted,
			RejectionReason:  why.reason,
		}),
		FailureCode:     why.code.String(),
		FailureMessage:  why.message,
		LastExtractedAt: &now,
	}
	if _, err := a.contentRepo.Upsert(ctx, content); err != nil {
		return err
	}

	doc.ContentStatus = document.ContentStatusIndexed
	doc.ContentError = ""
	doc.HasExtractedText = false
	doc.DetectedKind = ""
	doc.ShipmentDraftStatus = document.ShipmentDraftStatusUnavailable
	if err := a.documentRepo.UpdateIntelligence(ctx, &documentports.UpdateDocumentIntelligenceRequest{
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

	a.metrics.Document.RecordExtraction("skipped", "", why.reason)
	a.syncSearchProjection(ctx, doc, "")
	inboundmessageports.AnnounceDocumentAttached(ctx, a.triage, doc)

	return nil
}

func (a *Activities) ConvertDocumentActivity(
	ctx context.Context,
	input *ConvertDocumentInput,
) (*ConvertDocumentResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  input.OrganizationID,
		BuID:   input.BusinessUnitID,
		UserID: input.UserID,
	}

	doc, err := a.loadDocument(ctx, input.DocumentID, tenantInfo)
	if err != nil {
		return nil, err
	}
	content, err := a.contentForConversion(ctx, doc, tenantInfo)
	if err != nil {
		return nil, err
	}

	data, failure := a.readDocumentBytes(ctx, doc)
	if failure != nil {
		return a.conversionFailed(ctx, doc, content, failure)
	}

	converted, failure, err := a.convertDocumentBytes(ctx, doc, data, input.OCR)
	if err != nil {
		return nil, err
	}
	if failure != nil {
		return a.conversionFailed(ctx, doc, content, failure)
	}

	if err = a.storeConversion(ctx, doc, content, converted); err != nil {
		return nil, err
	}

	return &ConvertDocumentResult{
		DocumentID: doc.ID,
		ContentID:  content.ID,
		PageCount:  content.PageCount,
		Converted:  true,
		Status:     string(content.Status),
	}, nil
}

func (a *Activities) contentForConversion(
	ctx context.Context,
	doc *document.Document,
	tenantInfo pagination.TenantInfo,
) (*documentcontent.Content, error) {
	content, err := a.contentRepo.GetByDocumentID(ctx, doc.ID, tenantInfo)
	if err == nil {
		return content, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	now := timeutils.NowUnix()
	content = &documentcontent.Content{
		DocumentID:      doc.ID,
		OrganizationID:  doc.OrganizationID,
		BusinessUnitID:  doc.BusinessUnitID,
		Status:          documentcontent.StatusExtracting,
		Converter:       documentcontent.ConverterDocling,
		LastExtractedAt: &now,
	}

	return a.contentRepo.Upsert(ctx, content)
}

type conversionFailure struct {
	code    documentupload.FailureCode
	message string
}

func (a *Activities) conversionFailed(
	ctx context.Context,
	doc *document.Document,
	content *documentcontent.Content,
	failure *conversionFailure,
) (*ConvertDocumentResult, error) {
	content.Converter = documentcontent.ConverterDocling
	if err := a.markFailed(ctx, doc, content, failure.code.String(), failure.message); err != nil {
		return nil, err
	}

	return &ConvertDocumentResult{
		DocumentID: doc.ID,
		ContentID:  content.ID,
		Converted:  false,
		Status:     string(documentcontent.StatusFailed),
	}, nil
}

func (a *Activities) readDocumentBytes(
	ctx context.Context,
	doc *document.Document,
) ([]byte, *conversionFailure) {
	download, err := a.storage.Download(ctx, doc.StoragePath)
	if err != nil {
		return nil, &conversionFailure{
			code:    documentupload.FailureDownloadFailed,
			message: "Failed to download document",
		}
	}
	defer download.Body.Close()

	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(download.Body); err != nil {
		return nil, &conversionFailure{
			code:    documentupload.FailureReadFailed,
			message: "Failed to read document bytes",
		}
	}

	data := buf.Bytes()
	if a.encryption != nil && encryptionservice.IsEnvelope(string(data)) {
		data, err = a.encryption.DecryptBytesWithAAD(string(data), encryptionservice.AAD{
			Purpose:        encryptionservice.PurposeDocument,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			ResourceID:     doc.StoragePath,
		})
		if err != nil {
			return nil, &conversionFailure{
				code:    documentupload.FailureReadFailed,
				message: "Failed to decrypt document bytes",
			}
		}
	}

	return data, nil
}

type convertedFile struct {
	pages            []*PageExtractionResult
	pageCount        int
	converterVersion string
	raw              []byte
}

func (a *Activities) convertDocumentBytes(
	ctx context.Context,
	doc *document.Document,
	data []byte,
	ocr bool,
) (*convertedFile, *conversionFailure, error) {
	format := conversionFormatOf(doc.FileType, doc.OriginalName)
	if format == formatPlainText {
		return a.plainTextConversion(data), nil, nil
	}

	if limit := a.conversionCfg.GetMaxFileBytes(); int64(len(data)) > limit {
		return nil, &conversionFailure{
			code:    documentupload.FailureFileTooLarge,
			message: fmt.Sprintf("The file is larger than the %d MB extraction limit.", limit>>20),
		}, nil
	}

	var nativePages map[int]bool
	if format == formatPDF {
		pageCount, native := a.pdfTextProfile(ctx, data)
		if limit := a.conversionCfg.GetMaxPages(); pageCount > limit {
			return nil, &conversionFailure{
				code: documentupload.FailureTooManyPages,
				message: fmt.Sprintf(
					"The PDF has %d pages; extraction reads at most %d.",
					pageCount,
					limit,
				),
			}, nil
		}
		nativePages = native
	}

	converted, err := a.convertWithConverter(ctx, &kernelports.ConvertRequest{
		FileName:    doc.OriginalName,
		ContentType: doc.FileType,
		Body:        bytes.NewReader(data),
		Size:        int64(len(data)),
		OCR:         ocr,
	}, true)
	switch {
	case errors.Is(err, kernelports.ErrDocumentNotConvertible):
		return nil, &conversionFailure{
			code:    documentupload.FailureExtractionFailed,
			message: stringutils.TruncateRunes(err.Error(), 500),
		}, nil
	case err != nil:
		return nil, nil, err
	}

	pages := a.extractionPages(converted.Pages, nativePages)

	return &convertedFile{
		pages:            pages,
		pageCount:        len(pages),
		converterVersion: converted.ConverterVersion,
		raw:              converted.Raw,
	}, nil, nil
}

func (a *Activities) plainTextConversion(data []byte) *convertedFile {
	maxChars := a.cfg.GetMaxExtractedChars()
	bounded := data
	if len(bounded) > maxChars {
		bounded = bounded[:maxChars]
	}
	text := stringutils.TruncateAndTrim(string(bounded), maxChars)

	return &convertedFile{
		pages: []*PageExtractionResult{{
			PageNumber: 1,
			SourceKind: documentcontent.SourceKindNative,
			Text:       text,
		}},
		pageCount: 1,
	}
}

func (a *Activities) pdfTextProfile(ctx context.Context, data []byte) (int, map[int]bool) {
	if a.pdfReader == nil {
		return 0, nil
	}
	pdf, err := a.pdfReader.Open(ctx, data)
	if err != nil {
		return 0, nil
	}
	defer pdf.Close()

	pageCount := pdf.PageCount()
	if pageCount > a.conversionCfg.GetMaxPages() {
		return pageCount, nil
	}

	native := make(map[int]bool, pageCount)
	for page := range pageCount {
		text, textErr := pdf.PageText(ctx, page)
		native[page+1] = textErr == nil && strings.TrimSpace(text) != ""
	}

	return pageCount, native
}

func (a *Activities) convertWithConverter(
	ctx context.Context,
	req *kernelports.ConvertRequest,
	resumable bool,
) (*kernelports.ConvertedDocument, error) {
	if a.converter == nil || !a.converter.Enabled() {
		return nil, kernelports.ErrDocumentConverterUnavailable
	}

	data, err := readAllBody(req)
	if err != nil {
		return nil, err
	}

	var taskID kernelports.ConversionTaskID
	if resumable {
		taskID = resumedConversionTask(ctx)
	}
	for resubmits := 0; ; {
		if taskID == "" {
			submit := *req
			submit.Body = bytes.NewReader(data)
			if taskID, err = a.converter.Submit(ctx, &submit); err != nil {
				return nil, err
			}
		}
		recordConversionHeartbeat(ctx, taskID)

		converted, waitErr := a.awaitConversion(ctx, taskID)
		if !errors.Is(waitErr, kernelports.ErrConversionTaskNotFound) {
			return converted, waitErr
		}

		resubmits++
		if resubmits > maxConversionResubmits {
			return nil, fmt.Errorf(
				"%w: the converter lost the task %d times",
				kernelports.ErrDocumentConverterUnavailable,
				resubmits,
			)
		}
		a.logger.Warn("the document converter lost a task; submitting it again",
			zap.String("taskId", string(taskID)),
			zap.Int("resubmit", resubmits),
		)
		taskID = ""
	}
}

func readAllBody(req *kernelports.ConvertRequest) ([]byte, error) {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("read document for conversion: %w", err)
	}

	return data, nil
}

func (a *Activities) awaitConversion(
	ctx context.Context,
	taskID kernelports.ConversionTaskID,
) (*kernelports.ConvertedDocument, error) {
	for {
		status, err := a.converter.Status(ctx, taskID)
		if err != nil {
			return nil, err
		}
		switch status.State {
		case kernelports.ConversionStateSucceeded:
			return a.converter.Result(ctx, taskID)
		case kernelports.ConversionStateFailed:
			return nil, fmt.Errorf("%w: %s", kernelports.ErrDocumentNotConvertible, status.Message)
		case kernelports.ConversionStatePending, kernelports.ConversionStateRunning:
		}

		recordConversionHeartbeat(ctx, taskID)
		timer := time.NewTimer(conversionPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func resumedConversionTask(ctx context.Context) kernelports.ConversionTaskID {
	if !activity.IsActivity(ctx) || !activity.HasHeartbeatDetails(ctx) {
		return ""
	}
	var details conversionHeartbeat
	if err := activity.GetHeartbeatDetails(ctx, &details); err != nil {
		return ""
	}

	return kernelports.ConversionTaskID(details.TaskID)
}

func recordConversionHeartbeat(ctx context.Context, taskID kernelports.ConversionTaskID) {
	if !activity.IsActivity(ctx) {
		return
	}
	activity.RecordHeartbeat(ctx, conversionHeartbeat{TaskID: string(taskID)})
}

func (a *Activities) extractionPages(
	converted []kernelports.ConvertedPage,
	nativePages map[int]bool,
) []*PageExtractionResult {
	maxChars := a.cfg.GetMaxExtractedChars()
	pages := make([]*PageExtractionResult, 0, len(converted))
	for i := range converted {
		page := &converted[i]
		native := page.Native
		if hasNative, known := nativePages[page.Number]; known {
			native = hasNative
		}
		kind := sourceKindOf(page.OCR, native)

		result := &PageExtractionResult{
			PageNumber: page.Number,
			SourceKind: kind,
			Text:       stringutils.TruncateAndTrim(page.Text, maxChars),
			Markdown:   stringutils.TruncateAtBlock(page.Markdown, maxChars),
			Width:      int(math.Round(page.Width)),
			Height:     int(math.Round(page.Height)),
			Metadata: map[string]any{
				"converter": documentcontent.ConverterDocling,
			},
		}
		if kind != documentcontent.SourceKindNative {
			result.OCRConfidence = page.OCRConfidence
		}
		if lines := layoutLines(page.Lines); len(lines) > 0 {
			result.Metadata[documentcontent.MetadataLines] = lines
		}
		pages = append(pages, result)
	}

	return pages
}

func sourceKindOf(ocr, native bool) documentcontent.SourceKind {
	switch {
	case ocr && native:
		return documentcontent.SourceKindMixed
	case ocr:
		return documentcontent.SourceKindOCR
	default:
		return documentcontent.SourceKindNative
	}
}

func layoutLines(lines []kernelports.ConvertedLine) []documentcontent.LayoutLine {
	out := make([]documentcontent.LayoutLine, 0, min(len(lines), documentcontent.MaxLayoutLines))
	for i := range lines {
		if len(out) == documentcontent.MaxLayoutLines {
			break
		}
		out = append(out, documentcontent.LayoutLine{
			Text: lines[i].Text,
			X:    lines[i].X,
			Y:    lines[i].Y,
			W:    lines[i].W,
			H:    lines[i].H,
		})
	}

	return out
}

func (a *Activities) storeConversion(
	ctx context.Context,
	doc *document.Document,
	content *documentcontent.Content,
	converted *convertedFile,
) error {
	extracted := finalizeExtraction(converted.pages, a.cfg.GetMaxExtractedChars())

	rawKey := ""
	if len(converted.raw) > 0 {
		key, err := a.storeRawConversion(ctx, doc, converted.raw)
		if err != nil {
			return err
		}
		rawKey = key
	}

	content.Status = documentcontent.StatusExtracted
	content.ContentText = extracted.Text
	content.PageCount = converted.pageCount
	content.SourceKind = extracted.SourceKind
	content.Converter = documentcontent.ConverterDocling
	content.ConverterVersion = converted.converterVersion
	content.RawObjectKey = rawKey
	content.FailureCode = ""
	content.FailureMessage = ""
	if _, err := a.contentRepo.Upsert(ctx, content); err != nil {
		return err
	}

	return a.contentRepo.ReplacePages(ctx, content, buildContentPages(content, extracted.Pages))
}

func (a *Activities) storeRawConversion(
	ctx context.Context,
	doc *document.Document,
	raw []byte,
) (string, error) {
	if a.encryption == nil {
		return "", temporal.NewNonRetryableApplicationError(
			"Document encryption is not configured",
			temporaltype.ErrorTypeNonRetryable.String(),
			nil,
		)
	}

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(raw); err != nil {
		return "", fmt.Errorf("compress converted document: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("compress converted document: %w", err)
	}

	key := documentcontent.RawObjectKey(doc.OrganizationID, doc.ID)
	encrypted, err := a.encryption.EncryptBytesWithAADContext(ctx, compressed.Bytes(), encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		ResourceID:     key,
	})
	if err != nil {
		return "", fmt.Errorf("encrypt converted document: %w", err)
	}

	body := []byte(encrypted)
	if _, err = a.storage.Upload(ctx, &kernelports.UploadParams{
		Key:         key,
		ContentType: "application/gzip",
		Size:        int64(len(body)),
		Body:        bytes.NewReader(body),
		Metadata: map[string]string{
			"crypto_mode": encryptionservice.CryptoModeForCiphertext(encrypted),
			"converter":   string(documentcontent.ConverterDocling),
		},
	}); err != nil {
		return "", fmt.Errorf("store converted document: %w", err)
	}

	return key, nil
}

func (a *Activities) AnalyzeConvertedDocumentActivity(
	ctx context.Context,
	input *AnalyzeConvertedDocumentInput,
) (*ProcessDocumentIntelligenceResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  input.OrganizationID,
		BuID:   input.BusinessUnitID,
		UserID: input.UserID,
	}

	doc, err := a.loadDocument(ctx, input.DocumentID, tenantInfo)
	if err != nil {
		return nil, err
	}
	control, err := a.getDocumentControl(ctx, doc.OrganizationID, doc.BusinessUnitID)
	if err != nil {
		return nil, err
	}
	content, err := a.contentRepo.GetByDocumentID(ctx, doc.ID, tenantInfo)
	if err != nil {
		return nil, err
	}
	pages, err := a.contentRepo.ListPagesByDocumentID(ctx, doc.ID, tenantInfo)
	if err != nil {
		return nil, err
	}

	payload := &documentintelligencecontract.ProcessDocumentIntelligencePayload{
		BasePayload: input.BasePayload,
		DocumentID:  input.DocumentID,
	}
	extracted := storedExtraction(content, pages)
	outcome, err := a.analyzeExtraction(ctx, payload, doc, control, extracted)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	if content.LastExtractedAt != nil {
		now = *content.LastExtractedAt
	}
	if err = a.persistIndexedContentAndAssociate(ctx, &PersistIndexedContentPayload{
		Document:       doc,
		TenantInfo:     tenantInfo,
		Control:        control,
		Content:        content,
		Extracted:      outcome.Extracted,
		Classification: outcome.Classification,
		Intelligence:   outcome.Intelligence,
		AIDiagnostics:  outcome.AIDiagnostics,
		Timestamp:      now,
		PagesStored:    true,
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
		Signature:      outcome.Signature,
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

func storedExtraction(
	content *documentcontent.Content,
	pages []*documentcontent.Page,
) *ExtractionResult {
	results := make([]*PageExtractionResult, 0, len(pages))
	for _, page := range pages {
		results = append(results, &PageExtractionResult{
			PageNumber:           page.PageNumber,
			SourceKind:           page.SourceKind,
			Text:                 page.ExtractedText,
			Markdown:             page.Markdown,
			OCRConfidence:        page.OCRConfidence,
			PreprocessingApplied: page.PreprocessingApplied,
			Width:                page.Width,
			Height:               page.Height,
			Metadata:             page.Metadata,
		})
	}

	return &ExtractionResult{
		Text:       content.ContentText,
		PageCount:  content.PageCount,
		SourceKind: content.SourceKind,
		Pages:      results,
	}
}

func (a *Activities) analyzeExtraction(
	ctx context.Context,
	payload *documentintelligencecontract.ProcessDocumentIntelligencePayload,
	doc *document.Document,
	control *tenant.DocumentControl,
	extracted *ExtractionResult,
) (*ExtractionPipelineOutcome, error) {
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
	result, err := a.enrichWithAI(ctx, &EnrichmentPayload{
		Payload:        payload,
		Document:       doc,
		Control:        control,
		Extracted:      extracted,
		Features:       features,
		Fingerprint:    fingerprint,
		Classification: classification,
		Intelligence:   intelligence,
	})
	if err != nil {
		return nil, err
	}

	return &ExtractionPipelineOutcome{
		Extracted:      extracted,
		Classification: classification,
		Intelligence:   intelligence,
		AIDiagnostics:  result.AIDiagnostics,
		Signature:      result.Signature,
		EnqueueAsyncAI: result.EnqueueAsyncAI,
	}, nil
}

func (a *Activities) ReconcileDocumentExtractionActivity(
	ctx context.Context,
	payload *ReconcileDocumentIntelligencePayload,
) (*ReconcileDocumentIntelligenceResult, error) {
	if !a.workflowStarter.Enabled() {
		return &ReconcileDocumentIntelligenceResult{}, nil
	}

	olderThan := time.Now().Add(-time.Duration(payload.OlderThanSeconds) * time.Second).Unix()
	docs, err := a.contentRepo.ListPendingExtraction(
		ctx,
		&documentcontentports.ListPendingDocumentExtractionRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: payload.OrganizationID,
				BuID:  payload.BusinessUnitID,
			},
			OlderThan: olderThan,
			Limit:     payload.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	result := &ReconcileDocumentIntelligenceResult{}
	for _, doc := range docs {
		startErr := documentextractioncontract.Start(
			ctx,
			a.workflowStarter,
			&documentextractioncontract.ProcessDocumentExtractionPayload{
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
