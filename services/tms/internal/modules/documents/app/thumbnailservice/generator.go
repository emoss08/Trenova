package thumbnailservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

var (
	ErrPDFHasNoPages        = errors.New("PDF has no pages")
	ErrPageOutOfRange       = errors.New("the document has no such page")
	ErrUnreadable           = errors.New("the document could not be read")
	ErrImagingNotConfigured = errors.New("the imaging service is not configured")
)

const (
	ThumbnailWidth   = 240
	ThumbnailHeight  = 320
	thumbnailQuality = 80

	PageMaxWidth  = 1400
	PageMaxHeight = 1800
	pageQuality   = 82

	maxRenderBytes = 32 << 20
)

type Generator struct {
	imaging kernelports.Imaging
}

func NewGenerator(imaging kernelports.Imaging) *Generator {
	return &Generator{imaging: imaging}
}

func (g *Generator) SupportsThumbnail(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "image/") || ct == "application/pdf"
}

func (g *Generator) Generate(
	ctx context.Context,
	source kernelports.ImagingSource,
	priority kernelports.ImagingPriority,
) ([]byte, error) {
	var out []byte
	err := g.render(
		ctx,
		source,
		[]int{0},
		ThumbnailWidth,
		ThumbnailHeight,
		thumbnailQuality,
		priority,
		func(_ int, data []byte) error {
			out = data
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, ErrPDFHasNoPages
	}
	return out, nil
}

func (g *Generator) RenderPage(
	ctx context.Context,
	source kernelports.ImagingSource,
	page int,
	priority kernelports.ImagingPriority,
) ([]byte, error) {
	if page < 1 {
		return nil, ErrPageOutOfRange
	}

	var out []byte
	err := g.render(
		ctx,
		source,
		[]int{page - 1},
		PageMaxWidth,
		PageMaxHeight,
		pageQuality,
		priority,
		func(_ int, data []byte) error {
			out = data
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, ErrPageOutOfRange
	}
	return out, nil
}

func (g *Generator) RenderPages(
	ctx context.Context,
	source kernelports.ImagingSource,
	maxPages int,
	priority kernelports.ImagingPriority,
	yield func(page int, data []byte) error,
) (int, error) {
	return g.RenderPagesFrom(ctx, source, 1, maxPages, priority, yield)
}

func (g *Generator) RenderPagesFrom(
	ctx context.Context,
	source kernelports.ImagingSource,
	first, maxPages int,
	priority kernelports.ImagingPriority,
	yield func(page int, data []byte) error,
) (int, error) {
	if g.imaging == nil {
		return 0, ErrImagingNotConfigured
	}
	req := &kernelports.ImagingRenderRequest{
		Source:    source,
		FitWidth:  PageMaxWidth,
		FitHeight: PageMaxHeight,
		Format:    kernelports.ImagingWebP,
		Quality:   pageQuality,
		MaxPages:  maxPages,
		Priority:  priority,
	}
	if first > 1 {
		if first > maxPages {
			return 0, nil
		}
		req.Pages = make([]int, 0, maxPages-first+1)
		for page := first; page <= maxPages; page++ {
			req.Pages = append(req.Pages, page-1)
		}
		req.MaxPages = 0
	}
	summary, err := g.imaging.Render(ctx, req, func(artifact *kernelports.ImagingArtifact) error {
		if artifact.Body == nil {
			return nil
		}
		data, readErr := readArtifact(artifact)
		if readErr != nil {
			return readErr
		}
		return yield(artifact.Index+1, data)
	})
	if err != nil {
		return 0, classify(err)
	}
	return summary.PageCount, nil
}

func (g *Generator) render(
	ctx context.Context,
	source kernelports.ImagingSource,
	pages []int,
	width, height, quality int,
	priority kernelports.ImagingPriority,
	yield func(page int, data []byte) error,
) error {
	if g.imaging == nil {
		return ErrImagingNotConfigured
	}
	var pageErr error
	summary, err := g.imaging.Render(ctx, &kernelports.ImagingRenderRequest{
		Source:    source,
		Pages:     pages,
		FitWidth:  width,
		FitHeight: height,
		Format:    kernelports.ImagingWebP,
		Quality:   quality,
		Priority:  priority,
	}, func(artifact *kernelports.ImagingArtifact) error {
		if artifact.Body == nil {
			pageErr = artifact.Err
			return nil
		}
		data, readErr := readArtifact(artifact)
		if readErr != nil {
			return readErr
		}
		return yield(artifact.Index+1, data)
	})
	if err != nil {
		return classify(err)
	}
	if summary.PageCount == 0 {
		return ErrPDFHasNoPages
	}
	if pageErr != nil {
		return classify(pageErr)
	}
	return nil
}

func readArtifact(artifact *kernelports.ImagingArtifact) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(artifact.Body, maxRenderBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read the rendered page: %w", err)
	}
	if len(data) > maxRenderBytes {
		return nil, fmt.Errorf("the rendered page is larger than %d bytes", maxRenderBytes)
	}
	return data, nil
}

func classify(err error) error {
	switch {
	case errors.Is(err, kernelports.ErrImagingPageOutOfRange):
		return fmt.Errorf("%w: %w", ErrPageOutOfRange, err)
	case errors.Is(err, kernelports.ErrImagingInvalidDocument),
		errors.Is(err, kernelports.ErrImagingEncrypted),
		errors.Is(err, kernelports.ErrImagingUnsupported),
		errors.Is(err, kernelports.ErrImagingPixelBudget),
		errors.Is(err, kernelports.ErrImagingTooLarge):
		return fmt.Errorf("%w: %w", ErrUnreadable, err)
	default:
		return err
	}
}
