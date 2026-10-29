package shipmentcontrol

import (
	"github.com/emoss08/trenova/shared/pulid"
)

func NewDefaultShipmentControl(orgID, buID pulid.ID) *ShipmentControl {
	threshold := int16(30)

	return &ShipmentControl{
		OrganizationID:               orgID,
		BusinessUnitID:               buID,
		MaxShipmentWeightLimit:       80000,
		AutoDelayShipments:           true,
		AutoDelayShipmentsThreshold:  &threshold,
		DetentionThreshold:           new(threshold),
		AutoCancelShipmentsThreshold: new(threshold),
		AllowMoveRemovals:            true,
		CheckForDuplicateBOLs:        true,
		CheckHazmatSegregation:       true,
	}
}
