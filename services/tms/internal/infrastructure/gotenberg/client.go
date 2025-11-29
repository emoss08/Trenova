package gotenberg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ClientParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Client struct {
	endpoint   string
	httpClient *http.Client
	l          *zap.Logger
}

func NewClient(p ClientParams) *Client {
	timeout := 30 * time.Second
	if p.Config.Gotenberg != nil && p.Config.Gotenberg.Timeout > 0 {
		timeout = p.Config.Gotenberg.Timeout
	}

	endpoint := "http://localhost:3000"
	if p.Config.Gotenberg != nil && p.Config.Gotenberg.Endpoint != "" {
		endpoint = p.Config.Gotenberg.Endpoint
	}

	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		l: p.Logger.Named("gotenberg.client"),
	}
}

type PDFOptions struct {
	PageWidth    float64
	PageHeight   float64
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64
	Landscape    bool
}

func (c *Client) HTMLToPDF(ctx context.Context, html []byte, opts PDFOptions) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	htmlPart, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, fmt.Errorf("create html form file: %w", err)
	}
	if _, err = htmlPart.Write(html); err != nil {
		return nil, fmt.Errorf("write html content: %w", err)
	}

	if opts.PageWidth > 0 {
		if err = writer.WriteField("paperWidth", fmt.Sprintf("%.2f", opts.PageWidth)); err != nil {
			return nil, fmt.Errorf("write paper width: %w", err)
		}
	}
	if opts.PageHeight > 0 {
		if err = writer.WriteField("paperHeight", fmt.Sprintf("%.2f", opts.PageHeight)); err != nil {
			return nil, fmt.Errorf("write paper height: %w", err)
		}
	}

	if err = writer.WriteField("marginTop", fmt.Sprintf("%.2f", opts.MarginTop)); err != nil {
		return nil, fmt.Errorf("write margin top: %w", err)
	}
	if err = writer.WriteField("marginBottom", fmt.Sprintf("%.2f", opts.MarginBottom)); err != nil {
		return nil, fmt.Errorf("write margin bottom: %w", err)
	}
	if err = writer.WriteField("marginLeft", fmt.Sprintf("%.2f", opts.MarginLeft)); err != nil {
		return nil, fmt.Errorf("write margin left: %w", err)
	}
	if err = writer.WriteField("marginRight", fmt.Sprintf("%.2f", opts.MarginRight)); err != nil {
		return nil, fmt.Errorf("write margin right: %w", err)
	}

	if opts.Landscape {
		if err = writer.WriteField("landscape", "true"); err != nil {
			return nil, fmt.Errorf("write landscape: %w", err)
		}
	}

	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint+"/forms/chromium/convert/html",
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.l.Warn("failed to close response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"gotenberg returned status %d: %s",
			resp.StatusCode,
			string(respBody),
		)
	}

	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return pdfData, nil
}

func (c *Client) HTMLToPDFWithHeaderFooter(
	ctx context.Context,
	html, headerHTML, footerHTML []byte,
	opts PDFOptions,
) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	htmlPart, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, fmt.Errorf("create html form file: %w", err)
	}
	if _, err = htmlPart.Write(html); err != nil {
		return nil, fmt.Errorf("write html content: %w", err)
	}

	if len(headerHTML) > 0 {
		headerPart, hErr := writer.CreateFormFile("files", "header.html")
		if hErr != nil {
			return nil, fmt.Errorf("create header form file: %w", hErr)
		}
		if _, err = headerPart.Write(headerHTML); err != nil {
			return nil, fmt.Errorf("write header content: %w", err)
		}
	}

	if len(footerHTML) > 0 {
		footerPart, fErr := writer.CreateFormFile("files", "footer.html")
		if fErr != nil {
			return nil, fmt.Errorf("create footer form file: %w", fErr)
		}
		if _, err = footerPart.Write(footerHTML); err != nil {
			return nil, fmt.Errorf("write footer content: %w", err)
		}
	}

	if opts.PageWidth > 0 {
		if err = writer.WriteField("paperWidth", fmt.Sprintf("%.2f", opts.PageWidth)); err != nil {
			return nil, fmt.Errorf("write paper width: %w", err)
		}
	}
	if opts.PageHeight > 0 {
		if err = writer.WriteField("paperHeight", fmt.Sprintf("%.2f", opts.PageHeight)); err != nil {
			return nil, fmt.Errorf("write paper height: %w", err)
		}
	}

	if err = writer.WriteField("marginTop", fmt.Sprintf("%.2f", opts.MarginTop)); err != nil {
		return nil, fmt.Errorf("write margin top: %w", err)
	}
	if err = writer.WriteField("marginBottom", fmt.Sprintf("%.2f", opts.MarginBottom)); err != nil {
		return nil, fmt.Errorf("write margin bottom: %w", err)
	}
	if err = writer.WriteField("marginLeft", fmt.Sprintf("%.2f", opts.MarginLeft)); err != nil {
		return nil, fmt.Errorf("write margin left: %w", err)
	}
	if err = writer.WriteField("marginRight", fmt.Sprintf("%.2f", opts.MarginRight)); err != nil {
		return nil, fmt.Errorf("write margin right: %w", err)
	}

	if opts.Landscape {
		if err = writer.WriteField("landscape", "true"); err != nil {
			return nil, fmt.Errorf("write landscape: %w", err)
		}
	}

	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint+"/forms/chromium/convert/html",
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.l.Warn("failed to close response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"gotenberg returned status %d: %s",
			resp.StatusCode,
			string(respBody),
		)
	}

	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return pdfData, nil
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/health", http.NoBody)
	if err != nil {
		return fmt.Errorf("create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute health request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.l.Warn("failed to close response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gotenberg unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
