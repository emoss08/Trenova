package documentintelligencejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/referenceutils"
)

const (
	captureContentType = "application/pdf"
	// maxCaptureReferences bounds what a captured document offers as shipment
	// references. The extracted reference field comes first, then numbers from
	// the text in the order they appear.
	maxCaptureReferences = 12
)

// CaptureAnalyzer reads a captured document through the same extraction,
// classification and parsing rules a stored document goes through, without
// storing it or recording anything about it. A captured stack is read to
// suggest where its parts go; the document each part becomes is read again,
// properly, once it is filed.
type CaptureAnalyzer struct {
	activities *Activities
}

func NewCaptureAnalyzer(a *Activities) services.CaptureAnalyzer {
	return &CaptureAnalyzer{activities: a}
}

func (c *CaptureAnalyzer) Analyze(
	ctx context.Context,
	req *services.CaptureAnalysisRequest,
) (*services.CaptureAnalysis, error) {
	a := c.activities
	control, err := a.getDocumentControl(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
	if err != nil {
		return nil, err
	}

	doc := &document.Document{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		OriginalName:   req.FileName,
		FileType:       captureContentType,
	}

	extracted, err := a.extractContent(ctx, doc, req.PDF, control)
	if err != nil {
		return nil, err
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
		req.TenantInfo,
		doc.OriginalName,
		classification.ProviderFingerprint,
		extracted,
		intelligence,
	)

	analysis := &services.CaptureAnalysis{
		Kind:       intelligence.Kind,
		Confidence: intelligence.OverallConfidence,
		References: captureReferences(intelligence, extracted.Text),
	}
	if analysis.Confidence == 0 {
		analysis.Confidence = classification.Confidence
	}
	if inferred, ok := inferDocumentType(intelligence.Kind); ok {
		analysis.DocumentTypeCode = inferred.Code
	}

	return analysis, nil
}

func captureReferences(intelligence *DocumentIntelligenceAnalysis, text string) []string {
	extracted := ""
	if field, ok := intelligence.Fields["referenceNumber"]; ok && field != nil {
		extracted = field.Value
	}

	return referenceutils.Candidates(maxCaptureReferences, extracted, text)
}
