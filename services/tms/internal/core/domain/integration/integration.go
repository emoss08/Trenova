package integration

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type Integration struct {
	bun.BaseModel `bun:"table:integrations,alias:integ" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Type           Type           `json:"type"           bun:"type,type:integration_type,notnull"`
	Name           string         `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Description    string         `json:"description"    bun:"description,type:TEXT,nullzero"`
	Enabled        bool           `json:"enabled"        bun:"enabled,type:BOOLEAN,notnull,default:false"`
	BuiltBy        string         `json:"builtBy"        bun:"built_by,type:VARCHAR(100),nullzero"`
	Category       Category       `json:"category"       bun:"category,type:integration_category,notnull"`
	Configuration  map[string]any `json:"configuration"  bun:"configuration,type:jsonb,nullzero"`
	DocsURL        string         `json:"docsUrl"        bun:"docs_url,type:TEXT,nullzero"`
	Featured       bool           `json:"featured"       bun:"featured,type:BOOLEAN,notnull,default:false"`
	LogoURL        string         `json:"logoUrl"        bun:"logo_url,type:TEXT,nullzero"`
	WebsiteURL     string         `json:"websiteUrl"     bun:"website_url,type:TEXT,nullzero"`
	EnabledByID    pulid.ID       `json:"enabledById"    bun:"enabled_by_id,type:VARCHAR(100),nullzero"`
	Version        int64          `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	EnabledBy    *tenant.User         `json:"enabledBy,omitempty"    bun:"rel:belongs-to,join:enabled_by_id=id"`
}

func (i *Integration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("intg_")
		}
		i.CreatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}
