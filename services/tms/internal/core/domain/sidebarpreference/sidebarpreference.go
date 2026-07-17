package sidebarpreference

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*SidebarPreference)(nil)
	_ validationframework.TenantedEntity = (*SidebarPreference)(nil)
)

type SidebarPreference struct {
	bun.BaseModel `bun:"table:sidebar_preferences,alias:sbp" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID  `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Preferences    *Document `json:"preferences"    bun:"preferences,type:JSONB,notnull"`
	Version        int64     `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64     `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	User         *tenant.User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
}

func (sp *SidebarPreference) Validate(multiErr *errortypes.MultiError) {
	if sp.Preferences == nil {
		multiErr.Add("preferences", errortypes.ErrRequired, "Preferences document is required")
		return
	}

	sp.Preferences.Validate(multiErr)
}

func (sp *SidebarPreference) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if sp.ID.IsNil() {
			sp.ID = pulid.MustNew("sbp_")
		}
		sp.CreatedAt = now
	case *bun.UpdateQuery:
		sp.UpdatedAt = now
	}

	return nil
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
