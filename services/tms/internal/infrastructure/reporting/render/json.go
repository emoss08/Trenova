package render

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
)

var _ services.ReportRenderer = (*JSONRenderer)(nil)

type JSONRenderer struct{}

func NewJSON() *JSONRenderer { return &JSONRenderer{} }

func (r *JSONRenderer) Format() report.Format { return report.FormatJSON }

func (r *JSONRenderer) Render(
	ctx context.Context,
	req *services.ReportRenderRequest,
) (*services.ReportRenderStats, error) {
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

	return &services.ReportRenderStats{
		Rows:      req.Dataset.RowCount(),
		Truncated: truncated,
	}, nil
}

func jsonValue(value any) any {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case decimal.Decimal:
		return v.String()
	case time.Time:
		return v.Unix()
	default:
		return v
	}
}
