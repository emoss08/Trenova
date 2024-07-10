package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TractorAssignment struct {
	bun.BaseModel `bun:"table:tractor_assignments,alias:ta" json:"-"`

	ID          uuid.UUID  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Sequence    int        `bun:"type:integer,notnull" json:"sequence"`
	AssignedAt  time.Time  `bun:"type:TIMESTAMPTZ,notnull" json:"assignedAt"`
	CompletedAt *time.Time `bun:"type:TIMESTAMPTZ,nullzero" json:"completedAt"`
	Status      string     `bun:"type:varchar(20),notnull" json:"status"`

	TractorID      uuid.UUID `bun:"type:uuid,notnull" json:"tractorId"`
	ShipmentID     uuid.UUID `bun:"type:uuid,notnull" json:"shipmentId"`
	ShipmentMoveID uuid.UUID `bun:"type:uuid,notnull" json:"shipmentMoveId"`
	AssignedByID   uuid.UUID `bun:"type:uuid,notnull" json:"assignedById"`
	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	AssignedBy   *User         `bun:"rel:belongs-to,join:assigned_by_id=id" json:"assignedBy,omitempty"`
	Tractor      *Tractor      `bun:"rel:belongs-to,join:tractor_id=id" json:"tractor,omitempty"`
	Shipment     *Shipment     `bun:"rel:belongs-to,join:shipment_id=id" json:"shipment,omitempty"`
	ShipmentMove *ShipmentMove `bun:"rel:belongs-to,join:shipment_move_id=id" json:"shipmentMove,omitempty"`
}
