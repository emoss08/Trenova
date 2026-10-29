package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/dashboardtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/documenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/inboundtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ptodecisiontools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ratetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentcommenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetOf_NamesTheRecordFromItsArgument(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	target, ok := toolparams.TargetOf(
		map[string]any{"shipmentId": id.String()},
		"shipmentId",
		permission.ResourceShipment,
	)

	require.True(t, ok)
	assert.Equal(
		t,
		agenttoolports.ToolTarget{Resource: permission.ResourceShipment, ID: id},
		target,
	)
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
			_, ok := toolparams.TargetOf(params, "shipmentId", permission.ResourceShipment)
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
		tool     agenttoolports.AgentTool
		key      string
		resource permission.Resource
	}{
		{&shipmentcommenttools.AddShipmentCommentTool{}, "shipmentId", permission.ResourceShipment},
		// The document, not the shipment: attaching changes the document's
		// lineage, so that is the record whose version the proposal pins.
		{&documenttools.AttachDocumentTool{}, "documentId", permission.ResourceDocument},
		{&shipmenttools.PlaceShipmentHoldTool{}, "shipmentId", permission.ResourceShipment},
		{&shipmenttools.ReleaseShipmentHoldTool{}, "shipmentId", permission.ResourceShipment},
		{&shipmenttools.CancelShipmentTool{}, "shipmentId", permission.ResourceShipment},
		{&movetools.RecordStopActualTool{}, "moveId", permission.ResourceShipmentMove},
		{&movetools.AssignMoveTool{}, "shipmentMoveId", permission.ResourceShipmentMove},
		{&ptodecisiontools.ApproveWorkerPTOTool{}, "ptoId", permission.ResourceWorkerPTO},
		{&ptodecisiontools.RejectWorkerPTOTool{}, "ptoId", permission.ResourceWorkerPTO},
		{&ptodecisiontools.CancelWorkerPTOTool{}, "ptoId", permission.ResourceWorkerPTO},
		{
			&invoicetools.CorrectChargeCodeTool{},
			"billingQueueItemId",
			permission.ResourceBillingQueue,
		},
		{&dashboardtools.AddDashboardTileTool{}, "dashboardId", permission.ResourceDashboard},
		{&alerttools.ScheduleReportTool{}, "definitionId", permission.ResourceReport},
		{&inboundtools.LinkInboundMessageTool{}, "messageId", permission.ResourceInboundMessage},
		{&inboundtools.MarkInboundMessageTool{}, "messageId", permission.ResourceInboundMessage},
		// The message, not a new record: a reply to a message somebody has
		// since settled or relinked must be refused as stale.
		{&inboundtools.ReplyToInboundMessageTool{}, "messageId", permission.ResourceInboundMessage},
		{
			&carriertools.AssignMoveToCarrierTool{},
			"shipmentMoveId",
			permission.ResourceShipmentMove,
		},
		{
			&carriertools.CancelCarrierAssignmentTool{},
			"shipmentMoveId",
			permission.ResourceShipmentMove,
		},
		// The tender, whose version moves as it advances or ends, so a
		// withdrawal proposed against a tender that has since moved is stale.
		{&tendertools.CancelTenderTool{}, "tenderId", permission.ResourceTender},
		{
			&ratetools.GenerateRateConfirmationTool{},
			"shipmentMoveId",
			permission.ResourceShipmentMove,
		},
		{
			&ratetools.SendRateConfirmationTool{},
			"rateConfirmationId",
			permission.ResourceRateConfirmation,
		},
		{
			&ratetools.VoidRateConfirmationTool{},
			"rateConfirmationId",
			permission.ResourceRateConfirmation,
		},
		{
			&ratetools.RecordRateConfirmationConfirmedTool{},
			"rateConfirmationId",
			permission.ResourceRateConfirmation,
		},
	}

	for _, tc := range cases {
		targeted, ok := tc.tool.(agenttoolports.TargetedTool)
		require.True(t, ok, "%T names no target", tc.tool)
		target, found := targeted.Target(map[string]any{tc.key: id.String()})
		require.True(t, found, "%T", tc.tool)
		assert.Equal(t, tc.resource, target.Resource, "%T", tc.tool)
		assert.Equal(t, id, target.ID, "%T", tc.tool)
	}
}
