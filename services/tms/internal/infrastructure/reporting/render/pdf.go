package render

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ reportports.ReportRenderer = (*PDFRenderer)(nil)

type PDFParams struct {
	fx.In

	Config   *config.Config
	Renderer documenttemplateports.PDFRenderer
	Logger   *zap.Logger
}

type PDFRenderer struct {
	renderer documenttemplateports.PDFRenderer
	l        *zap.Logger
	maxRows  int64
}

func NewPDF(p PDFParams) *PDFRenderer {
	return &PDFRenderer{
		maxRows:  p.Config.GetReportingConfig().GetPDFMaxRows(),
		renderer: p.Renderer,
		l:        p.Logger.Named("reporting.pdf-renderer"),
	}
}

func (r *PDFRenderer) Format() report.Format { return report.FormatPDF }

// pdfCell carries the banded emphasis alongside the text so the template can
// colour an exception without re-deriving it.
type pdfCell struct {
	Text string
	Tone string
}

type pdfTemplateData struct {
	Title       string
	Description string
	GeneratedAt string
	RequestedBy string
	Params      []pdfParam
	Headers     []string
	Rows        [][]pdfCell
	Totals      []string
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
  tfoot td { border-top: 2px solid #111; border-bottom: none; background: #fff; font-weight: 600; }
  .notice { margin-top: 12px; color: #a00; font-size: 10px; }
  td.tone-positive { color: #15803d; font-weight: 600; }
  td.tone-warning  { color: #b45309; font-weight: 600; }
  td.tone-negative { color: #b91c1c; font-weight: 600; }
  td.tone-neutral  { color: #3f3f46; font-weight: 600; }
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
    {{range .Rows}}<tr>{{range .}}<td{{if .Tone}} class="tone-{{.Tone}}"{{end}}>{{.Text}}</td>{{end}}</tr>
    {{end}}
  </tbody>
  {{if .Totals}}<tfoot><tr>{{range .Totals}}<td>{{.}}</td>{{end}}</tr></tfoot>{{end}}
</table>
<div class="count">{{.RowCount}} rows</div>
{{if .Truncated}}<div class="notice">{{.Notice}}</div>{{end}}
</body>
</html>`))

type RowLimitError struct {
	MaxRows int64
}

func (e *RowLimitError) Error() string {
	return fmt.Sprintf(
		"PDF output is limited to %d rows; export this report as CSV or XLSX instead",
		e.MaxRows,
	)
}

func (r *PDFRenderer) Render(
	ctx context.Context,
	req *reportports.ReportRenderRequest,
) (*reportports.ReportRenderStats, error) {
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
	for _, name := range orderedParamNames(&req.Meta) {
		data.Params = append(
			data.Params,
			pdfParam{Name: name, Value: fmt.Sprint(req.Meta.Params[name])},
		)
	}
	for i := range schema {
		data.Headers = append(data.Headers, schema[i].Label)
	}

	if err := streamFormatted(ctx, req.Dataset, loc, func(row *reportfmt.Row) error {
		if int64(len(data.Rows)) >= r.maxRows {
			return &RowLimitError{MaxRows: r.maxRows}
		}

		record := make([]pdfCell, len(row.Cells))
		for i := range row.Cells {
			record[i] = pdfCell{Text: row.Cells[i].Text, Tone: string(row.Cells[i].Tone)}
		}
		data.Rows = append(data.Rows, record)
		return nil
	}); err != nil {
		return nil, err
	}

	totals, totalsErr := totalsCells(ctx, req.Dataset, loc)
	if totalsErr != nil {
		return nil, totalsErr
	}
	data.Totals = totals
	data.Truncated = req.Dataset.Truncated()
	data.RowCount = req.Dataset.RowCount()

	var html strings.Builder
	if err := pdfTemplate.Execute(&html, data); err != nil {
		return nil, fmt.Errorf("render PDF template: %w", err)
	}

	pdf, err := r.renderer.Render(ctx, &documenttemplateports.PDFRenderRequest{
		HTML:        html.String(),
		Title:       req.Meta.Title,
		PageSize:    documenttemplate.PageSizeLetter,
		Orientation: documenttemplate.OrientationLandscape,
		Margins:     reportPDFMargins,
	})
	if err != nil {
		return nil, err
	}

	if _, err = req.Sink.Write(pdf); err != nil {
		return nil, err
	}

	return &reportports.ReportRenderStats{
		Rows:      req.Dataset.RowCount(),
		Truncated: data.Truncated,
	}, nil
}

// reportPDFMargins is the 0.4in page box the report layout has always used,
// expressed in the millimeters the renderer takes.
var reportPDFMargins = documenttemplate.Margins{
	Top:    10.16,
	Bottom: 10.16,
	Left:   10.16,
	Right:  10.16,
}
