package models

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type EquipmentType struct {
	bun.BaseModel `bun:"table:equipment_types,alias:et" json:"-"`

	ID              uuid.UUID           `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status          property.Status     `bun:"status,type:status,default:'Active'" json:"status"`
	EquipmentClass  string              `bun:"type:VARCHAR(12),notnull,default:'Undefined'" json:"equipmentClass"`
	Code            string              `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description     string              `bun:"type:TEXT" json:"description"`
	CostPerMile     decimal.NullDecimal `bun:"type:NUMERIC(10,2),nullzero" json:"costPerMile"`
	FixedCost       decimal.NullDecimal `bun:"type:NUMERIC(10,2),nullzero" json:"fixedCost"`
	VariableCost    decimal.NullDecimal `bun:"type:NUMERIC(10,2),nullzero" json:"variableCost"`
	Height          decimal.NullDecimal `bun:"type:NUMERIC(10,2),nullzero" json:"height"`
	Length          decimal.NullDecimal `bun:"type:NUMERIC(10,2),nullzero" json:"length"`
	Width           decimal.NullDecimal `bun:"type:NUMERIC(10,2),nullzero" json:"width"`
	Weight          decimal.NullDecimal `bun:"type:NUMERIC(10,2),nullzero" json:"weight"`
	ExemptFromTolls bool                `bun:"type:BOOLEAN,notnull" json:"exemptFromTolls"`
	Color           string              `bun:"type:VARCHAR(10)" json:"color"`
	Version         int64               `bun:"type:BIGINT" json:"version"`
	CreatedAt       time.Time           `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt       time.Time           `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c EquipmentType) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(1, 10).Error("Code must be atleast 10 characters")),
		validation.Field(&c.Color, is.HexColor),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

func (c *EquipmentType) BeforeUpdate(_ context.Context) error {
	c.Version++

	return nil
}

func (c *EquipmentType) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
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
			Message: fmt.Sprintf("Version mismatch. The EquipmentType (ID: %s) has been updated by another user. Please refresh and try again.", c.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*EquipmentType)(nil)

func (c *EquipmentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
