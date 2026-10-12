package masterdatatools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/shared/stringutils"
)

var dashboardViewLabels = map[string]string{common.FieldTiles: "Tiles, in order"}

type dashboardView struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Visibility  string   `json:"visibility"`
	Tiles       []string `json:"tiles"`
}

func DashboardViewOf(entity *report.Dashboard) *dashboardView {
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

func (t *CreateDashboardTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := common.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	planned := reporting.NewDashboard(request)
	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceDashboard,
		Label:    planned.Name,
	}, DashboardViewOf(planned), toolpreview.Labels(dashboardViewLabels))
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would create the report dashboard %q with %s, %s.",
		planned.Name,
		common.CountOf(len(planned.Layout.Tiles), "tile"),
		dashboardAudience(planned.Visibility),
	), change)

	return common.WarnRefusal(preview, t.Validate(ctx, params))
}

func (t *AddDashboardTileTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := common.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, existing, err := t.request(ctx, &params)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(
				toolpreview.Build("Would add a tile to a dashboard."),
				err,
			), nil
		}

		return nil, err
	}

	updated := *existing
	reporting.ApplyDashboardSave(&updated, request)
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceDashboard,
		ID:       existing.ID,
		Label:    existing.Name,
		Version:  common.PreviewVersion(existing.Version),
	}, DashboardViewOf(existing), DashboardViewOf(&updated), toolpreview.Labels(dashboardViewLabels))
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would add a tile to the bottom of the dashboard %q, which is %s.",
		existing.Name,
		dashboardAudience(existing.Visibility),
	), change)

	return common.WarnRefusal(preview, t.Validate(ctx, params))
}
