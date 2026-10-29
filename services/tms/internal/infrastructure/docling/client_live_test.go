//go:build integration

package docling

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

const (
	liveURLEnv    = "TRENOVA_TEST_DOCLING_URL"
	liveAPIKeyEnv = "TRENOVA_TEST_DOCLING_API_KEY"
	liveAPIKey    = "integration-test-docling-key"
)

var (
	liveOnce    sync.Once
	liveBaseURL string
	liveKey     string
	liveErr     error
)

func startPinnedSidecar() (string, string, error) {
	if url := os.Getenv(liveURLEnv); url != "" {
		return url, os.Getenv(liveAPIKeyEnv), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        PinnedImage,
			ExposedPorts: []string{"5001/tcp"},
			Env: map[string]string{
				"DOCLING_SERVE_API_KEY":                liveAPIKey,
				"DOCLING_SERVE_ENABLE_UI":              "false",
				"DOCLING_SERVE_ENABLE_REMOTE_SERVICES": "false",
				"DOCLING_SERVE_ALLOW_EXTERNAL_PLUGINS": "false",
				"DOCLING_SERVE_ENG_KIND":               "local",
				"DOCLING_SERVE_ENG_LOC_NUM_WORKERS":    "2",
				"DOCLING_SERVE_LOAD_MODELS_AT_BOOT":    "true",
				"DOCLING_SERVE_SINGLE_USE_RESULTS":     "true",
				"DOCLING_SERVE_MAX_NUM_PAGES":          "50",
				"DOCLING_SERVE_MAX_FILE_SIZE":          "52428800",
				"DOCLING_SERVE_ENABLE_API_DOCS":        "false",
				"DOCLING_SERVE_SCRATCH_PATH":           "/tmp/docling",
				"DOCLING_SERVE_ARTIFACTS_PATH":         "/opt/app-root/src/.cache/docling/models",
			},
			WaitingFor: wait.ForHTTP("/health").WithPort("5001/tcp").
				WithStartupTimeout(5 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		return "", "", err
	}
	endpoint, err := container.PortEndpoint(ctx, "5001/tcp", "http")
	if err != nil {
		return "", "", err
	}

	return endpoint, liveAPIKey, nil
}

func liveClient(t *testing.T) *Client {
	t.Helper()

	liveOnce.Do(func() {
		liveBaseURL, liveKey, liveErr = startPinnedSidecar()
	})
	require.NoError(t, liveErr, "start the pinned docling-serve container")

	cfg := &config.Config{
		DocumentConversion: documentconversionconfig.Config{
			DoclingURL:     liveBaseURL,
			APIKey:         liveKey,
			RequestTimeout: 30 * time.Second,
			MaxConcurrent:  2,
		},
	}

	return New(Params{DocumentConversionConfig: &cfg.DocumentConversion, Logger: zap.NewNop()})
}

func convertLive(
	t *testing.T,
	client *Client,
	name, contentType string,
	ocr bool,
) (*kernelports.ConvertedDocument, error) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	id, err := client.Submit(ctx, &kernelports.ConvertRequest{
		FileName:    name,
		ContentType: contentType,
		Open: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		},
		Size: int64(len(data)),
		OCR:  ocr,
	})
	if err != nil {
		return nil, err
	}

	for {
		status, statusErr := client.Status(ctx, id)
		require.NoError(t, statusErr)
		if status.State == kernelports.ConversionStateFailed {
			return nil, errors.New(status.Message)
		}
		if status.Done() {
			break
		}
	}

	return client.Result(ctx, id)
}

func TestLiveSidecarIsThePinnedVersion(t *testing.T) {
	client := liveClient(t)
	require.NoError(t, client.Healthy(t.Context()))

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		liveBaseURL+"/v1/capabilities",
		http.NoBody,
	)
	require.NoError(t, err)
	req.Header.Set(apiKeyHeader, liveKey)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var capabilities struct {
		Versions map[string]string `json:"versions"`
		Stages   map[string]struct {
			Presets []struct {
				ID string `json:"id"`
			} `json:"presets"`
		} `json:"stages"`
	}
	require.NoError(t, sonic.Unmarshal(raw, &capabilities))
	assert.Equal(t, "1.36.0", capabilities.Versions["docling-serve"])

	presets := make([]string, 0)
	for _, preset := range capabilities.Stages["ocr"].Presets {
		presets = append(presets, preset.ID)
	}
	assert.Contains(t, presets, "rapidocr")

	unauthenticated, err := http.Post(liveBaseURL+submitPath, "text/plain", strings.NewReader("x"))
	require.NoError(t, err)
	defer unauthenticated.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, unauthenticated.StatusCode, "every call needs the key")

	docs, err := http.Get(liveBaseURL + "/openapi.json")
	require.NoError(t, err)
	defer docs.Body.Close()
	assert.Equal(t, http.StatusNotFound, docs.StatusCode, "API docs are off")
}

func TestLiveNativeRateConfirmation(t *testing.T) {
	client := liveClient(t)

	converted, err := convertLive(t, client, "rate-confirmation.pdf", "application/pdf", true)
	require.NoError(t, err)

	assert.Equal(t, "1.36.0", converted.ConverterVersion)
	require.Len(t, converted.Pages, 2)
	first := converted.Pages[0]
	assert.True(t, first.Native)
	assert.False(t, first.OCR, "a native PDF is not OCRed even with OCR on")
	assert.True(t, strings.HasPrefix(first.Markdown, PageHeaderMarker))
	assert.Contains(t, first.Markdown, "| Line haul | 1 | $1,850.00 | $1,850.00 |")
	assert.NotEmpty(t, first.Lines)
}

func TestLiveOCRIsHonoured(t *testing.T) {
	client := liveClient(t)

	scanned, err := convertLive(t, client, "faxed-bol.pdf", "application/pdf", true)
	require.NoError(t, err)
	require.Len(t, scanned.Pages, 1)
	assert.True(t, scanned.Pages[0].OCR)
	assert.Contains(t, scanned.Pages[0].Text, "HF-2026-10433")
	assert.Contains(
		t,
		scanned.Pages[0].Markdown,
		"| Qty | Type | Description | Weight | Class |",
		"the accurate table mode keeps the five columns",
	)

	unread, err := convertLive(t, client, "faxed-bol.pdf", "application/pdf", false)
	require.NoError(t, err)
	require.Len(t, unread.Pages, 1)
	assert.NotContains(
		t,
		unread.Pages[0].Text,
		"HF-2026-10433",
		"do_ocr=false reads no scanned text",
	)
	assert.False(t, unread.Pages[0].OCR)
}

func TestLiveOCREngineAndLanguageReachTheEngine(t *testing.T) {
	client := liveClient(t)
	data, err := os.ReadFile(filepath.Join("testdata", "photo-pod.jpg"))
	require.NoError(t, err)

	id, err := client.Submit(t.Context(), &kernelports.ConvertRequest{
		FileName:    "photo-pod.jpg",
		ContentType: "image/jpeg",
		Open: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		},
		Size:      int64(len(data)),
		OCR:       true,
		Languages: []string{"ko"},
	})
	require.NoError(t, err)

	var status kernelports.ConversionStatus
	for !status.Done() {
		status, err = client.Status(t.Context(), id)
		require.NoError(t, err)
	}
	assert.Equal(t, kernelports.ConversionStateFailed, status.State,
		"RapidOCR in the CPU image has no Korean model, so asking for one must fail: "+
			"a success would mean the language or engine was ignored")

	photo, err := convertLive(t, client, "photo-pod.jpg", "image/jpeg", true)
	require.NoError(t, err)
	require.Len(t, photo.Pages, 1)
	assert.True(t, photo.Pages[0].OCR)
	assert.Contains(t, photo.Pages[0].Text, "VG14-88213")
}

func TestLiveOfficeAndEmail(t *testing.T) {
	client := liveClient(t)

	cases := map[string]string{
		"rate-confirmation.docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"tendered-loads.xlsx":    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"load-tender.eml":        "message/rfc822",
	}
	for name, contentType := range cases {
		converted, err := convertLive(t, client, name, contentType, true)
		require.NoError(t, err, name)
		require.NotEmpty(t, converted.Pages, name)
		assert.True(t, converted.Pages[0].Native, name)
		assert.NotEmpty(t, strings.TrimSpace(converted.Pages[0].Markdown), name)
	}
}

func TestLiveLostTaskIsNotFound(t *testing.T) {
	client := liveClient(t)

	_, err := client.Status(t.Context(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, kernelports.ErrConversionTaskNotFound)
	_, err = client.Result(t.Context(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, kernelports.ErrConversionTaskNotFound)
}

func TestLiveCorruptFileIsNotConvertible(t *testing.T) {
	client := liveClient(t)

	id, err := client.Submit(t.Context(), &kernelports.ConvertRequest{
		FileName:    "broken.pdf",
		ContentType: "application/pdf",
		Open: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("this is not a pdf")), nil
		},
		Size: 17,
		OCR:  true,
	})
	require.NoError(t, err)

	var status kernelports.ConversionStatus
	for !status.Done() {
		status, err = client.Status(t.Context(), id)
		require.NoError(t, err)
	}
	if status.State == kernelports.ConversionStateFailed {
		return
	}
	_, err = client.Result(t.Context(), id)
	require.ErrorIs(t, err, kernelports.ErrDocumentNotConvertible)
}
