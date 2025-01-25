package shipment

import (
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

type Stop struct {
	bun.BaseModel `bun:"table:stops,alias:stp" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:",pk,type:VARCHAR(100),notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,pk,notnull,type:VARCHAR(100)" json:"organizationId"`

	// Relationship identifiers (Non-Primary-Keys)
	ShipmentMoveID pulid.ID `bun:"shipment_move_id,notnull,type:VARCHAR(100)" json:"shipmentMoveId"`
}
