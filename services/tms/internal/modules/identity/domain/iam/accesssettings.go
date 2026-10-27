package iam

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type AccessSettings struct {
	keel.Table `keel:"organization_access_settings"`

	OrganizationID        pulid.ID `json:"organizationId"        keel:"organization_id,pk,tenant"`
	BusinessUnitID        pulid.ID `json:"businessUnitId"        keel:"business_unit_id,pk,tenant"`
	RequireRoleActivation bool     `json:"requireRoleActivation" keel:"require_role_activation"`
	UpdatedByID           pulid.ID `json:"updatedById"           keel:"updated_by_id,nullzero"`
	Version               int64    `json:"version"               keel:"version,version"`
	CreatedAt             int64    `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt             int64    `json:"updatedAt"             keel:"updated_at,autoupdate"`
}

func DefaultAccessSettings(orgID, buID pulid.ID) *AccessSettings {
	return &AccessSettings{OrganizationID: orgID, BusinessUnitID: buID}
}
