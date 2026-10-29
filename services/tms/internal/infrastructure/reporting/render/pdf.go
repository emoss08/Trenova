package render

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/shared/i18n"
	"go.uber.org/fx"
)

var _ reportports.ReportRenderer = (*PDFRenderer)(nil)

type PDFParams struct {
	fx.In

	ReportingConfig *reportingconfig.Config
	Renderer        documenttemplateports.PDFRenderer
}

type PDFRenderer struct {
	renderer documenttemplateports.PDFRenderer
	maxRows  int64
}

func NewPDF(p PDFParams) *PDFRenderer {
	return &PDFRenderer{
		maxRows:  p.ReportingConfig.GetPDFMaxRows(),
		renderer: p.Renderer,
	}
}

func (r *PDFRenderer) Format() report.Format { return report.FormatPDF }

type pdfRow struct {
	Cells   []Cell
	Summary bool
}

type pdfParam struct {
	Name  string
	Value string
}

type pdfTemplateData struct {
	Title          string
	Description    string
	GeneratedLabel string
	GeneratedAt    string
	RequestedLabel string
	RequestedBy    string
	Params         []pdfParam
	Headers        []string
	Rows           []pdfRow
	Truncated      bool
	Notice         string
	RowCount       string
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
  tr.summary td { border-top: 2px solid #111; border-bottom: none; background: #fff; font-weight: 600; }
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
  <span>{{.GeneratedLabel}} {{.GeneratedAt}}</span>
  {{if .RequestedBy}}<span>{{.RequestedLabel}} {{.RequestedBy}}</span>{{end}}
  {{range .Params}}<span>{{.Name}}: {{.Value}}</span>{{end}}
</div>
<table>
  <thead><tr>{{range .Headers}}<th>{{.}}</th>{{end}}</tr></thead>
  <tbody>
    {{range .Rows}}<tr{{if .Summary}} class="summary"{{end}}>{{range .Cells}}<td{{if .Tone}} class="tone-{{.Tone}}"{{end}}>{{.Text}}</td>{{end}}</tr>
    {{end}}
  </tbody>
</table>
<div class="count">{{.RowCount}}</div>
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
	locale := i18n.FromContext(ctx)
	opts := Options(ctx, &req.Meta, 0)
	columns := req.Dataset.Columns()
	formatters := Formatters(columns, format.Options{Locale: opts.Locale, Location: opts.Location})

	data := pdfTemplateData{
		Title:          req.Meta.Title,
		Description:    req.Meta.Description,
		GeneratedLabel: i18n.Translate(locale, "Generated"),
		GeneratedAt: time.Unix(req.Meta.GeneratedAtUnix, 0).
			In(opts.Location).
			Format("2006-01-02 15:04 MST"),
		RequestedLabel: i18n.Translate(locale, "Requested by"),
		RequestedBy:    req.Meta.RequestedBy,
		Notice:         opts.Text.RowsTruncated,
	}
	for _, p := range opts.Meta.Params {
		data.Params = append(data.Params, pdfParam{Name: p.Name, Value: fmt.Sprint(p.Value)})
	}
	for i := range columns {
		data.Headers = append(data.Headers, columns[i].Label)
	}

	var detail int64
	for {
		batch, err := req.Dataset.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		for i := range batch.Len {
			level := batch.Level(i)
			if level == table.Detail {
				if detail >= r.maxRows {
					return nil, &RowLimitError{MaxRows: r.maxRows}
				}
				detail++
			}
			data.Rows = append(data.Rows, pdfRow{
				Cells:   FormatRow(formatters, batch, i, levelLabel(&opts.Text, level)),
				Summary: level != table.Detail,
			})
		}
	}
	data.Truncated = req.Dataset.Truncated()
	data.RowCount = i18n.Translate(locale, "{0, plural, one {# row} other {# rows}}", detail)

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

	written, err := req.Sink.Write(pdf)
	if err != nil {
		return nil, err
	}

	return &reportports.ReportRenderStats{
		Rows:      detail,
		Bytes:     int64(written),
		Truncated: data.Truncated,
	}, nil
}

var reportPDFMargins = documenttemplate.Margins{
	Top:    10.16,
	Bottom: 10.16,
	Left:   10.16,
	Right:  10.16,
}
