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

type ShipmentPermission string

const (
	// PermissionShipmentView is the permission to view shipment details
	PermissionShipmentView = ShipmentPermission("shipment.view")

	// PermissionShipmentEdit is the permission to edit shipment details
	PermissionShipmentEdit = ShipmentPermission("shipment.edit")

	// PermissionShipmentAdd is the permission to add a new shipment
	PermissionShipmentAdd = ShipmentPermission("shipment.add")

	// PermissionShipmentDelete is the permission to delete a shipment
	PermissionShipmentDelete = ShipmentPermission("shipment.delete")
)

// String returns the string representation of the ShipmentPermission
func (p ShipmentPermission) String() string {
	return string(p)
}

type Shipment struct {
	bun.BaseModel            `bun:"table:shipments,alias:sp" json:"-"`
	CreatedAt                time.Time                     `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt                time.Time                     `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	BusinessUnitID           uuid.UUID                     `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID           uuid.UUID                     `bun:"type:uuid,notnull" json:"organizationId"`
	ID                       uuid.UUID                     `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ProNumber                string                        `bun:"type:VARCHAR(12),notnull" json:"proNumber" queryField:"true"`
	Status                   property.ShipmentStatus       `bun:"type:shipment_status_enum,notnull,default:'New'" json:"status"`
	ShipmentTypeID           uuid.UUID                     `bun:"type:uuid,notnull" json:"shipmentTypeId"`
	RevenueCodeID            *uuid.UUID                    `bun:"type:uuid,nullzero" json:"revenueCodeId"`
	ServiceTypeID            *uuid.UUID                    `bun:"type:uuid,nullzero" json:"serviceTypeId"`
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
	TrailerTypeID            *uuid.UUID                    `bun:"type:uuid,nullzero" json:"trailerTypeId"`
	TractorTypeID            *uuid.UUID                    `bun:"type:uuid,nullzero" json:"tractorTypeId"`
	TemperatureMin           int                           `bun:"type:integer" json:"temperatureMin"`
	TemperatureMax           int                           `bun:"type:integer" json:"temperatureMax"`
	BillOfLading             string                        `bun:"type:VARCHAR(20)" json:"billOfLading"`
	VoidedComment            string                        `bun:"type:TEXT" json:"voidedComment"`
	AutoRated                bool                          `bun:",notnull,default:false" json:"autoRated"`
	EntryMethod              string                        `bun:"type:VARCHAR(20)" json:"entryMethod"`
	CreatedByID              *uuid.UUID                    `bun:"type:uuid,nullzero" json:"createdById"`
	IsHazardous              bool                          `bun:",notnull,default:false" json:"isHarzardous"`
	EstimatedDeliveryDate    *pgtype.Date                  `bun:"type:date,nullzero" json:"estimatedDeliveryDate"`
	ActualDeliveryDate       *pgtype.Date                  `bun:"type:date,nullzero" json:"actualDeliveryDate"`
	OriginLocationID         uuid.UUID                     `bun:"type:uuid" json:"originLocationId"`
	DestinationLocationID    uuid.UUID                     `bun:"type:uuid" json:"destinationLocationId"`
	CustomerID               uuid.UUID                     `bun:"type:uuid" json:"customerId"`
	Priority                 int                           `bun:"type:integer,notnull,default:0" json:"priority"`
	SpecialInstructions      string                        `bun:"type:TEXT,nullzero" json:"specialInstructions"`
	TrackingNumber           string                        `bun:"type:VARCHAR(50)" json:"trackingNumber"`
	TotalDistance            decimal.NullDecimal           `bun:"type:NUMERIC(10,2),nullzero" json:"totalDistance"`

	CreatedBy           *User                `bun:"rel:belongs-to,join:created_by_id=id" json:"-"`
	BusinessUnit        *BusinessUnit        `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization        *Organization        `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	ShipmentType        *ShipmentType        `bun:"rel:belongs-to,join:shipment_type_id=id" json:"-"`
	ServiceType         *ServiceType         `bun:"rel:belongs-to,join:service_type_id=id" json:"-"`
	TractorType         *EquipmentType       `bun:"rel:belongs-to,join:tractor_type_id=id" json:"-"`
	TrailerType         *EquipmentType       `bun:"rel:belongs-to,join:trailer_type_id=id" json:"-"`
	RevenueCode         *RevenueCode         `bun:"rel:belongs-to,join:revenue_code_id=id" json:"-"`
	OriginLocation      *Location            `bun:"rel:belongs-to,join:origin_location_id=id" json:"-"`
	DestinationLocation *Location            `bun:"rel:belongs-to,join:destination_location_id=id" json:"-"`
	Customer            *Customer            `bun:"rel:belongs-to,join:customer_id=id" json:"-"`
	AccessorialCharges  []*AccessorialCharge `bun:"rel:has-many,join:id=shipment_id" json:"-"`
	ShipmentMoves       []*ShipmentMove      `bun:"rel:has-many,join:id=shipment_id" json:"moves"`
}

func (s Shipment) Validate() error {
	return validation.ValidateStruct(
		&s,
		validation.Field(&s.Status, validation.Required.Error("Status is required")),
		validation.Field(&s.RatingMethod, validation.Required.Error("Rating method is required")),
		validation.Field(&s.TemperatureMin, validation.Max(s.TemperatureMax)),
		validation.Field(&s.EstimatedDeliveryDate, validation.By(s.validateDeliveryDate)),
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

// Helper method to set status and handle database updates
func (s *Shipment) setStatus(ctx context.Context, db *bun.DB, newStatus property.ShipmentStatus) error {
	s.Status = newStatus
	_, err := db.NewUpdate().Model(s).Column("status").WherePK().Exec(ctx)
	return err
}

// CalculateTotalChargeAmount updates the TotalChargeAmount based on FreightChargeAmount and OtherChargeAmount
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

	// if err := s.validateTotalChargeAmount(); err != nil {
	// 	if errors.As(err, &dbValidationErr) {
	// 		multiErr.Errors = append(multiErr.Errors, *dbValidationErr)
	// 	} else {
	// 		return err
	// 	}
	// }

	if err := s.validateDeliveryDate(s.EstimatedDeliveryDate); err != nil {
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

func (s Shipment) validateDeliveryDate(value any) error {
	estimatedDelivery, ok := value.(*pgtype.Date)
	if !ok {
		return fmt.Errorf("expected *pgtype.Date, got %T", value)
	}

	if s.ShipDate != nil && estimatedDelivery != nil {
		if estimatedDelivery.Time.Before(s.ShipDate.Time) {
			return errors.New("estimated delivery date must be after ship date")
		}
	}
	return nil
}

func (s Shipment) validateTotalChargeAmount() error {
	expectedTotal := s.FreightChargeAmount.Add(s.OtherChargeAmount)
	if s.TotalChargeAmount != expectedTotal {
		return &validator.DBValidationError{
			Field:   "totalChargeAmount",
			Message: fmt.Sprintf("Total charge amount must be the sum of freight charge amount and other charge amount. Expected %d, got %d", expectedTotal, s.TotalChargeAmount),
		}
	}

	return nil
}

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

func (s *Shipment) AssignTractorToMovement(ctx context.Context, db *bun.DB, tractorID uuid.UUID) error {
	if s.Status != property.ShipmentStatusNew {
		return &validator.BusinessLogicError{
			Message: "Tractor can only be assigned to a shipment that is in `New` status",
		}
	}

	// Start a transaction
	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Fetch all movements for this shipment
		var movements []*ShipmentMove
		err := tx.NewSelect().Model(&movements).Where("shipment_id = ?", s.ID).Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch shipment movements: %w", err)
		}

		if len(movements) == 0 {
			return &validator.DBValidationError{
				Field:   "movements",
				Message: "No movements found for this shipment",
			}
		}

		// Fetch the tractor to ensure it exists and is available
		tractor := new(Tractor)
		err = tx.NewSelect().Model(tractor).Where("id = ?", tractorID).Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch tractor: %w", err)
		}

		if tractor.Status != "Available" {
			return &validator.DBValidationError{
				Field:   "tractorId",
				Message: "Selected tractor is not available",
			}
		}

		for _, move := range movements {
			move.TractorID = tractorID
			_, err = tx.NewUpdate().Model(move).Column("tractor_id").WherePK().Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update movement with tractor: %w", err)
			}

			// Only assign workers if the movement is not in progress
			if move.Status == property.ShipmentMoveStatusNew {
				if err = move.AssignWorkersByTractorID(ctx, tx, tractorID); err != nil {
					return fmt.Errorf("failed to assign workers to movement: %w", err)
				}
			}
		}

		return nil
	})

	return err
}

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
