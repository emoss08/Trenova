package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingMoves struct {
	serviceports.ShipmentMoveService

	move  *shipment.ShipmentMove
	guard writeGuard
}

func (f *savingMoves) arrive(
	move *shipment.ShipmentMove,
	req *repositories.RecordStopActualRequest,
) error {
	stop := findStop(move, req.StopID)
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
	stop.Status = shipment.StopStatusInTransit
	move.Status = shipment.MoveStatusInTransit

	return nil
}

func (f *savingMoves) PreviewStopActual(
	_ context.Context,
	req *repositories.RecordStopActualRequest,
) (*serviceports.StopActualPlan, error) {
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

	return &serviceports.StopActualPlan{Before: before, After: after}, nil
}

func (f *savingMoves) RecordStopActual(
	_ context.Context,
	req *repositories.RecordStopActualRequest,
) (*shipment.ShipmentMove, error) {
	if err := f.guard.write(); err != nil {
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
		Status:  shipment.MoveStatusAssigned,
		Version: 3,
		Stops: []*shipment.Stop{{
			ID:       pulid.MustNew("stp_"),
			Type:     shipment.StopTypePickup,
			Status:   shipment.StopStatusNew,
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
	tool := newRecordStopActualTool(moves).(*recordStopActualTool)
	occurredAt := int64(1790000000)
	params := executeParams(map[string]any{
		"moveId":     move.ID.String(),
		"stopId":     move.Stops[0].ID.String(),
		"action":     "Arrive",
		"occurredAt": float64(occurredAt),
	})

	preview := previewWithoutWrites(t, &moves.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceShipmentMove, change.Resource)
	assert.Equal(t, "Pickup at Reno Cold Storage", change.Label)
	arrival := fieldByPath(t, change, "actualArrival")
	assert.False(t, arrival.Volatile, "a time the caller gave is part of what is approved")
	assert.Equal(t, "InTransit", fieldByPath(t, change, "moveStatus").After)
	assert.Contains(t, preview.Summary, "from Assigned to InTransit")

	request, err := tool.request(&params)
	require.NoError(t, err)
	require.NoError(t, tool.Execute(t.Context(), params))
	stopID := move.Stops[0].ID
	requireUpdateParity(t, change,
		stopActualViewOf(before, stopID), stopActualViewOf(move, stopID),
		stopActualOptions(request)...)
}

func TestRecordStopActual_PreviewMarksNowAsVolatile(t *testing.T) {
	t.Parallel()

	move := dispatchedMove()
	moves := &savingMoves{move: move}
	tool := newRecordStopActualTool(moves).(*recordStopActualTool)

	preview := previewWithoutWrites(t, &moves.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			"moveId": move.ID.String(),
			"stopId": move.Stops[0].ID.String(),
			"action": "Arrive",
		}))
	})

	assert.True(t, fieldByPath(t, previewChange(t, preview, 0), "actualArrival").Volatile)
	assert.Contains(t, preview.Summary, "as happening now")
}

func TestRecordStopActual_PreviewWarnsWhenTheServiceWouldRefuse(t *testing.T) {
	t.Parallel()

	move := dispatchedMove()
	arrived := int64(100)
	move.Stops[0].ActualArrival = &arrived
	moves := &savingMoves{move: move}
	tool := newRecordStopActualTool(moves).(*recordStopActualTool)

	preview := previewWithoutWrites(t, &moves.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			"moveId": move.ID.String(),
			"stopId": move.Stops[0].ID.String(),
			"action": "Arrive",
		}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
