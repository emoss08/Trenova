package models

import (
	"context"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type CommodityPermission string

const (
	// PermissionCommodityView is the permission to view commodity details
	PermissionCommodityView = CommodityPermission("commodity.view")

	// PermissionCommodityEdit is the permission to edit commodity details
	PermissionCommodityEdit = CommodityPermission("commodity.edit")

	// PermissionCommodityAdd is the permission to add a new commodity
	PermissionCommodityAdd = CommodityPermission("commodity.add")

	// PermissionCommodityDelete is the permission to delete a commodity
	PermissionCommodityDelete = CommodityPermission("commodity.delete")
)

// String returns the string representation of the CommodityPermission
func (p CommodityPermission) String() string {
	return string(p)
}

type Commodity struct {
	bun.BaseModel       `bun:"table:commodities,alias:com" json:"-"`
	CreatedAt           time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt           time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID                  uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name                string    `bun:"type:VARCHAR(100),notnull" json:"name" queryField:"true"`
	Status              string    `bun:"type:status_enum,notnull,default:'Active'" json:"status"`
	IsHazmat            bool      `bun:"type:boolean,notnull,default:false" json:"isHazmat"`
	UnitOfMeasure       string    `bun:"type:VARCHAR(50),notnull" json:"unitOfMeasure"`
	MinTemp             int       `bun:"type:integer,notnull" json:"minTemp"`
	MaxTemp             int       `bun:"type:integer,notnull" json:"maxTemp"`
	HazardousMaterialID uuid.UUID `bun:"type:uuid" json:"hazardousMaterialId"`
	BusinessUnitID      uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID      uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit      *BusinessUnit      `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization      *Organization      `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	HazardousMaterial *HazardousMaterial `bun:"rel:belongs-to,join:hazardous_material_id=id" json:"-"`
}

func (c Commodity) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Name, validation.Required, validation.Length(1, 50)),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*Commodity)(nil)

func (c *Commodity) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
