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

type RevenueCodePermission string

const (
	// PermissionRevenueCodeView is the permission to view revenue code details
	PermissionRevenueCodeView = RevenueCodePermission("revenuecode.view")

	// PermissionRevenueCodeEdit is the permission to edit revenue code details
	PermissionRevenueCodeEdit = RevenueCodePermission("revenuecode.edit")

	// PermissionRevenueCodeAdd is the permission to add a new revenue code
	PermissionRevenueCodeAdd = RevenueCodePermission("revenuecode.add")

	// PermissionRevenueCodeDelete is the permission to delete a revenue code
	PermissionRevenueCodeDelete = RevenueCodePermission("revenuecode.delete")
)

// String returns the string representation of the RevenueCodePermission
func (p RevenueCodePermission) String() string {
	return string(p)
}

type RevenueCode struct {
	bun.BaseModel    `bun:"table:revenue_codes,alias:rc" json:"-"`
	CreatedAt        time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt        time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID               uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status           property.Status `bun:"status,type:status" json:"status"`
	Code             string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description      string          `bun:"type:TEXT" json:"description"`
	Color            string          `bun:"type:VARCHAR(10)" json:"color"`
	RevenueAccountID *uuid.UUID      `bun:"type:uuid,nullzero" json:"revenueAccountId"`
	ExpenseAccountID *uuid.UUID      `bun:"type:uuid,nullzero" json:"expenseAccountId"`
	BusinessUnitID   uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID   uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	RevenueAccount *GeneralLedgerAccount `bun:"rel:belongs-to,join:revenue_account_id=id" json:"revenueAccount"`
	ExpenseAccount *GeneralLedgerAccount `bun:"rel:belongs-to,join:expense_account_id=id" json:"expenseAccount"`
	BusinessUnit   *BusinessUnit         `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization   *Organization         `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c RevenueCode) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(1, 10).Error("Code must be between 1 and 10 characters")),
		validation.Field(&c.Color, is.HexColor),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*RevenueCode)(nil)

func (c *RevenueCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
