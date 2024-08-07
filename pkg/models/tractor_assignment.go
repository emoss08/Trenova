// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package models

import (
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TractorAssignment struct {
	bun.BaseModel `bun:"table:tractor_assignments,alias:ta" json:"-"`

	ID          uuid.UUID                 `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Sequence    int                       `bun:"type:integer,notnull" json:"sequence"`
	AssignedAt  time.Time                 `bun:"type:TIMESTAMPTZ,notnull" json:"assignedAt"`
	CompletedAt *time.Time                `bun:"type:TIMESTAMPTZ,nullzero" json:"completedAt"`
	Status      property.AssignmentStatus `bun:"type:varchar(20),notnull" json:"status"`

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

// TODO(Wolfred): Add a validate method.
