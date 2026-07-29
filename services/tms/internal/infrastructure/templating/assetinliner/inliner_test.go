package assetinliner

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/emoss08/trenova/shared/imageutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func dataURI(t *testing.T, w, h int) string {
	t.Helper()
	return imageutils.ToDataURI(&imageutils.Image{
		Data:   pngBytes(t, w, h),
		Format: imageutils.FormatPNG,
	})
}

// fakeStorage serves objects from a map and records what was asked for.
type fakeStorage struct {
	objects   map[string][]byte
	requested []string
	err       error
}

func (f *fakeStorage) Download(_ context.Context, key string) (*storage.DownloadResult, error) {
	f.requested = append(f.requested, key)
	if f.err != nil {
		return nil, f.err
	}
	body, ok := f.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return &storage.DownloadResult{
		Body:        io.NopCloser(bytes.NewReader(body)),
		ContentType: "application/octet-stream",
		Size:        int64(len(body)),
	}, nil
}

func newInliner(t *testing.T, store ObjectReader, limits Limits) *Inliner {
	t.Helper()
	return New(Params{Storage: store, Logger: zap.NewNop(), Limits: limits})
}

func hasCode(diags templateengine.Diagnostics, code string) bool {
	for i := range diags {
		if diags[i].Code == code {
			return true
		}
	}
	return false
}

func TestInlineLeavesNonImageMarkupUntouched(t *testing.T) {
	in := `<h1>Invoice</h1><p>Balance due</p><table><tr><td>x</td></tr></table>`

	result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{HTML: in})
	require.NoError(t, err)
	assert.Equal(t, in, result.HTML)
	assert.Zero(t, result.Inlined)
}

func TestInlineResolvesStorageKeys(t *testing.T) {
	store := &fakeStorage{objects: map[string][]byte{
		"org/logo.png": pngBytes(t, 200, 50),
	}}

	result, err := newInliner(t, store, Limits{}).Inline(t.Context(), &Request{
		HTML: `<div><img src="org/logo.png" alt="logo"></div>`,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"org/logo.png"}, store.requested)
	assert.Contains(t, result.HTML, "data:image/png;base64,")
	assert.NotContains(t, result.HTML, "org/logo.png", "the storage key must not survive")
	assert.Contains(t, result.HTML, `alt="logo"`)
	assert.Equal(t, 1, result.Inlined)
	assert.Empty(t, result.Diags)
}

// TestInlineRefusesInternalAddresses is the SSRF assertion. The old invoice
// renderer fetched an organization-controlled logo URL with http.DefaultClient,
// which made a template a way to reach anything the API process could reach.
func TestInlineRefusesInternalAddresses(t *testing.T) {
	targets := []string{
		"http://127.0.0.1/x.png",
		"http://localhost/x.png",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5/x.png",
		"http://192.168.1.1/x.png",
		"http://172.16.0.1/x.png",
		"http://100.64.0.1/x.png",
		"http://[::1]/x.png",
	}

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{
				HTML:  `<img src="` + target + `">`,
				Field: "bodyHtml",
			})
			require.NoError(t, err, "a blocked asset must not fail the document")

			assert.NotContains(t, result.HTML, target)
			assert.NotContains(t, result.HTML, "<img")
			assert.Zero(t, result.Inlined)
			assert.True(t, hasCode(result.Diags, templateengine.CodeRemoteAssetBlocked),
				"the omission must be reported, never silent")
		})
	}
}

func TestInlineRefusesDangerousSchemes(t *testing.T) {
	schemes := []string{
		"file:///etc/passwd",
		"javascript:alert(1)",
		"vbscript:msgbox(1)",
		"about:blank",
		"blob:http://x/y",
		"view-source:http://x",
		"ftp://example.com/x.png",
		// A protocol-relative URL inherits the renderer's scheme and would be
		// fetched.
		"//evil.example/x.png",
	}

	for _, src := range schemes {
		t.Run(src, func(t *testing.T) {
			result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{
				HTML:  `<img src="` + src + `">`,
				Field: "bodyHtml",
			})
			require.NoError(t, err)

			assert.NotContains(t, result.HTML, "<img")
			assert.True(t, hasCode(result.Diags, templateengine.CodeRemoteAssetBlocked))
		})
	}
}

func TestInlineFetchesPublicRemoteAssets(t *testing.T) {
	// httptest binds to loopback, which httpsafe refuses by design, so a real
	// remote fetch cannot be exercised here without weakening the very protection
	// under test. Assert the refusal instead, and cover the success path through
	// storage, which is how organization logos actually arrive.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes(t, 10, 10))
	}))
	defer srv.Close()

	result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{
		HTML:  `<img src="` + srv.URL + `/logo.png">`,
		Field: "bodyHtml",
	})
	require.NoError(t, err)
	assert.True(t, hasCode(result.Diags, templateengine.CodeRemoteAssetBlocked),
		"a loopback target must be refused even when it really serves an image")
}

func TestInlineValidatesExistingDataURIs(t *testing.T) {
	t.Run("a real image passes through", func(t *testing.T) {
		result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{
			HTML: `<img src="` + dataURI(t, 20, 20) + `">`,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Inlined)
		assert.Contains(t, result.HTML, "data:image/png;base64,")
	})

	// A data: URI is not trusted just because it is inline. One claiming image/png
	// while carrying markup is how an inert image slot becomes a delivery channel.
	t.Run("markup labelled as an image is dropped", func(t *testing.T) {
		payload := "data:image/png;base64," +
			"PHN2Zz48c2NyaXB0PmFsZXJ0KDEpPC9zY3JpcHQ+PC9zdmc+"

		result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{
			HTML:  `<img src="` + payload + `">`,
			Field: "bodyHtml",
		})
		require.NoError(t, err)
		assert.NotContains(t, result.HTML, "<img")
		assert.True(t, hasCode(result.Diags, templateengine.CodeRemoteAssetBlocked))
	})

	t.Run("an svg data uri is dropped", func(t *testing.T) {
		result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{
			HTML:  `<img src="data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=">`,
			Field: "bodyHtml",
		})
		require.NoError(t, err)
		assert.NotContains(t, result.HTML, "<img")
	})
}

func TestInlineDropsAttributesItDoesNotRecognize(t *testing.T) {
	store := &fakeStorage{objects: map[string][]byte{"logo.png": pngBytes(t, 40, 40)}}

	result, err := newInliner(t, store, Limits{}).Inline(t.Context(), &Request{
		HTML: `<img src="logo.png" alt="ok" class="c" onerror="alert(1)" data-x="y">`,
	})
	require.NoError(t, err)

	assert.Contains(t, result.HTML, `alt="ok"`)
	assert.Contains(t, result.HTML, `class="c"`)
	assert.NotContains(t, result.HTML, "onerror")
	assert.NotContains(t, result.HTML, "data-x")
}

// TestInlineRewritesDuplicateSrc covers the browser-precedence case: a repeated
// attribute must not leave an unprocessed URL in the output.
func TestInlineRewritesDuplicateSrc(t *testing.T) {
	store := &fakeStorage{objects: map[string][]byte{"logo.png": pngBytes(t, 10, 10)}}

	result, err := newInliner(t, store, Limits{}).Inline(t.Context(), &Request{
		HTML:  `<img src="logo.png" src="http://169.254.169.254/x">`,
		Field: "bodyHtml",
	})
	require.NoError(t, err)

	assert.NotContains(t, result.HTML, "169.254.169.254")
}

func TestInlineSizesUnsizedImages(t *testing.T) {
	// A 4000px logo is the case that used to push content off the page.
	store := &fakeStorage{objects: map[string][]byte{"huge.png": pngBytes(t, 4000, 1000)}}

	result, err := newInliner(t, store, Limits{MaxAssetBytes: 8 << 20}).
		Inline(t.Context(), &Request{HTML: `<img src="huge.png">`})
	require.NoError(t, err)

	assert.Contains(t, result.HTML, `height="80"`)
	assert.Contains(t, result.HTML, `width="320"`)
}

func TestInlineRespectsAuthorDimensions(t *testing.T) {
	store := &fakeStorage{objects: map[string][]byte{"logo.png": pngBytes(t, 400, 100)}}

	result, err := newInliner(t, store, Limits{}).Inline(t.Context(), &Request{
		HTML: `<img src="logo.png" width="120">`,
	})
	require.NoError(t, err)

	assert.Contains(t, result.HTML, `width="120"`)
	assert.NotContains(t, result.HTML, `height=`, "the author sized it; leave it alone")
}

func TestInlineEnforcesPerAssetSizeCap(t *testing.T) {
	store := &fakeStorage{objects: map[string][]byte{"big.png": pngBytes(t, 500, 500)}}

	result, err := newInliner(t, store, Limits{MaxAssetBytes: 256}).Inline(t.Context(), &Request{
		HTML:  `<img src="big.png">`,
		Field: "bodyHtml",
	})
	require.NoError(t, err)

	assert.Zero(t, result.Inlined)
	assert.True(t, hasCode(result.Diags, templateengine.CodeRemoteAssetBlocked))
}

func TestInlineEnforcesAssetCountCap(t *testing.T) {
	store := &fakeStorage{objects: map[string][]byte{"a.png": pngBytes(t, 10, 10)}}

	var b strings.Builder
	for range 5 {
		b.WriteString(`<img src="a.png">`)
	}

	result, err := newInliner(t, store, Limits{MaxAssets: 2}).Inline(t.Context(), &Request{
		HTML:  b.String(),
		Field: "bodyHtml",
	})
	require.NoError(t, err)

	assert.Equal(t, 2, result.Inlined)
	assert.Equal(t, 2, strings.Count(result.HTML, "<img"))
	assert.True(t, hasCode(result.Diags, templateengine.CodeAssetBudgetExceeded))
}

func TestInlineEnforcesTotalByteBudget(t *testing.T) {
	body := pngBytes(t, 200, 200)
	store := &fakeStorage{objects: map[string][]byte{"a.png": body}}

	result, err := newInliner(t, store, Limits{
		MaxAssetBytes: int64(len(body)) + 1,
		MaxTotalBytes: int64(len(body)) + 1,
	}).Inline(t.Context(), &Request{
		HTML:  `<img src="a.png"><img src="a.png">`,
		Field: "bodyHtml",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Inlined)
	assert.True(t, hasCode(result.Diags, templateengine.CodeAssetBudgetExceeded))
}

func TestInlineKeepsSourcelessImages(t *testing.T) {
	result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{
		HTML: `<img alt="placeholder">`,
	})
	require.NoError(t, err)

	assert.Contains(t, result.HTML, "<img")
	assert.Contains(t, result.HTML, `alt="placeholder"`)
	assert.Zero(t, result.Inlined)
}

func TestInlineReportsMissingStorageObject(t *testing.T) {
	store := &fakeStorage{objects: map[string][]byte{}}

	result, err := newInliner(t, store, Limits{}).Inline(t.Context(), &Request{
		HTML:  `<img src="absent.png">`,
		Field: "headerHtml",
	})
	require.NoError(t, err, "a missing decorative image must not stop an invoice")

	require.NotEmpty(t, result.Diags)
	assert.Equal(t, "headerHtml", result.Diags[0].Field)
	assert.Equal(t, templateengine.SeverityWarning, result.Diags[0].Severity)
}

func TestInlineWithoutStorageClient(t *testing.T) {
	result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{
		HTML:  `<img src="logo.png">`,
		Field: "bodyHtml",
	})
	require.NoError(t, err)
	assert.True(t, hasCode(result.Diags, templateengine.CodeRemoteAssetBlocked))
}

func TestInlineEmptyInput(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\t"} {
		result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), &Request{HTML: in})
		require.NoError(t, err)
		assert.Empty(t, result.HTML)
	}
}

func TestInlineNilRequest(t *testing.T) {
	result, err := newInliner(t, nil, Limits{}).Inline(t.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, result.HTML)
}

func TestWebPTranscoderProducesPNG(t *testing.T) {
	// A minimal RIFF/WEBP header is enough to reach the decoder; a malformed body
	// must surface as an error rather than a panic.
	body := append([]byte("RIFF"), []byte{0, 0, 0, 0}...)
	body = append(body, []byte("WEBPVP8 ")...)

	_, err := NewWebPTranscoder()(body)
	assert.Error(t, err)
}
