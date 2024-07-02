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

type TractorPermission string

const (
	// PermissionTractorView is the permission to view tractor details
	PermissionTractorView = TractorPermission("tractor.view")

	// PermissionTractorEdit is the permission to edit tractor details
	PermissionTractorEdit = TractorPermission("tractor.edit")

	// PermissionTractorAdd is the permission to add a new tractor
	PermissionTractorAdd = TractorPermission("tractor.add")

	// PermissionTractorDelete is the permission to delete a tractor
	PermissionTractorDelete = TractorPermission("tractor.delete")
)

// String returns the string representation of the TractorPermission
func (p TractorPermission) String() string {
	return string(p)
}

type Tractor struct {
	bun.BaseModel           `bun:"table:tractors,alias:tr" json:"-"`
	CreatedAt               time.Time    `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt               time.Time    `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID                      uuid.UUID    `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Code                    string       `bun:"type:VARCHAR(50),notnull" json:"code" queryField:"true"`
	Status                  string       `bun:"type:equipment_status_enum,notnull" json:"status"`
	Model                   string       `bun:"type:VARCHAR(50)" json:"model"`
	Year                    *int         `bun:"type:INTEGER,nullzero" json:"year"`
	LicensePlateNumber      string       `bun:"type:VARCHAR(50)" json:"licensePlateNumber"`
	Vin                     string       `bun:"type:VARCHAR(17)" json:"vin"`
	IsLeased                bool         `bun:"type:boolean" json:"isLeased"`
	LeasedDate              *pgtype.Date `bun:"type:date,nullzero" json:"leasedDate"`
	EquipmentTypeID         uuid.UUID    `bun:"type:uuid,notnull" json:"equipmentTypeId"`
	EquipmentManufacturerID *uuid.UUID   `bun:"type:uuid,nullzero" json:"equipmentManufacturerId"`
	StateID                 *uuid.UUID   `bun:"type:uuid,nullzero" json:"stateId"`
	FleetCodeID             *uuid.UUID   `bun:"type:uuid,nullzero" json:"fleetCodeId"`
	BusinessUnitID          uuid.UUID    `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID          uuid.UUID    `bun:"type:uuid,notnull" json:"organizationId"`
	PrimaryWorkerID         uuid.UUID    `bun:"type:uuid" json:"primaryWorkerId"`
	SecondaryWorkerID       *uuid.UUID   `bun:"type:uuid" json:"secondaryWorkerId"`

	PrimaryWorker         *Worker                `bun:"rel:has-one,join:primary_worker_id=id" json:"primaryWorker"`
	SecondaryWorker       *Worker                `bun:"rel:belongs-to,join:secondary_worker_id=id" json:"-"`
	EquipmentType         *EquipmentType         `bun:"rel:belongs-to,join:equipment_type_id=id" json:"equipmentType"`
	EquipmentManufacturer *EquipmentManufacturer `bun:"rel:belongs-to,join:equipment_manufacturer_id=id" json:"-"`
	State                 *UsState               `bun:"rel:belongs-to,join:state_id=id" json:"-"`
	FleetCode             *FleetCode             `bun:"rel:belongs-to,join:fleet_code_id=id" json:"-"`
	BusinessUnit          *BusinessUnit          `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization          *Organization          `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c Tractor) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(1, 50)),
		validation.Field(&c.BusinessUnitID, validation.Required.Error("Business Unit is required. Please try again."), is.UUIDv4),
		validation.Field(&c.EquipmentTypeID, validation.Required.Error("Equipment Type is required. Please try again."), is.UUIDv4),
		validation.Field(&c.EquipmentManufacturerID, validation.Required.Error("Equipment Manufacturer is required. Please try again."), is.UUIDv4),
		validation.Field(&c.LeasedDate, validation.When(c.IsLeased, validation.NotNil)),
		validation.Field(&c.OrganizationID, validation.Required.Error("OrganizationID is required. Please try again."), is.UUIDv4),
		validation.Field(&c.SecondaryWorkerID, validation.By(validateWorkers(c.PrimaryWorkerID, c.SecondaryWorkerID))),
	)
}

func (c Tractor) DBValidate(ctx context.Context, db *bun.DB) error {
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

func (c Tractor) validateEquipmentClass(ctx context.Context, db *bun.DB) error {
	if c.EquipmentTypeID != uuid.Nil {
		et := new(EquipmentType)

		err := db.NewSelect().Model(et).Where("id = ?", c.EquipmentTypeID).Scan(ctx)
		if err != nil {
			return &validator.BusinessLogicError{Message: "Failed to fetch equipment type."}
		}

		if et.EquipmentClass != "Tractor" {
			return validator.DBValidationError{
				Field:   "equipmentTypeId",
				Message: "Equipment type must have a class of Tractor.",
			}
		}
	}

	return nil
}

func validateWorkers(pWorkerID uuid.UUID, sWorkerID *uuid.UUID) validation.RuleFunc {
	return func(_ any) error {
		if sWorkerID == nil {
			return nil
		}

		if pWorkerID == *sWorkerID {
			return errors.New("Secondary Worker cannot be the same as Primary Worker. Please try again.")
		}

		return nil
	}
}

var _ bun.BeforeAppendModelHook = (*Tractor)(nil)

func (c *Tractor) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
