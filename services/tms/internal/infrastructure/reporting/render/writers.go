package render

import (
	"context"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"go.uber.org/fx"
)

type WriterParams struct {
	fx.In

	ReportingConfig *reportingconfig.Config
}

type CSVRenderer struct {
	bom      bool
	maxBytes int64
}

func NewCSV(p WriterParams) *CSVRenderer {
	cfg := p.ReportingConfig
	return NewCSVRenderer(cfg.CSVIncludeBOM, cfg.GetMaxArtifactBytes())
}

func NewCSVRenderer(bom bool, maxBytes int64) *CSVRenderer {
	return &CSVRenderer{bom: bom, maxBytes: maxBytes}
}

func (r *CSVRenderer) Format() report.Format { return report.FormatCSV }

func (r *CSVRenderer) Render(
	ctx context.Context,
	req *reportports.ReportRenderRequest,
) (*reportports.ReportRenderStats, error) {
	s, err := export.CSV(ctx, req.Sink, req.Dataset, export.CSVOptions{
		Options: Options(ctx, &req.Meta, r.maxBytes),
		BOM:     r.bom,
	})
	if err != nil {
		return nil, err
	}
	return stats(s), nil
}

type JSONRenderer struct {
	maxBytes int64
}

func NewJSON(p WriterParams) *JSONRenderer {
	return &JSONRenderer{maxBytes: p.ReportingConfig.GetMaxArtifactBytes()}
}

func (r *JSONRenderer) Format() report.Format { return report.FormatJSON }

func (r *JSONRenderer) Render(
	ctx context.Context,
	req *reportports.ReportRenderRequest,
) (*reportports.ReportRenderStats, error) {
	s, err := export.JSON(ctx, req.Sink, req.Dataset, Options(ctx, &req.Meta, r.maxBytes))
	if err != nil {
		return nil, err
	}
	return stats(s), nil
}

type XLSXRenderer struct {
	maxBytes int64
}

func NewXLSX(p WriterParams) *XLSXRenderer {
	return &XLSXRenderer{maxBytes: p.ReportingConfig.GetMaxArtifactBytes()}
}

func (r *XLSXRenderer) Format() report.Format { return report.FormatXLSX }

func (r *XLSXRenderer) Render(
	ctx context.Context,
	req *reportports.ReportRenderRequest,
) (*reportports.ReportRenderStats, error) {
	s, err := export.XLSX(ctx, req.Sink, req.Dataset, Options(ctx, &req.Meta, r.maxBytes))
	if err != nil {
		return nil, err
	}
	return stats(s), nil
}
