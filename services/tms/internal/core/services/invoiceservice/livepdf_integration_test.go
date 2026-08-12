//go:build integration && !nofitz

package invoiceservice

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/gotenberg"
	"github.com/emoss08/trenova/pkg/templateengine"
	fitz "github.com/gen2brain/go-fitz"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestLiveInvoicePDFRendersThroughTheSidecar prints a real invoice end to end:
// the surviving data builder, the published context, the built-in template, and
// the Gotenberg sidecar.
//
// The unit tests prove each seam in isolation. This proves the replacement for
// 1,342 lines of imperative drawing actually produces a PDF with the customer's
// figures on it, which is the only claim that matters at cutover.
func TestLiveInvoicePDFRendersThroughTheSidecar(t *testing.T) {
	rendererURL := strings.TrimSpace(os.Getenv("RENDERER_GOTENBERG_URL"))
	if rendererURL == "" {
		t.Skip("RENDERER_GOTENBERG_URL is not set")
	}

	builder := &ContextBuilder{inliner: &stubInliner{}}

	entity := testInvoiceForPDF(
		testPULID(t, "inv_"), testPULID(t, "org_"), testPULID(t, "bu_"),
	)
	entity.Lines = []*invoice.InoviceLine{
		{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeFreight,
			Description: "Linehaul Chicago to Dallas",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromInt(2800),
			Amount:      decimal.NewFromInt(2800),
		},
		{
			LineNumber:  2,
			Type:        invoice.InvoiceLineTypeAccessorial,
			Description: "Detention Fee",
			Quantity:    decimal.NewFromInt(2),
			UnitPrice:   decimal.NewFromInt(75),
			Amount:      decimal.NewFromInt(150),
		},
	}
	entity.SubtotalAmount = decimal.NewFromInt(2800)
	entity.OtherAmount = decimal.NewFromInt(150)
	entity.TotalAmount = decimal.NewFromInt(2950)

	data, err := builder.BuildFrom(t.Context(), entity, testInvoicePDFDeliveryProfile())
	require.NoError(t, err)

	starter, err := starters.For(documenttemplate.KindInvoicePDF)
	require.NoError(t, err)

	engine, err := templateengine.New(templateengine.Config{})
	require.NoError(t, err)

	registry := documenttemplate.NewRegistry()
	compiled, diags := engine.Parse(&templateengine.ParseRequest{
		Field:        documenttemplate.ChannelPDF.Field(),
		Kind:         string(documenttemplate.KindInvoicePDF),
		Channel:      documenttemplate.ChannelPDF.Engine(),
		Source:       starter.BodyHTML,
		AllowedPaths: registry.Paths(documenttemplate.KindInvoicePDF),
		Strict:       true,
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	body, err := engine.Render(t.Context(), compiled, data)
	require.NoError(t, err)

	document := templateengine.WrapDocument(templateengine.WrapRequest{
		Title:    "Invoice " + entity.Number,
		BodyHTML: body,
		CSS:      starter.CSS,
	})

	renderer := gotenberg.New(gotenberg.Params{
		Config: &config.Config{
			Renderer: config.RendererConfig{
				GotenbergURL:   rendererURL,
				RequestTimeout: 60 * time.Second,
				MaxConcurrent:  2,
				MaxRetries:     2,
			},
		},
		Logger: zap.NewNop(),
	})

	pdf, err := renderer.Render(t.Context(), &services.PDFRenderRequest{
		HTML:        document,
		Title:       "Invoice " + entity.Number,
		PageSize:    starter.PageSize,
		Orientation: starter.Orientation,
		Margins:     starter.Margins,
	})
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(string(pdf), "%PDF-"), "output is not a PDF")

	// Bytes that parse as a PDF are not the claim. The claim is that a customer
	// can read what they owe, so this asserts against the text a reader sees.
	text := livePDFText(t, pdf)
	require.Contains(t, text, "INV-1001", "the invoice number is missing from the page")
	require.Contains(t, text, "Total USD 2950.00", "the amount billed is missing from the page")
	require.Contains(t, text, "Detention Fee", "an accessorial line is missing from the page")
	// BillingControl continuity, proven on the printed page rather than inferred
	// from the context: the balance and the authored footer both survive.
	require.Contains(t, text, "Balance Due USD 2950.00")
	require.Contains(t, text, "Thank you for your business.")
	require.NotContains(t, text, "ZgotmplZ")
}

// livePDFText extracts what the reader actually sees. Note that the stylesheet
// uppercases table headings, so assert on rendered text rather than source text.
func livePDFText(t *testing.T, pdf []byte) string {
	t.Helper()

	doc, err := fitz.NewFromMemory(pdf)
	require.NoError(t, err)
	defer func() { _ = doc.Close() }()

	var out strings.Builder
	for page := range doc.NumPage() {
		content, pErr := doc.Text(page)
		require.NoError(t, pErr)
		out.WriteString(content)
	}

	return out.String()
}
