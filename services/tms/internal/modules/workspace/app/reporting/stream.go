package reporting

import (
	"context"
	"io"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/reporting/render"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/timeutils"
)

type StreamCSVRequest struct {
	Request

	Title      string
	Definition *ir.Definition
	Params     map[string]any
}

func (s *Service) StreamCSV(
	ctx context.Context,
	req *StreamCSVRequest,
	sink io.Writer,
) (*reportports.ReportRenderStats, error) {
	prepared, err := s.Prepare(ctx, &PrepareParams{
		Tenant:     req.TenantInfo,
		Definition: req.Definition,
		Params:     req.Params,
		Mode:       reportaccess.ModeExport,
	})
	if err != nil {
		return nil, err
	}

	stream, err := s.engine.Run(prepared.Context, prepared.Request)
	if err != nil {
		return nil, EngineError(prepared.Context, prepared.Access, err)
	}
	defer stream.Close()

	stats, err := render.NewCSVRenderer(s.cfg.CSVIncludeBOM, s.cfg.GetMaxArtifactBytes()).
		Render(prepared.Context, &reportports.ReportRenderRequest{
			Dataset: stream,
			Sink:    sink,
			Meta: reportports.ReportRunMeta{
				Title:           req.Title,
				GeneratedAtUnix: timeutils.NowUnix(),
				Timezone:        prepared.Location.String(),
				Locale:          i18n.FromContext(ctx).String(),
				Params:          OrderedParams(req.Definition, req.Params),
			},
		})
	if err != nil {
		return nil, EngineError(prepared.Context, prepared.Access, err)
	}
	return stats, nil
}

func OrderedParams(def *ir.Definition, values map[string]any) []reportports.ReportParam {
	if len(values) == 0 {
		return nil
	}
	out := make([]reportports.ReportParam, 0, len(values))
	listed := make(map[string]bool, len(values))
	if def != nil {
		for i := range def.Parameters {
			p := &def.Parameters[i]
			value, ok := values[p.Name]
			if !ok {
				continue
			}
			name := p.Label
			if name == "" {
				name = p.Name
			}
			listed[p.Name] = true
			out = append(out, reportports.ReportParam{Name: name, Value: value})
		}
	}
	for _, name := range sortedNames(values) {
		if !listed[name] {
			out = append(out, reportports.ReportParam{Name: name, Value: values[name]})
		}
	}
	return out
}
