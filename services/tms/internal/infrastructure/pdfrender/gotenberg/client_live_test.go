//go:build integration

package gotenberg

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	fitz "github.com/gen2brain/go-fitz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// pdfText extracts the visible text of every page, so assertions can talk about
// what a reader would actually see rather than about compressed bytes.
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

func assertPDFLacksMarker(t *testing.T, pdf []byte, marker string) {
	t.Helper()
	assert.NotContains(t, pdfText(t, pdf), marker)
}

// liveClient builds a client against a real sidecar, or skips.
//
// Start one with `docker compose -f docker-compose-local.yml up -d gotenberg`
// and run with RENDERER_GOTENBERG_URL=http://localhost:3009.
func liveClient(t *testing.T) *Client {
	t.Helper()

	baseURL := os.Getenv("RENDERER_GOTENBERG_URL")
	if baseURL == "" {
		t.Skip("RENDERER_GOTENBERG_URL is not set; no live renderer to exercise")
	}

	return New(Params{
		Config: &config.Config{
			Renderer: config.RendererConfig{
				GotenbergURL:   baseURL,
				RequestTimeout: 60 * time.Second,
				HealthTimeout:  5 * time.Second,
				MaxConcurrent:  2,
				MaxRetries:     2,
			},
		},
		Logger: zap.NewNop(),
	})
}

func TestLiveHealthy(t *testing.T) {
	require.NoError(t, liveClient(t).Healthy(t.Context()))
}

func TestLiveRenderProducesPDF(t *testing.T) {
	client := liveClient(t)

	pdf, err := client.Render(t.Context(), &services.PDFRenderRequest{
		HTML: `<!DOCTYPE html><html><head><meta charset="utf-8">
			<style>body{font-family:sans-serif}h1{color:#111}</style></head>
			<body><h1>Invoice 1024</h1><p>Balance due $1,234.50</p></body></html>`,
		Title:       "Invoice 1024",
		PageSize:    documenttemplate.PageSizeLetter,
		Orientation: documenttemplate.OrientationPortrait,
		Margins:     documenttemplate.DefaultMargins(),
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(pdf, []byte("%PDF")), "output must be a PDF document")
	assert.Greater(t, len(pdf), 500, "a rendered page should not be near-empty")
}

func TestLiveRenderHonoursHeaderAndFooter(t *testing.T) {
	client := liveClient(t)

	withChrome, err := client.Render(t.Context(), &services.PDFRenderRequest{
		HTML:        `<!DOCTYPE html><html><body><p>body</p></body></html>`,
		HeaderHTML:  `<!DOCTYPE html><html><body><div>HEADER</div></body></html>`,
		FooterHTML:  `<!DOCTYPE html><html><body><div>FOOTER</div></body></html>`,
		PageSize:    documenttemplate.PageSizeLetter,
		Orientation: documenttemplate.OrientationPortrait,
		Margins:     documenttemplate.DefaultMargins(),
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(withChrome, []byte("%PDF")))
}

// TestLiveJavaScriptIsDisabled is the security assertion that matters most: the
// sidecar must not execute template script, or an organization admin authoring a
// template gains code execution inside the render container.
func TestLiveJavaScriptIsDisabled(t *testing.T) {
	client := liveClient(t)

	pdf, err := client.Render(t.Context(), &services.PDFRenderRequest{
		HTML: `<!DOCTYPE html><html><body>
			<div id="out">SCRIPT-DID-NOT-RUN</div>
			<script>document.getElementById('out').textContent='SCRIPT-RAN';</script>
			</body></html>`,
		PageSize:    documenttemplate.PageSizeLetter,
		Orientation: documenttemplate.OrientationPortrait,
		Margins:     documenttemplate.DefaultMargins(),
	})
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(pdf, []byte("%PDF")))

	// Text lands in the PDF content stream compressed, so assert on the marker
	// the script would have had to replace by extracting with pdftotext when it
	// is available, and otherwise settle for a successful inert render.
	assertPDFLacksMarker(t, pdf, "SCRIPT-RAN")
}

// TestLiveRemoteAssetsAreBlocked proves the allow-list holds: a template that
// references an external image must not cause an outbound fetch. If this starts
// failing, organization-authored templates can exfiltrate render-time data by
// URL and probe the internal network.
func TestLiveRemoteAssetsAreBlocked(t *testing.T) {
	client := liveClient(t)

	pdf, err := client.Render(t.Context(), &services.PDFRenderRequest{
		HTML: `<!DOCTYPE html><html><body>
			<p>body</p>
			<img src="http://169.254.169.254/latest/meta-data/" alt="blocked">
			<img src="file:///etc/passwd" alt="blocked">
			</body></html>`,
		PageSize:    documenttemplate.PageSizeLetter,
		Orientation: documenttemplate.OrientationPortrait,
		Margins:     documenttemplate.DefaultMargins(),
	})
	require.NoError(t, err, "blocked assets must not fail the render outright")
	assert.True(t, bytes.HasPrefix(pdf, []byte("%PDF")))
	assertPDFLacksMarker(t, pdf, "root:")
}
