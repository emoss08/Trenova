package docling

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testAPIKey = "test-docling-key"

func newTestClient(
	t *testing.T,
	baseURL string,
	mutate func(*documentconversionconfig.Config),
) *Client {
	t.Helper()

	cfg := &config.Config{
		DocumentConversion: documentconversionconfig.Config{
			DoclingURL:     baseURL,
			APIKey:         testAPIKey,
			RequestTimeout: 5 * time.Second,
			HealthTimeout:  time.Second,
			MaxConcurrent:  2,
			MaxRetries:     3,
		},
	}
	if mutate != nil {
		mutate(&cfg.DocumentConversion)
	}

	return New(Params{DocumentConversionConfig: &cfg.DocumentConversion, Logger: zap.NewNop()})
}

func convertRequest(body string) *kernelports.ConvertRequest {
	return &kernelports.ConvertRequest{
		FileName:    "../../Rate Con #48213.pdf",
		ContentType: "application/pdf",
		Open: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(body)), nil
		},
		Size: int64(len(body)),
		OCR:  true,
	}
}

type recordedSubmit struct {
	fileName    string
	contentType string
	body        string
	fields      map[string][]string
}

func readSubmit(t *testing.T, r *http.Request) recordedSubmit {
	t.Helper()

	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	require.NoError(t, err)
	reader := multipart.NewReader(r.Body, params["boundary"])

	recorded := recordedSubmit{fields: map[string][]string{}}
	for {
		part, partErr := reader.NextPart()
		if partErr == io.EOF {
			break
		}
		require.NoError(t, partErr)
		data, readErr := io.ReadAll(part)
		require.NoError(t, readErr)
		if part.FormName() == "files" {
			recorded.fileName = part.FileName()
			recorded.contentType = part.Header.Get("Content-Type")
			recorded.body = string(data)
			continue
		}
		recorded.fields[part.FormName()] = append(recorded.fields[part.FormName()], string(data))
	}

	return recorded
}

func TestSubmitSendsTheConfiguredRequest(t *testing.T) {
	t.Parallel()

	var recorded recordedSubmit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, submitPath, r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, testAPIKey, r.Header.Get(apiKeyHeader))
		recorded = readSubmit(t, r)
		_, _ = w.Write([]byte(`{"task_id":"task-1","task_status":"pending"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, func(cfg *documentconversionconfig.Config) {
		cfg.OCRLanguages = []string{"en", "es"}
		cfg.TableMode = "fast"
	})
	id, err := client.Submit(t.Context(), convertRequest("%PDF-1.4 body"))
	require.NoError(t, err)

	assert.Equal(t, kernelports.ConversionTaskID("task-1"), id)
	assert.Equal(t, "Rate-Con-48213.pdf", recorded.fileName)
	assert.Equal(t, "application/pdf", recorded.contentType)
	assert.Equal(t, "%PDF-1.4 body", recorded.body)
	assert.Equal(t, []string{"json"}, recorded.fields["to_formats"])
	assert.Equal(t, []string{"true"}, recorded.fields["do_ocr"])
	assert.Equal(t, []string{"rapidocr"}, recorded.fields["ocr_preset"])
	assert.Equal(t, []string{"en", "es"}, recorded.fields["ocr_lang"])
	assert.Equal(t, []string{"fast"}, recorded.fields["table_mode"])
	assert.Equal(t, []string{"placeholder"}, recorded.fields["image_export_mode"])
	assert.Equal(t, []string{"false"}, recorded.fields["include_images"])
	assert.Equal(t, []string{"600"}, recorded.fields["document_timeout"])
	assert.NotContains(t, recorded.fields, "pipeline")
}

func TestSubmitRequestLanguagesOverrideConfig(t *testing.T) {
	t.Parallel()

	var recorded recordedSubmit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded = readSubmit(t, r)
		_, _ = w.Write([]byte(`{"task_id":"task-1"}`))
	}))
	defer server.Close()

	req := convertRequest("x")
	req.OCR = false
	req.Languages = []string{"fr"}
	_, err := newTestClient(t, server.URL, nil).Submit(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, []string{"fr"}, recorded.fields["ocr_lang"])
	assert.Equal(t, []string{"false"}, recorded.fields["do_ocr"])
}

func TestSubmitWithVisionEndpointUsesTheVLMPipeline(t *testing.T) {
	t.Parallel()

	var recorded recordedSubmit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded = readSubmit(t, r)
		_, _ = w.Write([]byte(`{"task_id":"task-1"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, func(cfg *documentconversionconfig.Config) {
		cfg.VLMEndpoint = "http://vlm:8000/"
	})
	_, err := client.Submit(t.Context(), convertRequest("x"))
	require.NoError(t, err)

	assert.Equal(t, []string{"vlm"}, recorded.fields["pipeline"])
	require.Len(t, recorded.fields["vlm_pipeline_model_api"], 1)
	assert.Contains(
		t,
		recorded.fields["vlm_pipeline_model_api"][0],
		`"url":"http://vlm:8000/v1/chat/completions"`,
	)
	assert.Contains(t, recorded.fields["vlm_pipeline_model_api"][0], "granite-docling-258M")
}

func TestSubmitRetriesTransientFailures(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = readSubmit(t, r)
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"task_id":"task-3"}`))
	}))
	defer server.Close()

	id, err := newTestClient(t, server.URL, nil).Submit(t.Context(), convertRequest("x"))
	require.NoError(t, err)
	assert.Equal(t, kernelports.ConversionTaskID("task-3"), id)
	assert.Equal(t, int32(3), calls.Load())
}

func TestSubmitGivesUpAfterMaxRetries(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL, nil).Submit(t.Context(), convertRequest("x"))
	require.ErrorIs(t, err, kernelports.ErrDocumentConverterUnavailable)
	assert.Equal(t, int32(3), calls.Load())
}

func TestClientErrorIsNotConvertibleAndNotRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"detail":"unsupported format"}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL, nil).Submit(t.Context(), convertRequest("x"))
	require.ErrorIs(t, err, kernelports.ErrDocumentNotConvertible)
	assert.Contains(t, err.Error(), "unsupported format")
	assert.Equal(t, int32(1), calls.Load())
}

func TestRefusedAPIKeyIsAConfigurationFailureNotTheDocument(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					calls.Add(1)
					w.WriteHeader(status)
				}),
			)
			defer server.Close()

			_, err := newTestClient(t, server.URL, nil).Submit(t.Context(), convertRequest("x"))
			require.ErrorIs(t, err, kernelports.ErrDocumentConverterMisconfigured)
			require.NotErrorIs(t, err, kernelports.ErrDocumentConverterUnavailable)
			require.NotErrorIs(t, err, kernelports.ErrDocumentNotConvertible)
			assert.Equal(t, int32(1), calls.Load(), "a refused key is never retried")
		})
	}
}

func TestSubmitRefusesOversizedFilesBeforeUpload(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, func(cfg *documentconversionconfig.Config) {
		cfg.MaxFileBytes = 4
	})

	_, err := client.Submit(t.Context(), convertRequest("12345"))
	require.ErrorIs(t, err, kernelports.ErrDocumentNotConvertible)
	assert.Zero(t, calls.Load(), "a declared size over the limit is refused before any request")

	req := convertRequest("12345")
	req.Size = 2
	_, err = client.Submit(t.Context(), req)
	require.ErrorIs(
		t,
		err,
		kernelports.ErrDocumentNotConvertible,
		"an understated size is caught as the file streams, and not retried",
	)
	assert.LessOrEqual(t, calls.Load(), int32(1))
}

func TestStatusMapsTaskStates(t *testing.T) {
	t.Parallel()

	states := map[string]kernelports.ConversionStatus{
		"pending":         {State: kernelports.ConversionStatePending},
		"started":         {State: kernelports.ConversionStateRunning},
		"success":         {State: kernelports.ConversionStateSucceeded},
		"partial_success": {State: kernelports.ConversionStateSucceeded},
		"failure": {
			State:   kernelports.ConversionStateFailed,
			Message: "Internal processing error.",
		},
	}
	for status, want := range states {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, statusPath+"task-1", r.URL.Path)
			assert.Equal(t, "5", r.URL.Query().Get("wait"))
			assert.Equal(t, testAPIKey, r.Header.Get(apiKeyHeader))
			body := `{"task_id":"task-1","task_status":"` + status + `"`
			if status == "failure" {
				body += `,"error_message":"Internal processing error."`
			}
			_, _ = w.Write([]byte(body + "}"))
		}))

		got, err := newTestClient(t, server.URL, nil).Status(t.Context(), "task-1")
		server.Close()
		require.NoError(t, err, status)
		assert.Equal(t, want, got, status)
		assert.Equal(t, status != "pending" && status != "started", got.Done(), status)
	}
}

func TestLostTaskIsReportedAsNotFound(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Task not found."}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, nil)
	_, err := client.Status(t.Context(), "gone")
	require.ErrorIs(t, err, kernelports.ErrConversionTaskNotFound)

	_, err = client.Result(t.Context(), "gone")
	require.ErrorIs(t, err, kernelports.ErrConversionTaskNotFound)
	assert.Equal(
		t,
		int32(2),
		calls.Load(),
		"a lost task is not retried here; the caller submits again",
	)
}

func TestResultRendersTheDocument(t *testing.T) {
	t.Parallel()

	response, err := os.ReadFile(filepath.Join("testdata", "rate-confirmation.pdf.response.json"))
	require.NoError(t, err)

	var versionCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testAPIKey, r.Header.Get(apiKeyHeader))
		switch r.URL.Path {
		case resultPath + "task-1":
			_, _ = w.Write(response)
		case versionPath:
			versionCalls.Add(1)
			_, _ = w.Write([]byte(`{"docling-serve":"1.36.0","docling":"2.132.0"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, nil)
	for range 2 {
		converted, resultErr := client.Result(t.Context(), "task-1")
		require.NoError(t, resultErr)
		assert.Equal(t, "1.36.0", converted.ConverterVersion)
		assert.Len(t, converted.Pages, 2)
		assert.True(t, bytes.HasPrefix(converted.Raw, []byte("{")))
	}
	assert.Equal(t, int32(1), versionCalls.Load(), "the version is read once")
}

func TestResultOfAnUnreadableDocumentIsNotConvertible(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(
			[]byte(`{"document":{"filename":"x.pdf","json_content":null},"status":"failure",` +
				`"errors":[{"component_type":"user_input","error_message":"docling-parse could not load document"}],` +
				`"processing_time":0.01}`),
		)
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL, nil).Result(t.Context(), "task-1")
	require.ErrorIs(t, err, kernelports.ErrDocumentNotConvertible)
	assert.Contains(t, err.Error(), "docling-parse could not load document")
}

func TestResultLargerThanTheCapIsRefused(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), 2048))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, nil)
	client.resultLimit = 1024
	_, err := client.Result(t.Context(), "task-1")
	require.ErrorIs(t, err, kernelports.ErrDocumentNotConvertible)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestDisabledClientIsUnavailable(t *testing.T) {
	t.Parallel()

	for name, mutate := range map[string]func(*documentconversionconfig.Config){
		"no url": func(c *documentconversionconfig.Config) { c.DoclingURL = "" },
		"no api key": func(c *documentconversionconfig.Config) {
			c.DoclingURL = "http://docling:5001"
			c.APIKey = ""
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := newTestClient(t, "http://docling:5001", mutate)
			assert.False(t, client.Enabled())

			_, err := client.Submit(t.Context(), convertRequest("x"))
			require.ErrorIs(t, err, kernelports.ErrDocumentConverterUnavailable)
			_, err = client.Status(t.Context(), "task")
			require.ErrorIs(t, err, kernelports.ErrDocumentConverterUnavailable)
			_, err = client.Result(t.Context(), "task")
			require.ErrorIs(t, err, kernelports.ErrDocumentConverterUnavailable)
			require.ErrorIs(
				t,
				client.Healthy(t.Context()),
				kernelports.ErrDocumentConverterUnavailable,
			)
		})
	}
}

func TestHealthy(t *testing.T) {
	t.Parallel()

	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, healthPath, r.URL.Path)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer healthy.Close()
	require.NoError(t, newTestClient(t, healthy.URL, nil).Healthy(t.Context()))

	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()
	require.ErrorIs(
		t,
		newTestClient(t, unhealthy.URL, nil).Healthy(t.Context()),
		kernelports.ErrDocumentConverterUnavailable,
	)
}

func TestConcurrentCallsAreBoundedByMaxConcurrent(t *testing.T) {
	t.Parallel()

	var inFlight, peak atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := inFlight.Add(1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		<-release
		inFlight.Add(-1)
		_, _ = w.Write([]byte(`{"task_id":"t","task_status":"pending"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, nil)
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			_, _ = client.Status(t.Context(), "t")
		})
	}
	time.Sleep(200 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.LessOrEqual(t, peak.Load(), int32(2))
}

func TestSanitizeFileName(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"../../etc/passwd.pdf":         "passwd.pdf",
		`C:\Users\dana\Rate Con 1.pdf`: "Rate-Con-1.pdf",
		"скан.jpg":                     "document.jpg",
		"":                             "document.pdf",
		"invoice.PDF":                  "invoice.pdf",
		"report.d%c":                   "report.pdf",
	}
	for name, want := range cases {
		assert.Equal(t, want, SanitizeFileName(name, "application/pdf"), name)
	}
	assert.Equal(t, "loads.xlsx", SanitizeFileName(
		"loads",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	))
}
