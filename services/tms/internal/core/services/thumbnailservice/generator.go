package thumbnailservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/go-fitz"
)

var (
	ErrPDFHasNoPages   = errors.New("PDF has no pages")
	ErrRendererCrashed = errors.New("PDF renderer crashed")
)

const (
	DefaultMaxWidth    = 300
	DefaultMaxHeight   = 400
	DefaultWebPQuality = 80

	RenderCommandName = "render-thumbnail"
	RenderExitNoPages = 3

	renderTimeout        = 90 * time.Second
	maxRenderErrorOutput = 4096
)

type Generator struct {
	maxWidth     int
	maxHeight    int
	webpQuality  float32
	pdfInProcess bool
}

func NewGenerator() *Generator {
	return &Generator{
		maxWidth:    DefaultMaxWidth,
		maxHeight:   DefaultMaxHeight,
		webpQuality: DefaultWebPQuality,
	}
}

func NewInProcessGenerator() *Generator {
	g := NewGenerator()
	g.pdfInProcess = true
	return g
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
		if g.pdfInProcess {
			return g.generateFromPDF(reader)
		}
		return g.generateFromPDFIsolated(ctx, reader)
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

	return g.encodeThumbnail(img)
}

func (g *Generator) generateFromPDFIsolated(
	ctx context.Context,
	reader io.Reader,
) ([]byte, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF data: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	renderCtx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(
		renderCtx,
		exe,
		RenderCommandName,
		"--content-type",
		"application/pdf",
	)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		if ctxErr := renderCtx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("PDF rendering timed out: %w", ctxErr)
		}

		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			switch exitErr.ExitCode() {
			case RenderExitNoPages:
				return nil, ErrPDFHasNoPages
			case -1:
				return nil, fmt.Errorf(
					"%w: %s: %s",
					ErrRendererCrashed,
					exitErr.String(),
					renderErrorOutput(&stderr),
				)
			default:
				return nil, fmt.Errorf("failed to render PDF: %s", renderErrorOutput(&stderr))
			}
		}

		return nil, fmt.Errorf("failed to run PDF renderer: %w", err)
	}

	return stdout.Bytes(), nil
}

func (g *Generator) encodeThumbnail(img image.Image) ([]byte, error) {
	thumbnail := imaging.Fit(img, g.maxWidth, g.maxHeight, imaging.Lanczos)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, thumbnail, &webp.Options{Quality: g.webpQuality}); err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return buf.Bytes(), nil
}

func renderErrorOutput(buf *bytes.Buffer) string {
	s := strings.TrimSpace(buf.String())
	if s == "" {
		return "no error output"
	}
	if len(s) > maxRenderErrorOutput {
		s = s[len(s)-maxRenderErrorOutput:]
	}
	return s
}
