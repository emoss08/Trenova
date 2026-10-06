package pdfreader

import (
	"context"
	"errors"
	"fmt"
	"image"
	"math"
	"strings"
	"sync/atomic"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
)

const (
	pointsPerInch = 72
	maxDPI        = 1200
	layoutScale   = 8
)

var (
	errDocumentClosed = errors.New("pdf document is closed")
	errInstanceKilled = errors.New("pdf reader instance was stopped")
)

type document struct {
	instance  pdfium.Pdfium
	handle    references.FPDF_DOCUMENT
	pages     int
	maxPixels int
	opened    bool
	closed    bool
	killed    atomic.Bool
}

var _ services.PDFDocument = (*document)(nil)

func (d *document) PageCount() int { return d.pages }

func (d *document) RenderPage(ctx context.Context, page, dpi int) (*image.RGBA, error) {
	if err := d.checkPage(page); err != nil {
		return nil, err
	}
	if dpi < 1 || dpi > maxDPI {
		return nil, fmt.Errorf("render page %d: dpi %d is outside 1-%d", page, dpi, maxDPI)
	}

	var out *image.RGBA
	err := d.guard(ctx, "render page", func() error {
		size, err := d.instance.GetPageSize(&requests.GetPageSize{Page: d.pageRef(page)})
		if err != nil {
			return err
		}

		rendered, err := d.instance.RenderPageInDPI(&requests.RenderPageInDPI{
			Page: d.pageRef(page),
			DPI:  boundedDPI(size.Width, size.Height, dpi, d.maxPixels),
		})
		if err != nil {
			return err
		}
		defer rendered.Cleanup()

		out, err = opaqueCopy(&rendered.Result)
		return err
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (d *document) PageText(ctx context.Context, page int) (string, error) {
	if err := d.checkPage(page); err != nil {
		return "", err
	}

	var text string
	err := d.guard(ctx, "read page text", func() error {
		resp, err := d.instance.GetPageText(&requests.GetPageText{Page: d.pageRef(page)})
		if err != nil {
			return err
		}
		text = resp.Text
		return nil
	})

	return text, err
}

func (d *document) PageLayout(ctx context.Context, page int) (*services.PDFPageLayout, error) {
	if err := d.checkPage(page); err != nil {
		return nil, err
	}

	var layout *services.PDFPageLayout
	err := d.guard(ctx, "read page layout", func() error {
		size, err := d.instance.GetPageSize(&requests.GetPageSize{Page: d.pageRef(page)})
		if err != nil {
			return err
		}

		structured, err := d.instance.GetPageTextStructured(&requests.GetPageTextStructured{
			Page: d.pageRef(page),
			Mode: requests.GetPageTextStructuredModeRects,
		})
		if err != nil {
			return err
		}

		runs := make([]textRun, 0, len(structured.Rects))
		for _, rect := range structured.Rects {
			text := strings.Join(strings.Fields(rect.Text), " ")
			if text == "" {
				continue
			}

			box, boxErr := d.deviceBox(page, size, rect.PointPosition)
			if boxErr != nil {
				return boxErr
			}
			box.text = text
			runs = append(runs, box)
		}

		layout = &services.PDFPageLayout{
			Width:  size.Width,
			Height: size.Height,
			Lines:  groupLines(runs),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return layout, nil
}

func (d *document) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true

	if d.killed.Load() {
		return nil
	}

	var err error
	if d.opened {
		if _, closeErr := d.instance.FPDF_CloseDocument(
			&requests.FPDF_CloseDocument{Document: d.handle},
		); closeErr != nil {
			err = fmt.Errorf("close pdf: %w", closeErr)
		}
	}

	return errors.Join(err, d.instance.Close())
}

func (d *document) release() {
	d.closed = true
	if !d.killed.Load() {
		_ = d.instance.Close()
	}
}

func (d *document) guard(ctx context.Context, op string, fn func() error) error {
	if d.closed {
		return fmt.Errorf("%s: %w", op, errDocumentClosed)
	}
	if d.killed.Load() {
		return fmt.Errorf("%s: %w", op, errInstanceKilled)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	stop := context.AfterFunc(ctx, func() {
		if d.killed.CompareAndSwap(false, true) {
			_ = d.instance.Kill()
		}
	})
	err := fn()
	stop()

	if d.killed.Load() {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%s: %w", op, ctxErr)
		}
		return fmt.Errorf("%s: %w", op, errInstanceKilled)
	}
	if err != nil {
		return fmt.Errorf("%s: %w: %w", op, services.ErrPDFUnreadable, err)
	}

	return nil
}

func (d *document) checkPage(page int) error {
	if page < 0 || page >= d.pages {
		return fmt.Errorf("page %d is outside the document's %d pages", page, d.pages)
	}
	return nil
}

func (d *document) pageRef(page int) requests.Page {
	return requests.Page{ByIndex: &requests.PageByIndex{Document: d.handle, Index: page}}
}

func (d *document) deviceBox(
	page int,
	size *responses.GetPageSize,
	pos responses.CharPosition,
) (textRun, error) {
	sizeX := int(math.Round(size.Width * layoutScale))
	sizeY := int(math.Round(size.Height * layoutScale))

	toDevice := func(x, y float64) (float64, float64, error) {
		resp, err := d.instance.FPDF_PageToDevice(&requests.FPDF_PageToDevice{
			Page:  d.pageRef(page),
			SizeX: sizeX,
			SizeY: sizeY,
			PageX: x,
			PageY: y,
		})
		if err != nil {
			return 0, 0, err
		}
		return float64(resp.DeviceX) / layoutScale, float64(resp.DeviceY) / layoutScale, nil
	}

	x1, y1, err := toDevice(pos.Left, pos.Top)
	if err != nil {
		return textRun{}, err
	}
	x2, y2, err := toDevice(pos.Right, pos.Bottom)
	if err != nil {
		return textRun{}, err
	}

	return textRun{
		left:   min(x1, x2),
		top:    min(y1, y2),
		right:  max(x1, x2),
		bottom: max(y1, y2),
	}, nil
}

func boundedDPI(widthPt, heightPt float64, dpi, maxPixels int) int {
	if maxPixels <= 0 || widthPt <= 0 || heightPt <= 0 {
		return dpi
	}

	scale := float64(dpi) / pointsPerInch
	pixels := widthPt * scale * heightPt * scale
	if pixels <= float64(maxPixels) {
		return dpi
	}

	return max(1, int(float64(dpi)*math.Sqrt(float64(maxPixels)/pixels)))
}

func opaqueCopy(result *responses.RenderPage) (*image.RGBA, error) {
	src, ok := result.RenderedImage.(*image.RGBA)
	if !ok || src == nil {
		return nil, fmt.Errorf("unexpected rendered image type %T", result.RenderedImage)
	}

	bounds := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	rowBytes := bounds.Dx() * 4
	for y := range bounds.Dy() {
		srcOff := src.PixOffset(bounds.Min.X, bounds.Min.Y+y)
		copy(out.Pix[y*out.Stride:y*out.Stride+rowBytes], src.Pix[srcOff:srcOff+rowBytes])
	}

	if result.HasTransparency {
		flattenOnWhite(out.Pix)
	}

	return out, nil
}

func flattenOnWhite(pix []uint8) {
	for i := 0; i+3 < len(pix); i += 4 {
		a := uint32(pix[i+3])
		if a == 0xff {
			continue
		}
		pix[i] = blendOnWhite(pix[i], a)
		pix[i+1] = blendOnWhite(pix[i+1], a)
		pix[i+2] = blendOnWhite(pix[i+2], a)
		pix[i+3] = 0xff
	}
}

func blendOnWhite(c uint8, a uint32) uint8 {
	blended := (uint32(c)*a + 0xff*(0xff-a)) / 0xff
	return uint8(blended) //nolint:gosec // a <= 255 keeps the blend within 0-255
}
