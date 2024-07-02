package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type EquipmentManufacturerPermission string

const (
	// PermissionEquipmentManufacturerView is the permission to view equipment manufacturer details
	PermissionEquipmentManufacturerView = EquipmentManufacturerPermission("equipmentmanufacturer.view")

	// PermissionEquipmentManufacturerEdit is the permission to edit equipment manufacturer details
	PermissionEquipmentManufacturerEdit = EquipmentManufacturerPermission("equipmentmanufacturer.edit")

	// PermissionEquipmentManufacturerAdd is the permission to add a new equipment manufacturer
	PermissionEquipmentManufacturerAdd = EquipmentManufacturerPermission("equipmentmanufacturer.add")

	// PermissionEquipmentManufacturerDelete is the permission to delete an equipment manufacturer
	PermissionEquipmentManufacturerDelete = EquipmentManufacturerPermission("equipmentmanufacturer.delete")
)

// String returns the string representation of the EquipmentManufacturerPermission
func (p EquipmentManufacturerPermission) String() string {
	return string(p)
}

type EquipmentManufacturer struct {
	bun.BaseModel  `bun:"table:equipment_manufacturers,alias:em" json:"-"`
	CreatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status `bun:"status,type:status" json:"status"`
	Name           string          `bun:"type:VARCHAR,notnull" json:"name" queryField:"true"`
	Description    string          `bun:"type:TEXT" json:"description"`
	BusinessUnitID uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func (c EquipmentManufacturer) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Name, validation.Required, validation.Length(1, 10)),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*EquipmentManufacturer)(nil)

func (c *EquipmentManufacturer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
