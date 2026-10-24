package tenant

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type SSOIdentityLink struct {
	keel.Table `keel:"sso_identity_links"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:ssoil_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	SSOConfigID    pulid.ID `json:"ssoConfigId" keel:"sso_config_id"`
	UserID         pulid.ID `json:"userId" keel:"user_id"`
	Issuer         string   `json:"issuer" keel:"issuer"`
	Subject        string   `json:"subject" keel:"subject"`
	EmailAtLink    string   `json:"emailAtLink" keel:"email_at_link"`
	LastLoginAt    int64    `json:"lastLoginAt" keel:"last_login_at"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
}
