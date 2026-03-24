package pagefavorite

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
	_ bun.BeforeAppendModelHook          = (*PageFavorite)(nil)
	_ validationframework.TenantedEntity = (*PageFavorite)(nil)
)

type PageFavorite struct {
	bun.BaseModel `bun:"table:page_favorites,alias:pf" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	PageURL        string   `json:"pageUrl"        bun:"page_url,type:VARCHAR(500),notnull"`
	PageTitle      string   `json:"pageTitle"      bun:"page_title,type:VARCHAR(255),notnull"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	User         *tenant.User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
}

func (pf *PageFavorite) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if pf.ID.IsNil() {
			pf.ID = pulid.MustNew("pf_")
		}
		pf.CreatedAt = now
	case *bun.UpdateQuery:
		pf.UpdatedAt = now
	}

	return nil
}

func (pf *PageFavorite) GetID() pulid.ID {
	return pf.ID
}

func (pf *PageFavorite) GetOrganizationID() pulid.ID {
	return pf.OrganizationID
}

func (pf *PageFavorite) GetBusinessUnitID() pulid.ID {
	return pf.BusinessUnitID
}

func (pf *PageFavorite) GetTableName() string {
	return "page_favorites"
}

func (pf *PageFavorite) Validate(_ *errortypes.MultiError) {
}
