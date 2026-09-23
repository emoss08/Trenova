package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDashboardLister struct {
	dashboards []*report.Dashboard
}

func (f *fakeDashboardLister) ListDashboards(
	_ context.Context,
	_ *reporting.ListDashboardsRequest,
) ([]*report.Dashboard, error) {
	return f.dashboards, nil
}

/*
The dashboard list says what is on each dashboard.

It claimed to and returned a tile count. "Is on-time already on the ops page"
could only be answered by guessing, and a model that guesses no builds a second
dashboard.
*/
func TestListDashboards_SaysWhatIsOnEachAndSearchesByName(t *testing.T) {
	t.Parallel()

	reportID := pulid.MustNew("rdef_")
	tool := &listDashboardsTool{dashboards: &fakeDashboardLister{dashboards: []*report.Dashboard{
		{
			ID:   pulid.MustNew("rdash_"),
			Name: "Operations",
			Layout: &report.DashboardLayout{Tiles: []report.DashboardTile{
				{Kind: report.TileKindKPI, Title: "On time", DefinitionID: reportID},
				{Kind: report.TileKindTable, CannedKey: "ar_aging"},
			}},
		},
		{ID: pulid.MustNew("rdash_"), Name: "Billing", Category: "Finance"},
	}}}

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "operations"}))
	require.NoError(t, err)

	rows := result.(searchOutcome).Items.([]dashboardRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "Operations", rows[0].Name)
	assert.Equal(t, []dashboardTileRow{
		{Kind: "kpi", Title: "On time", DefinitionID: reportID.String()},
		{Kind: "table", CannedKey: "ar_aging"},
	}, rows[0].Tiles)
}
