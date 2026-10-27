package render

import (
	"context"
	"errors"
	"io"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
)

var _ reportports.ReportRenderer = (*JSONRenderer)(nil)

type JSONRenderer struct{}

func NewJSON() *JSONRenderer { return &JSONRenderer{} }

func (r *JSONRenderer) Format() report.Format { return report.FormatJSON }

func (r *JSONRenderer) Render(
	ctx context.Context,
	req *reportports.ReportRenderRequest,
) (*reportports.ReportRenderStats, error) {
	envelope := newRowsEnvelope(req.Sink, req.Dataset.Schema())
	if err := envelope.WriteHead(&req.Meta); err != nil {
		return nil, err
	}

	for {
		row, err := req.Dataset.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if err = envelope.WriteRow(row); err != nil {
			return nil, err
		}
	}

	totals, err := req.Dataset.Totals(ctx)
	if err != nil {
		return nil, err
	}

	truncated := req.Dataset.Truncated()
	if err = envelope.WriteTail(totals, req.Dataset.RowCount(), truncated); err != nil {
		return nil, err
	}

	return &reportports.ReportRenderStats{
		Rows:      req.Dataset.RowCount(),
		Truncated: truncated,
	}, nil
}
