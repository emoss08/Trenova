package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Define a map of valid status transitions
var validShipmentStatusTransitions = map[property.ShipmentStatus][]property.ShipmentStatus{
	property.ShipmentStatusNew: {
		property.ShipmentStatusInProgress,
		property.ShipmentStatusHold,
		property.ShipmentStatusVoided,
	},
	property.ShipmentStatusInProgress: {
		property.ShipmentStatusCompleted,
		property.ShipmentStatusHold,
		property.ShipmentStatusVoided,
	},
	property.ShipmentStatusCompleted: {
		property.ShipmentStatusBilled,
		property.ShipmentStatusVoided,
	},
	property.ShipmentStatusBilled: {
		property.ShipmentStatusVoided,
	},
	property.ShipmentStatusHold: {
		property.ShipmentStatusInProgress,
		property.ShipmentStatusVoided,
	},
	property.ShipmentStatusVoided: {}, // No valid transitions from Voided
}

type Shipment struct {
	bun.BaseModel `bun:"table:shipments,alias:sp" json:"-"`

	ID                       uuid.UUID                     `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ProNumber                string                        `bun:"type:VARCHAR(12),notnull" json:"proNumber" queryField:"true"`
	Status                   property.ShipmentStatus       `bun:"type:shipment_status_enum,notnull,default:'New'" json:"status"`
	RatingUnit               int                           `bun:"type:integer,notnull" json:"ratingUnit"`
	RatingMethod             property.ShipmentRatingMethod `bun:"type:rating_method_enum,notnull,default:'FlatRate'" json:"ratingMethod"`
	OtherChargeAmount        decimal.Decimal               `bun:"type:NUMERIC(19,4),notnull,default:0" json:"otherChargeAmount"`
	FreightChargeAmount      decimal.Decimal               `bun:"type:NUMERIC(19,4),notnull,default:0" json:"freightChargeAmount"`
	TotalChargeAmount        decimal.Decimal               `bun:"type:NUMERIC(19,4),notnull,default:0" json:"totalChargeAmount"`
	Pieces                   decimal.NullDecimal           `bun:"type:NUMERIC(10,2),nullzero" json:"pieces"`
	Weight                   decimal.NullDecimal           `bun:"type:NUMERIC(10,2),nullzero" json:"weight"`
	ReadyToBill              bool                          `bun:",notnull,default:false" json:"readyToBill"`
	BillDate                 *pgtype.Date                  `bun:"type:date,nullzero" json:"billDate"`
	ShipDate                 *pgtype.Date                  `bun:"type:date,nullzero" json:"shipDate"`
	Billed                   bool                          `bun:",notnull,default:false" json:"billed"`
	TransferredToBilling     bool                          `bun:",notnull,default:false" json:"transferredToBilling"`
	TransferredToBillingDate *pgtype.Date                  `bun:"type:date,nullzero" json:"transferredToBillingDate"`
	TemperatureMin           int                           `bun:"type:integer" json:"temperatureMin"`
	TemperatureMax           int                           `bun:"type:integer" json:"temperatureMax"`
	BillOfLading             string                        `bun:"type:VARCHAR(20)" json:"billOfLading"`
	VoidedComment            string                        `bun:"type:TEXT" json:"voidedComment"`
	AutoRated                bool                          `bun:",notnull,default:false" json:"autoRated"`
	EntryMethod              string                        `bun:"type:VARCHAR(20)" json:"entryMethod"`
	IsHazardous              bool                          `bun:",notnull,default:false" json:"isHarzardous"`
	EstimatedDeliveryDate    *pgtype.Date                  `bun:"type:date,nullzero" json:"estimatedDeliveryDate"`
	ActualDeliveryDate       *pgtype.Date                  `bun:"type:date,nullzero" json:"actualDeliveryDate"`
	Priority                 int                           `bun:"type:integer,notnull,default:0" json:"priority"`
	SpecialInstructions      string                        `bun:"type:TEXT,nullzero" json:"specialInstructions"`
	TrackingNumber           string                        `bun:"type:VARCHAR(50)" json:"trackingNumber"`
	TotalDistance            decimal.NullDecimal           `bun:"type:NUMERIC(10,2),nullzero" json:"totalDistance"`
	Version                  int64                         `bun:"type:BIGINT" json:"version"`
	CreatedAt                time.Time                     `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt                time.Time                     `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	CreatedByID           *uuid.UUID `bun:"type:uuid,nullzero" json:"createdById"`
	BusinessUnitID        uuid.UUID  `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID        uuid.UUID  `bun:"type:uuid,notnull" json:"organizationId"`
	ShipmentTypeID        uuid.UUID  `bun:"type:uuid,notnull" json:"shipmentTypeId"`
	RevenueCodeID         *uuid.UUID `bun:"type:uuid,nullzero" json:"revenueCodeId"`
	ServiceTypeID         *uuid.UUID `bun:"type:uuid,nullzero" json:"serviceTypeId"`
	OriginLocationID      uuid.UUID  `bun:"type:uuid" json:"originLocationId"`
	DestinationLocationID uuid.UUID  `bun:"type:uuid" json:"destinationLocationId"`
	CustomerID            uuid.UUID  `bun:"type:uuid" json:"customerId"`
	TrailerTypeID         *uuid.UUID `bun:"type:uuid,nullzero" json:"trailerTypeId"`
	TractorTypeID         *uuid.UUID `bun:"type:uuid,nullzero" json:"tractorTypeId"`

	CreatedBy           *User           `bun:"rel:belongs-to,join:created_by_id=id" json:"-"`
	BusinessUnit        *BusinessUnit   `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization        *Organization   `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	ShipmentType        *ShipmentType   `bun:"rel:belongs-to,join:shipment_type_id=id" json:"-"`
	ServiceType         *ServiceType    `bun:"rel:belongs-to,join:service_type_id=id" json:"-"`
	TractorType         *EquipmentType  `bun:"rel:belongs-to,join:tractor_type_id=id" json:"-"`
	TrailerType         *EquipmentType  `bun:"rel:belongs-to,join:trailer_type_id=id" json:"-"`
	RevenueCode         *RevenueCode    `bun:"rel:belongs-to,join:revenue_code_id=id" json:"-"`
	OriginLocation      *Location       `bun:"rel:belongs-to,join:origin_location_id=id" json:"-"`
	DestinationLocation *Location       `bun:"rel:belongs-to,join:destination_location_id=id" json:"-"`
	Customer            *Customer       `bun:"rel:belongs-to,join:customer_id=id" json:"-"`
	ShipmentMoves       []*ShipmentMove `bun:"rel:has-many,join:id=shipment_id" json:"moves"`
}

// Validate validates the Shipment struct.
func (s Shipment) Validate() error {
	return validation.ValidateStruct(
		&s,
		validation.Field(&s.Status, validation.Required.Error("Status is required")),
		validation.Field(&s.RatingMethod, validation.Required.Error("Rating method is required")),
		validation.Field(&s.TemperatureMin, validation.Max(s.TemperatureMax)),
		validation.Field(&s.BillOfLading, validation.Required.Error("Bill of lading is required")),
		validation.Field(&s.ShipmentTypeID, validation.Required.Error("Shipment Type is required")),
		validation.Field(&s.RatingUnit, validation.Required.Error("Rating unit is required")),
		validation.Field(&s.BusinessUnitID, validation.Required),
		validation.Field(&s.OrganizationID, validation.Required),
		validation.Field(&s.OtherChargeAmount, validation.Required.Error("Other charge amount is required")),
		validation.Field(&s.FreightChargeAmount, validation.When(s.RatingMethod == property.ShipmentRatingMethodFlatRate,
			validation.Required.Error("Freight charge amount is required when rating method is flat rate"),
		)),
		validation.Field(&s.TotalDistance, validation.When(s.RatingMethod == property.ShipmentRatingMethodPerMile,
			validation.Required.Error("Total distance is required when rating method is per mile"),
		)),
		validation.Field(&s.ShipmentMoves))
}

func (s *Shipment) BeforeUpdate(_ context.Context) error {
	s.Version++

	return nil
}

func (s *Shipment) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := s.Version

	if err := s.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().Model(s).WherePK().Where("version = ?", ov).Exec(ctx)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return &validator.BusinessLogicError{
			Message: fmt.Sprintf("Version mismatch. The Shipment (ID: %s) has been updated by another user. Please refresh and try again.", s.ID),
		}
	}

	return nil
}

// UpdateShipmentstatus updates the shipment status based on its movements.
func (s *Shipment) UpdateStatus(ctx context.Context, db *bun.DB) error {
	// Fetch all movements for this shipment
	var movements []*ShipmentMove
	err := db.NewSelect().Model(&movements).Where("shipment_id = ?", s.ID).Scan(ctx)
	if err != nil {
		return validator.BusinessLogicError{Message: err.Error()}
	}

	allCompleted := true
	anyInProgress := false

	for _, move := range movements {
		switch move.Status {
		case property.ShipmentMoveStatusCompleted:
			continue
		case property.ShipmentMoveStatusInProgress:
			anyInProgress = true
			allCompleted = false
		case property.ShipmentMoveStatusVoided:
			allCompleted = false
		case property.ShipmentMoveStatusNew:
			allCompleted = false
		default:
			allCompleted = false
		}
	}

	var newStatus property.ShipmentStatus

	switch {
	case allCompleted:
		newStatus = property.ShipmentStatusCompleted
	case anyInProgress:
		newStatus = property.ShipmentStatusInProgress
	default:
		newStatus = property.ShipmentStatusNew
	}

	return s.setStatus(ctx, db, newStatus)
}

// GetMoveByID fetches a shipment move by its ID.
func (s Shipment) GetMoveByID(ctx context.Context, db *bun.DB, moveID uuid.UUID) (*ShipmentMove, error) {
	var move ShipmentMove
	err := db.NewSelect().Model(&move).Where("id = ?", moveID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &move, nil
}

// Helper method to set status and handle database updates.
func (s *Shipment) setStatus(ctx context.Context, db *bun.DB, newStatus property.ShipmentStatus) error {
	s.Status = newStatus
	_, err := db.NewUpdate().Model(s).Column("status").WherePK().Exec(ctx)
	return err
}

// CalculateTotalChargeAmount updates the TotalChargeAmount based on FreightChargeAmount and OtherChargeAmount.
func (s *Shipment) CalculateTotalChargeAmount() {
	s.TotalChargeAmount = s.FreightChargeAmount.Add(s.OtherChargeAmount)
}

// MarkReadyToBill marks the Shipment as ready to bill
func (s *Shipment) MarkReadyToBill() error {
	if s.Status != property.ShipmentStatusCompleted {
		return &validator.DBValidationError{
			Field:   "markReadyToBill",
			Message: "Shipment must be completed before it can be marked as ready to bill",
		}
	}

	s.ReadyToBill = true
	return nil
}

func (s Shipment) DBValidate(ctx context.Context, db *bun.DB) error {
	var multiErr validator.MultiValidationError
	var dbValidationErr *validator.DBValidationError

	if err := s.Validate(); err != nil {
		return err
	}

	if err := s.validateStatusTransition(ctx, db); err != nil {
		if errors.As(err, &dbValidationErr) {
			multiErr.Errors = append(multiErr.Errors, *dbValidationErr)
		} else {
			return err
		}
	}

	// if err := validateTotalChargeAmount(s.FreightChargeAmount, s.OtherChargeAmount, s.TotalChargeAmount); err != nil {
	// 	if errors.As(err, &dbValidationErr) {
	// 		multiErr.Errors = append(multiErr.Errors, *dbValidationErr)
	// 	} else {
	// 		return err
	// 	}
	// }

	if err := validateDeliveryDate(s.ShipDate, s.EstimatedDeliveryDate); err != nil {
		if errors.As(err, &dbValidationErr) {
			multiErr.Errors = append(multiErr.Errors, *dbValidationErr)
		} else {
			return err
		}
	}

	if len(multiErr.Errors) > 0 {
		return multiErr
	}

	return nil
}

// validateDeliveryDate validates that the estimated delivery date is after the ship date.
func validateDeliveryDate(shipDate, delDate *pgtype.Date) error {
	if shipDate != nil && delDate != nil {
		if delDate.Time.Before(shipDate.Time) {
			return errors.New("estimated delivery date must be after ship date")
		}
	}
	return nil
}

// validateTotalChargeAmount validates that the TotalChargeAmount is the sum of FreightChargeAmount and OtherChargeAmount.
func validateTotalChargeAmount(freightChargeAmount, otherChargeAmount, totalChargeAmount decimal.Decimal) error {
	expectedTotal := freightChargeAmount.Add(otherChargeAmount)
	if totalChargeAmount != expectedTotal {
		return &validator.DBValidationError{
			Field:   "totalChargeAmount",
			Message: fmt.Sprintf("Total charge amount must be the sum of freight charge amount and other charge amount. Expected %d, got %d", expectedTotal, totalChargeAmount),
		}
	}

	return nil
}

// validateStatusTransition validates that the status transition is valid.
func (s Shipment) validateStatusTransition(ctx context.Context, db *bun.DB) error {
	if s.ID == uuid.Nil {
		// This is a new shipment, so we only need to validae that the status is New
		if s.Status != property.ShipmentStatusNew {
			return &validator.DBValidationError{
				Field:   "status",
				Message: "New shipments must have a status of New",
			}
		}

		return nil
	}

	var currentStatus property.ShipmentStatus

	err := db.NewSelect().
		Model((*Shipment)(nil)).
		Where("id = ?", s.ID).
		Scan(ctx, &currentStatus)
	if err != nil {
		return &validator.BusinessLogicError{Message: "Failed to fetch current shipment status"}
	}

	// Check if the new status is different from the cureent status.
	if s.Status == currentStatus {
		return nil
	}

	// Check if the transition is valid
	validNextStatuses, exists := validShipmentStatusTransitions[currentStatus]
	if !exists {
		return &validator.DBValidationError{
			Field:   "status",
			Message: "Invalid status transition",
		}
	}

	for _, validStatus := range validNextStatuses {
		if s.Status == validStatus {
			return nil
		}
	}

	return &validator.DBValidationError{
		Field:   "status",
		Message: fmt.Sprintf("Invalid status transition from %s to %s", currentStatus, s.Status),
	}
}

// InsertShipment inserts a new shipment into the database.
func (s *Shipment) InsertShipment(ctx context.Context, db *bun.DB) error {
	if s.ProNumber == "" {
		proNumber, err := GenerateProNumber(ctx, db, s.OrganizationID)
		if err != nil {
			return validator.BusinessLogicError{Message: err.Error()}
		}

		s.ProNumber = proNumber
	}

	s.CalculateTotalChargeAmount()

	if err := s.DBValidate(ctx, db); err != nil {
		return err
	}

	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(s).Exec(ctx); err != nil {
			return err
		}

		return nil
	})

	return err
}

var _ bun.BeforeAppendModelHook = (*Shipment)(nil)

func (s *Shipment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		s.UpdatedAt = time.Now()
	}
	return nil
}
