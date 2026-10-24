package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*SSOIdentityLink)(nil)

type SSOIdentityLink struct {
	bun.BaseModel `bun:"table:sso_identity_links,alias:ssoil" json:"-"`
	keel.Table    `keel:"sso_identity_links"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:ssoil_"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	SSOConfigID    pulid.ID `json:"ssoConfigId"    bun:"sso_config_id,type:VARCHAR(100),notnull" keel:"sso_config_id"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull" keel:"user_id"`
	Issuer         string   `json:"issuer"         bun:"issuer,type:VARCHAR(500),notnull" keel:"issuer"`
	Subject        string   `json:"subject"        bun:"subject,type:VARCHAR(255),notnull" keel:"subject"`
	EmailAtLink    string   `json:"emailAtLink"    bun:"email_at_link,type:VARCHAR(320),notnull" keel:"email_at_link"`
	LastLoginAt    int64    `json:"lastLoginAt"    bun:"last_login_at,notnull" keel:"last_login_at"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,notnull" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,notnull" keel:"updated_at,autoupdate"`
}

func (l *SSOIdentityLink) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("ssoil_")
		}
		l.CreatedAt = now
		l.UpdatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}

	return nil
}
