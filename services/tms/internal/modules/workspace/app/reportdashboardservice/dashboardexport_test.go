package reportdashboardservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportingtest"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportDashboard_NeedsTheExportOperationOnEveryTile(t *testing.T) {
	t.Parallel()

	dashboard := dashboardWith([]report.DashboardTile{{
		ID:           "t1",
		Kind:         report.TileKindTable,
		DefinitionID: tileDefID,
	}}, nil)
	svc := tileTestService(t, dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: sharedDefinition("shipment", nil),
	})
	engine, ok := svc.engine.(*reportingtest.Engine)
	require.True(t, ok)

	_, err := svc.ExportDashboard(t.Context(), &ExportDashboardRequest{
		Request:     tileRequest(),
		DashboardID: pulid.MustNew("rdash_"),
	})
	require.Error(t, err)

	previews := engine.Recorded("preview")
	require.Len(t, previews, 1, "each tile runs once")
	access, ok := previews[0].Request.Policy.(*reportaccess.Access)
	require.True(t, ok)
	assert.Equal(t, reportaccess.ModeExport, access.Mode(), "an export decides access as an export")
}
