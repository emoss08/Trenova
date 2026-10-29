package worldwatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	watchedShipment = "shp_01J9WATCHWATCHWATCHWATCH0"
	otherShipment   = "shp_01J9OTHEROTHEROTHEROTHER0"
)

func TestWorldState_ReadsWatchTheRecordsAGetCallNamed(t *testing.T) {
	t.Parallel()

	world := OpenWorld("0.1-0", nil, 100)
	world.Read(&llmports.ToolCall{
		Name: "shipment_get",
		Arguments: map[string]any{
			"shipmentId":   watchedShipment,
			"customerID":   otherShipment,
			"includeMoves": true,
			"note":         "shp_01J9NOTANIDNOTANIDNOTANI0",
			"tractorId":    "short",
		},
	}, "S-1001")
	world.Read(&llmports.ToolCall{
		Name:      "shipment_list",
		Arguments: map[string]any{"customerId": "cus_01J9LISTLISTLISTLISTLIST0"},
	}, "3 shipments")

	ids := make([]string, 0, len(world.Records))
	for _, record := range world.Records {
		ids = append(ids, record.ID)
	}
	assert.Equal(t, []string{otherShipment, watchedShipment}, ids,
		"only id arguments of a get_ call are watched, in a fixed order")
	assert.Equal(t, "shipment", world.Records[1].Type)
	assert.Equal(t, "S-1001", world.Records[1].Label)
}

func TestWorldState_ItsOwnWriteIsNotNews(t *testing.T) {
	t.Parallel()

	world := OpenWorld(
		"0.1-0",
		[]agentpagecontext.EntityRef{{Type: "shipment", ID: watchedShipment}},
		100,
	)
	world.Wrote(&agentruntimeports.PendingAction{
		Executed: true,
		Target: &agentruntimeports.ProposalTarget{
			Resource: "shipment",
			ID:       pulid.ID(watchedShipment),
		},
	}, 200)
	world.Wrote(&agentruntimeports.PendingAction{
		Executed: false,
		Target: &agentruntimeports.ProposalTarget{
			Resource: "shipment",
			ID:       pulid.ID(otherShipment),
		},
	}, 200)

	news := world.News(
		&realtimeports.RecordChanges{Cursor: "0.3-0", Changes: []realtimeports.WatchedRecordChange{
			{RecordID: watchedShipment, Action: "updated", At: 205},
		}},
		210,
	)
	assert.Empty(t, news, "the turn's own write is not a change made elsewhere")
	assert.Len(t, world.Records, 1, "a write only proposed is not watched")

	news = world.News(&realtimeports.RecordChanges{Changes: []realtimeports.WatchedRecordChange{
		{RecordID: watchedShipment, Action: "updated", At: 200 + ownWriteGrace + 1},
	}}, 300)
	assert.Len(t, news, 1, "a later change to the same record is news again")
	assert.Equal(t, "0.3-0", world.Cursor, "an answer without a cursor keeps the last one")
}

func TestWorldState_DueOnlyAfterTheInterval(t *testing.T) {
	t.Parallel()

	world := OpenWorld("0.1-0", []agentpagecontext.EntityRef{{ID: watchedShipment}}, 100)
	assert.Nil(t, world.Due(100+WorldCheckEvery-1))
	require.NotNil(t, world.Due(100+WorldCheckEvery))

	var none *State
	assert.Nil(t, none.Due(1000), "a turn that watches nothing is never due")
	assert.Nil(t, OpenWorld("", nil, 0), "no cursor, no watch")
}

func TestWorldChangeNotice_SaysWhoChangedIt(t *testing.T) {
	t.Parallel()

	actor := runtimetest.Actor()
	notice := ChangeNotice([]realtimeports.WatchedRecordChange{
		{RecordID: watchedShipment, Label: "shipment S-1001", Action: "updated",
			ActorType: "session_user", ActorUserID: actor.UserID.String(), Fields: []string{"status"}},
		{RecordID: otherShipment, Label: otherShipment, Action: "deleted", ActorType: "agent"},
	}, actor)

	assert.Contains(t, notice, "shipment S-1001 ("+watchedShipment+") was changed by the person "+
		"you are working for, outside this conversation: status.")
	assert.Contains(t, notice, otherShipment+" was deleted by an agent.")
	assert.NotContains(t, notice, otherShipment+" ("+otherShipment+")")
}
