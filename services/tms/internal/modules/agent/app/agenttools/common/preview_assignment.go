package common

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
)

var assignedStatusLabels = map[string]string{"coverageType": "Covered by"}

type assignedMoveView struct {
	Status       string `json:"status"`
	CoverageType string `json:"coverageType"`
}

type ShipmentStatusView struct {
	Status string `json:"status"`
}

func assignedMoveOf(entity *shipment.Shipment, moveID pulid.ID) *assignedMoveView {
	move := entity.FindMove(moveID)
	if move == nil {
		return &assignedMoveView{}
	}

	return &assignedMoveView{Status: string(move.Status), CoverageType: string(move.CoverageType)}
}

// coverageChanges is what a change of cover does to the move and its
// shipment: the move's status and who covers it, and the shipment's status
// when the coordinator moves it. A record that does not change is left out.
func CoverageChanges(
	before, after *shipment.Shipment,
	moveID pulid.ID,
) ([]*agent.RecordChange, error) {
	shipmentLabel := before.ProNumber
	changes := make([]*agent.RecordChange, 0, 2)
	move, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		ID:       moveID,
		Label:    "Move on " + shipmentLabel,
		Version:  MoveVersion(before.FindMove(moveID)),
	}, assignedMoveOf(before, moveID),
		assignedMoveOf(after, moveID),
		toolpreview.Labels(assignedStatusLabels),
	)
	if err != nil {
		return nil, err
	}
	if len(move.Fields) > 0 {
		changes = append(changes, move)
	}

	shipmentChange, err := ShipmentStatusChange(before, after)
	if err != nil || shipmentChange == nil {
		return changes, err
	}

	return append(changes, shipmentChange), nil
}

// shipmentStatusChange is the shipment's status before and after, or nil when
// the write leaves it where it is.
func ShipmentStatusChange(before, after *shipment.Shipment) (*agent.RecordChange, error) {
	if before.Status == after.Status {
		return nil, nil //nolint:nilnil // an unmoved shipment is no change and no error
	}

	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       before.ID,
		Label:    before.ProNumber,
		Version:  PreviewVersion(before.Version),
	}, &ShipmentStatusView{Status: string(before.Status)},
		&ShipmentStatusView{Status: string(after.Status)},
	)
}

func MoveVersion(move *shipment.ShipmentMove) *int64 {
	if move == nil {
		return nil
	}

	return PreviewVersion(move.Version)
}
