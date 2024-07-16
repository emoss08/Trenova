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

type DivisionCode struct {
	bun.BaseModel `bun:"table:division_codes,alias:dc" json:"-"`

	ID          uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status      property.Status `bun:"status,type:status" json:"status"`
	Code        string          `bun:"type:VARCHAR(4),notnull" json:"code" queryField:"true"`
	Description string          `bun:"type:TEXT" json:"description"`
	Color       string          `bun:"type:VARCHAR(10)" json:"color"`
	Version     int64           `bun:"type:BIGINT" json:"version"`
	CreatedAt   time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt   time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	CashAccountID    *uuid.UUID `bun:"type:uuid,nullzero" json:"cashAccountId"`
	ApAccountID      *uuid.UUID `bun:"type:uuid,nullzero" json:"apAccountId"`
	ExpenseAccountID *uuid.UUID `bun:"type:uuid,nullzero" json:"expenseAccountId"`
	BusinessUnitID   uuid.UUID  `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID   uuid.UUID  `bun:"type:uuid,notnull" json:"organizationId"`

	CashAccount    *GeneralLedgerAccount `bun:"rel:belongs-to,join:cash_account_id=id" json:"-"`
	ApAccount      *GeneralLedgerAccount `bun:"rel:belongs-to,join:ap_account_id=id" json:"-"`
	ExpenseAccount *GeneralLedgerAccount `bun:"rel:belongs-to,join:expense_account_id=id" json:"-"`
	BusinessUnit   *BusinessUnit         `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization   *Organization         `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
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
