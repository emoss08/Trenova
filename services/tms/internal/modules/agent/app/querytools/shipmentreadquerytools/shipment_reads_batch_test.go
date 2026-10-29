package shipmentreadquerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeNewestComments struct {
	comments []*shipment.ShipmentComment
	calls    int
	asked    []pulid.ID
}

func (f *fakeNewestComments) NewestComments(
	_ context.Context,
	req *shipmentports.NewestCommentsRequest,
) ([]*shipment.ShipmentComment, error) {
	f.calls++
	f.asked = append(f.asked, req.ShipmentIDs...)

	return f.comments, nil
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
	tool := shipmentquerytools.NewShipmentListTool(&fakeShipmentRepo{items: items}, shipmentquerytools.PartySources{}).
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
The same question took a shipment_get per held load. shipment_get now takes
every load at once and reads them, their holds and their newest comments in
one read each.
*/
func TestGetShipment_SeveralLoadsAreOneTableReadOnceForEachSource(t *testing.T) {
	t.Parallel()

	first := &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "S-1",
		Status: shipment.StatusDelayed}
	second := &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "S-2",
		Status: shipment.StatusDelayed}
	missing := pulid.MustNew("shp_")
	repo := &fakeTrackingShipments{listed: []*shipment.Shipment{second, first}}
	holds := &fakeActiveHolds{holds: []*shipmenthold.ShipmentHold{
		noteHold(first.ID, shipmenthold.HoldSourceUser, "Holding for a new appointment."),
	}}
	comments := &fakeNewestComments{comments: []*shipment.ShipmentComment{
		{
			ID: pulid.MustNew("shc_"), ShipmentID: first.ID, Comment: "Receiver closed",
			Source: shipment.CommentSourceUser,
		},
		{
			ID: pulid.MustNew("shc_"), ShipmentID: second.ID, Comment: "ETA 4pm",
			Source: shipment.CommentSourceIntegration,
		},
	}}
	tool := newGetShipmentTool(repo, nil, nil, &querytooltest.FakePermissions{Allowed: true}).
		withBatchReads(holds, comments)

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"shipmentIds": []any{first.ID.String(), second.ID.String(), missing.String()},
	}))
	require.NoError(t, err)

	outcome, ok := result.(*shipmentReadsOutcome)
	require.True(t, ok)
	require.Len(t, outcome.Items, 2)
	assert.Equal(t, "S-1", outcome.Items[0].ProNumber, "rows come in the order named")
	require.Len(t, outcome.Items[0].ActiveHolds, 1)
	assert.Equal(t, "Holding for a new appointment.", outcome.Items[0].ActiveHolds[0].Notes)
	require.NotNil(t, outcome.Items[0].NewestComment)
	assert.Equal(t, "Receiver closed", outcome.Items[0].NewestComment.Comment)
	assert.Empty(t, outcome.Items[1].ActiveHolds)
	assert.Contains(t, outcome.Note, missing.String())

	assert.Equal(t, 1, repo.lists)
	assert.Equal(t, 1, holds.calls)
	assert.Equal(t, 1, comments.calls)
	assert.ElementsMatch(t, []pulid.ID{first.ID, second.ID}, comments.asked)

	require.Len(t, outcome.TaintedRecords(), 1, "the integration's comment marks the read")
	assert.Equal(t, comments.comments[1].ID.String(), outcome.TaintedRecords()[0].ID)
	assert.NotEmpty(t, outcome.CommentsNote)
	assert.Equal(t, shipmentReadColumns, encodedColumns(t, outcome))
}

func TestGetShipment_SeveralLoadsReadOneAtATimeOnlyInFull(t *testing.T) {
	t.Parallel()

	tool := newGetShipmentTool(&fakeTrackingShipments{}, nil, nil,
		&querytooltest.FakePermissions{Allowed: true})

	_, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"shipmentIds": []any{pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()},
		"detail":      "full",
	}))
	require.ErrorContains(t, err, "detail full reads one shipment")
}

func encodedColumns(t *testing.T, result any) []string {
	t.Helper()

	raw, ok := querytooltest.EncodedDocument(t, result)["columns"].([]any)
	require.True(t, ok)
	columns := make([]string, 0, len(raw))
	for _, column := range raw {
		columns = append(columns, column.(string))
	}

	return columns
}
