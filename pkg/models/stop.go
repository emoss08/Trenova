package models

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Stop struct {
	bun.BaseModel    `bun:"table:stops,alias:st" json:"-"`
	Status           property.ShipmentMoveStatus `bun:"type:VARCHAR(50),notnull" json:"status"`
	Type             property.StopType           `bun:"type:stop_type_enum,default:'Pickup',notnull" json:"type"`
	AddressLine      string                      `bun:"type:TEXT" json:"addressLine"`
	Notes            string                      `bun:"type:TEXT,nullzero" json:"notes"`
	SequenceNumber   int                         `bun:"type:INTEGER,notnull" json:"sequenceNumber"`
	Pieces           *float64                    `bun:"type:NUMERIC(10,2),default:0" json:"pieces"`
	Weight           *float64                    `bun:"type:NUMERIC(10,2),default:0" json:"weight"`
	PlannedArrival   time.Time                   `bun:"type:TIMESTAMP" json:"plannedArrival"`
	PlannedDeparture time.Time                   `bun:"type:TIMESTAMP" json:"plannedDeparture"`
	ActualArrival    *time.Time                  `bun:"type:TIMESTAMP,nullzero" json:"actualArrival"`
	ActualDeparture  *time.Time                  `bun:"type:TIMESTAMP,nullzero" json:"actualDeparture"`
	CreatedAt        time.Time                   `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt        time.Time                   `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	ID             uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`
	ShipmentMoveID uuid.UUID `bun:"type:uuid,notnull" json:"shipmentMoveId"`
	LocationID     uuid.UUID `bun:"type:uuid,notnull" json:"locationId"`

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
			return errors.New("Planned arrival must be before planned departure. Please try again.")
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
	if newStatus == property.ShipmentMoveStatusInProgress && s.ActualArrival == nil {
		now := time.Now()
		s.ActualArrival = &now
	} else if newStatus == property.ShipmentMoveStatusCompleted && s.ActualDeparture == nil {
		now := time.Now()
		s.ActualDeparture = &now
	}
	_, err := db.NewUpdate().Model(s).Column("status", "actual_arrival", "actual_departure").WherePK().Exec(ctx)
	return err
}
