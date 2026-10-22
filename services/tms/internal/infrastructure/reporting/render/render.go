package render

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/reportfmt"
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

func formatColumns(schema []reportports.ReportResultColumn) []reportfmt.Column {
	columns := make([]reportfmt.Column, len(schema))
	for i := range schema {
		columns[i] = reportfmt.Column{Display: &schema[i].Display, Type: schema[i].Type}
	}
	return columns
}

func formatRows(
	ctx context.Context,
	schema []reportports.ReportResultColumn,
	rows []reportports.ReportRow,
	loc *time.Location,
) ([]reportfmt.Row, error) {
	batch := reportfmt.NewBatch(formatColumns(schema), loc)
	out := make([]reportfmt.Row, 0, len(rows))
	for i, row := range rows {
		batch.Add(row)
		if !batch.Full() && i < len(rows)-1 {
			continue
		}
		formatted, err := batch.Format(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, formatted...)
	}
	return out, nil
}

func streamFormatted(
	ctx context.Context,
	dataset reportports.ReportDatasetReader,
	loc *time.Location,
	write func(row *reportfmt.Row) error,
) error {
	batch := reportfmt.NewBatch(formatColumns(dataset.Schema()), loc)
	flush := func() error {
		rows, err := batch.Format(ctx)
		if err != nil {
			return err
		}
		for i := range rows {
			if err = write(&rows[i]); err != nil {
				return err
			}
		}
		return nil
	}

	for {
		row, err := dataset.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		batch.Add(row)
		if batch.Full() {
			if err = flush(); err != nil {
				return err
			}
		}
	}

	return flush()
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
	formatted, err := reportfmt.FormatRow(ctx, formatColumns(schema), row, loc)
	if err != nil {
		return nil, err
	}

	cells := make([]string, len(schema))
	for i := range schema {
		if i >= len(row) || row[i] == nil {
			continue
		}
		cells[i] = formatted[i].Text
	}
	if len(cells) > 0 && row[0] == nil {
		cells[0] = totalsLabel
	}

	return cells, nil
}
