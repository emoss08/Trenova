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
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type Stop struct {
	bun.BaseModel `bun:"table:stops,alias:st" json:"-"`

	ID               uuid.UUID                   `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status           property.ShipmentMoveStatus `bun:"type:VARCHAR(50),notnull" json:"status"`
	Type             property.StopType           `bun:"type:stop_type_enum,default:'Pickup',notnull" json:"type"`
	AddressLine      string                      `bun:"type:TEXT" json:"addressLine"`
	Notes            string                      `bun:"type:TEXT,nullzero" json:"notes"`
	SequenceNumber   int                         `bun:"type:INTEGER,notnull" json:"sequenceNumber"`
	Pieces           decimal.NullDecimal         `bun:"type:NUMERIC(10,2),default:0" json:"pieces"`
	Weight           decimal.NullDecimal         `bun:"type:NUMERIC(10,2),default:0" json:"weight"`
	PlannedArrival   time.Time                   `bun:"type:TIMESTAMP" json:"plannedArrival"`
	PlannedDeparture time.Time                   `bun:"type:TIMESTAMP" json:"plannedDeparture"`
	ActualArrival    *time.Time                  `bun:"type:TIMESTAMP,nullzero" json:"actualArrival"`
	ActualDeparture  *time.Time                  `bun:"type:TIMESTAMP,nullzero" json:"actualDeparture"`
	Version          int64                       `bun:"type:BIGINT" json:"version"`
	CreatedAt        time.Time                   `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt        time.Time                   `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	LocationID     uuid.UUID `bun:"type:uuid,notnull" json:"locationId"`
	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`
	ShipmentMoveID uuid.UUID `bun:"type:uuid,notnull" json:"shipmentMoveId"`

	Location     *Location     `bun:"rel:belongs-to,join:location_id=id" json:"location"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	ShipmentMove *ShipmentMove `bun:"rel:belongs-to,join:shipment_move_id=id" json:"-"`
}

func (s Stop) Validate() error {
	return validation.ValidateStruct(
		&s,
		validation.Field(&s.Status, validation.Required),
		validation.Field(&s.Type, validation.Required),
		validation.Field(&s.PlannedArrival, validation.By(validateAppointmentWindow(s.PlannedArrival, s.PlannedDeparture))),
	)
}

func (s *Stop) BeforeUpdate(_ context.Context) error {
	s.Version++

	return nil
}

func (s *Stop) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := s.Version

	if err := s.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(s).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return &validator.BusinessLogicError{
			Message: fmt.Sprintf("Version mismatch. The Stop (ID: %s) has been updated by another user. Please refresh and try again.", s.ID),
		}
	}

	return nil
}

func (s *Stop) UpdateStatus(ctx context.Context, db *bun.DB, newStatus property.ShipmentMoveStatus) error {
	if err := s.setStatus(ctx, db, newStatus); err != nil {
		return err
	}

	// Update the associated movement
	move := &ShipmentMove{ID: s.ShipmentMoveID}
	if err := move.UpdateStatus(ctx, db); err != nil {
		return err
	}

	// Fetch and update the associated shipment
	shipment := new(Shipment)
	err := db.NewSelect().Model(shipment).Where("id = ?", move.ShipmentID).Scan(ctx)
	if err != nil {
		return err
	}

	return shipment.UpdateStatus(ctx, db)
}

func validateAppointmentWindow(plannedArrival, plannedDepart time.Time) validation.RuleFunc {
	return func(_ any) error {
		if plannedArrival.After(plannedDepart) {
			return validator.DBValidationError{Field: "PlannedArrival", Message: "Planned arrival must be before planned departure"}
		}

		return nil
	}
}

var _ bun.BeforeAppendModelHook = (*Stop)(nil)

func (s *Stop) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		s.UpdatedAt = time.Now()
	}
	return nil
}

// Helper method to set status and handle database updates
func (s *Stop) setStatus(ctx context.Context, db *bun.DB, newStatus property.ShipmentMoveStatus) error {
	s.Status = newStatus
	now := time.Now()

	switch {
	case newStatus == property.ShipmentMoveStatusInProgress && s.ActualArrival != nil:
		s.ActualArrival = &now
	case newStatus == property.ShipmentMoveStatusCompleted && s.ActualDeparture != nil:
		s.ActualDeparture = &now
	}

	_, err := db.NewUpdate().Model(s).Column("status", "actual_arrival", "actual_departure").WherePK().Exec(ctx)
	return err
}
