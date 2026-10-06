package reporting

import (
	"context"
	"io"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/reporting/render"
	"github.com/emoss08/trenova/shared/timeutils"
)

type StreamCSVRequest struct {
	Request

	Title      string
	Definition *report.Definition
	Params     map[string]any
}

// StreamCSV runs a definition whole and writes it as CSV as the rows arrive.
//
// It is the download of something already on screen: a preview the person
// is looking at. It runs synchronously, under the same row cap and statement
// timeout a run has, rather than as a stored run, because nothing about it
// needs keeping once the file has been written.
func (s *Service) StreamCSV(
	ctx context.Context,
	req *StreamCSVRequest,
	sink io.Writer,
) (*services.ReportRenderStats, error) {
	timezone := s.orgTimezone(ctx, req.TenantInfo)
	compiled, err := s.compiler.Compile(ctx, &services.ReportCompileRequest{
		Definition:  req.Definition,
		Tenant:      req.TenantInfo,
		Principal:   req.Principal,
		Params:      req.Params,
		OrgTimezone: timezone,
		NowUnix:     timeutils.NowUnix(),
	})
	if err != nil {
		return nil, err
	}

	reader, err := s.executor.Open(ctx, &services.OpenReportDatasetRequest{
		Compiled: compiled,
		MaxRows:  s.cfg.GetMaxRows(),
		Timeout:  s.cfg.GetStatementTimeout(),
	})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return render.NewCSVRenderer(s.cfg.CSVIncludeBOM).Render(ctx, &services.ReportRenderRequest{
		Dataset: reader,
		Sink:    sink,
		Meta: services.ReportRunMeta{
			Title:           req.Title,
			GeneratedAtUnix: timeutils.NowUnix(),
			Timezone:        timezone,
			Params:          req.Params,
		},
	})
}
