package masterdatatools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDashboards struct {
	dashboardWriter
	reports   map[pulid.ID]bool
	canned    map[string]bool
	dashboard *report.Dashboard
}

func (f *fakeDashboards) UnavailableTileTargets(
	_ context.Context,
	_ pagination.TenantInfo,
	tiles []report.DashboardTile,
) []reporting.UnavailableTile {
	var missing []reporting.UnavailableTile
	for i := range tiles {
		switch {
		case tiles[i].CannedKey != "":
			if !f.canned[tiles[i].CannedKey] {
				missing = append(missing, reporting.UnavailableTile{Index: i, Canned: true})
			}
		case !f.reports[tiles[i].DefinitionID]:
			missing = append(missing, reporting.UnavailableTile{Index: i})
		}
	}

	return missing
}

func (f *fakeDashboards) GetDashboard(
	_ context.Context,
	req *reporting.GetDashboardRequest,
) (*report.Dashboard, error) {
	if f.dashboard == nil || f.dashboard.ID != req.DashboardID {
		return nil, errortypes.NewNotFoundError("Dashboard not found")
	}

	return f.dashboard, nil
}

/*
An invented report id is refused before anybody approves the dashboard.

The Homepage Widget Builder proposed a dashboard with "on_time_percentage" as
its report id. Validation only checked that the field was filled, so the card
went to a person, was approved, and failed on execute. The refusal now comes
while the model can still fix the call, and names the tool that has the id.
*/
func TestCreateDashboard_RefusesAReportIDThatDoesNotExistBeforeProposing(t *testing.T) {
	t.Parallel()

	known := pulid.MustNew("rdef_")
	unknown := pulid.MustNew("rdef_")
	tool := &CreateDashboardTool{
		dashboards: &fakeDashboards{reports: map[pulid.ID]bool{known: true}},
	}

	err := tool.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name": "Operations",
		"tiles": []any{
			map[string]any{"kind": "table", "definitionId": known.String()},
			map[string]any{"kind": "kpi", "definitionId": "on_time_percentage", "columnId": "pct"},
			map[string]any{"kind": "chart", "definitionId": unknown.String()},
		},
	}))

	errs := agenttoolstest.FieldErrors(t, err)
	assert.NotContains(t, errs, "tiles[0].definitionId")
	assert.Contains(t, errs["tiles[1].definitionId"], "report_list")
	assert.Contains(t, errs["tiles[1].definitionId"], "on_time_percentage")
	assert.Contains(t, errs["tiles[2].definitionId"], "report_list")
}

// A built-in report is drawn by its key; a key that names nothing is refused
// the same way.
func TestCreateDashboard_DrawsABuiltInReportByKey(t *testing.T) {
	t.Parallel()

	tool := &CreateDashboardTool{
		dashboards: &fakeDashboards{canned: map[string]bool{"ar_aging": true}},
	}

	require.NoError(t, tool.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name":  "Receivables",
		"tiles": []any{map[string]any{"kind": "table", "cannedKey": "ar_aging"}},
	})))

	err := tool.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name":  "Receivables",
		"tiles": []any{map[string]any{"kind": "table", "cannedKey": "made_up"}},
	}))
	assert.Contains(t, agenttoolstest.FieldErrors(t, err)["tiles[0].cannedKey"], "report_list")

	tiles, err := buildTiles([]map[string]any{{"kind": "table", "cannedKey": "ar_aging"}}, 0)
	require.NoError(t, err)
	assert.Equal(t, "ar_aging", tiles[0].CannedKey)
}

func TestCreateDashboard_RefusesATileNamingBothKindsOfReport(t *testing.T) {
	t.Parallel()

	err := (&CreateDashboardTool{}).validateArgs(map[string]any{
		"name": "Both",
		"tiles": []any{map[string]any{
			"kind": "table", "definitionId": pulid.MustNew("rdef_").String(), "cannedKey": "ar_aging",
		}},
	})

	assert.Contains(t, agenttoolstest.FieldErrors(t, err), "tiles[0]")
}

func TestAddDashboardTile_RefusesADashboardThatIsNotThereBeforeProposing(t *testing.T) {
	t.Parallel()

	known := pulid.MustNew("rdef_")
	tool := &AddDashboardTileTool{dashboards: &fakeDashboards{
		reports:   map[pulid.ID]bool{known: true},
		dashboard: &report.Dashboard{ID: pulid.MustNew("rdash_")},
	}}

	err := tool.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"dashboardId": pulid.MustNew("rdash_").String(),
		"tile":        map[string]any{"kind": "table", "definitionId": known.String()},
	}))

	assert.Contains(t, agenttoolstest.FieldErrors(t, err)["dashboardId"], "report_list_dashboards")
}
