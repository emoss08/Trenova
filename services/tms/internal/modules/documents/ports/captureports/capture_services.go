package captureports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

// CaptureQRCode is a QR code as its modules: one string per row, top to
// bottom, "1" for a dark module. It carries no quiet zone; whoever draws it
// leaves the four-module margin the symbology requires.
type CaptureQRCode struct {
	Size    int      `json:"size"`
	Modules []string `json:"modules"`
}

// CaptureQREncoder draws the code a cover sheet carries, so the sheet can be
// printed wherever it is laid out.
type CaptureQREncoder interface {
	Encode(payload string) (*CaptureQRCode, error)
}

// CaptureAnalysisRequest is one proposed document to read before a person
// files it.
type CaptureAnalysisRequest struct {
	TenantInfo pagination.TenantInfo
	FileName   string
	PDF        []byte
}

// CaptureAnalysis is what the document pipeline made of a proposed document.
type CaptureAnalysis struct {
	// Kind is the classifier's reading, such as a rate confirmation or a POD,
	// empty when it could not tell.
	Kind       string
	Confidence float64
	// DocumentTypeCode is the tenant document type code the kind maps to.
	DocumentTypeCode string
	// References are the numbers the document names that could identify a
	// shipment: PRO, BOL, load and reference numbers.
	References []string
}

// CaptureAnalyzer reads a proposed document with the same classifier and
// parsing rules a stored document goes through, without storing it.
type CaptureAnalyzer interface {
	Analyze(ctx context.Context, req *CaptureAnalysisRequest) (*CaptureAnalysis, error)
}
