package thumbnailservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/gen2brain/webp"
)

var (
	ErrPDFHasNoPages        = errors.New("PDF has no pages")
	ErrPDFReaderUnavailable = errors.New("PDF reader is not configured")
	ErrPageOutOfRange       = errors.New("the document has no such page")
)

const (
	DefaultMaxWidth    = 300
	DefaultMaxHeight   = 400
	DefaultWebPQuality = 80

	PageMaxWidth    = 1400
	PageMaxHeight   = 1800
	pageWebPQuality = 82
	pageRenderDPI   = 150

	renderDPI     = 96
	renderTimeout = 90 * time.Second
)

type frame struct {
	maxWidth  int
	maxHeight int
	quality   int
	dpi       int
}

var pageFrame = frame{
	maxWidth:  PageMaxWidth,
	maxHeight: PageMaxHeight,
	quality:   pageWebPQuality,
	dpi:       pageRenderDPI,
}

type Generator struct {
	maxWidth    int
	maxHeight   int
	webpQuality int
	pdfReader   kernelports.PDFReader
}

func NewGenerator(pdfReader kernelports.PDFReader) *Generator {
	return &Generator{
		maxWidth:    DefaultMaxWidth,
		maxHeight:   DefaultMaxHeight,
		webpQuality: DefaultWebPQuality,
		pdfReader:   pdfReader,
	}
}

func (g *Generator) SupportsThumbnail(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "image/") || ct == "application/pdf"
}

func (g *Generator) Generate(
	ctx context.Context,
	reader io.Reader,
	contentType string,
) ([]byte, error) {
	return g.render(ctx, reader, contentType, 0, frame{
		maxWidth:  g.maxWidth,
		maxHeight: g.maxHeight,
		quality:   g.webpQuality,
		dpi:       renderDPI,
	})
}

func (g *Generator) RenderPage(
	ctx context.Context,
	reader io.Reader,
	contentType string,
	page int,
) ([]byte, error) {
	if page < 1 {
		return nil, ErrPageOutOfRange
	}

	return g.render(ctx, reader, contentType, page-1, pageFrame)
}

func (g *Generator) render(
	ctx context.Context,
	reader io.Reader,
	contentType string,
	pageIndex int,
	f frame,
) ([]byte, error) {
	ct := strings.ToLower(contentType)

	if ct == "application/pdf" {
		return g.renderPDF(ctx, reader, pageIndex, f)
	}

	if strings.HasPrefix(ct, "image/") {
		if pageIndex != 0 {
			return nil, ErrPageOutOfRange
		}
		return g.renderImage(reader, f)
	}

	return nil, fmt.Errorf("unsupported content type: %s", contentType)
}

func (g *Generator) renderImage(reader io.Reader, f frame) ([]byte, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return encode(img, f)
}

func (g *Generator) renderPDF(
	ctx context.Context,
	reader io.Reader,
	pageIndex int,
	f frame,
) ([]byte, error) {
	if g.pdfReader == nil {
		return nil, ErrPDFReaderUnavailable
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF data: %w", err)
	}

	renderCtx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()

	doc, err := g.pdfReader.Open(renderCtx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	if doc.PageCount() == 0 {
		return nil, ErrPDFHasNoPages
	}
	if pageIndex >= doc.PageCount() {
		return nil, ErrPageOutOfRange
	}

	img, err := doc.RenderPage(renderCtx, pageIndex, f.dpi)
	if err != nil {
		return nil, fmt.Errorf("failed to render PDF page: %w", err)
	}

	return encode(img, f)
}

func encode(img image.Image, f frame) ([]byte, error) {
	fitted := imaging.Fit(img, f.maxWidth, f.maxHeight, imaging.Lanczos)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, fitted, webp.Options{Quality: f.quality}); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}
