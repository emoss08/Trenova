package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ChargeTypePermission string

const (
	// PermissionChargeTypeView is the permission to view charge type details
	PermissionChargeTypeView = ChargeTypePermission("chargetype.view")

	// PermissionChargeTypeEdit is the permission to edit charge type details
	PermissionChargeTypeEdit = ChargeTypePermission("chargetype.edit")

	// PermissionChargeTypeAdd is the permission to add a new charge type
	PermissionChargeTypeAdd = ChargeTypePermission("chargetype.add")

	// PermissionChargeTypeDelete is the permission to delete an charge type
	PermissionChargeTypeDelete = ChargeTypePermission("chargetype.delete")
)

// String returns the string representation of the ChargeTypePermission
func (p ChargeTypePermission) String() string {
	return string(p)
}

type ChargeType struct {
	bun.BaseModel  `bun:"table:charge_types,alias:ct" json:"-"`
	CreatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status `bun:"status,type:status" json:"status"`
	Name           string          `bun:"type:VARCHAR(50),notnull" json:"name" queryField:"true"`
	Description    string          `bun:"type:TEXT" json:"description"`
	BusinessUnitID uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c ChargeType) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Name, validation.Required),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*ChargeType)(nil)

func (c *ChargeType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
