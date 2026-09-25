package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/pagination"
)

// ErrPageInspectionUnavailable is returned by an inspector that cannot render
// pages in this build. Capture still works without it: pages are kept and
// filed, and only what inspection would have found (thumbnails, blank pages,
// cover sheets) is missing.
var ErrPageInspectionUnavailable = errors.New("page inspection is unavailable in this build")

// CapturePageInspection is what reading one captured page found.
type CapturePageInspection struct {
	WidthPx  int
	HeightPx int
	// Thumbnail is a small JPEG of the page for the intake queue.
	Thumbnail []byte
	// InkCoverage is the share of the page, margins excluded, that is dark
	// enough to be content. A page under the blank threshold carries nothing.
	InkCoverage float64
	// Codes are the payloads of every QR code found on the page.
	Codes []string
}

// CapturePageInspector renders a one-page PDF and reads it.
type CapturePageInspector interface {
	Inspect(ctx context.Context, pdf []byte) (*CapturePageInspection, error)
}

// CaptureAssemblyPage is one page going into an assembled document.
type CaptureAssemblyPage struct {
	PDF []byte
	// Rotation is clockwise degrees, one of 0, 90, 180 or 270.
	Rotation int
}

// CapturePDFAssembler turns captured pages into documents and back.
type CapturePDFAssembler interface {
	// Assemble joins pages, in order and rotated as asked, into one PDF.
	Assemble(ctx context.Context, pages []CaptureAssemblyPage) ([]byte, error)
	// PageCount reports how many pages a PDF holds, failing on anything that
	// is not a readable PDF.
	PageCount(ctx context.Context, pdf []byte) (int, error)
	// Split separates a multi-page PDF into one PDF per page, keeping each
	// page's own content stream, so a printed page keeps its text.
	Split(ctx context.Context, pdf []byte) ([][]byte, error)
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
