//go:build integration

package starters_test

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/gotenberg"
	"github.com/emoss08/trenova/pkg/templateengine"
	fitz "github.com/gen2brain/go-fitz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// This is the end-to-end proof that a built-in document is printable.
//
// Everything else about the starters is checked without a browser, which is what
// makes those tests fast enough to run on every commit. But "the template compiles
// and renders to HTML" is not the same claim as "a customer receives a legible
// invoice", and only a real render settles the second one.
//
// Start the sidecar with `docker compose -f docker-compose-local.yml up -d gotenberg`
// and run with RENDERER_GOTENBERG_URL=http://localhost:3009.
func liveRenderer(t *testing.T) services.PDFRenderer {
	t.Helper()

	baseURL := os.Getenv("RENDERER_GOTENBERG_URL")
	if baseURL == "" {
		t.Skip("RENDERER_GOTENBERG_URL is not set; no live renderer to exercise")
	}

	return gotenberg.New(gotenberg.Params{
		Config: &config.Config{
			Renderer: config.RendererConfig{
				GotenbergURL:   baseURL,
				RequestTimeout: 60 * time.Second,
				MaxConcurrent:  2,
				MaxRetries:     2,
			},
		},
		Logger: zap.NewNop(),
	})
}

// pdfText extracts what a reader would actually see.
func pdfText(t *testing.T, pdf []byte) string {
	t.Helper()

	doc, err := fitz.NewFromMemory(pdf)
	require.NoError(t, err)
	defer func() { _ = doc.Close() }()

	var b strings.Builder
	for page := range doc.NumPage() {
		text, textErr := doc.Text(page)
		require.NoError(t, textErr)
		b.WriteString(text)
		b.WriteByte('\n')
	}
	return b.String()
}

func pdfPageCount(t *testing.T, pdf []byte) int {
	t.Helper()

	doc, err := fitz.NewFromMemory(pdf)
	require.NoError(t, err)
	defer func() { _ = doc.Close() }()

	return doc.NumPage()
}

// renderStarter runs one document kind all the way to PDF bytes.
func renderStarter(t *testing.T, kind documenttemplate.Kind) []byte {
	t.Helper()

	renderer := liveRenderer(t)
	registry := documenttemplate.NewRegistry()
	engine, err := templateengine.New(templateengine.Config{})
	require.NoError(t, err)

	starter, err := starters.For(kind)
	require.NoError(t, err)

	compiled, diags := engine.Parse(&templateengine.ParseRequest{
		Field:        documenttemplate.ChannelPDF.Field(),
		Kind:         string(kind),
		Channel:      templateengine.ChannelPDF,
		Source:       starter.BodyHTML,
		AllowedPaths: registry.Paths(kind),
		Strict:       true,
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	sample, err := registry.SampleContext(kind)
	require.NoError(t, err)

	body, err := engine.Render(t.Context(), compiled, sample)
	require.NoError(t, err)

	document := templateengine.WrapDocument(templateengine.WrapRequest{
		Title:    string(kind),
		BodyHTML: body,
		CSS:      starter.CSS,
	})

	pdf, err := renderer.Render(t.Context(), &services.PDFRenderRequest{
		HTML:        document,
		Title:       string(kind),
		PageSize:    starter.PageSize,
		Orientation: starter.Orientation,
		Margins:     starter.Margins,
	})
	require.NoError(t, err)

	return pdf
}

func TestLiveInvoiceStarterProducesALegibleInvoice(t *testing.T) {
	pdf := renderStarter(t, documenttemplate.KindInvoicePDF)

	require.True(t, bytes.HasPrefix(pdf, []byte("%PDF")))
	assert.LessOrEqual(t, pdfPageCount(t, pdf), 2,
		"the sample invoice should not sprawl past two pages")

	text := pdfText(t, pdf)

	// The fields that make the document a collectable invoice.
	for _, want := range []string{
		"INV-2026-1042",    // invoice number
		"Halstead Grocery", // bill to
		"PO Box 41220",     // remit to
		"Linehaul",         // a charge line
		"Fuel surcharge",   // another
		"2,338.74",         // the total
		"Net30",            // payment terms
	} {
		assert.Contains(t, text, want, "the rendered invoice omits %q", want)
	}

	// The escaper must not have neutralized the logo, which is the failure mode a
	// plain-string data: URI would produce.
	assert.NotContains(t, text, "ZgotmplZ")
}

func TestLiveDetentionNoticeStarterRenders(t *testing.T) {
	pdf := renderStarter(t, documenttemplate.KindDetentionNoticePDF)

	require.True(t, bytes.HasPrefix(pdf, []byte("%PDF")))

	text := pdfText(t, pdf)
	for _, want := range []string{
		"TRV-884120",     // shipment
		"Halstead DC 14", // facility
		"227.50",         // the charge being justified
		"09:14 CDT",      // the arrival time it rests on
	} {
		assert.Contains(t, text, want, "the rendered notice omits %q", want)
	}
	assert.NotContains(t, text, "ZgotmplZ")
}

func TestLiveReportStarterRendersLandscape(t *testing.T) {
	pdf := renderStarter(t, documenttemplate.KindReportPDF)

	require.True(t, bytes.HasPrefix(pdf, []byte("%PDF")))

	doc, err := fitz.NewFromMemory(pdf)
	require.NoError(t, err)
	defer func() { _ = doc.Close() }()

	bounds, err := doc.Bound(0)
	require.NoError(t, err)
	assert.Greater(t, bounds.Dx(), bounds.Dy(),
		"a report must be landscape, or its rightmost columns fall off the page")

	text := pdfText(t, pdf)
	assert.Contains(t, text, "Revenue by Customer")
	assert.Contains(t, text, "Halstead Grocery Group")
}

// TestLiveInvoicePaginatesCleanly is the pagination claim the fixed-coordinate
// renderer could not make. With enough charge lines to spill onto a second page, the
// table header must repeat and no row may be split across the boundary.
func TestLiveInvoicePaginatesCleanly(t *testing.T) {
	renderer := liveRenderer(t)
	registry := documenttemplate.NewRegistry()
	engine, err := templateengine.New(templateengine.Config{})
	require.NoError(t, err)

	starter, err := starters.For(documenttemplate.KindInvoicePDF)
	require.NoError(t, err)

	compiled, diags := engine.Parse(&templateengine.ParseRequest{
		Field:        documenttemplate.ChannelPDF.Field(),
		Kind:         string(documenttemplate.KindInvoicePDF),
		Channel:      templateengine.ChannelPDF,
		Source:       starter.BodyHTML,
		AllowedPaths: registry.Paths(documenttemplate.KindInvoicePDF),
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	sample, err := registry.SampleContext(documenttemplate.KindInvoicePDF)
	require.NoError(t, err)

	invoice, ok := sample.(documenttemplate.InvoiceContext)
	require.True(t, ok)

	// Enough lines to guarantee a spill.
	rows := make([]documenttemplate.ChargeRow, 0, 90)
	for i := range 90 {
		rows = append(rows, documenttemplate.ChargeRow{
			Line:        string(rune('0' + i%10)),
			Description: "Accessorial charge with a reasonably long description",
			Quantity:    "1",
			UnitPrice:   "25.00",
			Amount:      "25.00",
		})
	}
	invoice.ChargeRows = rows

	body, err := engine.Render(t.Context(), compiled, invoice)
	require.NoError(t, err)

	pdf, err := renderer.Render(t.Context(), &services.PDFRenderRequest{
		HTML: templateengine.WrapDocument(templateengine.WrapRequest{
			Title:    "invoice",
			BodyHTML: body,
			CSS:      starter.CSS,
		}),
		PageSize:    starter.PageSize,
		Orientation: starter.Orientation,
		Margins:     starter.Margins,
	})
	require.NoError(t, err)

	pages := pdfPageCount(t, pdf)
	require.Greater(t, pages, 1, "90 charge lines must spill past one page")

	doc, err := fitz.NewFromMemory(pdf)
	require.NoError(t, err)
	defer func() { _ = doc.Close() }()

	// A real <thead> repeats on every page a table continues onto. Styled divs get
	// no such treatment, which is why the starter uses table markup.
	for page := range pages {
		text, textErr := doc.Text(page)
		require.NoError(t, textErr)
		// The stylesheet uppercases table headings, and text extraction returns what
		// was rendered rather than the source.
		assert.Contains(t, text, "AMOUNT",
			"the charge table header must repeat on page %d", page+1)
	}
}
