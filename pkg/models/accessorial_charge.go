package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AccessorialchargePermission string

const (
	// PermissionAccessorialChargeView is the permission to view accessorial charge details
	PermissionAccessorialChargeView = AccessorialchargePermission("accessorialcharge.view")

	// PermissionAccessorialChargeEdit is the permission to edit accessorial charge details
	PermissionAccessorialChargeEdit = AccessorialchargePermission("accessorialcharge.edit")

	// PermissionAccessorialChargeAdd is the permission to add a new accessorial charge
	PermissionAccessorialChargeAdd = AccessorialchargePermission("accessorialcharge.add")

	// PermissionAccessorialChargeDelete is the permission to delete an accessorial charge
	PermissionAccessorialChargeDelete = AccessorialchargePermission("accessorialcharge.delete")
)

// String returns the string representation of the AccessorialchargePermission
func (p AccessorialchargePermission) String() string {
	return string(p)
}

type AccessorialCharge struct {
	bun.BaseModel  `bun:"table:accessorial_charges,alias:ac" json:"-"`
	CreatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status `bun:"status,type:status" json:"status"`
	Code           string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description    string          `bun:"type:TEXT" json:"description"`
	IsDetention    bool            `bun:"is_detention,type:BOOLEAN,default:false" json:"isDetention"`
	Method         string          `bun:"method,type:fuel_method_enum,notnull" json:"method"`
	Amount         string          `bun:"amount,type:numeric(19,2),notnull,default:0" json:"amount"`
	BusinessUnitID uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c AccessorialCharge) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(1, 10).Error("Code must be between 1 and 10 characters")),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
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
