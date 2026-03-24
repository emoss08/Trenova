package thumbnailservice

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/go-fitz"
)

var ErrPDFHasNoPages = errors.New("PDF has no pages")

const (
	DefaultMaxWidth    = 300
	DefaultMaxHeight   = 400
	DefaultWebPQuality = 80
)

type Generator struct {
	maxWidth    int
	maxHeight   int
	webpQuality float32
}

func NewGenerator() *Generator {
	return &Generator{
		maxWidth:    DefaultMaxWidth,
		maxHeight:   DefaultMaxHeight,
		webpQuality: DefaultWebPQuality,
	}
}

func (g *Generator) SupportsThumbnail(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "image/") || ct == "application/pdf"
}

func (g *Generator) Generate(reader io.Reader, contentType string) ([]byte, error) {
	ct := strings.ToLower(contentType)

	if ct == "application/pdf" {
		return g.generateFromPDF(reader)
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

	thumbnail := imaging.Fit(img, g.maxWidth, g.maxHeight, imaging.Lanczos)

	var buf bytes.Buffer
	if err = webp.Encode(&buf, thumbnail, &webp.Options{Quality: g.webpQuality}); err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *Generator) generateFromPDF(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF data: %w", err)
	}

	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	if doc.NumPage() == 0 {
		return nil, ErrPDFHasNoPages
	}

	img, err := doc.Image(0)
	if err != nil {
		return nil, fmt.Errorf("failed to render PDF page: %w", err)
	}

	thumbnail := imaging.Fit(img, g.maxWidth, g.maxHeight, imaging.Lanczos)

	var buf bytes.Buffer
	if err = webp.Encode(&buf, thumbnail, &webp.Options{Quality: g.webpQuality}); err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return buf.Bytes(), nil
}
