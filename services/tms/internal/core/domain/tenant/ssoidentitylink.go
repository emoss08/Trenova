package tenant

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*SSOIdentityLink)(nil)

type SSOIdentityLink struct {
	bun.BaseModel `bun:"table:sso_identity_links,alias:ssoil" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	SSOConfigID    pulid.ID `json:"ssoConfigId"    bun:"sso_config_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Issuer         string   `json:"issuer"         bun:"issuer,type:VARCHAR(500),notnull"`
	Subject        string   `json:"subject"        bun:"subject,type:VARCHAR(255),notnull"`
	EmailAtLink    string   `json:"emailAtLink"    bun:"email_at_link,type:VARCHAR(320),notnull"`
	LastLoginAt    int64    `json:"lastLoginAt"    bun:"last_login_at,notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,notnull"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,notnull"`
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
