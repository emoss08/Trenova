package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetOf_NamesTheRecordFromItsArgument(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	target, ok := targetOf(map[string]any{"shipmentId": id.String()}, "shipmentId", permission.ResourceShipment)

	require.True(t, ok)
	assert.Equal(t, serviceports.ToolTarget{Resource: permission.ResourceShipment, ID: id}, target)
}

// A missing or malformed id yields no target and the call is still made: the
// tool's own Execute reports the bad argument in its own words.
func TestTargetOf_YieldsNothingForABadArgument(t *testing.T) {
	t.Parallel()

	for name, params := range map[string]map[string]any{
		"missing":   {},
		"empty":     {"shipmentId": ""},
		"malformed": {"shipmentId": "not-a-pulid"},
		"wrongType": {"shipmentId": 42},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, ok := targetOf(params, "shipmentId", permission.ResourceShipment)
			assert.False(t, ok)
		})
	}
}

// Every tool that changes one record says which. A tool added later that
// forgets is caught here, since the staleness check depends on it.
func TestSingleRecordTools_NameTheirTarget(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("x_")
	cases := []struct {
		tool     serviceports.AgentTool
		key      string
		resource permission.Resource
	}{
		{&addShipmentCommentTool{}, "shipmentId", permission.ResourceShipment},
		{&placeShipmentHoldTool{}, "shipmentId", permission.ResourceShipment},
		{&releaseShipmentHoldTool{}, "shipmentId", permission.ResourceShipment},
		{&cancelShipmentTool{}, "shipmentId", permission.ResourceShipment},
		{&recordStopActualTool{}, "moveId", permission.ResourceShipmentMove},
		{&assignMoveTool{}, "shipmentMoveId", permission.ResourceShipmentMove},
		{&approveWorkerPTOTool{}, "ptoId", permission.ResourceWorkerPTO},
		{&rejectWorkerPTOTool{}, "ptoId", permission.ResourceWorkerPTO},
		{&cancelWorkerPTOTool{}, "ptoId", permission.ResourceWorkerPTO},
		{&correctChargeCodeTool{}, "billingQueueItemId", permission.ResourceBillingQueue},
		{&addDashboardTileTool{}, "dashboardId", permission.ResourceDashboard},
		{&scheduleReportTool{}, "definitionId", permission.ResourceReport},
	}

	for _, tc := range cases {
		targeted, ok := tc.tool.(serviceports.TargetedTool)
		require.True(t, ok, "%T names no target", tc.tool)
		target, found := targeted.Target(map[string]any{tc.key: id.String()})
		require.True(t, found, "%T", tc.tool)
		assert.Equal(t, tc.resource, target.Resource, "%T", tc.tool)
		assert.Equal(t, id, target.ID, "%T", tc.tool)
	}
}
