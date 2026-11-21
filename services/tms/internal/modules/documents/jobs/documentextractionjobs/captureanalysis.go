package documentextractionjobs

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentaiextractionjobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/referenceutils"
)

const (
	captureContentType = "application/pdf"
	// maxCaptureReferences bounds what a captured document offers as shipment
	// references. The extracted reference field comes first, then numbers from
	// the text in the order they appear.
	maxCaptureReferences = 12
)

// CaptureAnalyzer reads a captured document through the same conversion and
// classification a stored document goes through, without storing it or
// recording anything about it. A captured stack is read to suggest where its
// parts go; the document each part becomes is read again, properly, once it is
// filed.
type CaptureAnalyzer struct {
	activities *Activities
}

func NewCaptureAnalyzer(a *Activities) captureports.CaptureAnalyzer {
	return &CaptureAnalyzer{activities: a}
}

func (c *CaptureAnalyzer) Analyze(
	ctx context.Context,
	req *captureports.CaptureAnalysisRequest,
) (*captureports.CaptureAnalysis, error) {
	a := c.activities
	control, err := a.support.GetDocumentControl(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
	if err != nil {
		return nil, err
	}

	gate, err := a.gateReading(ctx, control, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if !gate.open() || !a.converter.Enabled() {
		return &captureports.CaptureAnalysis{}, nil
	}

	pdf, err := req.Assemble(ctx)
	if err != nil {
		return nil, fmt.Errorf("assemble the captured document: %w", err)
	}
	if int64(len(pdf)) > a.conversionCfg.GetMaxFileBytes() {
		return &captureports.CaptureAnalysis{}, nil
	}

	converted, err := a.convertWithConverter(ctx, &kernelports.ConvertRequest{
		FileName:    req.FileName,
		ContentType: captureContentType,
		Open: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(pdf)), nil
		},
		Size: int64(len(pdf)),
		OCR:  gate.ocr,
	}, false)
	if err != nil {
		return nil, err
	}

	extracted := documentanalysis.FinalizeExtraction(
		a.extractionPages(converted.Pages, nil),
		a.cfg.GetMaxExtractedChars(),
	)
	features := documentanalysis.ExtractDocumentFeatures(
		req.FileName,
		extracted.Pages,
		extracted.Text,
	)
	fingerprint := documentanalysis.DetectProviderFingerprint(
		req.FileName,
		extracted.Text,
		features,
	)
	classification := documentanalysis.ClassifyDocumentWithControl(
		req.FileName,
		extracted.Text,
		control,
		features,
		fingerprint,
	)

	return captureAnalysisOf(
		documentanalysis.AnalyzeDocument(classification, extracted),
		classification,
		extracted.Text,
		control.EnableAutoDocumentTypeAssociate,
	), nil
}

func captureAnalysisOf(
	intelligence *documentanalysis.DocumentIntelligenceAnalysis,
	classification *documentanalysis.ClassificationResult,
	text string,
	associatesType bool,
) *captureports.CaptureAnalysis {
	analysis := &captureports.CaptureAnalysis{
		Kind:       intelligence.Kind,
		Confidence: intelligence.OverallConfidence,
		References: captureReferences(intelligence, text),
	}
	if analysis.Confidence == 0 {
		analysis.Confidence = classification.Confidence
	}
	if !associatesType {
		return analysis
	}
	if inferred, ok := documentaiextractionjobs.InferDocumentType(intelligence.Kind); ok {
		analysis.DocumentTypeCode = inferred.Code
	}

	return analysis
}

func captureReferences(
	intelligence *documentanalysis.DocumentIntelligenceAnalysis,
	text string,
) []string {
	extracted := ""
	if field, ok := intelligence.Fields["referenceNumber"]; ok && field != nil {
		extracted = field.Value
	}

	return referenceutils.Candidates(maxCaptureReferences, extracted, text)
}
