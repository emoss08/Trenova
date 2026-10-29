package captureflow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const testPageMarker = "/Type /Page "

type pageReading struct {
	InkCoverage float64
	Thumbnail   []byte
	Codes       []string
}

func pdfPage(t *testing.T, shade uint8) []byte {
	t.Helper()
	return testPDF(fmt.Sprintf("%s/Shade %d", testPageMarker, shade))
}

func testPDF(pages ...string) []byte {
	var out bytes.Buffer
	out.WriteString("%PDF-1.7\n")
	for _, page := range pages {
		out.WriteString(page)
		out.WriteByte('\n')
	}
	out.WriteString("%%EOF\n")
	return out.Bytes()
}

func testPages(data []byte) ([]string, error) {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, fmt.Errorf("%w: not a PDF", kernelports.ErrImagingInvalidDocument)
	}
	var pages []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, testPageMarker) {
			pages = append(pages, line)
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("%w: no pages", kernelports.ErrImagingInvalidDocument)
	}
	return pages, nil
}

func mergePDFs(t *testing.T, files ...[]byte) []byte {
	t.Helper()
	var pages []string
	for _, file := range files {
		filePages, err := testPages(file)
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		pages = append(pages, filePages...)
	}
	return testPDF(pages...)
}

func countPages(data []byte) (int, error) {
	pages, err := testPages(data)
	return len(pages), err
}

type fakeImaging struct{ w *world }

func readSource(ctx context.Context, source kernelports.ImagingSource) ([]byte, error) {
	body, err := source.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

func (f *fakeImaging) Probe(
	ctx context.Context,
	req *kernelports.ImagingProbeRequest,
) (*kernelports.ImagingProbe, error) {
	data, err := readSource(ctx, req.Source)
	if err != nil {
		return nil, err
	}
	pages, err := testPages(data)
	if err != nil {
		return nil, err
	}
	return &kernelports.ImagingProbe{Kind: kernelports.ImagingKindPDF, PageCount: len(pages)}, nil
}

func (f *fakeImaging) Render(
	context.Context,
	*kernelports.ImagingRenderRequest,
	func(*kernelports.ImagingArtifact) error,
) (*kernelports.ImagingRenderSummary, error) {
	return nil, errors.New("capture does not render")
}

func (f *fakeImaging) InspectCapture(
	ctx context.Context,
	req *kernelports.ImagingCaptureRequest,
	yield func(*kernelports.ImagingCaptureResult) error,
) (*kernelports.ImagingCaptureSummary, error) {
	summary := &kernelports.ImagingCaptureSummary{}
	for page, err := range req.Pages() {
		if err != nil {
			return nil, err
		}
		data, readErr := readSource(ctx, page.Source)
		if readErr != nil {
			return nil, readErr
		}
		result := &kernelports.ImagingCaptureResult{Index: page.Index}
		f.w.mu.Lock()
		f.w.inspected++
		reading, known := f.w.inspections[string(data)]
		f.w.mu.Unlock()
		switch {
		case len(data) == 0:
			result.Err = &kernelports.ImagingPageError{
				Reason:  kernelports.ErrImagingInvalidDocument,
				Message: "empty",
			}
			summary.Failed++
		case known:
			result.Width, result.Height = 100, 130
			result.Thumbnail = reading.Thumbnail
			result.InkCoverage = reading.InkCoverage
			for _, code := range reading.Codes {
				result.Codes = append(
					result.Codes,
					kernelports.ImagingBarcode{Format: "qr_code", Text: code},
				)
			}
			summary.Inspected++
		default:
			result.Width, result.Height = 100, 130
			result.Thumbnail = []byte("jpeg")
			result.InkCoverage = 0.2
			summary.Inspected++
		}
		if err = yield(result); err != nil {
			return nil, err
		}
	}
	return summary, nil
}

func (f *fakeImaging) Assemble(
	ctx context.Context,
	req *kernelports.ImagingAssembleRequest,
	yield func(*kernelports.ImagingArtifact) error,
) ([]int, error) {
	sources := make([][]string, 0, len(req.Sources))
	for _, source := range req.Sources {
		data, err := readSource(ctx, source)
		if err != nil {
			return nil, err
		}
		pages, err := testPages(data)
		if err != nil {
			return nil, err
		}
		sources = append(sources, pages)
	}
	counts := make([]int, 0, len(req.Outputs))
	for index, output := range req.Outputs {
		var pages []string
		for _, r := range output {
			if r.Source < 0 || r.Source >= len(sources) {
				return nil, fmt.Errorf(
					"%w: no source %d",
					kernelports.ErrImagingInvalidDocument,
					r.Source,
				)
			}
			last := r.Last
			if last == imagingkit.ToLastPage {
				last = len(sources[r.Source]) - 1
			}
			if r.First < 0 || last >= len(sources[r.Source]) || r.First > last {
				return nil, kernelports.ErrImagingPageOutOfRange
			}
			for _, page := range sources[r.Source][r.First : last+1] {
				if r.Rotation != 0 {
					page = fmt.Sprintf("%s /Turn %d", page, r.Rotation)
				}
				pages = append(pages, page)
			}
		}
		data := testPDF(pages...)
		if err := yield(&kernelports.ImagingArtifact{
			Index:       index,
			ContentType: "application/pdf",
			Size:        int64(len(data)),
			Body:        bytes.NewReader(data),
		}); err != nil {
			return nil, err
		}
		counts = append(counts, len(pages))
	}
	return counts, nil
}

func (f *fakeImaging) NormalizeImage(
	context.Context,
	*kernelports.ImagingNormalizeRequest,
	func(*kernelports.ImagingArtifact) error,
) (*kernelports.ImagingNormalizeSummary, error) {
	return nil, errors.New("capture does not normalize images")
}
