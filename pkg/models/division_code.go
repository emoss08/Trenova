package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type DivisionCodePermission string

const (
	// PermissionDivisionCodeView is the permission to view division code details
	PermissionDivisionCodeView = DivisionCodePermission("divisioncode.view")

	// PermissionDivisionCodeEdit is the permission to edit division code details
	PermissionDivisionCodeEdit = DivisionCodePermission("divisioncode.edit")

	// PermissionDivisionCodeAdd is the permission to add a necw division code
	PermissionDivisionCodeAdd = DivisionCodePermission("divisioncode.add")

	// PermissionDivisionCodeDelete is the permission to delete an division code
	PermissionDivisionCodeDelete = DivisionCodePermission("divisioncode.delete")
)

// String returns the string representation of the DivisionCodePermission
func (p DivisionCodePermission) String() string {
	return string(p)
}

type DivisionCode struct {
	bun.BaseModel    `bun:"table:division_codes,alias:dc" json:"-"`
	CreatedAt        time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt        time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID               uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status           property.Status `bun:"status,type:status" json:"status"`
	Code             string          `bun:"type:VARCHAR(4),notnull" json:"code" queryField:"true"`
	Description      string          `bun:"type:TEXT" json:"description"`
	Color            string          `bun:"type:VARCHAR(10)" json:"color"`
	CashAccountID    *uuid.UUID      `bun:"type:uuid,nullzero" json:"cashAccountId"`
	ApAccountID      *uuid.UUID      `bun:"type:uuid,nullzero" json:"apAccountId"`
	ExpenseAccountID *uuid.UUID      `bun:"type:uuid,nullzero" json:"expenseAccountId"`
	BusinessUnitID   uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID   uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	CashAccount    *GeneralLedgerAccount `bun:"rel:belongs-to,join:cash_account_id=id" json:"-"`
	ApAccount      *GeneralLedgerAccount `bun:"rel:belongs-to,join:ap_account_id=id" json:"-"`
	ExpenseAccount *GeneralLedgerAccount `bun:"rel:belongs-to,join:expense_account_id=id" json:"-"`
	BusinessUnit   *BusinessUnit         `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization   *Organization         `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c DivisionCode) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(4, 4).Error("Code must be 4 characters")),
		validation.Field(&c.Color, is.HexColor),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*DivisionCode)(nil)

func (c *DivisionCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
