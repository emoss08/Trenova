package models

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type HazardousMaterialPermission string

const (
	// PermissionHazardousMaterialView is the permission to view hazardous material details
	PermissionHazardousMaterialView = HazardousMaterialPermission("hazardousmaterial.view")

	// PermissionHazardousMaterialEdit is the permission to edit hazardous material details
	PermissionHazardousMaterialEdit = HazardousMaterialPermission("hazardousmaterial.edit")

	// PermissionHazardousMaterialAdd is the permission to add a new hazardous material
	PermissionHazardousMaterialAdd = HazardousMaterialPermission("hazardousmaterial.add")

	// PermissionHazardousMaterialDelete is the permission to delete a hazardous material
	PermissionHazardousMaterialDelete = HazardousMaterialPermission("hazardousmaterial.delete")
)

// String returns the string representation of the HazardousMaterialPermission
func (p HazardousMaterialPermission) String() string {
	return string(p)
}

type HazardousMaterial struct {
	bun.BaseModel `bun:"table:hazardous_materials,alias:hm" json:"-"`

	ID                 uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name               string    `bun:"type:VARCHAR(50),notnull" json:"name" queryField:"true"`
	Status             string    `bun:"type:status_enum,notnull,default:'Active'" json:"status"`
	HazardClass        string    `bun:"type:VARCHAR(16),notnull:default:'HazardClass1And1'" json:"hazardClass"`
	ERGNumber          string    `bun:"type:VARCHAR" json:"ergNumber"`
	Description        string    `bun:"type:TEXT" json:"description"`
	PackingGroup       string    `bun:"type:VARCHAR,default:'PackingGroup1'" json:"packingGroup"`
	ProperShippingName string    `bun:"type:TEXT" json:"properShippingName"`
	Version            int64     `bun:"type:BIGINT" json:"version"`
	CreatedAt          time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt          time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func (c HazardousMaterial) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Name, validation.Required, validation.Length(1, 50)),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

func (c *HazardousMaterial) BeforeUpdate(_ context.Context) error {
	c.Version++

	return nil
}

func (c *HazardousMaterial) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
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
			Message: fmt.Sprintf("Version mismatch. The FleetCode (ID: %s) has been updated by another user. Please refresh and try again.", c.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*HazardousMaterial)(nil)

func (c *HazardousMaterial) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
