package reporting

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/infrastructure/reporting/render"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// TileFilterOverride carries one tile's ad-hoc narrowing. Tile filters are
// viewer state the dashboard never persists, so the caller has to supply both
// the fields and the chosen values or the export would quietly disagree with
// the screen it was taken from.
type TileFilterOverride struct {
	Filters []report.DashboardFilter
	Values  map[string]any
}

type ExportDashboardRequest struct {
	Request

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

// ExportDashboard runs every tile of a dashboard and returns one workbook.
//
// It executes synchronously rather than through Temporal because a dashboard
// is already a synchronous surface: the browser runs the same tiles, with the
// same preview row limits, every time it renders. An export is that same work
// once more, bounded by MaxDashboardTiles.
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
	for i := range tiles {
		tile := &tiles[i]
		// Each tile reuses the preview path, so an exported figure is the same
		// figure the dashboard shows — same compiler, same executor, same row
		// limit. A tile that fails takes the export down with it rather than
		// shipping a workbook with a silently missing sheet.
		definition := applyTileFilterOverride(tile.Definition, req.TileFilters[tile.TileID])

		result, previewErr := s.runPreview(ctx, &PreviewRequest{
			Request:    req.Request,
			Definition: definition,
			Params:     tile.Params,
		})
		if previewErr != nil {
			s.l.Error("dashboard tile failed to export",
				zap.String("dashboardId", req.DashboardID.String()),
				zap.String("tileId", tile.TileID),
				zap.Error(previewErr))
			return nil, previewErr
		}

		sheets = append(sheets, render.DashboardSheet{
			Title:     tile.Title,
			Schema:    result.Columns,
			Rows:      result.Rows,
			Totals:    result.Totals,
			Truncated: result.Truncated,
		})
	}

	var buf bytes.Buffer
	renderErr := render.NewXLSX().RenderDashboard(&render.DashboardRenderRequest{
		Title:       dashboard.Name,
		Description: dashboard.Description,
		GeneratedAt: timeutils.NowUnix(),
		Timezone:    s.orgTimezone(ctx, req.TenantInfo),
		RequestedBy: req.TenantInfo.UserID.String(),
		Filters:     req.FilterValues,
		Params:      req.ParamValues,
		Sheets:      sheets,
		Sink:        &buf,
	})
	if renderErr != nil {
		s.l.Error("failed to render dashboard export", zap.Error(renderErr))
		return nil, renderErr
	}

	return &DashboardExport{
		FileName:    dashboardExportFileName(dashboard),
		ContentType: dashboardExportContentType,
		Body:        buf.Bytes(),
	}, nil
}

// applyTileFilterOverride folds a tile's own narrowing on top of the
// dashboard's shared filters, in that order — the same precedence the screen
// applies, so both override a matching condition rather than stacking a
// second one on the same field.
func applyTileFilterOverride(
	definition *report.Definition,
	override TileFilterOverride,
) *report.Definition {
	if len(override.Filters) == 0 || len(override.Values) == 0 {
		return definition
	}
	scoped := &report.DashboardLayout{Filters: override.Filters}
	return scoped.ApplyFilters(definition, override.Values)
}

// dashboardExportFileName keeps the dashboard recognisable on disk and dates
// the file, since an export is a snapshot rather than a living document. It
// uses the same sanitiser as a report download so the two agree.
func dashboardExportFileName(dashboard *report.Dashboard) string {
	stamp := time.Unix(timeutils.NowUnix(), 0).UTC().Format("2006-01-02")
	return fmt.Sprintf(
		"%s %s.xlsx",
		fileutils.SanitizeDisplayFilename(dashboard.Name, "dashboard", maxDownloadNameLen),
		stamp,
	)
}
