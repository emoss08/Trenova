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
	"github.com/uptrace/bun"
)

type DelayCode struct {
	bun.BaseModel `bun:"table:delay_codes,alias:dc" json:"-"`

	ID               uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status           property.Status `bun:"status,type:status" json:"status"`
	Code             string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description      string          `bun:"type:TEXT,notnull" json:"description"`
	FCarrierOrDriver bool            `bun:"type:BOOLEAN,default:false" json:"fCarrierOrDriver"`
	Color            string          `bun:"type:VARCHAR(10)" json:"color"`
	Version          int64           `bun:"type:BIGINT" json:"version"`
	CreatedAt        time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt        time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func (d DelayCode) Validate() error {
	return validation.ValidateStruct(
		&d,
		validation.Field(&d.Code, validation.Required),
		validation.Field(&d.Color, is.HexColor),
		validation.Field(&d.BusinessUnitID, validation.Required),
		validation.Field(&d.OrganizationID, validation.Required),
	)
}

func (d *DelayCode) BeforeUpdate(_ context.Context) error {
	d.Version++

	return nil
}

func (d *DelayCode) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
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
			Message: fmt.Sprintf("Version mismatch. The DelayCode (ID: %s) has been updated by another user. Please refresh and try again.", d.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*DelayCode)(nil)

func (d *DelayCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		d.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		d.UpdatedAt = time.Now()
	}
	return nil
}
