package sidebarpreference

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*SidebarPreference)(nil)
)

type SidebarPreference struct {
	keel.Table `keel:"sidebar_preferences"`

	ID             pulid.ID  `json:"id" keel:"id,pk,autoid:sbp_"`
	OrganizationID pulid.ID  `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID  `json:"userId" keel:"user_id"`
	Preferences    *Document `json:"preferences" keel:"preferences,jsonb"`
	Version        int64     `json:"version" keel:"version,version"`
	CreatedAt      int64     `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64     `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	User         *tenant.User         `json:"user,omitempty" keel:"belongs:user_id"`
}

func (sp *SidebarPreference) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(sp,
		validation.Field(&sp.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&sp.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&sp.UserID, validation.Required.Error("User is required")),
		validation.Field(&sp.Preferences,
			validation.Required.Error("Preferences document is required"),
		),
	))

	if sp.Preferences != nil {
		sp.Preferences.Validate(multiErr)
	}
}

func (sp *SidebarPreference) GetID() pulid.ID {
	return sp.ID
}

func (sp *SidebarPreference) GetOrganizationID() pulid.ID {
	return sp.OrganizationID
}

func (sp *SidebarPreference) GetBusinessUnitID() pulid.ID {
	return sp.BusinessUnitID
}

func (sp *SidebarPreference) GetTableName() string {
	return "sidebar_preferences"
}
