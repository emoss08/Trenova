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

	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentextractioncontract"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentintelligencecontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
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

	reasonConverterUnconfigured = "converter_not_configured"

	extractionOffOnServerMessage = "Document extraction is turned off on this server. " +
		"An administrator needs to set ai.documentExtraction to true, " +
		"then extract the document again."
	extractionNotInPlanMessage  = "Your plan does not include document extraction."
	extractionNoProviderMessage = "No AI provider is assigned to document extraction. " +
		"In AI settings, add a provider or assign an existing one to the Document extraction " +
		"task, then extract the document again."
	converterUnavailableMessage = "Document extraction is not set up on this server: " +
		"the document converter is not configured."
	converterUnreachableMessage = "The document converter could not be reached. " +
		"Extract the document again once it is running."
	converterMisconfiguredMessage = "The document converter refused this server's API key. " +
		"An administrator needs to set documentConversion.apiKey to the converter's key, " +
		"then extract the document again."
	conversionFailedMessage = "The document could not be read. Extract it again to retry."

	errorTypeConverterUnavailable = "DocumentConverterUnavailable"
)

type ExtractionRoute string

const (
	ExtractionRouteSkip    ExtractionRoute = "skip"
	ExtractionRouteDocling ExtractionRoute = "docling"
	ExtractionRouteStopped ExtractionRoute = "stopped"
)

type PlanDocumentExtractionResult struct {
	Route                    ExtractionRoute                                     `json:"route"`
	OCR                      bool                                                `json:"ocr"`
	ConversionTimeoutSeconds int64                                               `json:"conversionTimeoutSeconds"`
	Result                   *documentanalysis.ProcessDocumentIntelligenceResult `json:"result,omitempty"`
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

type MarkConversionFailedInput struct {
	temporaltype.BasePayload

	DocumentID pulid.ID `json:"documentId"`
	Code       string   `json:"code"`
	Message    string   `json:"message"`
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

var officeContentTypes = map[string]struct{}{
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": {},
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
	case documentanalysis.IsPDFType(mediaType, ext):
		return formatPDF
	case mediaType == "message/rfc822" || ext == ".eml":
		return formatEmail
	case documentanalysis.IsPlainTextType(mediaType, ext):
		return formatPlainText
	}
	if _, ok := convertibleImageTypes[mediaType]; ok {
		return formatImage
	}
	if _, ok := officeContentTypes[mediaType]; ok {
		return formatOffice
	}
	switch ext {
	case ".docx", ".xlsx", ".pptx":
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

	control, err := a.support.GetDocumentControl(ctx, doc.OrganizationID, doc.BusinessUnitID)
	if err != nil {
		return nil, err
	}
	if !control.EnableDocumentIntelligence {
		a.metrics.Document.RecordExtraction("skipped", "", "control_disabled")
		return stoppedPlan(ExtractionRouteSkip, doc), nil
	}

	return a.planConversion(ctx, doc, control, tenantInfo)
}

func (a *Activities) planConversion(
	ctx context.Context,
	doc *document.Document,
	control *documentcontrol.DocumentControl,
	tenantInfo pagination.TenantInfo,
) (*PlanDocumentExtractionResult, error) {
	availability, err := a.aiDocumentService.ExtractionAvailability(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !availability.Available() {
		if err = a.markExtractionUnavailable(
			ctx,
			doc,
			modelUnavailable(availability),
		); err != nil {
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

	if err = a.markExtractionStarted(ctx, doc, tenantInfo); err != nil {
		return nil, err
	}

	return &PlanDocumentExtractionResult{
		Route:                    ExtractionRouteDocling,
		OCR:                      control.EnableOCR,
		ConversionTimeoutSeconds: int64(a.conversionCfg.GetConversionTimeout().Seconds()),
	}, nil
}

func modelUnavailable(
	availability documentaiextractionports.ExtractionAvailability,
) *extractionUnavailable {
	message := extractionNoProviderMessage
	switch availability {
	case documentaiextractionports.ExtractionOffOnServer:
		message = extractionOffOnServerMessage
	case documentaiextractionports.ExtractionNotInPlan:
		message = extractionNotInPlanMessage
	case documentaiextractionports.ExtractionAvailable,
		documentaiextractionports.ExtractionNoProviderAssigned:
	}

	return &extractionUnavailable{
		code:    documentupload.FailureExtractionUnavailable,
		reason:  string(availability),
		message: message,
	}
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
		Result: &documentanalysis.ProcessDocumentIntelligenceResult{
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
	analysis := &documentanalysis.DocumentIntelligenceAnalysis{
		Kind:          "",
		ReviewStatus:  documentanalysis.ReviewStatusUnavailable,
		MissingFields: []string{},
		Signals:       []string{},
		Conflicts:     []*documentanalysis.ReviewConflict{},
		Fields:        make(map[string]*documentanalysis.ReviewField),
		Stops:         []*documentanalysis.IntelligenceStop{},
	}
	content := &documentcontent.Content{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		Status:         documentcontent.StatusIndexed,
		StructuredData: documentanalysis.BuildStructuredData(
			analysis,
			&documentanalysis.AIDiagnostics{
				FallbackAnalysis: analysis,
				AcceptanceStatus: documentanalysis.AIAcceptanceStatusNotAttempted,
				RejectionReason:  why.reason,
			},
		),
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
		},
	); err != nil {
		return err
	}

	a.metrics.Document.RecordExtraction("skipped", "", why.reason)
	a.support.SyncSearchProjection(ctx, doc, "")
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

	converted, failure, err := a.convertDocument(ctx, doc, tenantInfo, input.OCR)
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

func (a *Activities) MarkConversionFailedActivity(
	ctx context.Context,
	input *MarkConversionFailedInput,
) error {
	tenantInfo := pagination.TenantInfo{
		OrgID:  input.OrganizationID,
		BuID:   input.BusinessUnitID,
		UserID: input.UserID,
	}

	doc, err := a.loadDocument(ctx, input.DocumentID, tenantInfo)
	if err != nil {
		if appErr, ok := errors.AsType[*temporal.ApplicationError](err); ok &&
			appErr.Type() == temporaltype.ErrorTypeResourceNotFound.String() {
			return nil
		}
		return err
	}
	content, err := a.contentForConversion(ctx, doc, tenantInfo)
	if err != nil {
		return err
	}

	_, err = a.conversionFailed(ctx, doc, content, &conversionFailure{
		code:    documentupload.FailureCode(input.Code),
		message: input.Message,
	})

	return err
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

func (a *Activities) documentSource(
	doc *document.Document,
	tenantInfo pagination.TenantInfo,
) kernelports.ImagingSource {
	return storedobject.Source(
		a.storage,
		a.encryption,
		doc.StoragePath,
		storedobject.AAD(tenantInfo, doc.StoragePath),
		doc.FileType,
		doc.FileSize,
	)
}

const decryptFailedMessage = "Failed to decrypt document bytes"

func unreadable(err error) bool {
	return errors.Is(err, encryptionservice.ErrBlobTampered) ||
		errors.Is(err, encryptionservice.ErrBlobAAD) ||
		errors.Is(err, encryptionservice.ErrBlobFormat) ||
		errors.Is(err, storedobject.ErrNotSealed)
}

func (a *Activities) readPlainText(
	ctx context.Context,
	source kernelports.ImagingSource,
) ([]byte, *conversionFailure, error) {
	body, err := source.Open(ctx)
	if err != nil {
		if unreadable(err) {
			return nil, &conversionFailure{
				code:    documentupload.FailureReadFailed,
				message: decryptFailedMessage,
			}, nil
		}
		return nil, nil, err
	}
	defer body.Close()

	data, err := io.ReadAll(io.LimitReader(body, int64(a.cfg.GetMaxExtractedChars())))
	if err != nil {
		if unreadable(err) {
			return nil, &conversionFailure{
				code:    documentupload.FailureReadFailed,
				message: decryptFailedMessage,
			}, nil
		}
		return nil, nil, err
	}
	return data, nil, nil
}

type convertedFile struct {
	pages            []*documentanalysis.PageExtractionResult
	pageCount        int
	converterVersion string
	raw              []byte
}

func (a *Activities) convertDocument(
	ctx context.Context,
	doc *document.Document,
	tenantInfo pagination.TenantInfo,
	ocr bool,
) (*convertedFile, *conversionFailure, error) {
	if a.encryption == nil {
		return nil, &conversionFailure{
			code:    documentupload.FailureReadFailed,
			message: "Document encryption is not configured",
		}, nil
	}
	source := a.documentSource(doc, tenantInfo)
	format := conversionFormatOf(doc.FileType, doc.OriginalName)
	if format == formatPlainText {
		data, failure, err := a.readPlainText(ctx, source)
		if failure != nil || err != nil {
			return nil, failure, err
		}
		return a.plainTextConversion(data), nil, nil
	}

	if limit := a.conversionCfg.GetMaxFileBytes(); doc.FileSize > limit {
		return nil, &conversionFailure{
			code:    documentupload.FailureFileTooLarge,
			message: fmt.Sprintf("The file is larger than the %d MB extraction limit.", limit>>20),
		}, nil
	}

	var nativePages map[int]bool
	if format == formatPDF {
		pageCount, native, err := a.pdfTextProfile(ctx, source)
		if err != nil {
			return nil, nil, err
		}
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
		Open:        source.Open,
		Size:        doc.FileSize,
		OCR:         ocr,
	}, true)
	switch {
	case unreadable(err):
		return nil, &conversionFailure{
			code:    documentupload.FailureReadFailed,
			message: decryptFailedMessage,
		}, nil
	case errors.Is(err, kernelports.ErrDocumentNotConvertible):
		return nil, &conversionFailure{
			code:    documentupload.FailureExtractionFailed,
			message: stringutils.TruncateRunes(err.Error(), 500),
		}, nil
	case errors.Is(err, kernelports.ErrDocumentConverterMisconfigured):
		a.logger.Error("the document converter refused the configured API key",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
		return nil, &conversionFailure{
			code:    documentupload.FailureConverterMisconfigured,
			message: converterMisconfiguredMessage,
		}, nil
	case errors.Is(err, kernelports.ErrDocumentConverterUnavailable):
		return nil, nil, temporal.NewApplicationErrorWithCause(
			err.Error(),
			errorTypeConverterUnavailable,
			err,
		)
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
		pages: []*documentanalysis.PageExtractionResult{{
			PageNumber: 1,
			SourceKind: documentcontent.SourceKindNative,
			Text:       text,
		}},
		pageCount: 1,
	}
}

func (a *Activities) pdfTextProfile(
	ctx context.Context,
	source kernelports.ImagingSource,
) (pageCount int, native map[int]bool, err error) {
	if a.imaging == nil {
		return 0, nil, nil
	}
	source.ContentType = "application/pdf"
	probe, err := a.imaging.Probe(ctx, &kernelports.ImagingProbeRequest{
		Source:      source,
		TextProfile: true,
		Priority:    kernelports.ImagingBatch,
	})
	if err != nil {
		if probeIsVerdict(err) {
			return 0, nil, nil
		}
		return 0, nil, fmt.Errorf("probe the PDF before converting it: %w", err)
	}
	if probe.Encrypted {
		return 0, nil, nil
	}

	pageCount = probe.PageCount
	if pageCount > a.conversionCfg.GetMaxPages() {
		return pageCount, nil, nil
	}

	native = make(map[int]bool, len(probe.Pages))
	for _, page := range probe.Pages {
		native[page.Index+1] = page.HasText
	}

	return pageCount, native, nil
}

func probeIsVerdict(err error) bool {
	return unreadable(err) ||
		errors.Is(err, kernelports.ErrImagingInvalidDocument) ||
		errors.Is(err, kernelports.ErrImagingEncrypted) ||
		errors.Is(err, kernelports.ErrImagingUnsupported) ||
		errors.Is(err, kernelports.ErrImagingTooLarge) ||
		errors.Is(err, kernelports.ErrImagingPixelBudget) ||
		errors.Is(err, kernelports.ErrImagingInternal)
}

func (a *Activities) convertWithConverter(
	ctx context.Context,
	req *kernelports.ConvertRequest,
	resumable bool,
) (*kernelports.ConvertedDocument, error) {
	if a.converter == nil || !a.converter.Enabled() {
		return nil, kernelports.ErrDocumentConverterUnavailable
	}

	var taskID kernelports.ConversionTaskID
	if resumable {
		taskID = resumedConversionTask(ctx)
	}
	for resubmits := 0; ; {
		if taskID == "" {
			var err error
			if taskID, err = a.converter.Submit(ctx, req); err != nil {
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
) []*documentanalysis.PageExtractionResult {
	maxChars := a.cfg.GetMaxExtractedChars()
	pages := make([]*documentanalysis.PageExtractionResult, 0, len(converted))
	for i := range converted {
		page := &converted[i]
		native := page.Native
		if hasNative, known := nativePages[page.Number]; known {
			native = hasNative
		}
		kind := sourceKindOf(page.OCR, native)

		result := &documentanalysis.PageExtractionResult{
			PageNumber: page.Number,
			SourceKind: kind,
			Text:       stringutils.TruncateAndTrim(page.Text, maxChars),
			Markdown:   stringutils.TruncateAtBlock(page.Markdown, maxChars),
			Width:      int(math.Round(page.Width)),
			Height:     int(math.Round(page.Height)),
			Metadata:   map[string]any{},
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
	extracted := documentanalysis.FinalizeExtraction(converted.pages, a.cfg.GetMaxExtractedChars())

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
	content.ConverterVersion = converted.converterVersion
	content.RawObjectKey = rawKey
	content.FailureCode = ""
	content.FailureMessage = ""
	if _, err := a.contentRepo.Upsert(ctx, content); err != nil {
		return err
	}

	return a.contentRepo.ReplacePages(
		ctx,
		content,
		documentanalysis.BuildContentPages(content, extracted.Pages),
	)
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
	if _, err := storedobject.Write(ctx, a.storage, a.encryption, &storedobject.WriteRequest{
		Key:         key,
		ContentType: "application/gzip",
		AAD: storedobject.AAD(
			pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
			key,
		),
		Body: bytes.NewReader(compressed.Bytes()),
		Size: int64(compressed.Len()),
	}); err != nil {
		return "", fmt.Errorf("store converted document: %w", err)
	}

	return key, nil
}

func (a *Activities) AnalyzeConvertedDocumentActivity(
	ctx context.Context,
	input *AnalyzeConvertedDocumentInput,
) (*documentanalysis.ProcessDocumentIntelligenceResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  input.OrganizationID,
		BuID:   input.BusinessUnitID,
		UserID: input.UserID,
	}

	doc, err := a.loadDocument(ctx, input.DocumentID, tenantInfo)
	if err != nil {
		return nil, err
	}
	control, err := a.support.GetDocumentControl(ctx, doc.OrganizationID, doc.BusinessUnitID)
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
	if err = a.persistIndexedContentAndAssociate(ctx, &documentanalysis.PersistIndexedContentPayload{
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

	if err = a.finalizeProcessDocumentIntelligence(ctx, &documentanalysis.FinalizeIntelligenceParams{
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

	return &documentanalysis.ProcessDocumentIntelligenceResult{
		DocumentID: doc.ID,
		Status:     string(doc.ContentStatus),
		Kind:       outcome.Classification.Kind,
	}, nil
}

func storedExtraction(
	content *documentcontent.Content,
	pages []*documentcontent.Page,
) *documentanalysis.ExtractionResult {
	results := make([]*documentanalysis.PageExtractionResult, 0, len(pages))
	for _, page := range pages {
		results = append(results, &documentanalysis.PageExtractionResult{
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

	return &documentanalysis.ExtractionResult{
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
	control *documentcontrol.DocumentControl,
	extracted *documentanalysis.ExtractionResult,
) (*documentanalysis.ExtractionPipelineOutcome, error) {
	features := documentanalysis.ExtractDocumentFeatures(
		doc.OriginalName,
		extracted.Pages,
		extracted.Text,
	)
	fingerprint := documentanalysis.DetectProviderFingerprint(
		doc.OriginalName,
		extracted.Text,
		features,
	)
	classification := documentanalysis.ClassifyDocumentWithControl(
		doc.OriginalName,
		extracted.Text,
		control,
		features,
		fingerprint,
	)
	intelligence := documentanalysis.AnalyzeDocument(classification, extracted)
	result, err := a.enrichWithAI(ctx, &documentanalysis.EnrichmentPayload{
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

	return &documentanalysis.ExtractionPipelineOutcome{
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
	payload *documentanalysis.ReconcileDocumentIntelligencePayload,
) (*documentanalysis.ReconcileDocumentIntelligenceResult, error) {
	if !a.workflowStarter.Enabled() {
		return &documentanalysis.ReconcileDocumentIntelligenceResult{}, nil
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

	result := &documentanalysis.ReconcileDocumentIntelligenceResult{}
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
