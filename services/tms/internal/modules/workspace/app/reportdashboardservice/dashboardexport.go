package reportdashboardservice

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/reporting/render"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type TileFilterOverride struct {
	Filters []ir.DashboardFilter
	Values  map[string]any
}

type ExportDashboardRequest struct {
	reporting.Request

	DashboardID  pulid.ID
	FilterValues map[string]any
	ParamValues  map[string]any
	TileFilters  map[string]TileFilterOverride
}

type DashboardExport struct {
	FileName    string
	ContentType string
	Body        []byte
}

const dashboardExportContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func (s *Service) ExportDashboard(
	ctx context.Context,
	req *ExportDashboardRequest,
) (*DashboardExport, error) {
	dashboard, err := s.GetDashboard(ctx, &GetDashboardRequest{
		Request:     req.Request,
		DashboardID: req.DashboardID,
	})
	if err != nil {
		return nil, err
	}

	tiles, err := s.ResolveDashboardTiles(ctx, &ResolveDashboardTilesRequest{
		Request:      req.Request,
		DashboardID:  req.DashboardID,
		FilterValues: req.FilterValues,
		ParamValues:  req.ParamValues,
	})
	if err != nil {
		return nil, err
	}
	if len(tiles) == 0 {
		return nil, errortypes.NewValidationError(
			"tiles", errortypes.ErrInvalid,
			"This dashboard has no tiles that query data",
		)
	}

	sheets := make([]render.DashboardSheet, 0, len(tiles))
	timezone := ""
	for i := range tiles {
		tile := &tiles[i]
		definition := applyTileFilterOverride(tile.Definition, req.TileFilters[tile.TileID])

		prepared, prepErr := s.reporting.Prepare(ctx, &reporting.PrepareParams{
			Tenant:     req.TenantInfo,
			Definition: definition,
			Params:     tile.Params,
			Mode:       reportaccess.ModeExport,
		})
		if prepErr != nil {
			return nil, prepErr
		}
		timezone = prepared.Location.String()

		result, previewErr := s.engine.Preview(prepared.Context, &prepared.Request)
		if previewErr != nil {
			s.l.Error("dashboard tile failed to export",
				zap.String("dashboardId", req.DashboardID.String()),
				zap.String("tileId", tile.TileID),
				zap.Error(previewErr))
			return nil, reporting.EngineError(prepared.Context, prepared.Access, previewErr)
		}

		sheets = append(sheets, render.DashboardSheet{Title: tile.Title, Result: result.Reader()})
	}

	var buf bytes.Buffer
	if renderErr := render.RenderDashboard(ctx, &render.DashboardRenderRequest{
		Title:       dashboard.Name,
		Description: dashboard.Description,
		GeneratedAt: timeutils.NowUnix(),
		Timezone:    timezone,
		Locale:      i18n.FromContext(ctx).String(),
		RequestedBy: req.TenantInfo.UserID.String(),
		Filters:     dashboardValues(dashboard.Layout, req.FilterValues),
		Params:      paramValues(req.ParamValues),
		Sheets:      sheets,
		Sink:        &buf,
	}); renderErr != nil {
		s.l.Error("failed to render dashboard export", zap.Error(renderErr))
		return nil, renderErr
	}

	return &DashboardExport{
		FileName:    dashboardExportFileName(dashboard),
		ContentType: dashboardExportContentType,
		Body:        buf.Bytes(),
	}, nil
}

func applyTileFilterOverride(
	definition *ir.Definition,
	override TileFilterOverride,
) *ir.Definition {
	if len(override.Filters) == 0 || len(override.Values) == 0 {
		return definition
	}
	applied, _ := ir.ApplyFilters(
		definition,
		report.FilterValues(override.Filters, override.Values),
		ir.Params{},
	)
	return applied
}

func dashboardValues(
	layout *report.DashboardLayout,
	values map[string]any,
) []render.DashboardValue {
	if layout == nil {
		return nil
	}
	out := make([]render.DashboardValue, 0, len(layout.Filters))
	for i := range layout.Filters {
		filter := &layout.Filters[i]
		value, ok := values[filter.ID]
		if !ok || value == nil {
			continue
		}
		name := filter.Label
		if name == "" {
			name = filter.ID
		}
		out = append(out, render.DashboardValue{Name: name, Value: value})
	}
	return out
}

func paramValues(values map[string]any) []render.DashboardValue {
	out := make([]render.DashboardValue, 0, len(values))
	for _, name := range slices.Sorted(maps.Keys(values)) {
		out = append(out, render.DashboardValue{Name: name, Value: values[name]})
	}
	return out
}

func dashboardExportFileName(dashboard *report.Dashboard) string {
	stamp := time.Unix(timeutils.NowUnix(), 0).UTC().Format("2006-01-02")
	return fmt.Sprintf(
		"%s %s.xlsx",
		fileutils.SanitizeDisplayFilename(
			dashboard.Name,
			"dashboard",
			reporting.MaxDownloadNameLen,
		),
		stamp,
	)
}
