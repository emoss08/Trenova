package pdfreader_test

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/pdfreader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPage struct {
	width, height float64
	rotate        int
	content       string
	transparent   bool
}

func buildPDF(pages ...testPage) []byte {
	var objects []string
	add := func(body string) int {
		objects = append(objects, body)
		return len(objects)
	}

	catalog := add("")
	pagesObj := add("")
	font := add("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	gstate := add("<< /Type /ExtGState /ca 0.5 /CA 0.5 >>")

	kids := make([]string, 0, len(pages))
	for _, page := range pages {
		stream := add(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(page.content), page.content))
		extra := ""
		if page.rotate != 0 {
			extra += fmt.Sprintf(" /Rotate %d", page.rotate)
		}
		if page.transparent {
			extra += " /Group << /Type /Group /S /Transparency /CS /DeviceRGB >>"
		}
		pageObj := add(fmt.Sprintf(
			"<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %g %g] /Contents %d 0 R "+
				"/Resources << /Font << /F1 %d 0 R >> /ExtGState << /GS1 %d 0 R >> >>%s >>",
			pagesObj, page.width, page.height, stream, font, gstate, extra,
		))
		kids = append(kids, fmt.Sprintf("%d 0 R", pageObj))
	}
	objects[catalog-1] = fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesObj)
	objects[pagesObj-1] = fmt.Sprintf(
		"<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(kids),
	)

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i, body := range objects {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, catalog, xref)

	return buf.Bytes()
}

func letterPage(content string) testPage {
	return testPage{width: 612, height: 792, content: content}
}

const invoiceContent = "BT /F1 12 Tf 72 700 Td (Rate: $1,359.56) Tj ET\n" +
	"BT /F1 12 Tf 400 700 Td (PO 77812) Tj ET\n" +
	"BT /F1 12 Tf 72 680 Td (Line two) Tj ET\n" +
	"0 0 1 rg 300 300 50 50 re f"

func newReader(t *testing.T, cfg *config.PDFReaderConfig) *pdfreader.Reader {
	t.Helper()
	reader := pdfreader.NewReader(cfg, nil)
	t.Cleanup(func() { require.NoError(t, reader.Close()) })
	return reader
}

func openDoc(t *testing.T, reader *pdfreader.Reader, data []byte) services.PDFDocument {
	t.Helper()
	doc, err := reader.Open(t.Context(), data)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, doc.Close()) })
	return doc
}

func TestReaderReadsTextAndLayout(t *testing.T) {
	t.Parallel()

	reader := newReader(t, nil)
	doc := openDoc(t, reader, buildPDF(letterPage(invoiceContent)))

	require.Equal(t, 1, doc.PageCount())

	text, err := doc.PageText(t.Context(), 0)
	require.NoError(t, err)
	assert.Contains(t, text, "Rate: $1,359.56")
	assert.Contains(t, text, "PO 77812")

	layout, err := doc.PageLayout(t.Context(), 0)
	require.NoError(t, err)
	assert.InDelta(t, 612, layout.Width, 0.01)
	assert.InDelta(t, 792, layout.Height, 0.01)
	require.Len(t, layout.Lines, 3)

	rate, po, second := layout.Lines[0], layout.Lines[1], layout.Lines[2]
	assert.Equal(t, "Rate: $1,359.56", rate.Text)
	assert.InDelta(t, 72, rate.Left, 2)
	assert.InDelta(t, 792-700-9, rate.Top, 4)
	assert.Greater(t, rate.Width, 60.0)
	assert.Less(t, rate.Width, 100.0)
	assert.Greater(t, rate.Height, 6.0)

	assert.Equal(t, "PO 77812", po.Text)
	assert.InDelta(t, 400, po.Left, 2)
	assert.InDelta(t, rate.Top, po.Top, 1)

	assert.Equal(t, "Line two", second.Text)
	assert.Greater(t, second.Top, rate.Top+10)
}

func TestReaderMapsRotatedPagesToTheirDisplayedOrientation(t *testing.T) {
	t.Parallel()

	reader := newReader(t, nil)
	page := letterPage("BT /F1 12 Tf 72 700 Td (Rotated) Tj ET")
	page.rotate = 90
	doc := openDoc(t, reader, buildPDF(page))

	layout, err := doc.PageLayout(t.Context(), 0)
	require.NoError(t, err)
	assert.InDelta(t, 792, layout.Width, 0.01)
	assert.InDelta(t, 612, layout.Height, 0.01)
	require.Len(t, layout.Lines, 1)

	line := layout.Lines[0]
	assert.InDelta(t, 72, line.Top, 4)
	assert.Greater(t, line.Height, line.Width/2)
	assert.LessOrEqual(t, line.Left+line.Width, layout.Width)

	img, err := doc.RenderPage(t.Context(), 0, 72)
	require.NoError(t, err)
	assert.Equal(t, 792, img.Bounds().Dx())
	assert.Equal(t, 612, img.Bounds().Dy())
}

func TestReaderRendersOpaquePages(t *testing.T) {
	t.Parallel()

	reader := newReader(t, nil)
	transparent := letterPage("/GS1 gs 1 0 0 rg 100 100 200 200 re f")
	transparent.transparent = true
	doc := openDoc(t, reader, buildPDF(letterPage(invoiceContent), transparent))

	img, err := doc.RenderPage(t.Context(), 0, 72)
	require.NoError(t, err)
	assert.Equal(t, 612, img.Bounds().Dx())
	assert.Equal(t, 792, img.Bounds().Dy())
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, img.RGBAAt(5, 5))
	assert.Equal(t, color.RGBA{0, 0, 255, 255}, img.RGBAAt(325, 792-325))

	hiDPI, err := doc.RenderPage(t.Context(), 0, 144)
	require.NoError(t, err)
	assert.Equal(t, 1224, hiDPI.Bounds().Dx())

	flat, err := doc.RenderPage(t.Context(), 1, 72)
	require.NoError(t, err)
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, flat.RGBAAt(5, 5))
	tinted := flat.RGBAAt(200, 792-200)
	assert.Equal(t, uint8(255), tinted.A)
	assert.Greater(t, tinted.R, uint8(200))
	assert.Greater(t, tinted.G, uint8(100))
	assert.Less(t, tinted.G, uint8(160))
}

func TestReaderCapsRenderSize(t *testing.T) {
	t.Parallel()

	reader := newReader(t, &config.PDFReaderConfig{MaxRenderPixels: 100_000})
	doc := openDoc(t, reader, buildPDF(letterPage(invoiceContent)))

	img, err := doc.RenderPage(t.Context(), 0, 300)
	require.NoError(t, err)
	assert.LessOrEqual(t, img.Bounds().Dx()*img.Bounds().Dy(), 100_000)
	assert.Greater(t, img.Bounds().Dx()*img.Bounds().Dy(), 50_000)
}

func TestReaderRejectsBadInput(t *testing.T) {
	t.Parallel()

	reader := newReader(t, nil)

	_, err := reader.Open(t.Context(), nil)
	require.ErrorIs(t, err, services.ErrPDFUnreadable)

	_, err = reader.Open(t.Context(), []byte("this is not a pdf"))
	require.ErrorIs(t, err, services.ErrPDFUnreadable)

	doc := openDoc(t, reader, buildPDF(letterPage(invoiceContent)))
	_, err = doc.RenderPage(t.Context(), 1, 72)
	require.Error(t, err)
	_, err = doc.RenderPage(t.Context(), 0, 0)
	require.Error(t, err)
	_, err = doc.PageText(t.Context(), -1)
	require.Error(t, err)
}

func TestReaderStopsOnCancelledContext(t *testing.T) {
	t.Parallel()

	reader := newReader(t, nil)
	doc, err := reader.Open(t.Context(), buildPDF(letterPage(invoiceContent)))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = doc.RenderPage(ctx, 0, 72)
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, doc.Close())

	again := openDoc(t, reader, buildPDF(letterPage(invoiceContent)))
	_, err = again.PageText(t.Context(), 0)
	require.NoError(t, err)
}

func TestReaderServesConcurrentDocuments(t *testing.T) {
	t.Parallel()

	reader := newReader(t, &config.PDFReaderConfig{MaxInstances: 2})
	data := buildPDF(letterPage(invoiceContent))

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Go(func() {
			doc, err := reader.Open(t.Context(), data)
			if err != nil {
				errs <- err
				return
			}
			defer doc.Close()

			text, err := doc.PageText(t.Context(), 0)
			if err == nil && !strings.Contains(text, "PO 77812") {
				err = fmt.Errorf("unexpected text %q", text)
			}
			errs <- err
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
}

func TestReaderRefusesWorkAfterClose(t *testing.T) {
	t.Parallel()

	reader := pdfreader.NewReader(nil, nil)
	require.NoError(t, reader.Close())

	_, err := reader.Open(t.Context(), buildPDF(letterPage(invoiceContent)))
	require.Error(t, err)
}
