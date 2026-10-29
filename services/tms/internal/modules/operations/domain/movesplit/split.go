package movesplit

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
)

type MoveSplitSpec struct {
	NewDeliveryLocationID pulid.ID
	RelayPickupStart      int64
	RelayPickupEnd        *int64
	NewDeliveryStart      int64
	NewDeliveryEnd        *int64
	Pieces                *int64
	Weight                *int64
}

type MoveSplit struct {
	RelayStop *shipment.Stop
	NewMove   *shipment.ShipmentMove
}

func PlanMoveSplit(original *shipment.ShipmentMove, spec *MoveSplitSpec) *MoveSplit {
	relay := *original.Stops[1]
	relay.Type = shipmentvocab.StopTypeSplitDelivery
	relay.Status = shipmentvocab.StopStatusNew

	newMove := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: original.BusinessUnitID,
		OrganizationID: original.OrganizationID,
		ShipmentID:     original.ShipmentID,
		Status:         shipmentvocab.MoveStatusNew,
		Loaded:         true,
		Sequence:       original.Sequence + 1,
		Distance:       original.Distance,
	}
	newMove.Stops = []*shipment.Stop{
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       original.BusinessUnitID,
			OrganizationID:       original.OrganizationID,
			ShipmentMoveID:       newMove.ID,
			LocationID:           relay.LocationID,
			Status:               shipmentvocab.StopStatusNew,
			Type:                 shipmentvocab.StopTypeSplitPickup,
			Sequence:             0,
			Pieces:               spec.Pieces,
			Weight:               spec.Weight,
			ScheduledWindowStart: spec.RelayPickupStart,
			ScheduledWindowEnd:   spec.RelayPickupEnd,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       original.BusinessUnitID,
			OrganizationID:       original.OrganizationID,
			ShipmentMoveID:       newMove.ID,
			LocationID:           spec.NewDeliveryLocationID,
			Status:               shipmentvocab.StopStatusNew,
			Type:                 shipmentvocab.StopTypeDelivery,
			Sequence:             1,
			Pieces:               spec.Pieces,
			Weight:               spec.Weight,
			ScheduledWindowStart: spec.NewDeliveryStart,
			ScheduledWindowEnd:   spec.NewDeliveryEnd,
		},
	}

	return &MoveSplit{RelayStop: &relay, NewMove: newMove}
}
