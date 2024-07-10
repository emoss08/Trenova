package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type ShipmentMove struct {
	bun.BaseModel     `bun:"table:shipment_moves,alias:sm" json:"-"`
	ID                uuid.UUID                   `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status            property.ShipmentMoveStatus `bun:"type:VARCHAR(50),notnull" json:"status"`
	IsLoaded          bool                        `bun:"type:BOOLEAN,default:false" json:"isLoaded"`
	SequenceNumber    int                         `bun:"type:INTEGER,notnull" json:"sequenceNumber"`
	EstimatedDistance decimal.NullDecimal         `bun:"type:NUMERIC(10,2),nullzero" json:"estimatedDistance"`
	ActualDistance    decimal.NullDecimal         `bun:"type:NUMERIC(10,2),nullzero" json:"actualDistance"`
	EstimatedCost     decimal.NullDecimal         `bun:"type:NUMERIC(19,4),nullzero" json:"estimatedCost"`
	ActualCost        decimal.NullDecimal         `bun:"type:NUMERIC(19,4),nullzero" json:"actualCost"`
	Notes             string                      `bun:"type:TEXT,nullzero" json:"notes"`
	CreatedAt         time.Time                   `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt         time.Time                   `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	ShipmentID        uuid.UUID  `bun:"type:uuid,notnull" json:"shipmentId"`
	TractorID         uuid.UUID  `bun:"type:uuid,notnull" json:"tractorId"`
	TrailerID         uuid.UUID  `bun:"type:uuid,notnull" json:"trailerId"`
	PrimaryWorkerID   uuid.UUID  `bun:"type:uuid,notnull" json:"primaryWorkerId"`
	SecondaryWorkerID *uuid.UUID `bun:"type:uuid,nullzero" json:"secondaryWorkerId"`
	BusinessUnitID    uuid.UUID  `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID    uuid.UUID  `bun:"type:uuid,notnull" json:"organizationId"`

	Shipment        *Shipment     `bun:"rel:belongs-to,join:shipment_id=id" json:"shipment,omitempty"`
	Tractor         *Tractor      `bun:"rel:belongs-to,join:tractor_id=id" json:"tractor,omitempty"`
	Trailer         *Trailer      `bun:"rel:belongs-to,join:trailer_id=id" json:"trailer,omitempty"`
	PrimaryWorker   *Worker       `bun:"rel:belongs-to,join:primary_worker_id=id" json:"primaryWorker,omitempty"`
	SecondaryWorker *Worker       `bun:"rel:belongs-to,join:secondary_worker_id=id" json:"secondaryWorker,omitempty"`
	BusinessUnit    *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization    *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	Stops           []*Stop       `bun:"rel:has-many,join:id=shipment_move_id" json:"stops,omitempty"`
}

var _ bun.BeforeAppendModelHook = (*ShipmentMove)(nil)

func (m *ShipmentMove) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		m.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		m.UpdatedAt = time.Now()
	}
	return nil
}

// Helper method to set status and handle database updates
func (m *ShipmentMove) setStatus(ctx context.Context, db *bun.DB, newStatus property.ShipmentMoveStatus) error {
	m.Status = newStatus
	_, err := db.NewUpdate().Model(m).Column("status").WherePK().Exec(ctx)
	return err
}

// UpdateMoveStatus updates the movement status based on its stops
func (m *ShipmentMove) UpdateStatus(ctx context.Context, db *bun.DB) error {
	// Fetch all stops for this movement
	var stops []*Stop
	err := db.NewSelect().Model(&stops).Where("shipment_move_id = ?", m.ID).Order("sequence_number ASC").Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch stops: %w", err)
	}

	if len(stops) == 0 {
		return errors.New("movement has no stops")
	}

	allCompleted := true
	anyInProgress := false

	for _, stop := range stops {
		switch stop.Status {
		case property.ShipmentMoveStatusCompleted:
			continue
		case property.ShipmentMoveStatusInProgress:
			anyInProgress = true
			allCompleted = false
		case property.ShipmentMoveStatusNew:
			allCompleted = false
		case property.ShipmentMoveStatusVoided:
			allCompleted = false
		default:
			allCompleted = false
		}
	}

	var newStatus property.ShipmentMoveStatus
	switch {
	case allCompleted:
		newStatus = property.ShipmentMoveStatusCompleted
	case anyInProgress:
		newStatus = property.ShipmentMoveStatusInProgress
	default:
		newStatus = property.ShipmentMoveStatusNew
	}

	return m.setStatus(ctx, db, newStatus)
}

// ValidateStopSequence ensures stops are in a valid order
func (m *ShipmentMove) ValidateStopSequence() error {
	if len(m.Stops) < 2 {
		return errors.New("movement must have at least two stops")
	}

	// Check if the first stop is a pickup
	if m.Stops[0].Type != property.StopTypePickup && m.Stops[0].Type != property.StopTypeSplitPickup {
		return errors.New("first stop must be Pickup or SplitPickup")
	}

	// Check if the last stop is a delivery
	lastStop := m.Stops[len(m.Stops)-1]
	if lastStop.Type != property.StopTypeDelivery && lastStop.Type != property.StopTypeDropOff {
		return errors.New("last stop must be Delivery or DropOff")
	}

	// Validate intermediate stops and sequence numbers
	for i, stop := range m.Stops {
		// Validate sequence number
		if stop.SequenceNumber != i+1 {
			return fmt.Errorf("incorrect sequence number for stop at position %d", i+1)
		}

		// Validate stop types
		if i == 0 {
			// First stop validation already done
			continue
		} else if i == len(m.Stops)-1 {
			// Last stop validation already done
			continue
		}

		// Intermediate stops can be any type
		switch stop.Type {
		case property.StopTypePickup, property.StopTypeSplitPickup, property.StopTypeSplitDrop, property.StopTypeDelivery, property.StopTypeDropOff:
			// All types are allowed for intermediate stops
		default:
			return fmt.Errorf("invalid stop type at position %d", i+1)
		}
	}

	return nil
}

func (m *ShipmentMove) AssignTractor(ctx context.Context, tx bun.Tx, tractorID uuid.UUID) error {
	if m.Status != property.ShipmentMoveStatusNew {
		return validator.BusinessLogicError{
			Message: "Movement must be in New status to assign a tractor",
		}
	}

	tractor := new(Tractor)
	err := tx.NewSelect().Model(tractor).Where("id = ?", tractorID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch tractor: %w", err)
	}

	m.TractorID = tractorID
	_, err = tx.NewUpdate().Model(m).Column("tractor_id").WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update movement with tractor: %w", err)
	}

	// Check if workers are available
	if tractor.PrimaryWorkerID == uuid.Nil {
		return validator.DBValidationError{
			Field:   "primaryWorker",
			Message: "No primary worker assigned to the selected tractor",
		}
	}

	// Assign primary worker
	m.PrimaryWorkerID = tractor.PrimaryWorkerID

	// Assign secondary worker if available
	m.SecondaryWorkerID = tractor.SecondaryWorkerID

	_, err = tx.NewUpdate().Model(m).Column("primary_worker_id", "secondary_worker_id").WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update movement with workers: %w", err)
	}

	return nil
}
