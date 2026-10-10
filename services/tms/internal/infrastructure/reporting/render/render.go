package render

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"go.uber.org/fx"
)

type Registry struct {
	renderers map[report.Format]reportports.ReportRenderer
}

type RegistryParams struct {
	fx.In

	Renderers []reportports.ReportRenderer `group:"report_renderers"`
}

func NewRegistry(p RegistryParams) *Registry {
	registry := &Registry{
		renderers: make(map[report.Format]reportports.ReportRenderer, len(p.Renderers)),
	}
	for _, renderer := range p.Renderers {
		registry.renderers[renderer.Format()] = renderer
	}
	return registry
}

func (r *Registry) For(format report.Format) (reportports.ReportRenderer, error) {
	renderer, ok := r.renderers[format]
	if !ok {
		return nil, fmt.Errorf("no renderer registered for format %q", format)
	}
	return renderer, nil
}

func asRenderer(constructor any) fx.Option {
	return fx.Provide(fx.Annotate(
		constructor,
		fx.As(new(reportports.ReportRenderer)),
		fx.ResultTags(`group:"report_renderers"`),
	))
}

var Module = fx.Module("report-renderers",
	asRenderer(NewCSV),
	asRenderer(NewJSON),
	asRenderer(NewXLSX),
	asRenderer(NewPDF),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(reportports.ReportRendererRegistry)),
		),
	),
)

func metaLocation(meta *reportports.ReportRunMeta) *time.Location {
	if meta.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(meta.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func formatCell(
	column *reportports.ReportResultColumn,
	value any,
	loc *time.Location,
) string {
	return column.Display.String(value, loc)
}

const truncationNotice = "Results were truncated at the row limit; narrow your filters or use a larger export format."

// totalsLabel names the grand-total row in the leading grouping column, which
// the compiler leaves empty precisely so the label has somewhere to live.
const totalsLabel = "Total"

// totalsCells formats the grand-total row, or returns nil when the report did
// not ask for one. Grouping columns come back blank apart from the label.
func totalsCells(
	ctx context.Context,
	dataset reportports.ReportDatasetReader,
	loc *time.Location,
) ([]string, error) {
	row, err := dataset.Totals(ctx)
	if err != nil || row == nil {
		return nil, err
	}

	schema := dataset.Schema()
	cells := make([]string, len(schema))
	for i := range schema {
		if i >= len(row) || row[i] == nil {
			continue
		}
		cells[i] = formatCell(&schema[i], row[i], loc)
	}
	if len(cells) > 0 && row[0] == nil {
		cells[0] = totalsLabel
	}

	return cells, nil
}
