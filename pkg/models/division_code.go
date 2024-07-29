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

	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type DivisionCode struct {
	ID          uuid.UUID       `xorm:"pk uuid default uuid_generate_v4()" json:"id"`
	Status      property.Status `xorm:"status VARCHAR(255)" json:"status"`
	Code        string          `xorm:"VARCHAR(4) notnull" json:"code" queryField:"true"`
	Description string          `xorm:"TEXT" json:"description"`
	Color       string          `xorm:"VARCHAR(10)" json:"color"`
	Version     int64           `xorm:"BIGINT" json:"version"`
	CreatedAt   time.Time       `xorm:"created notnull default current_timestamp" json:"createdAt"`
	UpdatedAt   time.Time       `xorm:"updated notnull default current_timestamp" json:"updatedAt"`

	CashAccountID    *uuid.UUID `xorm:"uuid null" json:"cashAccountId"`
	ApAccountID      *uuid.UUID `xorm:"uuid null" json:"apAccountId"`
	ExpenseAccountID *uuid.UUID `xorm:"uuid null" json:"expenseAccountId"`
	BusinessUnitID   uuid.UUID  `xorm:"uuid notnull" json:"businessUnitId"`
	OrganizationID   uuid.UUID  `xorm:"uuid notnull" json:"organizationId"`

	CashAccount    *GeneralLedgerAccount `xorm:"-" json:"-"`
	ApAccount      *GeneralLedgerAccount `xorm:"-" json:"-"`
	ExpenseAccount *GeneralLedgerAccount `xorm:"-" json:"-"`
	BusinessUnit   *BusinessUnit         `xorm:"-" json:"-"`
	Organization   *Organization         `xorm:"-" json:"-"`
}

func (d DivisionCode) Validate() error {
	return validation.ValidateStruct(
		&d,
		validation.Field(&d.Code, validation.Required, validation.Length(4, 4).Error("Code must be 4 characters")),
		validation.Field(&d.Color, is.HexColor),
		validation.Field(&d.BusinessUnitID, validation.Required),
		validation.Field(&d.OrganizationID, validation.Required),
	)
}

func (d *DivisionCode) BeforeUpdate(_ context.Context) error {
	d.Version++

	return nil
}

func (d *DivisionCode) Insert(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	if err := d.Validate(); err != nil {
		return err
	}

	if _, err := tx.NewInsert().Model(d).Returning("*").Exec(ctx); err != nil {
		return err
	}

	auditService.LogAction(
		constants.TableDivisionCode,
		d.ID.String(),
		property.AuditLogActionCreate,
		user,
		d.OrganizationID,
		d.BusinessUnitID,
		audit.WithDiff(nil, d),
	)

	return nil
}

func (d *DivisionCode) UpdateOne(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	original := new(DivisionCode)
	if err := tx.NewSelect().Model(original).Where("id = ?", d.ID).Scan(ctx); err != nil {
		return validator.BusinessLogicError{Message: err.Error()}
	}

	if err := d.Validate(); err != nil {
		return err
	}

	if err := d.OptimisticUpdate(ctx, tx); err != nil {
		return err
	}

	auditService.LogAction(
		constants.TableDivisionCode,
		d.ID.String(),
		property.AuditLogActionUpdate,
		user,
		d.OrganizationID,
		d.BusinessUnitID,
		audit.WithDiff(original, d),
	)

	return nil
}

func (d *DivisionCode) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := d.Version

	if err := d.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(d).
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
			Message: fmt.Sprintf("Version mismatch. The DivisionCode (ID: %s) has been updated by another user. Please refresh and try again.", d.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*DivisionCode)(nil)

func (d *DivisionCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		d.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		d.UpdatedAt = time.Now()
	}
	return nil
}
