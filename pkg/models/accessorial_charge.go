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
	"github.com/shopspring/decimal"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AccessorialCharge struct {
	bun.BaseModel `bun:"table:accessorial_charges,alias:ac" json:"-"`

	ID          uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status      property.Status `bun:"status,type:status_enum,default:'Active'" json:"status"`
	Code        string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description string          `bun:"type:TEXT" json:"description"`
	IsDetention bool            `bun:"is_detention,type:BOOLEAN,default:false" json:"isDetention"`
	Method      string          `bun:"method,type:fuel_method_enum,notnull" json:"method"`
	Amount      decimal.Decimal `bun:"amount,type:numeric(19,2),notnull,default:0" json:"amount"`
	Version     int64           `bun:"type:BIGINT" json:"version"`
	CreatedAt   time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt   time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c AccessorialCharge) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(1, 10).Error("Code must be between 1 and 10 characters")),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
		validation.Field(&c.Description, validation.Required),
		validation.Field(&c.Method, validation.Required),
	)
}

func (c *AccessorialCharge) Insert(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	if err := c.Validate(); err != nil {
		return err
	}

	if _, err := tx.NewInsert().Model(c).Returning("*").Exec(ctx); err != nil {
		return err
	}

	auditService.LogAction(
		constants.TableAccessorialCharge,
		c.ID.String(),
		property.AuditLogActionCreate,
		user,
		c.OrganizationID,
		c.BusinessUnitID,
		audit.WithDiff(nil, c),
	)

	return nil
}

func (c *AccessorialCharge) UpdateOne(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	original := new(AccessorialCharge)
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
		constants.TableAccessorialCharge,
		c.ID.String(),
		property.AuditLogActionUpdate,
		user,
		c.OrganizationID,
		c.BusinessUnitID,
		audit.WithDiff(original, c),
	)

	return nil
}

func (c *AccessorialCharge) BeforeUpdate(_ context.Context) error {
	c.Version++

	return nil
}

func (c *AccessorialCharge) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
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
			Message: fmt.Sprintf("Version mismatch. The AccessorialCharge (ID: %s) has been updated by another user. Please refresh and try again.", c.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*AccessorialCharge)(nil)

func (c *AccessorialCharge) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
