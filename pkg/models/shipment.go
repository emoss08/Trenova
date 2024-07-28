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
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/constants"

	"github.com/rs/zerolog/log"

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
	IsHazardous              bool                          `bun:",notnull,default:false" json:"isHazardous"`
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

// DBValidate validates the Shipment struct against the database.
func (s *Shipment) DBValidate(ctx context.Context, tx bun.IDB) error {
	var multiErr validator.MultiValidationError

	if err := s.Validate(); err != nil {
		return err
	}

	shipControl, err := QueryShipmentControlByOrgID(ctx, tx, s.OrganizationID)
	if err != nil {
		log.Error().Err(err).Str("orgID", s.OrganizationID.String()).Msg("Failed to fetch shipment controls")
		return err
	}

	// Validate the revenue if EnforceRevenue is enabled in Shipment Controls
	s.validateRevenueCode(shipControl, multiErr)

	// Validate the voided comment if EnforceVoidedComm is enabled in Shipment Controls
	s.validateEnforceVoidedComm(shipControl, multiErr)

	// Validate the origin and destination locations are not the same if enabled in Shipment Controls
	s.CompareOriginDestination(shipControl, multiErr)

	// Validate the delivery date is after the ship date
	s.validateDeliveryDate(s.ShipDate, s.EstimatedDeliveryDate, multiErr)

	// Check for duplicate BOLs
	if err = s.checkForDuplicateBOLs(ctx, tx, s.OrganizationID, multiErr); err != nil {
		return err
	}

	// Validate the status transition
	if err = s.validateStatusTransition(ctx, tx, multiErr); err != nil {
		return err
	}

	if len(multiErr.Errors) > 0 {
		return multiErr
	}

	return nil
}

// UpdateStatus updates the shipment status based on its movements.
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
func (s *Shipment) GetMoveByID(ctx context.Context, db *bun.DB, moveID uuid.UUID) (*ShipmentMove, error) {
	var move ShipmentMove
	err := db.NewSelect().Model(&move).Where("id = ?", moveID).Scan(ctx)
	if err != nil {
		log.Error().Err(err).Str("moveID", moveID.String()).Msg("Failed to fetch shipment move")
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

func (s *Shipment) UpdateOne(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	//original := new(Customer)
	//if err := tx.NewSelect().Model(original).Where("id = ?", c.ID).Scan(ctx); err != nil {
	//	return err
	//}
	//
	//if err := c.OptimisticUpdate(ctx, tx); err != nil {
	//	return err
	//}
	//
	//auditService.LogAction(
	//	constants.TableCustomer,
	//	c.ID.String(),
	//	property.AuditLogActionUpdate,
	//	user,
	//	c.OrganizationID,
	//	c.BusinessUnitID,
	//	audit.WithDiff(original, c),
	//)
	//
	//return nil
	return errors.New("not implemented")
}

// checkForDuplicateBOLs checks for duplicate BOLs in the database by organization ID.
func (s *Shipment) checkForDuplicateBOLs(ctx context.Context, tx bun.IDB, orgID uuid.UUID, multiErr validator.MultiValidationError) error {
	var duplicateBOLs []*string
	err := tx.NewSelect().
		Model((*Shipment)(nil)).
		ColumnExpr("DISTINCT bill_of_lading").
		Where("organization_id = ?", orgID).
		Where("bill_of_lading = ?", s.BillOfLading).
		Scan(ctx, &duplicateBOLs)
	if err != nil {
		log.Error().Err(err).Str("orgID", orgID.String()).Msg("Failed to fetch shipments with duplicate BOLs")
		return &validator.BusinessLogicError{Message: "Failed to fetch shipments with duplicate BOLs"}
	}

	if len(duplicateBOLs) > 0 {
		multiErr.Errors = append(multiErr.Errors, validator.DBValidationError{
			Field:   "billOfLading",
			Message: fmt.Sprintf("Bill of lading %s already exists in the system. Please try again.", s.BillOfLading),
		})
	}

	return nil
}

// validateRevenueCode validates that the revenue code is set if EnforceRevCode is enabled in Shipment Controls.
func (s *Shipment) validateRevenueCode(sc *ShipmentControl, multiErr validator.MultiValidationError) {
	if sc.EnforceRevCode && s.RevenueCodeID == nil {
		multiErr.Errors = append(multiErr.Errors, validator.DBValidationError{
			Field:   "revenueCodeId",
			Message: "Organization enforces a revenue code when creating shipments. Please try again.",
		})
	}
}

// validateEnforceVoidedComm validates that the voided comment is set if EnforceVoidedComm is enabled in Shipment Controls.
func (s *Shipment) validateEnforceVoidedComm(sc *ShipmentControl, multiErr validator.MultiValidationError) {
	if sc.EnforceVoidedComm && s.Status == property.ShipmentStatusVoided && s.VoidedComment == "" {
		multiErr.Errors = append(multiErr.Errors, validator.DBValidationError{
			Field:   "voidedComment",
			Message: "Organization requires a voided comment. Please try again.",
		})
	}
}

// CompareOriginDestination compares the origin and destination locations and adds an error if they are the same.
func (s *Shipment) CompareOriginDestination(sc *ShipmentControl, multiErr validator.MultiValidationError) {
	if sc.CompareOriginDestination && s.OriginLocationID == s.DestinationLocationID {
		multiErr.Errors = append(multiErr.Errors, validator.DBValidationError{
			Field:   "destinationLocationId",
			Message: "Organization does not allow the origin and destination locations to be the same. Please try again.",
		})
	}
}

// validateStatusTransition validates that the status transition is valid.
func (s *Shipment) validateStatusTransition(ctx context.Context, tx bun.IDB, multiErr validator.MultiValidationError) error {
	if s.ID == uuid.Nil {
		// This is a new shipment, so we only need to validate that the status is New
		if s.Status != property.ShipmentStatusNew {
			multiErr.Errors = append(multiErr.Errors, validator.DBValidationError{
				Field:   "status",
				Message: "New shipments must have a status of New",
			})
		}

		return nil
	}

	var currentStatus property.ShipmentStatus

	err := tx.NewSelect().
		Model((*Shipment)(nil)).
		Where("id = ?", s.ID).
		Scan(ctx, &currentStatus)
	if err != nil {
		log.Error().Err(err).Str("shipmentID", s.ID.String()).Msg("Failed to fetch current shipment status")
		return &validator.BusinessLogicError{Message: "Failed to fetch current shipment status"}
	}

	// Check if the new status is different from the current status.
	if s.Status == currentStatus {
		return nil
	}

	// Check if the transition is valid
	validNextStatuses, exists := validShipmentStatusTransitions[currentStatus]
	if !exists {
		multiErr.Errors = append(multiErr.Errors, validator.DBValidationError{
			Field:   "status",
			Message: fmt.Sprintf("Invalid status transition from %s to %s", currentStatus, s.Status),
		})
	}

	for _, validStatus := range validNextStatuses {
		if s.Status == validStatus {
			return nil
		}
	}

	multiErr.Errors = append(multiErr.Errors, validator.DBValidationError{
		Field:   "status",
		Message: fmt.Sprintf("Invalid status transition from %s to %s", currentStatus, s.Status),
	})

	return nil
}

// validateDeliveryDate validates that the estimated delivery date is after the ship date.
func (s *Shipment) validateDeliveryDate(shipDate, delDate *pgtype.Date, multiErr validator.MultiValidationError) {
	if shipDate != nil && delDate != nil {
		if delDate.Time.Before(shipDate.Time) {
			multiErr.Errors = append(multiErr.Errors, validator.DBValidationError{
				Field:   "estimatedDeliveryDate",
				Message: "Estimated delivery date must be after the ship date",
			})
		}
	}
}

// Insert inserts a new shipment into the database.
func (s *Shipment) Insert(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	if s.ProNumber == "" {
		proNumber, err := GenerateProNumber(ctx, tx, s.OrganizationID)
		if err != nil {
			return validator.BusinessLogicError{Message: err.Error()}
		}

		s.ProNumber = proNumber
	}

	shipControl, err := QueryShipmentControlByOrgID(ctx, tx, s.OrganizationID)
	if err != nil {
		return err
	}

	// Calculate the total charge amount if AutoTotalShipment is enabled in Shipment Controls
	if shipControl.AutoTotalShipment {
		s.CalculateTotalChargeAmount()
	}

	if err = s.DBValidate(ctx, tx); err != nil {
		return err
	}

	if _, err = tx.NewInsert().Model(s).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to insert shipment")
		return err
	}

	auditService.LogAction(
		constants.TableShipment,
		s.ID.String(),
		property.AuditLogActionCreate,
		user,
		s.OrganizationID,
		s.BusinessUnitID,
		audit.WithDiff(nil, s),
	)

	return err
}

func (s *Shipment) BeforeUpdate(_ context.Context) error {
	s.Version++

	return nil
}

// OptimisticUpdate updates the Shipment in the database with optimistic locking.
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
