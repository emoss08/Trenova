package render

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ services.ReportRenderer = (*PDFRenderer)(nil)

type PDFParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type PDFRenderer struct {
	maxRows int64
	l       *zap.Logger
}

func NewPDF(p PDFParams) *PDFRenderer {
	return &PDFRenderer{
		maxRows: p.Config.GetReportingConfig().GetPDFMaxRows(),
		l:       p.Logger.Named("reporting.pdf-renderer"),
	}
}

func (r *PDFRenderer) Format() report.Format { return report.FormatPDF }

type pdfTemplateData struct {
	Title       string
	Description string
	GeneratedAt string
	RequestedBy string
	Params      []pdfParam
	Headers     []string
	Rows        [][]string
	Truncated   bool
	Notice      string
	RowCount    int64
}

type pdfParam struct {
	Name  string
	Value string
}

var pdfTemplate = template.Must(template.New("report").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  :root { color-scheme: light; }
  * { box-sizing: border-box; }
  body { font-family: -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; margin: 32px; color: #111; }
  h1 { font-size: 20px; margin: 0 0 4px; }
  .description { color: #555; font-size: 12px; margin: 0 0 12px; }
  .meta { color: #777; font-size: 10px; margin-bottom: 16px; }
  .meta span { margin-right: 16px; }
  table { border-collapse: collapse; width: 100%; font-size: 10px; }
  th { text-align: left; border-bottom: 2px solid #111; padding: 4px 8px; font-weight: 600; }
  td { border-bottom: 1px solid #ddd; padding: 4px 8px; }
  tr:nth-child(even) td { background: #fafafa; }
  .notice { margin-top: 12px; color: #a00; font-size: 10px; }
  .count { margin-top: 8px; color: #777; font-size: 10px; }
</style>
</head>
<body>
<h1>{{.Title}}</h1>
{{if .Description}}<p class="description">{{.Description}}</p>{{end}}
<div class="meta">
  <span>Generated {{.GeneratedAt}}</span>
  {{if .RequestedBy}}<span>Requested by {{.RequestedBy}}</span>{{end}}
  {{range .Params}}<span>{{.Name}}: {{.Value}}</span>{{end}}
</div>
<table>
  <thead><tr>{{range .Headers}}<th>{{.}}</th>{{end}}</tr></thead>
  <tbody>
    {{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>
    {{end}}
  </tbody>
</table>
<div class="count">{{.RowCount}} rows</div>
{{if .Truncated}}<div class="notice">{{.Notice}}</div>{{end}}
</body>
</html>`))

func (r *PDFRenderer) Render(
	ctx context.Context,
	req *services.ReportRenderRequest,
) (*services.ReportRenderStats, error) {
	schema := req.Dataset.Schema()
	loc := metaLocation(&req.Meta)

	data := pdfTemplateData{
		Title:       req.Meta.Title,
		Description: req.Meta.Description,
		GeneratedAt: time.Unix(req.Meta.GeneratedAtUnix, 0).
			In(loc).
			Format("2006-01-02 15:04 MST"),
		RequestedBy: req.Meta.RequestedBy,
		Notice:      truncationNotice,
	}
	for name, value := range req.Meta.Params {
		data.Params = append(data.Params, pdfParam{Name: name, Value: fmt.Sprint(value)})
	}
	for i := range schema {
		data.Headers = append(data.Headers, schema[i].Label)
	}

	for {
		row, err := req.Dataset.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		if int64(len(data.Rows)) >= r.maxRows {
			return nil, fmt.Errorf(
				"PDF output is limited to %d rows; export this report as CSV or XLSX instead",
				r.maxRows,
			)
		}

		record := make([]string, len(schema))
		for i := range schema {
			record[i] = formatCell(&schema[i], row[i], loc)
		}
		data.Rows = append(data.Rows, record)
	}

	data.Truncated = req.Dataset.Truncated()
	data.RowCount = req.Dataset.RowCount()

	var html strings.Builder
	if err := pdfTemplate.Execute(&html, data); err != nil {
		return nil, fmt.Errorf("render PDF template: %w", err)
	}

	pdf, err := r.printToPDF(ctx, html.String())
	if err != nil {
		return nil, err
	}

	if _, err = req.Sink.Write(pdf); err != nil {
		return nil, err
	}

	return &services.ReportRenderStats{
		Rows:      req.Dataset.RowCount(),
		Truncated: data.Truncated,
	}, nil
}

func (r *PDFRenderer) printToPDF(ctx context.Context, html string) ([]byte, error) {
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", "new"),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-background-networking", true),
		)...,
	)
	defer cancelAlloc()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	renderCtx, cancelRender := context.WithTimeout(browserCtx, 2*time.Minute)
	defer cancelRender()

	var pdf []byte
	err := chromedp.Run(renderCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, ftErr := page.GetFrameTree().Do(ctx)
			if ftErr != nil {
				return ftErr
			}
			return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var pErr error
			pdf, _, pErr = page.PrintToPDF().
				WithPrintBackground(true).
				WithLandscape(true).
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			return pErr
		}),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"render PDF via headless chromium (is a chromium binary installed in the reporting worker image?): %w",
			err,
		)
	}

	return pdf, nil
}
