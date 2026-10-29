package dashboardtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingDashboards struct {
	fakeDashboards

	saved *report.Dashboard
	guard agenttoolstest.WriteGuard
}

func (f *savingDashboards) CreateDashboard(
	_ context.Context,
	req *reporting.SaveDashboardRequest,
) (*report.Dashboard, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.saved = reporting.NewDashboard(req)
	f.saved.ID = pulid.MustNew("rdb_")

	return f.saved, nil
}

func (f *savingDashboards) UpdateDashboard(
	_ context.Context,
	req *reporting.SaveDashboardRequest,
) (*report.Dashboard, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	updated := *f.dashboard
	reporting.ApplyDashboardSave(&updated, req)
	f.saved = &updated

	return f.saved, nil
}

func TestCreateDashboard_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	reportID := pulid.MustNew("rdef_")
	dashboards := &savingDashboards{fakeDashboards: fakeDashboards{
		reports: map[pulid.ID]bool{reportID: true},
	}}
	tool := NewCreateDashboardTool(nil).(*CreateDashboardTool)
	tool.dashboards = dashboards
	params := agenttoolstest.ExecuteParams(map[string]any{
		"name":   "Weekly revenue",
		"shared": true,
		"tiles": []any{
			map[string]any{"kind": "chart", "definitionId": reportID.String(), "title": "By week"},
			map[string]any{"kind": "table", "definitionId": reportID.String()},
		},
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&dashboards.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Empty(t, preview.Warnings)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceDashboard, change.Resource)
	assert.Equal(t, "Weekly revenue", change.Label)
	tiles := agenttoolstest.FieldByPath(t, change, "tiles")
	assert.Equal(t, "Tiles, in order", tiles.Label)
	assert.Contains(t, tiles.After, "By week")
	assert.Contains(t, preview.Summary, "2 tiles")

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change, DashboardViewOf(dashboards.saved),
		toolpreview.Labels(dashboardViewLabels))
}

func TestCreateDashboard_PreviewWarnsOnAReportThatIsNotThere(t *testing.T) {
	t.Parallel()

	dashboards := &savingDashboards{}
	tool := NewCreateDashboardTool(nil).(*CreateDashboardTool)
	tool.dashboards = dashboards

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&dashboards.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"name": "Broken",
				"tiles": []any{
					map[string]any{
						"kind":         "table",
						"definitionId": pulid.MustNew("rdef_").String(),
					},
				},
			}))
		},
	)

	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

func TestAddDashboardTile_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	params := agenttoolstest.ExecuteParams(map[string]any{})
	existing := &report.Dashboard{
		ID:         pulid.MustNew("rdb_"),
		Name:       "Operations",
		OwnerID:    params.Actor.UserID,
		Visibility: report.VisibilityPrivate,
		Version:    9,
		Layout: &report.DashboardLayout{Tiles: []report.DashboardTile{
			{ID: "tile_1", Kind: report.TileKindText, Title: "Notes", Text: "x", W: 12, H: 2},
		}},
	}
	dashboards := &savingDashboards{fakeDashboards: fakeDashboards{
		canned:    map[string]bool{"on_time": true},
		dashboard: existing,
	}}
	tool := NewAddDashboardTileTool(nil).(*AddDashboardTileTool)
	tool.dashboards = dashboards
	params.Params = map[string]any{
		"dashboardId": existing.ID.String(),
		"tile":        map[string]any{"kind": "chart", "cannedKey": "on_time", "title": "On time"},
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&dashboards.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Empty(t, preview.Warnings)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, existing.ID, change.EntityID)
	tiles := agenttoolstest.FieldByPath(t, change, "tiles")
	assert.NotContains(t, tiles.Before, "On time")
	assert.Contains(t, tiles.After, "On time")

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(
		t,
		change,
		DashboardViewOf(existing),
		DashboardViewOf(dashboards.saved),
		toolpreview.Labels(dashboardViewLabels),
	)
}
