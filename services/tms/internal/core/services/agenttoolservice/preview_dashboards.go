package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/stringutils"
)

var (
	_ serviceports.ToolPreviewer = (*createDashboardTool)(nil)
	_ serviceports.ToolPreviewer = (*addDashboardTileTool)(nil)
)

var dashboardViewLabels = map[string]string{"tiles": "Tiles, in order"}

type dashboardView struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Visibility  string   `json:"visibility"`
	Tiles       []string `json:"tiles"`
}

func dashboardViewOf(entity *report.Dashboard) *dashboardView {
	view := &dashboardView{
		Name:        entity.Name,
		Description: entity.Description,
		Category:    entity.Category,
		Visibility:  string(entity.Visibility),
		Tiles:       []string{},
	}
	if entity.Layout == nil {
		return view
	}

	view.Tiles = make([]string, 0, len(entity.Layout.Tiles))
	for idx := range entity.Layout.Tiles {
		tile := &entity.Layout.Tiles[idx]
		view.Tiles = append(view.Tiles, stringutils.FirstNonEmpty(
			tile.Title,
			fmt.Sprintf("%s tile", tile.Kind),
		))
	}

	return view
}

func dashboardAudience(visibility report.Visibility) string {
	if visibility == report.VisibilityShared {
		return "shared with everyone in the organization"
	}

	return "private to you"
}

func (t *createDashboardTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	planned := reporting.NewDashboard(request)
	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceDashboard,
		Label:    planned.Name,
	}, dashboardViewOf(planned), toolpreview.Labels(dashboardViewLabels))
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would create the report dashboard %q with %s, %s.",
		planned.Name,
		countOf(len(planned.Layout.Tiles), "tile"),
		dashboardAudience(planned.Visibility),
	), change)

	return warnRefusal(preview, t.Validate(ctx, params))
}

func (t *addDashboardTileTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	request, existing, err := t.request(ctx, params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build("Would add a tile to a dashboard."), err), nil
		}

		return nil, err
	}

	updated := *existing
	reporting.ApplyDashboardSave(&updated, request)
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceDashboard,
		ID:       existing.ID,
		Label:    existing.Name,
		Version:  previewVersion(existing.Version),
	}, dashboardViewOf(existing), dashboardViewOf(&updated), toolpreview.Labels(dashboardViewLabels))
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would add a tile to the bottom of the dashboard %q, which is %s.",
		existing.Name,
		dashboardAudience(existing.Visibility),
	), change)

	return warnRefusal(preview, t.Validate(ctx, params))
}
