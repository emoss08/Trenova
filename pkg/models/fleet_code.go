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
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"

	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type FleetCode struct {
	bun.BaseModel `bun:"table:fleet_codes,alias:fl" json:"-"`

	ID           uuid.UUID           `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status       property.Status     `bun:"status,type:status" json:"status"`
	Code         string              `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description  string              `bun:"type:VARCHAR(100)" json:"description"`
	RevenueGoal  decimal.NullDecimal `bun:"type:numeric(10,2),nullzero" json:"revenueGoal"`
	DeadheadGoal decimal.NullDecimal `bun:"type:numeric(10,2),nullzero" json:"deadheadGoal"`
	MileageGoal  decimal.NullDecimal `bun:"type:numeric(10,2),nullzero" json:"mileageGoal"`
	Color        string              `bun:"type:VARCHAR(10)" json:"color"`
	Version      int64               `bun:"type:BIGINT" json:"version"`
	CreatedAt    time.Time           `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt    time.Time           `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	ManagerID      *uuid.UUID `bun:"type:uuid" json:"managerId"`
	BusinessUnitID uuid.UUID  `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID  `bun:"type:uuid,notnull" json:"organizationId"`

	Manager      *User         `bun:"rel:belongs-to,join:manager_id=id" json:"manager"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func (f FleetCode) Validate() error {
	return validation.ValidateStruct(
		&f,
		validation.Field(&f.Code, validation.Required),
		validation.Field(&f.Color, is.HexColor),
		validation.Field(&f.BusinessUnitID, validation.Required),
		validation.Field(&f.OrganizationID, validation.Required),
	)
}

func (f *FleetCode) BeforeUpdate(_ context.Context) error {
	f.Version++

	return nil
}

func (f *FleetCode) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := f.Version

	if err := f.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(f).
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
			Message: fmt.Sprintf("Version mismatch. The FleetCode (ID: %s) has been updated by another user. Please refresh and try again.", f.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*FleetCode)(nil)

func (f *FleetCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		f.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		f.UpdatedAt = time.Now()
	}
	return nil
}
