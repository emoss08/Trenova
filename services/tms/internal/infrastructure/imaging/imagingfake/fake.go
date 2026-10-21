package imagingfake

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type Fake struct {
	mu sync.Mutex

	ProbeFunc  func(ctx context.Context, req *kernelports.ImagingProbeRequest) (*kernelports.ImagingProbe, error)
	RenderFunc func(
		ctx context.Context,
		req *kernelports.ImagingRenderRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingRenderSummary, error)
	InspectCaptureFunc func(
		ctx context.Context,
		req *kernelports.ImagingCaptureRequest,
		yield func(*kernelports.ImagingCaptureResult) error,
	) (*kernelports.ImagingCaptureSummary, error)
	AssembleFunc func(
		ctx context.Context,
		req *kernelports.ImagingAssembleRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) ([]int, error)
	NormalizeImageFunc func(
		ctx context.Context,
		req *kernelports.ImagingNormalizeRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingNormalizeSummary, error)

	PageCount int
	Calls     []string
}

var _ kernelports.Imaging = (*Fake)(nil)

func New() *Fake {
	return &Fake{PageCount: 1}
}

func (f *Fake) record(call string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, call)
}

func (f *Fake) CallCount(call string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.Calls {
		if c == call {
			n++
		}
	}
	return n
}

func Read(ctx context.Context, source kernelports.ImagingSource) ([]byte, error) {
	body, err := source.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

func (f *Fake) Probe(
	ctx context.Context,
	req *kernelports.ImagingProbeRequest,
) (*kernelports.ImagingProbe, error) {
	f.record("probe")
	if f.ProbeFunc != nil {
		return f.ProbeFunc(ctx, req)
	}
	if _, err := Read(ctx, req.Source); err != nil {
		return nil, err
	}
	pages := make([]kernelports.ImagingPageInfo, 0, f.PageCount)
	for i := range f.PageCount {
		pages = append(pages, kernelports.ImagingPageInfo{
			Index: i, WidthPoints: 612, HeightPoints: 792, HasText: true, CharCount: 100,
		})
	}
	return &kernelports.ImagingProbe{
		Kind:      kernelports.ImagingKindPDF,
		PageCount: f.PageCount,
		Pages:     pages,
	}, nil
}

func (f *Fake) Render(
	ctx context.Context,
	req *kernelports.ImagingRenderRequest,
	yield func(*kernelports.ImagingArtifact) error,
) (*kernelports.ImagingRenderSummary, error) {
	f.record("render")
	if f.RenderFunc != nil {
		return f.RenderFunc(ctx, req, yield)
	}
	if _, err := Read(ctx, req.Source); err != nil {
		return nil, err
	}
	pages := req.Pages
	if len(pages) == 0 {
		for i := range f.PageCount {
			pages = append(pages, i)
		}
	}
	rendered := 0
	for _, page := range pages {
		if page >= f.PageCount {
			if err := yield(&kernelports.ImagingArtifact{
				Index: page,
				Err: &kernelports.ImagingPageError{
					Reason:  kernelports.ErrImagingPageOutOfRange,
					Message: kernelports.ErrImagingPageOutOfRange.Error(),
				},
			}); err != nil {
				return nil, err
			}
			continue
		}
		data := []byte("RIFF-fake-page")
		if err := yield(&kernelports.ImagingArtifact{
			Index:       page,
			ContentType: "image/webp",
			Width:       max(req.FitWidth, 1),
			Height:      max(req.FitHeight, 1),
			Size:        int64(len(data)),
			Body:        bytes.NewReader(data),
		}); err != nil {
			return nil, err
		}
		rendered++
	}
	return &kernelports.ImagingRenderSummary{PageCount: f.PageCount, Rendered: rendered}, nil
}

func (f *Fake) InspectCapture(
	ctx context.Context,
	req *kernelports.ImagingCaptureRequest,
	yield func(*kernelports.ImagingCaptureResult) error,
) (*kernelports.ImagingCaptureSummary, error) {
	f.record("inspect_capture")
	if f.InspectCaptureFunc != nil {
		return f.InspectCaptureFunc(ctx, req, yield)
	}
	inspected := 0
	for page, err := range req.Pages() {
		if err != nil {
			return nil, err
		}
		if _, err = Read(ctx, page.Source); err != nil {
			return nil, err
		}
		if err = yield(&kernelports.ImagingCaptureResult{
			Index:                page.Index,
			Width:                1275,
			Height:               1650,
			Thumbnail:            []byte("fake-thumbnail"),
			ThumbnailContentType: "image/jpeg",
			InkCoverage:          0.05,
		}); err != nil {
			return nil, err
		}
		inspected++
	}
	return &kernelports.ImagingCaptureSummary{Inspected: inspected}, nil
}

func (f *Fake) Assemble(
	ctx context.Context,
	req *kernelports.ImagingAssembleRequest,
	yield func(*kernelports.ImagingArtifact) error,
) ([]int, error) {
	f.record("assemble")
	if f.AssembleFunc != nil {
		return f.AssembleFunc(ctx, req, yield)
	}
	for _, source := range req.Sources {
		if _, err := Read(ctx, source); err != nil {
			return nil, err
		}
	}
	counts := make([]int, 0, len(req.Outputs))
	for index, output := range req.Outputs {
		pages := 0
		for _, r := range output {
			pages += r.Last - r.First + 1
		}
		data := []byte("%PDF-1.7 fake")
		if err := yield(&kernelports.ImagingArtifact{
			Index:       index,
			ContentType: "application/pdf",
			Size:        int64(len(data)),
			Body:        bytes.NewReader(data),
		}); err != nil {
			return nil, err
		}
		counts = append(counts, pages)
	}
	return counts, nil
}

func (f *Fake) NormalizeImage(
	ctx context.Context,
	req *kernelports.ImagingNormalizeRequest,
	yield func(*kernelports.ImagingArtifact) error,
) (*kernelports.ImagingNormalizeSummary, error) {
	f.record("normalize_image")
	if f.NormalizeImageFunc != nil {
		return f.NormalizeImageFunc(ctx, req, yield)
	}
	if _, err := Read(ctx, req.Source); err != nil {
		return nil, err
	}
	data := []byte("\x89PNG fake")
	if err := yield(&kernelports.ImagingArtifact{
		ContentType: "image/png",
		Width:       1,
		Height:      1,
		Size:        int64(len(data)),
		Body:        bytes.NewReader(data),
	}); err != nil {
		return nil, err
	}
	return &kernelports.ImagingNormalizeSummary{
		SourceFormat: "webp",
		SourceWidth:  1,
		SourceHeight: 1,
	}, nil
}
