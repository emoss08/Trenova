package render

import (
	"context"
	"encoding/csv"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"go.uber.org/fx"
)

var _ reportports.ReportRenderer = (*CSVRenderer)(nil)

type CSVParams struct {
	fx.In

	Config *config.Config
}

type CSVRenderer struct {
	includeBOM bool
}

func NewCSV(p CSVParams) *CSVRenderer {
	return &CSVRenderer{includeBOM: p.Config.GetReportingConfig().CSVIncludeBOM}
}

// NewCSVRenderer is a CSV renderer for a caller that streams a dataset itself
// rather than through a run.
func NewCSVRenderer(includeBOM bool) *CSVRenderer {
	return &CSVRenderer{includeBOM: includeBOM}
}

func (r *CSVRenderer) Format() report.Format { return report.FormatCSV }

func (r *CSVRenderer) Render(
	ctx context.Context,
	req *reportports.ReportRenderRequest,
) (*reportports.ReportRenderStats, error) {
	if r.includeBOM {
		if _, err := req.Sink.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			return nil, err
		}
	}

	writer := csv.NewWriter(req.Sink)
	schema := req.Dataset.Schema()
	loc := metaLocation(&req.Meta)

	header := make([]string, len(schema))
	for i := range schema {
		header[i] = schema[i].Label
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	record := make([]string, len(schema))
	if err := streamFormatted(ctx, req.Dataset, loc, func(row *reportfmt.Row) error {
		for i := range row.Cells {
			record[i] = row.Cells[i].Text
		}
		return writer.Write(record)
	}); err != nil {
		return nil, err
	}

	totals, err := totalsCells(ctx, req.Dataset, loc)
	if err != nil {
		return nil, err
	}
	if totals != nil {
		if err = writer.Write(totals); err != nil {
			return nil, err
		}
	}

	truncated := req.Dataset.Truncated()
	if truncated {
		notice := make([]string, len(schema))
		notice[0] = truncationNotice
		if err = writer.Write(notice); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err = writer.Error(); err != nil {
		return nil, err
	}

	return &reportports.ReportRenderStats{
		Rows:      req.Dataset.RowCount(),
		Truncated: truncated,
	}, nil
}
