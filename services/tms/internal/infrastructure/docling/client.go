package docling

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cenkalti/backoff/v5"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const (
	ConverterName = "docling"

	submitPath  = "/v1/convert/file/async"
	statusPath  = "/v1/status/poll/"
	resultPath  = "/v1/result/"
	healthPath  = "/health"
	versionPath = "/version"

	apiKeyHeader = "X-Api-Key" //nolint:gosec // a header name, not a credential

	formFalse = "false"

	maxResultBytes       = 64 << 20
	maxErrorBodyBytes    = 2 << 10
	initialRetryInterval = 250 * time.Millisecond
	statusPollWait       = 5 * time.Second
)

var errRejected = errors.New("document converter rejected the request")

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Client struct {
	http    *http.Client
	sem     *semaphore.Weighted
	l       *zap.Logger
	baseURL string
	cfg     *config.DocumentConversionConfig

	resultLimit int64

	versionMu sync.Mutex
	version   string
}

var _ kernelports.DocumentConverter = (*Client)(nil)

func New(p Params) *Client {
	cfg := p.Config.GetDocumentConversionConfig()
	maxConcurrent := cfg.GetMaxConcurrent()

	return &Client{
		baseURL: cfg.GetDoclingURL(),
		cfg:     cfg,
		http: &http.Client{
			Timeout: cfg.GetRequestTimeout() + statusPollWait,
			Transport: &http.Transport{
				MaxIdleConns:        maxConcurrent,
				MaxIdleConnsPerHost: maxConcurrent,
				MaxConnsPerHost:     maxConcurrent,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		sem:         semaphore.NewWeighted(int64(maxConcurrent)),
		l:           p.Logger.Named("docling"),
		resultLimit: maxResultBytes,
	}
}

func (c *Client) Enabled() bool { return c.cfg.Enabled() }

func (c *Client) Healthy(ctx context.Context) error {
	if !c.Enabled() {
		return kernelports.ErrDocumentConverterUnavailable
	}

	healthCtx, cancel := context.WithTimeout(ctx, c.cfg.GetHealthTimeout())
	defer cancel()

	req, err := c.newRequest(healthCtx, http.MethodGet, healthPath, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", kernelports.ErrDocumentConverterUnavailable, err)
	}
	defer drain(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"%w: health returned %d",
			kernelports.ErrDocumentConverterUnavailable,
			resp.StatusCode,
		)
	}

	return nil
}

func (c *Client) Submit(
	ctx context.Context,
	req *kernelports.ConvertRequest,
) (kernelports.ConversionTaskID, error) {
	if !c.Enabled() {
		return "", kernelports.ErrDocumentConverterUnavailable
	}
	if req == nil || req.Body == nil {
		return "", fmt.Errorf("%w: the request has no file", kernelports.ErrDocumentNotConvertible)
	}
	if req.Size > c.cfg.GetMaxFileBytes() {
		return "", c.tooLarge()
	}

	data, err := io.ReadAll(io.LimitReader(req.Body, c.cfg.GetMaxFileBytes()+1))
	if err != nil {
		return "", fmt.Errorf("read document: %w", err)
	}
	if int64(len(data)) > c.cfg.GetMaxFileBytes() {
		return "", c.tooLarge()
	}

	body, contentType, err := c.buildSubmitBody(req, data)
	if err != nil {
		return "", err
	}

	raw, err := c.call(ctx, http.MethodPost, submitPath, body, contentType, maxErrorBodyBytes)
	if err != nil {
		return "", err
	}

	var submitted submitResponse
	if err = sonic.Unmarshal(raw, &submitted); err != nil {
		return "", fmt.Errorf("decode submit response: %w", err)
	}
	if strings.TrimSpace(submitted.TaskID) == "" {
		return "", fmt.Errorf(
			"%w: the converter returned no task id",
			kernelports.ErrDocumentConverterUnavailable,
		)
	}

	return kernelports.ConversionTaskID(submitted.TaskID), nil
}

func (c *Client) tooLarge() error {
	return fmt.Errorf(
		"%w: the file is larger than the %d MB limit",
		kernelports.ErrDocumentNotConvertible,
		c.cfg.GetMaxFileBytes()>>20,
	)
}

func (c *Client) Status(
	ctx context.Context,
	id kernelports.ConversionTaskID,
) (kernelports.ConversionStatus, error) {
	if !c.Enabled() {
		return kernelports.ConversionStatus{}, kernelports.ErrDocumentConverterUnavailable
	}

	path := statusPath + url.PathEscape(string(id)) +
		"?wait=" + strconv.Itoa(int(statusPollWait/time.Second))
	raw, err := c.call(ctx, http.MethodGet, path, nil, "", maxErrorBodyBytes)
	if err != nil {
		return kernelports.ConversionStatus{}, err
	}

	var status statusResponse
	if err = sonic.Unmarshal(raw, &status); err != nil {
		return kernelports.ConversionStatus{}, fmt.Errorf("decode status response: %w", err)
	}

	switch status.TaskStatus {
	case "pending":
		return kernelports.ConversionStatus{State: kernelports.ConversionStatePending}, nil
	case "started":
		return kernelports.ConversionStatus{State: kernelports.ConversionStateRunning}, nil
	case conversionStatusSuccess, conversionStatusPartialSuccess:
		return kernelports.ConversionStatus{State: kernelports.ConversionStateSucceeded}, nil
	default:
		message := strings.TrimSpace(status.ErrorMessage)
		if message == "" {
			message = "the converter could not read the document (" + status.TaskStatus + ")"
		}
		return kernelports.ConversionStatus{
			State:   kernelports.ConversionStateFailed,
			Message: message,
		}, nil
	}
}

func (c *Client) Result(
	ctx context.Context,
	id kernelports.ConversionTaskID,
) (*kernelports.ConvertedDocument, error) {
	if !c.Enabled() {
		return nil, kernelports.ErrDocumentConverterUnavailable
	}

	raw, err := c.call(
		ctx,
		http.MethodGet,
		resultPath+url.PathEscape(string(id)),
		nil,
		"",
		c.resultLimit,
	)
	if err != nil {
		return nil, err
	}

	converted, err := convertResult(raw)
	if err != nil {
		return nil, err
	}
	converted.ConverterVersion = c.converterVersion(ctx)

	return converted, nil
}

func convertResult(raw []byte) (*kernelports.ConvertedDocument, error) {
	var envelope convertResponse
	if err := sonic.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode conversion result: %w", err)
	}
	if envelope.Status != conversionStatusSuccess &&
		envelope.Status != conversionStatusPartialSuccess {
		return nil, fmt.Errorf(
			"%w: %s",
			kernelports.ErrDocumentNotConvertible,
			describeErrors(envelope.Errors, envelope.Status),
		)
	}

	content := bytes.TrimSpace(envelope.Document.JSONContent)
	if len(content) == 0 || bytes.Equal(content, []byte("null")) {
		return nil, fmt.Errorf(
			"%w: the converter returned no document",
			kernelports.ErrDocumentNotConvertible,
		)
	}

	doc, err := decodeDocument(content)
	if err != nil {
		return nil, fmt.Errorf("decode converted document: %w", err)
	}

	return &kernelports.ConvertedDocument{
		Converter: ConverterName,
		Pages:     renderPages(doc, envelope.Confidence),
		Raw:       bytes.Clone(content),
	}, nil
}

func describeErrors(items []conversionError, status string) string {
	messages := make([]string, 0, len(items))
	for i := range items {
		if message := strings.TrimSpace(items[i].ErrorMessage); message != "" {
			messages = append(messages, message)
		}
	}
	if len(messages) == 0 {
		return "conversion ended with status " + strconv.Quote(status)
	}

	return strings.Join(messages, "; ")
}

func (c *Client) converterVersion(ctx context.Context) string {
	c.versionMu.Lock()
	defer c.versionMu.Unlock()
	if c.version != "" {
		return c.version
	}

	raw, err := c.call(ctx, http.MethodGet, versionPath, nil, "", maxErrorBodyBytes)
	if err != nil {
		c.l.Warn("could not read the docling-serve version", zap.Error(err))
		return ""
	}
	var version versionResponse
	if err = sonic.Unmarshal(raw, &version); err != nil {
		c.l.Warn("could not decode the docling-serve version", zap.Error(err))
		return ""
	}
	c.version = strings.TrimSpace(version.DoclingServe)

	return c.version
}

func (c *Client) call(
	ctx context.Context,
	method, path string,
	body []byte,
	contentType string,
	limit int64,
) ([]byte, error) {
	if err := c.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("wait for converter slot: %w", err)
	}
	defer c.sem.Release(1)

	operation := func() ([]byte, error) {
		return c.do(ctx, method, path, body, contentType, limit)
	}

	return backoff.Retry(
		ctx,
		operation,
		backoff.WithMaxTries(uint(c.cfg.GetMaxRetries())),
		backoff.WithBackOff(newRetryBackOff()),
	)
}

func newRetryBackOff() backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = initialRetryInterval
	b.Multiplier = 2
	b.RandomizationFactor = 0.5
	return b
}

func (c *Client) do(
	ctx context.Context,
	method, path string,
	body []byte,
	contentType string,
	limit int64,
) ([]byte, error) {
	var reader io.Reader = http.NoBody
	if body != nil {
		reader = bytes.NewReader(body)
	}
	requestCtx, cancel := context.WithTimeout(ctx, c.cfg.GetRequestTimeout()+statusPollWait)
	defer cancel()

	req, err := c.newRequest(requestCtx, method, path, reader)
	if err != nil {
		return nil, backoff.Permanent(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, backoff.Permanent(ctx.Err())
		}
		return nil, fmt.Errorf("%w: %w", kernelports.ErrDocumentConverterUnavailable, err)
	}
	defer drain(resp)

	switch {
	case resp.StatusCode == http.StatusOK:
		return readLimited(resp.Body, limit)
	case resp.StatusCode == http.StatusNotFound && path != submitPath:
		return nil, backoff.Permanent(fmt.Errorf(
			"%w: %s",
			kernelports.ErrConversionTaskNotFound,
			readErrorBody(resp.Body),
		))
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, backoff.Permanent(fmt.Errorf(
			"%w: the converter refused the API key (%d)",
			kernelports.ErrDocumentConverterMisconfigured,
			resp.StatusCode,
		))
	case retryableStatus(resp.StatusCode):
		return nil, fmt.Errorf(
			"%w: converter returned %d: %s",
			kernelports.ErrDocumentConverterUnavailable,
			resp.StatusCode,
			readErrorBody(resp.Body),
		)
	case resp.StatusCode >= http.StatusInternalServerError:
		return nil, backoff.Permanent(fmt.Errorf(
			"%w: converter returned %d: %s",
			kernelports.ErrDocumentConverterUnavailable,
			resp.StatusCode,
			readErrorBody(resp.Body),
		))
	default:
		return nil, backoff.Permanent(fmt.Errorf(
			"%w: %w (%d): %s",
			kernelports.ErrDocumentNotConvertible,
			errRejected,
			resp.StatusCode,
			readErrorBody(resp.Body),
		))
	}
}

func (c *Client) newRequest(
	ctx context.Context,
	method, path string,
	body io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("build converter request: %w", err)
	}
	if c.cfg.APIKey != "" {
		req.Header.Set(apiKeyHeader, c.cfg.APIKey)
	}

	return req, nil
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, fmt.Errorf(
			"%w: read converter response: %w",
			kernelports.ErrDocumentConverterUnavailable,
			err,
		)
	}
	if int64(len(raw)) > limit {
		return nil, backoff.Permanent(fmt.Errorf(
			"%w: the converter's response exceeds the %d MB limit",
			kernelports.ErrDocumentNotConvertible,
			limit>>20,
		))
	}

	return raw, nil
}

func retryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func readErrorBody(r io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(r, maxErrorBodyBytes))
	if err != nil {
		return "unreadable error body"
	}
	message := strings.TrimSpace(string(raw))
	if message == "" {
		return "no detail"
	}

	return message
}

func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBodyBytes))
	_ = resp.Body.Close()
}

func (c *Client) buildSubmitBody(
	req *kernelports.ConvertRequest,
	data []byte,
) (body []byte, contentType string, err error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	header := make(textproto.MIMEHeader, 2)
	header.Set(
		"Content-Disposition",
		multipart.FileContentDisposition("files", SanitizeFileName(req.FileName, req.ContentType)),
	)
	header.Set("Content-Type", fileContentType(req.ContentType))
	part, err := w.CreatePart(header)
	if err != nil {
		return nil, "", fmt.Errorf("create converter file part: %w", err)
	}
	if _, err = part.Write(data); err != nil {
		return nil, "", fmt.Errorf("write converter file part: %w", err)
	}

	for _, field := range c.formFields(req) {
		if err = w.WriteField(field[0], field[1]); err != nil {
			return nil, "", fmt.Errorf("write converter field %q: %w", field[0], err)
		}
	}

	if err = w.Close(); err != nil {
		return nil, "", fmt.Errorf("finalize converter request: %w", err)
	}

	return buf.Bytes(), w.FormDataContentType(), nil
}

func (c *Client) formFields(req *kernelports.ConvertRequest) [][2]string {
	languages := req.Languages
	if len(languages) == 0 {
		languages = c.cfg.GetOCRLanguages()
	}

	fields := make([][2]string, 0, 12+len(languages))
	fields = append(fields,
		[2]string{"to_formats", "json"},
		[2]string{"do_ocr", strconv.FormatBool(req.OCR)},
		[2]string{"ocr_preset", c.cfg.GetOCREngine()},
		[2]string{"table_mode", c.cfg.GetTableMode()},
		[2]string{"image_export_mode", "placeholder"},
		[2]string{"include_images", formFalse},
		[2]string{"include_page_images", formFalse},
		[2]string{"abort_on_error", formFalse},
		[2]string{"document_timeout", strconv.Itoa(int(c.cfg.GetConversionTimeout().Seconds()))},
	)
	for _, language := range languages {
		fields = append(fields, [2]string{"ocr_lang", language})
	}
	if endpoint := c.cfg.GetVLMEndpoint(); endpoint != "" {
		fields = append(fields,
			[2]string{"pipeline", "vlm"},
			[2]string{"vlm_pipeline_model_api", vlmModelAPI(endpoint, c.cfg.GetVLMModel())},
		)
	}

	return fields
}

func vlmModelAPI(endpoint, model string) string {
	encoded, err := sonic.MarshalString(map[string]any{
		"url":             endpoint + "/v1/chat/completions",
		"params":          map[string]any{"model": model, "max_completion_tokens": 4096},
		"prompt":          "Convert this page to docling.",
		"response_format": "doctags",
		"timeout":         120,
	})
	if err != nil {
		return "{}"
	}

	return encoded
}

func fileContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" || strings.ContainsAny(contentType, "\r\n") {
		return "application/octet-stream"
	}

	return contentType
}

func SanitizeFileName(name, contentType string) string {
	base := filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	ext := strings.ToLower(filepath.Ext(base))
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	var b strings.Builder
	b.Grow(len(stem))
	lastWasSeparator := false
	for _, r := range stem {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			lastWasSeparator = false
		case r == '-', r == ' ', r == '.':
			if !lastWasSeparator && b.Len() > 0 {
				b.WriteByte('-')
				lastWasSeparator = true
			}
		}
	}
	cleanStem := strings.Trim(b.String(), "-")
	if cleanStem == "" {
		cleanStem = "document"
	}

	cleanExt := sanitizeExtension(ext)
	if cleanExt == "" {
		cleanExt = extensionFor(contentType)
	}

	return cleanStem + cleanExt
}

func sanitizeExtension(ext string) string {
	if len(ext) < 2 || len(ext) > 6 {
		return ""
	}
	for _, r := range ext[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ""
		}
	}

	return ext
}

func extensionFor(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "application/pdf":
		return ".pdf"
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/tiff":
		return ".tiff"
	case "image/bmp":
		return ".bmp"
	case "image/webp":
		return ".webp"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ".pptx"
	case "message/rfc822":
		return ".eml"
	case "text/html":
		return ".html"
	case "text/csv":
		return ".csv"
	default:
		return ""
	}
}
