package models

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/pkg/validator"
	"github.com/jackc/pgx/v5/pgtype"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TrailerPermission string

const (
	// PermissionTrailerView is the permission to view trailer details
	PermissionTrailerView = TrailerPermission("trailer.view")

	// PermissionTrailerEdit is the permission to edit trailer details
	PermissionTrailerEdit = TrailerPermission("trailer.edit")

	// PermissionTrailerAdd is the permission to add a new trailer
	PermissionTrailerAdd = TrailerPermission("trailer.add")

	// PermissionTrailerDelete is the permission to delete a trailer
	PermissionTrailerDelete = TrailerPermission("trailer.delete")
)

// String returns the string representation of the TrailerPermission
func (p TrailerPermission) String() string {
	return string(p)
}

type Trailer struct {
	bun.BaseModel              `bun:"table:trailers,alias:tr" json:"-"`
	CreatedAt                  time.Time    `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt                  time.Time    `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID                         uuid.UUID    `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Code                       string       `bun:"type:VARCHAR(50),notnull" json:"code" queryField:"true"`
	Status                     string       `bun:"type:equipment_status_enum,notnull" json:"status"`
	Model                      string       `bun:"type:VARCHAR(50)" json:"model"`
	Year                       *int         `bun:"type:INTEGER,nullzero" json:"year"`
	LicensePlateNumber         string       `bun:"type:VARCHAR(50)" json:"licensePlateNumber"`
	Vin                        string       `bun:"type:VARCHAR(17)" json:"vin"`
	LastInspectionDate         *pgtype.Date `bun:"type:date,nullzero" json:"lastInspectionDate"`
	RegistrationNumber         string       `bun:"type:VARCHAR(50)" json:"registrationNumber"`
	RegistrationExpirationDate *pgtype.Date `bun:"type:date,nullzero" json:"registrationExpirationDate"`
	EquipmentTypeID            uuid.UUID    `bun:"type:uuid,notnull" json:"equipmentTypeId"`
	EquipmentManufacturerID    *uuid.UUID   `bun:"type:uuid,nullzero" json:"equipmentManufacturerId"`
	StateID                    *uuid.UUID   `bun:"type:uuid,nullzero" json:"stateId"`
	RegistrationStateID        *uuid.UUID   `bun:"type:uuid,nullzero" json:"RegistrationStateId"`
	FleetCodeID                *uuid.UUID   `bun:"type:uuid,nullzero" json:"fleetCodeId"`
	BusinessUnitID             uuid.UUID    `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID             uuid.UUID    `bun:"type:uuid,notnull" json:"organizationId"`

	EquipmentType         *EquipmentType         `bun:"rel:belongs-to,join:equipment_type_id=id" json:"equipmentType"`
	EquipmentManufacturer *EquipmentManufacturer `bun:"rel:belongs-to,join:equipment_manufacturer_id=id" json:"-"`
	State                 *UsState               `bun:"rel:belongs-to,join:state_id=id" json:"-"`
	RegistrationState     *UsState               `bun:"rel:belongs-to,join:registration_state_id=id" json:"-"`
	FleetCode             *FleetCode             `bun:"rel:belongs-to,join:fleet_code_id=id" json:"-"`
	BusinessUnit          *BusinessUnit          `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization          *Organization          `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c Trailer) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(1, 50)),
		validation.Field(&c.BusinessUnitID, validation.Required.Error("Business Unit is required. Please try again."), is.UUIDv4),
		validation.Field(&c.EquipmentTypeID, validation.Required.Error("Equipment Type is required. Please try again."), is.UUIDv4),
		validation.Field(&c.EquipmentManufacturerID, validation.Required.Error("Equipment Manufacturer is required. Please try again."), is.UUIDv4),
		validation.Field(&c.OrganizationID, validation.Required.Error("OrganizationID is required. Please try again."), is.UUIDv4),
	)
}

func (c Trailer) DBValidate(ctx context.Context, db *bun.DB) error {
	var multiErr validator.MultiValidationError

	if err := c.Validate(); err != nil {
		return err
	}

	if err := c.validateEquipmentClass(ctx, db); err != nil {
		// If the error is a DBValidationError, we can add it to the multiErr
		var dbValidationErr *validator.DBValidationError

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

func (c Trailer) validateEquipmentClass(ctx context.Context, db *bun.DB) error {
	if c.EquipmentTypeID != uuid.Nil {
		et := new(EquipmentType)

		err := db.NewSelect().Model(et).Where("id = ?", c.EquipmentTypeID).Scan(ctx)
		if err != nil {
			return &validator.BusinessLogicError{Message: "Failed to fetch equipment type."}
		}

		if et.EquipmentClass != "Trailer" {
			return &validator.DBValidationError{
				Field:   "equipmentTypeId",
				Message: "Equipment type must have equip. class 'Trailer'. Please try again.",
			}
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*Trailer)(nil)

func (c *Trailer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
