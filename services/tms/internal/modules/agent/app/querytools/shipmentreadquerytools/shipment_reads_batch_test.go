package shipmentreadquerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeNewestComments struct {
	comments []*shipment.ShipmentComment
	totals   map[pulid.ID]int
	calls    int
	asked    []pulid.ID
	per      int
}

func (f *fakeNewestComments) NewestComments(
	_ context.Context,
	req *shipmentports.NewestCommentsRequest,
) (map[pulid.ID]*shipmentports.ShipmentCommentDigest, error) {
	f.calls++
	f.asked = append(f.asked, req.ShipmentIDs...)
	f.per = req.PerShipment

	out := make(map[pulid.ID]*shipmentports.ShipmentCommentDigest)
	for _, comment := range f.comments {
		digest, ok := out[comment.ShipmentID]
		if !ok {
			digest = &shipmentports.ShipmentCommentDigest{ShipmentID: comment.ShipmentID}
			out[comment.ShipmentID] = digest
		}
		digest.Newest = append(digest.Newest, comment)
		digest.Total = max(f.totals[comment.ShipmentID], len(digest.Newest))
	}

	return out, nil
}

func noteHold(
	shipmentID pulid.ID,
	source shipmenthold.HoldSource,
	note string,
) *shipmenthold.ShipmentHold {
	return &shipmenthold.ShipmentHold{
		ID:             pulid.MustNew("shh_"),
		ShipmentID:     shipmentID,
		Type:           holdreason.HoldTypeOperational,
		Severity:       holdreason.HoldSeverityBlocking,
		ReasonCode:     "MISSED_PICKUP",
		Source:         source,
		Notes:          note,
		StartedAt:      1_793_000_000,
		BlocksDispatch: true,
	}
}

/*
Asked which delayed loads were held and what the last note said, Haiku listed
the delayed loads, saw two held, and opened each with shipment_get for the
hold's note. The list row now carries the note a person or a hold rule wrote,
and when the hold started; a hold an integration raised keeps its text out.
*/
func TestShipmentList_RowsCarryTheHoldsNoteAndWhenItStarted(t *testing.T) {
	t.Parallel()

	items, _ := heldShipments()
	holds := &fakeActiveHolds{holds: []*shipmenthold.ShipmentHold{
		noteHold(items[0].ID, shipmenthold.HoldSourceUser, "Waiting on a corrected BOL."),
		noteHold(items[1].ID, shipmenthold.HoldSourceEDI, "partner text"),
	}}
	tool := shipmentquerytools.NewShipmentListTool(&fakeShipmentRepo{items: items}, &shipmentquerytools.PartySources{}).
		WithHolds(holds, &querytooltest.FakePermissions{Allowed: true})

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{}))
	require.NoError(t, err)

	rows, ok := querytooltest.EncodedDocument(t, result)["items"].([]any)
	require.True(t, ok)
	held := querytooltest.ObjectAt(t, rows, 0, "activeHolds", 0)
	assert.Equal(t, "Waiting on a corrected BOL.", held["note"])
	assert.NotNil(t, held["startedAt"])
	assert.NotContains(t, querytooltest.ObjectAt(t, rows, 1, "activeHolds", 0), "note",
		"a hold an integration raised shows what it blocks, not its text")
}

/*
Asked for new reefer loads with their comments and tracking, an agent listed them and then
read each in full. shipment_get now reads every load named as the same full record one load
gets, with its holds and its newest comments and how many there are, one read per source.
*/
func TestGetShipment_SeveralLoadsAreFullRecordsReadOnceForEachSource(t *testing.T) {
	t.Parallel()

	first := &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "S-1",
		Status: shipmentvocab.StatusDelayed}
	second := &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "S-2",
		Status: shipmentvocab.StatusDelayed}
	missing := pulid.MustNew("shp_")
	repo := &fakeTrackingShipments{listed: []*shipment.Shipment{second, first}}
	holds := &fakeActiveHolds{holds: []*shipmenthold.ShipmentHold{
		noteHold(first.ID, shipmenthold.HoldSourceUser, "Holding for a new appointment."),
	}}
	comments := &fakeNewestComments{
		comments: []*shipment.ShipmentComment{
			{
				ID: pulid.MustNew("shc_"), ShipmentID: first.ID, Comment: "Receiver closed",
				Source: shipmentvocab.CommentSourceUser,
			},
			{
				ID: pulid.MustNew("shc_"), ShipmentID: second.ID, Comment: "ETA 4pm",
				Source: shipmentvocab.CommentSourceIntegration,
			},
		},
		totals: map[pulid.ID]int{first.ID: 9},
	}
	tool := newGetShipmentTool(repo, nil, nil, &querytooltest.FakePermissions{Allowed: true}).
		withBatchReads(holds, comments)

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"shipmentIds": []any{first.ID.String(), second.ID.String(), missing.String()},
	}))
	require.NoError(t, err)

	document := querytooltest.EncodedDocument(t, result)
	rows, ok := document["items"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 2)
	firstRow, ok := rows[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "S-1", firstRow["proNumber"], "records come in the order named")
	assert.Contains(t, firstRow, "rating", "each record is the full summary, not a table row")
	assert.Contains(t, firstRow, "moves")
	assert.InDelta(t, 9, firstRow["commentCount"], 0)
	held := querytooltest.ObjectAt(t, rows, 0, "activeHolds", 0)
	assert.Equal(t, "Holding for a new appointment.", held["notes"])
	comment := querytooltest.ObjectAt(t, rows, 0, "recentComments", 0)
	assert.Equal(t, "Receiver closed", comment["comment"])
	assert.Contains(t, document["note"], missing.String())

	assert.Equal(t, 1, repo.lists)
	assert.Equal(t, 1, holds.calls)
	assert.Equal(t, 1, comments.calls)
	assert.Equal(t, batchCommentsPerShipment, comments.per)
	assert.ElementsMatch(t, []pulid.ID{first.ID, second.ID}, comments.asked)

	carrier, ok := result.(agent.TaintCarrier)
	require.True(t, ok)
	require.Len(t, carrier.TaintedRecords(), 1, "the integration's comment marks the read")
	assert.Equal(t, comments.comments[1].ID.String(), carrier.TaintedRecords()[0].ID)
	assert.NotEmpty(t, document["commentsNote"])
}

func TestGetShipment_SeveralLoadsReadInFull(t *testing.T) {
	t.Parallel()

	first := &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "S-1"}
	second := &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "S-2"}
	tool := newGetShipmentTool(
		&fakeTrackingShipments{listed: []*shipment.Shipment{first, second}},
		nil, nil, &querytooltest.FakePermissions{Allowed: true},
	)

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"shipmentIds": []any{first.ID.String(), second.ID.String()},
		"detail":      "full",
	}))
	require.NoError(t, err)

	rows, ok := querytooltest.EncodedDocument(t, result)["items"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 2)
	row, ok := rows[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "S-2", row["proNumber"])
	assert.Contains(t, row, "activeHolds")
}
