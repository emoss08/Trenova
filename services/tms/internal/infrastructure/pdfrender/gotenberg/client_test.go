package gotenberg

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestClient(t *testing.T, baseURL string, mutate func(*config.RendererConfig)) *Client {
	t.Helper()

	cfg := &config.Config{
		Renderer: config.RendererConfig{
			GotenbergURL:   baseURL,
			RequestTimeout: 5 * time.Second,
			HealthTimeout:  time.Second,
			MaxConcurrent:  4,
			MaxRetries:     3,
		},
	}
	if mutate != nil {
		mutate(&cfg.Renderer)
	}

	return New(Params{Config: cfg, Logger: zap.NewNop()})
}

// capturedRequest is the parsed multipart body the fake renderer received.
type capturedRequest struct {
	files  map[string]string
	fields map[string]string
	header http.Header
}

func parseMultipart(t *testing.T, r *http.Request) capturedRequest {
	t.Helper()

	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	require.NoError(t, err)

	captured := capturedRequest{
		files:  map[string]string{},
		fields: map[string]string{},
		header: r.Header.Clone(),
	}

	mr := multipart.NewReader(r.Body, params["boundary"])
	for {
		part, partErr := mr.NextPart()
		if errors.Is(partErr, io.EOF) {
			break
		}
		require.NoError(t, partErr)

		body, readErr := io.ReadAll(part)
		require.NoError(t, readErr)

		if part.FileName() != "" {
			captured.files[part.FileName()] = string(body)
		} else {
			captured.fields[part.FormName()] = string(body)
		}
		require.NoError(t, part.Close())
	}

	return captured
}

func TestRenderSubmitsExpectedMultipartShape(t *testing.T) {
	var captured capturedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, convertPath, r.URL.Path)
		captured = parseMultipart(t, r)
		_, _ = w.Write([]byte("%PDF-1.7 ok"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, nil)

	pdf, err := client.Render(t.Context(), &services.PDFRenderRequest{
		HTML:        "<p>body</p>",
		HeaderHTML:  "<span>head</span>",
		FooterHTML:  "<span>foot</span>",
		Title:       "Invoice 1024",
		TraceID:     "trace-abc",
		PageSize:    documenttemplate.PageSizeLetter,
		Orientation: documenttemplate.OrientationPortrait,
		Margins:     documenttemplate.Margins{Top: 25.4, Bottom: 12.7, Left: 25.4, Right: 12.7},
	})
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.7 ok", string(pdf))

	assert.Equal(t, "<p>body</p>", captured.files[indexPartName])
	assert.Equal(t, "<span>head</span>", captured.files[headerPartName])
	assert.Equal(t, "<span>foot</span>", captured.files[footerPartName])

	assert.Equal(t, "trace-abc", captured.header.Get("Gotenberg-Trace"))
	assert.Equal(t, "Invoice-1024.pdf", captured.header.Get("Gotenberg-Output-Filename"))

	// 25.4mm is exactly one inch; 12.7mm is exactly half.
	assert.Equal(t, "1.00", captured.fields["marginTop"])
	assert.Equal(t, "0.50", captured.fields["marginBottom"])
	assert.Equal(t, "1.00", captured.fields["marginLeft"])
	assert.Equal(t, "0.50", captured.fields["marginRight"])

	assert.Equal(t, "true", captured.fields["printBackground"])
	assert.Equal(t, "false", captured.fields["preferCssPageSize"])
	assert.Equal(t, "print", captured.fields["emulatedMediaType"])
	assert.Equal(t, "true", captured.fields["skipNetworkIdleEvent"])
}

func TestRenderOmitsEmptyHeaderAndFooter(t *testing.T) {
	var captured capturedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = parseMultipart(t, r)
		_, _ = w.Write([]byte("pdf"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv.URL, nil).Render(t.Context(), &services.PDFRenderRequest{
		HTML:       "<p>body</p>",
		HeaderHTML: "   ",
	})
	require.NoError(t, err)

	assert.Len(t, captured.files, 1)
	assert.NotContains(t, captured.files, headerPartName)
	assert.NotContains(t, captured.files, footerPartName)
}

func TestRenderPageGeometry(t *testing.T) {
	tests := []struct {
		name        string
		pageSize    documenttemplate.PageSize
		orientation documenttemplate.Orientation
		wantWidth   string
		wantHeight  string
		wantLand    string
	}{
		{"letter portrait", documenttemplate.PageSizeLetter, documenttemplate.OrientationPortrait, "8.50", "11.00", "false"},
		{"letter landscape", documenttemplate.PageSizeLetter, documenttemplate.OrientationLandscape, "8.50", "11.00", "true"},
		{"a4 portrait", documenttemplate.PageSizeA4, documenttemplate.OrientationPortrait, "8.27", "11.69", "false"},
		{"a4 landscape", documenttemplate.PageSizeA4, documenttemplate.OrientationLandscape, "8.27", "11.69", "true"},
		{"legal portrait", documenttemplate.PageSizeLegal, documenttemplate.OrientationPortrait, "8.50", "14.00", "false"},
		{"legal landscape", documenttemplate.PageSizeLegal, documenttemplate.OrientationLandscape, "8.50", "14.00", "true"},
		// An unset page size must not produce a zero-area page.
		{"unset falls back to letter", documenttemplate.PageSize(""), documenttemplate.OrientationPortrait, "8.50", "11.00", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured capturedRequest
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured = parseMultipart(t, r)
				_, _ = w.Write([]byte("pdf"))
			}))
			defer srv.Close()

			_, err := newTestClient(t, srv.URL, nil).Render(t.Context(), &services.PDFRenderRequest{
				HTML:        "<p>x</p>",
				PageSize:    tt.pageSize,
				Orientation: tt.orientation,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.wantWidth, captured.fields["paperWidth"])
			assert.Equal(t, tt.wantHeight, captured.fields["paperHeight"])
			assert.Equal(t, tt.wantLand, captured.fields["landscape"])
		})
	}
}

func TestRenderInvalidMarginsFallBackToDefault(t *testing.T) {
	var captured capturedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = parseMultipart(t, r)
		_, _ = w.Write([]byte("pdf"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv.URL, nil).Render(t.Context(), &services.PDFRenderRequest{
		HTML:    "<p>x</p>",
		Margins: documenttemplate.Margins{Top: -5, Bottom: 900},
	})
	require.NoError(t, err)

	// DefaultMargins is 20mm on every side.
	assert.Equal(t, "0.79", captured.fields["marginTop"])
	assert.Equal(t, "0.79", captured.fields["marginRight"])
}

func TestRenderRetryMatrix(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantCalls int32
		wantErrIs error
	}{
		{"429 retries", http.StatusTooManyRequests, 3, services.ErrPDFRendererUnavailable},
		{"502 retries", http.StatusBadGateway, 3, services.ErrPDFRendererUnavailable},
		{"503 retries", http.StatusServiceUnavailable, 3, services.ErrPDFRendererUnavailable},
		{"504 retries", http.StatusGatewayTimeout, 3, services.ErrPDFRendererUnavailable},
		// Bad markup cannot be fixed by asking again.
		{"400 does not retry", http.StatusBadRequest, 1, errBadRequest},
		{"409 does not retry", http.StatusConflict, 1, errBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte("renderer detail"))
			}))
			defer srv.Close()

			client := newTestClient(t, srv.URL, func(c *config.RendererConfig) {
				c.MaxRetries = 3
			})

			_, err := client.Render(t.Context(), &services.PDFRenderRequest{HTML: "<p>x</p>"})
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErrIs)
			assert.Contains(t, err.Error(), "renderer detail")
			assert.Equal(t, tt.wantCalls, calls.Load())
		})
	}
}

func TestRenderSucceedsAfterRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("pdf-bytes"))
	}))
	defer srv.Close()

	pdf, err := newTestClient(t, srv.URL, nil).
		Render(t.Context(), &services.PDFRenderRequest{HTML: "<p>x</p>"})
	require.NoError(t, err)
	assert.Equal(t, "pdf-bytes", string(pdf))
	assert.Equal(t, int32(2), calls.Load())
}

func TestRenderSemaphoreLimitsConcurrency(t *testing.T) {
	const maxConcurrent = 2

	var (
		mu      sync.Mutex
		inWork  int
		highest int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		inWork++
		if inWork > highest {
			highest = inWork
		}
		mu.Unlock()

		time.Sleep(25 * time.Millisecond)

		mu.Lock()
		inWork--
		mu.Unlock()

		_, _ = w.Write([]byte("pdf"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, func(c *config.RendererConfig) {
		c.MaxConcurrent = maxConcurrent
	})

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			_, err := client.Render(t.Context(), &services.PDFRenderRequest{HTML: "<p>x</p>"})
			assert.NoError(t, err)
		})
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.LessOrEqual(t, highest, maxConcurrent)
}

func TestRenderRejectsOversizedPDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 128)))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, func(c *config.RendererConfig) {
		c.MaxPDFBytes = 64
	})

	_, err := client.Render(t.Context(), &services.PDFRenderRequest{HTML: "<p>x</p>"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the 64 byte limit")
}

func TestDisabledRendererIsUnavailableNotAnError(t *testing.T) {
	client := newTestClient(t, "", nil)

	assert.False(t, client.Enabled())

	_, err := client.Render(t.Context(), &services.PDFRenderRequest{HTML: "<p>x</p>"})
	assert.ErrorIs(t, err, services.ErrPDFRendererUnavailable)
	assert.ErrorIs(t, client.Healthy(t.Context()), services.ErrPDFRendererUnavailable)
}

func TestRenderRejectsEmptyHTML(t *testing.T) {
	client := newTestClient(t, "http://renderer.invalid", nil)

	_, err := client.Render(t.Context(), &services.PDFRenderRequest{HTML: "   "})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no HTML")
}

func TestHealthy(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"ok", http.StatusOK, false},
		{"unhealthy", http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, healthPath, r.URL.Path)
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			err := newTestClient(t, srv.URL, nil).Healthy(t.Context())
			if tt.wantErr {
				assert.ErrorIs(t, err, services.ErrPDFRendererUnavailable)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Invoice 1024", "Invoice-1024.pdf"},
		{"report.2026", "report-2026.pdf"},
		{"../../etc/passwd", "etcpasswd.pdf"},
		{`bad"header` + "\r\nInjected: yes", "badheaderInjected-yes.pdf"},
		{"", "document.pdf"},
		{"!!!", "document.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := sanitizeFilename(tt.in)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "\r")
			assert.NotContains(t, got, "\n")
			assert.NotContains(t, got, "/")
		})
	}
}
