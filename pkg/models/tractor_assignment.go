// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
