package movetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingMoves struct {
	shipmentmoveports.ShipmentMoveService

	move  *shipment.ShipmentMove
	guard agenttoolstest.WriteGuard
}

func (f *savingMoves) arrive(
	move *shipment.ShipmentMove,
	req *shipmentmoveports.RecordStopActualRequest,
) error {
	stop := shipmentpreview.FindStop(move, req.StopID)
	if stop == nil {
		return errortypes.NewNotFoundError("Stop not found on this load")
	}
	if stop.ActualArrival != nil {
		return errortypes.NewBusinessError("You've already arrived at this stop")
	}
	at := timeutils.NowUnix()
	if req.OccurredAt != nil {
		at = *req.OccurredAt
	}
	stop.ActualArrival = &at
	stop.Status = shipmentvocab.StopStatusInTransit
	move.Status = shipmentvocab.MoveStatusInTransit

	return nil
}

func (f *savingMoves) PreviewStopActual(
	_ context.Context,
	req *shipmentmoveports.RecordStopActualRequest,
) (*shipmentmoveports.StopActualPlan, error) {
	before := new(shipment.ShipmentMove)
	after := new(shipment.ShipmentMove)
	if err := jsonutils.Convert(f.move, before); err != nil {
		return nil, err
	}
	if err := jsonutils.Convert(f.move, after); err != nil {
		return nil, err
	}
	if err := f.arrive(after, req); err != nil {
		return nil, err
	}

	return &shipmentmoveports.StopActualPlan{Before: before, After: after}, nil
}

func (f *savingMoves) RecordStopActual(
	_ context.Context,
	req *shipmentmoveports.RecordStopActualRequest,
) (*shipment.ShipmentMove, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	if err := f.arrive(f.move, req); err != nil {
		return nil, err
	}

	return f.move, nil
}

func dispatchedMove() *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		ID:      pulid.MustNew("smv_"),
		Status:  shipmentvocab.MoveStatusAssigned,
		Version: 3,
		Stops: []*shipment.Stop{{
			ID:       pulid.MustNew("stp_"),
			Type:     shipmentvocab.StopTypePickup,
			Status:   shipmentvocab.StopStatusNew,
			Location: &location.Location{Name: "Reno Cold Storage"},
		}},
	}
}

func TestRecordStopActual_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	move := dispatchedMove()
	before := new(shipment.ShipmentMove)
	require.NoError(t, jsonutils.Convert(move, before))
	moves := &savingMoves{move: move}
	tool := newRecordStopActualTool(moves, reportsFor(moves), nil).(*RecordStopActualTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":     move.ID.String(),
		"stopId":     move.Stops[0].ID.String(),
		"action":     "Arrive",
		"occurredAt": "2026-09-21T14:13:20",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&moves.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceShipmentMove, change.Resource)
	assert.Equal(t, "Pickup at Reno Cold Storage", change.Label)
	arrival := agenttoolstest.FieldByPath(t, change, "actualArrival")
	assert.False(t, arrival.Volatile, "a time the caller gave is part of what is approved")
	assert.Equal(t, "InTransit", agenttoolstest.FieldByPath(t, change, "moveStatus").After)
	assert.Contains(t, preview.Summary, "from Assigned to InTransit")

	request, err := tool.request(t.Context(), &params)
	require.NoError(t, err)
	require.NoError(t, tool.Execute(t.Context(), params))
	stopID := move.Stops[0].ID
	agenttoolstest.RequireUpdateParity(t, change,
		stopActualViewOf(before, stopID), stopActualViewOf(move, stopID),
		stopActualOptions(request)...)
}

func TestRecordStopActual_PreviewMarksNowAsVolatile(t *testing.T) {
	t.Parallel()

	move := dispatchedMove()
	moves := &savingMoves{move: move}
	tool := newRecordStopActualTool(moves, reportsFor(moves), nil).(*RecordStopActualTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&moves.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"moveId": move.ID.String(),
				"stopId": move.Stops[0].ID.String(),
				"action": "Arrive",
			}))
		},
	)

	assert.True(
		t,
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			"actualArrival",
		).Volatile,
	)
	assert.Contains(t, preview.Summary, "as happening now")
}

func TestRecordStopActual_PreviewWarnsWhenTheServiceWouldRefuse(t *testing.T) {
	t.Parallel()

	move := dispatchedMove()
	arrived := int64(100)
	move.Stops[0].ActualArrival = &arrived
	moves := &savingMoves{move: move}
	tool := newRecordStopActualTool(moves, reportsFor(moves), nil).(*RecordStopActualTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&moves.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"moveId": move.ID.String(),
				"stopId": move.Stops[0].ID.String(),
				"action": "Arrive",
			}))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}
