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
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/jackc/pgx/v5/pgtype"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Trailer struct {
	bun.BaseModel `bun:"table:trailers,alias:tr" json:"-"`

	ID                         uuid.UUID    `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Code                       string       `bun:"type:VARCHAR(50),notnull" json:"code" queryField:"true"`
	Status                     string       `bun:"type:equipment_status_enum,notnull" json:"status"`
	Model                      string       `bun:"type:VARCHAR(50)" json:"model"`
	Year                       int          `bun:"type:INTEGER,nullzero" json:"year"`
	LicensePlateNumber         string       `bun:"type:VARCHAR(50)" json:"licensePlateNumber"`
	Vin                        string       `bun:"type:VARCHAR(17)" json:"vin"`
	LastInspectionDate         *pgtype.Date `bun:"type:date,nullzero" json:"lastInspectionDate"`
	RegistrationNumber         string       `bun:"type:VARCHAR(50)" json:"registrationNumber"`
	RegistrationExpirationDate *pgtype.Date `bun:"type:date,nullzero" json:"registrationExpirationDate"`
	Version                    int64        `bun:"type:BIGINT" json:"version"`
	CreatedAt                  time.Time    `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt                  time.Time    `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	EquipmentTypeID         uuid.UUID  `bun:"type:uuid,notnull" json:"equipmentTypeId"`
	EquipmentManufacturerID *uuid.UUID `bun:"type:uuid,nullzero" json:"equipmentManufacturerId"`
	StateID                 *uuid.UUID `bun:"type:uuid,nullzero" json:"stateId"`
	FleetCodeID             *uuid.UUID `bun:"type:uuid,nullzero" json:"fleetCodeId"`
	RegistrationStateID     *uuid.UUID `bun:"type:uuid,nullzero" json:"RegistrationStateId"`
	BusinessUnitID          uuid.UUID  `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID          uuid.UUID  `bun:"type:uuid,notnull" json:"organizationId"`

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
		validation.Field(&c.OrganizationID, validation.Required.Error("OrganizationID is required. Please try again."), is.UUIDv4),
	)
}

func (c *Trailer) Insert(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	if err := c.Validate(); err != nil {
		return err
	}

	if _, err := tx.NewInsert().Model(c).Returning("*").Exec(ctx); err != nil {
		return err
	}

	auditService.LogAction(
		constants.TableTrailer,
		c.ID.String(),
		property.AuditLogActionCreate,
		user,
		c.OrganizationID,
		c.BusinessUnitID,
		audit.WithDiff(nil, c),
	)

	return nil
}

func (c *Trailer) UpdateOne(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	original := new(Trailer)
	if err := tx.NewSelect().Model(original).Where("id = ?", c.ID).Scan(ctx); err != nil {
		return validator.BusinessLogicError{Message: err.Error()}
	}

	if err := c.Validate(); err != nil {
		return err
	}

	if err := c.OptimisticUpdate(ctx, tx); err != nil {
		return err
	}

	auditService.LogAction(
		constants.TableTrailer,
		c.ID.String(),
		property.AuditLogActionUpdate,
		user,
		c.OrganizationID,
		c.BusinessUnitID,
		audit.WithDiff(original, c),
	)

	return nil
}

func (c *Trailer) BeforeUpdate(_ context.Context) error {
	c.Version++

	return nil
}

func (c *Trailer) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := c.Version

	if err := c.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(c).
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
			Message: fmt.Sprintf("Version mismatch. The Trailer (ID: %s) has been updated by another user. Please refresh and try again.", c.ID),
		}
	}

	return nil
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
