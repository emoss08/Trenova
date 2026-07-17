package render

import (
	"context"
	"encoding/csv"
	"errors"
	"io"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

var _ services.ReportRenderer = (*CSVRenderer)(nil)

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

func (r *CSVRenderer) Format() report.Format { return report.FormatCSV }

func (r *CSVRenderer) Render(
	ctx context.Context,
	req *services.ReportRenderRequest,
) (*services.ReportRenderStats, error) {
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
	for {
		row, err := req.Dataset.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		for i := range schema {
			record[i] = formatCell(&schema[i], row[i], loc)
		}
		if err = writer.Write(record); err != nil {
			return nil, err
		}
	}

	truncated := req.Dataset.Truncated()
	if truncated {
		notice := make([]string, len(schema))
		notice[0] = truncationNotice
		if err := writer.Write(notice); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return &services.ReportRenderStats{
		Rows:      req.Dataset.RowCount(),
		Truncated: truncated,
	}, nil
}
