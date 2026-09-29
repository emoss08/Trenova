package shipment

import "github.com/emoss08/trenova/shared/pulid"

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
	RelayStop *Stop
	NewMove   *ShipmentMove
}

func PlanMoveSplit(original *ShipmentMove, spec *MoveSplitSpec) *MoveSplit {
	relay := *original.Stops[1]
	relay.Type = StopTypeSplitDelivery
	relay.Status = StopStatusNew

	newMove := &ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: original.BusinessUnitID,
		OrganizationID: original.OrganizationID,
		ShipmentID:     original.ShipmentID,
		Status:         MoveStatusNew,
		Loaded:         true,
		Sequence:       original.Sequence + 1,
		Distance:       original.Distance,
	}
	newMove.Stops = []*Stop{
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       original.BusinessUnitID,
			OrganizationID:       original.OrganizationID,
			ShipmentMoveID:       newMove.ID,
			LocationID:           relay.LocationID,
			Status:               StopStatusNew,
			Type:                 StopTypeSplitPickup,
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
			Status:               StopStatusNew,
			Type:                 StopTypeDelivery,
			Sequence:             1,
			Pieces:               spec.Pieces,
			Weight:               spec.Weight,
			ScheduledWindowStart: spec.NewDeliveryStart,
			ScheduledWindowEnd:   spec.NewDeliveryEnd,
		},
	}

	return &MoveSplit{RelayStop: &relay, NewMove: newMove}
}
