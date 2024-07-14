package models

import (
	"context"
	"fmt"
	"time"

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
	ShipmentID     uuid.UUID `bun:"type:uuid" json:"shipmentId"`

	Shipment     *Shipment     `bun:"rel:belongs-to,join:shipment_id=id" json:"-"`
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
			Message: fmt.Sprintf("Version mismatch. The DivisionCode (ID: %s) has been updated by another user. Please refresh and try again.", c.ID),
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
