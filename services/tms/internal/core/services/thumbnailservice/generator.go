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
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/gen2brain/webp"
)

var (
	ErrPDFHasNoPages        = errors.New("PDF has no pages")
	ErrPDFReaderUnavailable = errors.New("PDF reader is not configured")
)

const (
	DefaultMaxWidth    = 300
	DefaultMaxHeight   = 400
	DefaultWebPQuality = 80

	renderDPI     = 96
	renderTimeout = 90 * time.Second
)

type Generator struct {
	maxWidth    int
	maxHeight   int
	webpQuality int
	pdfReader   services.PDFReader
}

func NewGenerator(pdfReader services.PDFReader) *Generator {
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
	ct := strings.ToLower(contentType)

	if ct == "application/pdf" {
		return g.generateFromPDF(ctx, reader)
	}

	if strings.HasPrefix(ct, "image/") {
		return g.generateFromImage(reader)
	}

	return nil, fmt.Errorf("unsupported content type: %s", contentType)
}

func (g *Generator) generateFromImage(reader io.Reader) ([]byte, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return g.encodeThumbnail(img)
}

func (g *Generator) generateFromPDF(ctx context.Context, reader io.Reader) ([]byte, error) {
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

	img, err := doc.RenderPage(renderCtx, 0, renderDPI)
	if err != nil {
		return nil, fmt.Errorf("failed to render PDF page: %w", err)
	}

	return g.encodeThumbnail(img)
}

func (g *Generator) encodeThumbnail(img image.Image) ([]byte, error) {
	thumbnail := imaging.Fit(img, g.maxWidth, g.maxHeight, imaging.Lanczos)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, thumbnail, webp.Options{Quality: g.webpQuality}); err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return buf.Bytes(), nil
}
