package gotenberg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const (
	convertPath = "/forms/chromium/convert/html"
	healthPath  = "/health"

	indexPartName  = "index.html"
	headerPartName = "header.html"
	footerPartName = "footer.html"

	// initialRetryInterval is short because the failures worth retrying are a
	// busy queue or a browser that is mid-restart, not a slow render.
	initialRetryInterval = 250 * time.Millisecond
)

// errBadRequest marks a 4xx from the renderer. The markup is at fault, so the
// caller gets a business error and no retry.
var errBadRequest = errors.New("renderer rejected the request")

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
	cfg     *config.RendererConfig
}

var _ services.PDFRenderer = (*Client)(nil)

func New(p Params) *Client {
	cfg := p.Config.GetRendererConfig()
	maxConcurrent := cfg.GetMaxConcurrent()

	return &Client{
		baseURL: cfg.GetGotenbergURL(),
		cfg:     cfg,
		// A plain client on purpose: the target is an internal address that
		// shared/httpsafe is designed to refuse. Untrusted URLs never reach
		// here — the renderer's own allow-list confines it to data: URIs.
		http: &http.Client{
			Timeout: cfg.GetRequestTimeout(),
			Transport: &http.Transport{
				MaxIdleConns:        maxConcurrent,
				MaxIdleConnsPerHost: maxConcurrent,
				MaxConnsPerHost:     maxConcurrent,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		sem: semaphore.NewWeighted(int64(maxConcurrent)),
		l:   p.Logger.Named("pdfrender.gotenberg"),
	}
}

func (c *Client) Enabled() bool { return c.baseURL != "" }

func (c *Client) Healthy(ctx context.Context) error {
	if !c.Enabled() {
		return services.ErrPDFRendererUnavailable
	}

	healthCtx, cancel := context.WithTimeout(ctx, c.cfg.GetHealthTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(
		healthCtx,
		http.MethodGet,
		c.baseURL+healthPath,
		http.NoBody,
	)
	if err != nil {
		return fmt.Errorf("build renderer health request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", services.ErrPDFRendererUnavailable, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"%w: health returned %d",
			services.ErrPDFRendererUnavailable,
			resp.StatusCode,
		)
	}

	return nil
}

func (c *Client) Render(ctx context.Context, req *services.PDFRenderRequest) ([]byte, error) {
	if !c.Enabled() {
		return nil, services.ErrPDFRendererUnavailable
	}
	if req == nil || strings.TrimSpace(req.HTML) == "" {
		return nil, errors.New("render request has no HTML")
	}

	// Queue in-process so a burst of previews lines up here instead of piling
	// onto the browser. Acquire honours the caller's deadline.
	if err := c.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("wait for renderer slot: %w", err)
	}
	defer c.sem.Release(1)

	body, contentType, err := buildMultipartBody(req)
	if err != nil {
		return nil, err
	}

	operation := func() ([]byte, error) {
		return c.postConvert(ctx, body, contentType, req)
	}

	// MaxRetries is validated to 0..5, so the conversion cannot wrap.
	maxTries := uint(c.cfg.GetMaxRetries())

	pdf, err := backoff.Retry(
		ctx,
		operation,
		backoff.WithMaxTries(maxTries),
		backoff.WithBackOff(newRetryBackOff()),
	)
	if err != nil {
		return nil, err
	}

	return pdf, nil
}

func newRetryBackOff() backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = initialRetryInterval
	b.Multiplier = 2
	b.RandomizationFactor = 0.5
	return b
}

func (c *Client) postConvert(
	ctx context.Context,
	body []byte,
	contentType string,
	renderReq *services.PDFRenderRequest,
) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+convertPath,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, backoff.Permanent(fmt.Errorf("build render request: %w", err))
	}
	httpReq.Header.Set("Content-Type", contentType)
	if renderReq.TraceID != "" {
		httpReq.Header.Set("Gotenberg-Trace", renderReq.TraceID)
	}
	if renderReq.Title != "" {
		httpReq.Header.Set("Gotenberg-Output-Filename", sanitizeFilename(renderReq.Title))
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		// Transport failures are the retryable case: a restarting browser, a
		// dropped keep-alive, a sidecar that has not finished starting.
		return nil, fmt.Errorf("%w: %w", services.ErrPDFRendererUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusOK:
		pdf, readErr := io.ReadAll(io.LimitReader(resp.Body, c.cfg.GetMaxPDFBytes()+1))
		if readErr != nil {
			return nil, fmt.Errorf("read rendered PDF: %w", readErr)
		}
		if int64(len(pdf)) > c.cfg.GetMaxPDFBytes() {
			return nil, backoff.Permanent(fmt.Errorf(
				"rendered PDF exceeds the %d byte limit; reduce the document size",
				c.cfg.GetMaxPDFBytes(),
			))
		}
		return pdf, nil

	case retryableStatus(resp.StatusCode):
		return nil, fmt.Errorf(
			"%w: renderer returned %d: %s",
			services.ErrPDFRendererUnavailable,
			resp.StatusCode,
			readErrorBody(resp.Body),
		)

	default:
		// 4xx means the markup or the page setup is wrong. Retrying cannot help
		// and would triple the latency of a template the author needs to fix.
		return nil, backoff.Permanent(fmt.Errorf(
			"%w (%d): %s",
			errBadRequest,
			resp.StatusCode,
			readErrorBody(resp.Body),
		))
	}
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

// maxErrorBodyBytes bounds how much of a renderer error we quote back.
const maxErrorBodyBytes = 2 << 10

func readErrorBody(r io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(r, maxErrorBodyBytes))
	if err != nil {
		return "unreadable error body"
	}
	msg := strings.TrimSpace(string(raw))
	if msg == "" {
		return "no detail"
	}
	return msg
}

func buildMultipartBody(
	req *services.PDFRenderRequest,
) (body []byte, contentType string, err error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err = writeFilePart(w, indexPartName, req.HTML); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(req.HeaderHTML) != "" {
		if err = writeFilePart(w, headerPartName, req.HeaderHTML); err != nil {
			return nil, "", err
		}
	}
	if strings.TrimSpace(req.FooterHTML) != "" {
		if err = writeFilePart(w, footerPartName, req.FooterHTML); err != nil {
			return nil, "", err
		}
	}

	pageSize := req.PageSize
	if !pageSize.IsValid() {
		pageSize = documenttemplate.PageSizeLetter
	}
	width, height := pageSize.Dimensions()

	margins := req.Margins
	if !margins.IsValid() {
		margins = documenttemplate.DefaultMargins()
	}
	inches := margins.Inches()

	fields := [...]struct {
		name  string
		value string
	}{
		{"paperWidth", formatInches(width)},
		{"paperHeight", formatInches(height)},
		{"landscape", strconv.FormatBool(req.Orientation.IsLandscape())},
		{"marginTop", formatInches(inches.Top)},
		{"marginBottom", formatInches(inches.Bottom)},
		{"marginLeft", formatInches(inches.Left)},
		{"marginRight", formatInches(inches.Right)},
		{"printBackground", "true"},
		{"preferCssPageSize", "false"},
		{"emulatedMediaType", "print"},
		// Every asset is inlined as a data: URI before we get here, so there is
		// no network activity to idle on and waiting for it only adds latency.
		{"skipNetworkIdleEvent", "true"},
	}
	for _, f := range fields {
		if err = w.WriteField(f.name, f.value); err != nil {
			return nil, "", fmt.Errorf("write render field %q: %w", f.name, err)
		}
	}

	if err = w.Close(); err != nil {
		return nil, "", fmt.Errorf("finalize render request: %w", err)
	}

	return buf.Bytes(), w.FormDataContentType(), nil
}

func writeFilePart(w *multipart.Writer, name, content string) error {
	part, err := w.CreateFormFile("files", name)
	if err != nil {
		return fmt.Errorf("create render part %q: %w", name, err)
	}
	if _, err = io.WriteString(part, content); err != nil {
		return fmt.Errorf("write render part %q: %w", name, err)
	}
	return nil
}

func formatInches(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// sanitizeFilename keeps the output filename header to characters that cannot
// break header framing or escape a directory. Separator runs collapse so a
// title like "../../etc/passwd" does not come back as "----etcpasswd".
func sanitizeFilename(title string) string {
	var b strings.Builder
	b.Grow(len(title) + len(".pdf"))

	lastWasSeparator := false
	for _, r := range title {
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

	name := strings.Trim(b.String(), "-")
	if name == "" {
		return "document.pdf"
	}
	return name + ".pdf"
}
