package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// OrganizationPermission is a type for organization permissions
type OrganizationPermission string

const (
	// PermissionOrganizationView is the permission to view organization details
	PermissionOrganizationView = OrganizationPermission("organization.view")

	// PermissionOrganizationEdit is the permission to edit organization details
	PermissionOrganizationEdit = OrganizationPermission("organization.edit")

	// PermissionOrganizationAdd is the permission to add a new organization
	PermissionOrganizationAdd = OrganizationPermission("organization.add")

	// PermissionOrganizationDelete is the permission to delete an organization
	PermissionOrganizationDelete = OrganizationPermission("organization.delete")

	// PermissionOrganizationChangeLogo is the permission to change the logo of the organization
	PermissionOrganizationChangeLogo = OrganizationPermission("organization.change_logo")
)

// String returns the string representation of the OrganizationPermission
func (p OrganizationPermission) String() string {
	return string(p)
}

type Organization struct {
	bun.BaseModel  `bun:"organizations" alias:"o" json:"-"`
	CreatedAt      time.Time                 `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time                 `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID                 `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name           string                    `bun:"name,notnull" json:"name" queryField:"true"`
	ScacCode       string                    `bun:",notnull" json:"scacCode"`
	DOTNumber      string                    `bun:"dot_number" json:"dotNumber"`
	LogoURL        string                    `bun:"logo_url" json:"logoUrl"`
	OrgType        property.OrganizationType `bun:"org_type,notnull,default:'Asset'" json:"orgType"`
	AddressLine1   string                    `bun:"address_line_1" json:"addressLine1"`
	AddressLine2   string                    `bun:"address_line_2" json:"addressLine2"`
	City           string                    `bun:"city" json:"city"`
	PostalCode     string                    `bun:"postal_code" json:"postalCode"`
	Timezone       string                    `bun:"timezone,notnull,default:'America/New_York'" json:"timezone"`
	BusinessUnitID uuid.UUID                 `bun:"type:uuid" json:"businessUnitId"`
	StateID        uuid.UUID                 `bun:"type:uuid,notnull" json:"stateId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	State        *UsState      `bun:"rel:belongs-to,join:state_id=id" json:"state"`
}

func (o Organization) Validate() error {
	return validation.ValidateStruct(
		&o,
		validation.Field(&o.BusinessUnitID, validation.Required),
		validation.Field(&o.Name, validation.Required),
		validation.Field(&o.ScacCode, validation.Required, validation.Length(4, 4).Error("SCAC code must be 4 characters")),
		// validation.Field(&o.DOTNumber, validation.Required, validation.Length(12, 12).Error("DOT number must be 12 characters")),
		validation.Field(&o.OrgType, validation.Required),
		validation.Field(&o.AddressLine1, validation.Length(0, 150).Error("Address line 1 must be less than 150 characters")),
		validation.Field(&o.City, validation.Required),
		validation.Field(&o.StateID, validation.Required),
		validation.Field(&o.Timezone, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*Organization)(nil)

func (o *Organization) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		o.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		o.UpdatedAt = time.Now()
	}
	return nil
}
