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

type ShipmentTypePermission string

const (
	// PermissionShipmentTypeView is the permission to view shipment type details
	PermissionShipmentTypeView = ShipmentTypePermission("shipmenttype.view")

	// PermissionShipmentTypeEdit is the permission to edit shipment type details
	PermissionShipmentTypeEdit = ShipmentTypePermission("shipmenttype.edit")

	// PermissionShipmentTypeAdd is the permission to add a necw shipment type
	PermissionShipmentTypeAdd = ShipmentTypePermission("shipmenttype.add")

	// PermissionShipmentTypeDelete is the permission to delete an shipment type
	PermissionShipmentTypeDelete = ShipmentTypePermission("shipmenttype.delete")
)

// String returns the string representation of the ShipmentTypePermission
func (p ShipmentTypePermission) String() string {
	return string(p)
}

type ShipmentType struct {
	bun.BaseModel  `bun:"table:shipment_types,alias:st" json:"-"`
	CreatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status `bun:"status,type:status" json:"status"`
	Code           string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Color          string          `bun:"type:VARCHAR(10)" json:"color"`
	Description    string          `bun:"type:TEXT" json:"description"`
	BusinessUnitID uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c ShipmentType) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(1, 10).Error("Code must be between 1 and 10 characters")),
		validation.Field(&c.Color, is.HexColor),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*ShipmentType)(nil)

func (c *ShipmentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
