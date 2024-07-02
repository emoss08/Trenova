package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ServiceTypePermission string

const (
	// PermissionServiceTypeView is the permission to view service type details
	PermissionServiceTypeView = ServiceTypePermission("servicetype.view")

	// PermissionServiceTypeEdit is the permission to edit service type details
	PermissionServiceTypeEdit = ServiceTypePermission("servicetype.edit")

	// PermissionServiceTypeAdd is the permission to add a necw service type
	PermissionServiceTypeAdd = ServiceTypePermission("servicetype.add")

	// PermissionServiceTypeDelete is the permission to delete an service type
	PermissionServiceTypeDelete = ServiceTypePermission("servicetype.delete")
)

// String returns the string representation of the ServiceTypePermission
func (p ServiceTypePermission) String() string {
	return string(p)
}

type ServiceType struct {
	bun.BaseModel  `bun:"table:service_types,alias:st" json:"-"`
	CreatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status `bun:"status,type:status" json:"status"`
	Code           string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description    string          `bun:"type:TEXT" json:"description"`
	BusinessUnitID uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c ServiceType) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(4, 4).Error("Code must be 4 characters")),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*ServiceType)(nil)

func (c *ServiceType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
